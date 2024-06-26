// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Verifier is a struct that can be used to verify whether the shoot-rsyslog-relp extension is working as expected.
type Verifier struct {
	log                        logr.Logger
	client                     kubernetes.Interface
	rsyslogRelpEchoServerPodIf clientcorev1.PodInterface
	rsyslogEchoServerPodName   string
	providerType               string
	nodeName                   string
	projectName                string
	shootName                  string
	shootUID                   string
	rootPodExecutor            framework.RootPodExecutor
}

type logEntry struct {
	program           string
	severity          string
	message           string
	shouldBeForwarded bool
}

// NewVerifier creates a new Verifier.
func NewVerifier(log logr.Logger,
	client kubernetes.Interface,
	echoServerPodIf clientcorev1.PodInterface,
	echoServerPodName, providerType, projectName, shootName, shootUID string) *Verifier {
	return &Verifier{
		log:                        log,
		client:                     client,
		rsyslogRelpEchoServerPodIf: echoServerPodIf,
		rsyslogEchoServerPodName:   echoServerPodName,
		providerType:               providerType,
		projectName:                projectName,
		shootName:                  shootName,
		shootUID:                   shootUID,
	}
}

// SetEchoServerPodIfAndName sets the clientcorev1.PodInterface and the name of the rsyslog-relp-echo-server pod to the Verifier.
func (v *Verifier) SetEchoServerPodIfAndName(echoServerPodIf clientcorev1.PodInterface, echoServerPodName string) {
	v.rsyslogRelpEchoServerPodIf = echoServerPodIf
	v.rsyslogEchoServerPodName = echoServerPodName
}

// VerifyExtensionForNode verifies whether the shoot-rsyslog-relp extension has properly configured
// the rsyslog service running on the given node.
func (v *Verifier) VerifyExtensionForNode(ctx context.Context, nodeName string) {
	v.setPodExecutor(framework.NewRootPodExecutor(v.log, v.client, &nodeName, "kube-system"))
	defer v.cleanPodExecutor(ctx)

	v.setNodeName(nodeName)
	defer v.setNodeName("")

	v.verifyThatRsyslogIsActiveAndConfigured(ctx)
	v.verifyLogForwarding(
		ctx,
		logEntry{program: "test-program", severity: "1", message: "this should get sent to echo server", shouldBeForwarded: true},
		logEntry{program: "test-program", severity: "3", message: "this should not get sent to echo server", shouldBeForwarded: false},
		logEntry{program: "other-program", severity: "1", message: "this should not get sent to echo server", shouldBeForwarded: false},
	)
}

// VerifyExtensionDisabledForNode verifies whether the configuration done by the shoot-rsyslog-relp extension
// has been properly cleaned up from the given node after the extension is disabled.
func (v *Verifier) VerifyExtensionDisabledForNode(ctx context.Context, nodeName string) {
	v.setPodExecutor(framework.NewRootPodExecutor(v.log, v.client, &nodeName, "kube-system"))
	defer v.cleanPodExecutor(ctx)

	v.setNodeName(nodeName)
	defer v.setNodeName("")

	v.verifyLogForwarding(ctx,
		logEntry{program: "test-program", severity: "1", message: "this should not get sent to echo server", shouldBeForwarded: false},
	)
}

func (v *Verifier) verifyThatRsyslogIsActiveAndConfigured(ctx context.Context) {
	EventuallyWithOffset(2, func(g Gomega) {
		response, _ := ExecCommand(ctx, v.log, v.rootPodExecutor, "sh -c 'test -f /etc/rsyslog.d/60-audit.conf && systemctl is-active rsyslog.service' &>/dev/null && echo 'configured' || echo 'not configured'")
		g.Expect(string(response)).To(Equal("configured\n"), fmt.Sprintf("Expected the /etc/rsyslog.d/60-audit.conf file to exist and the rsyslog service to be active on node %s", v.nodeName))
	}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
}

func (v *Verifier) verifyLogForwarding(ctx context.Context, logEntries ...logEntry) {
	timeBeforeLogGeneration := metav1.Now()
	ExpectWithOffset(2, v.generateLogs(ctx, logEntries)).To(Succeed(), fmt.Sprintf("Expected to successfully generate logs for node %s", v.nodeName))

	forwardedLogMatchers, notForwardedLogMatchers := v.constructRegexMatchers(logEntries)
	if len(forwardedLogMatchers) > 0 {
		EventuallyWithOffset(2, func(g Gomega) {
			logLines, err := v.getLogs(ctx, timeBeforeLogGeneration)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(logLines).To(ContainElements(forwardedLogMatchers...))
		}).WithTimeout(1*time.Minute).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected logs for node %s to be present in rsyslog-relp-echo-server", v.nodeName))
	}

	if len(notForwardedLogMatchers) > 0 {
		ConsistentlyWithOffset(2, func(g Gomega) {
			logLines, err := v.getLogs(ctx, timeBeforeLogGeneration)
			g.Expect(err).NotTo(HaveOccurred())
			// Iterate over all matchers to ensure that none of them are contained in the log lines.
			for _, notSentLogsMatcher := range notForwardedLogMatchers {
				g.Expect(logLines).NotTo(ContainElement(notSentLogsMatcher))
			}
		}).WithTimeout(30*time.Second).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected logs for node %s to NOT be present in rsyslog-relp-echo-server", v.nodeName))
	}
}

func (v *Verifier) setPodExecutor(rootPodExecutor framework.RootPodExecutor) {
	v.rootPodExecutor = rootPodExecutor
}

func (v *Verifier) cleanPodExecutor(ctx context.Context) {
	ExpectWithOffset(2, v.rootPodExecutor.Clean(ctx)).To(Succeed())
}

func (v *Verifier) setNodeName(nodeName string) {
	v.nodeName = nodeName
}

func (v *Verifier) generateLogs(ctx context.Context, logEntries []logEntry) error {
	command := "sh -c '"
	for _, logEntry := range logEntries {
		command += fmt.Sprintf("echo %s | systemd-cat -t %s -p %s; ", logEntry.message, logEntry.program, logEntry.severity)
	}
	command += "'"

	if _, err := ExecCommand(ctx, v.log, v.rootPodExecutor, command); err != nil {
		return err
	}
	return nil
}

func (v *Verifier) getLogs(ctx context.Context, timeBeforeLogGeneration metav1.Time) ([]string, error) {
	logs, err := kubernetes.GetPodLogs(ctx, v.rsyslogRelpEchoServerPodIf, v.rsyslogEchoServerPodName, &corev1.PodLogOptions{SinceTime: &timeBeforeLogGeneration})
	if err != nil {
		return nil, err
	}

	var logLines []string
	scanner := bufio.NewScanner(bytes.NewReader(logs))
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
	}
	return logLines, nil
}

func (v *Verifier) constructRegexMatchers(logEntries []logEntry) ([]interface{}, []interface{}) {
	var (
		expectedNodeHostName    string
		forwardedLogMatchers    []interface{}
		notForwardedLogMatchers []interface{}
	)

	switch {
	case v.providerType == "aws":
		// On aws the nodes are named by adding the regional domain name after the instance, e.g.:
		// `ip-xxx-xxx-xxx-xxx.ec2.<region>.internal`. However the rsyslog `hostname` property only returns the
		// instance name - `ip-xxx-xxx-xxx-xxx`.
		expectedNodeHostName = strings.SplitN(v.nodeName, ".", 2)[0]
	case v.providerType == "alicloud":
		// On alicloud the name of a node is made of lower case characters, e.g. `izgw846obiag360olq8sdaz`.
		// However, the rsyslog `hostname` property can also contain upper case characters, e.g. `iZgw846obiag360olq8sdaZ`.
		// This is why we use the (?i:...) - to turn on case-insensitive mode for the hostname matching.
		expectedNodeHostName = "(?i:" + v.nodeName + ")"
	default:
		expectedNodeHostName = v.nodeName
	}

	for _, logEntry := range logEntries {
		matchRegexp := MatchRegexp(fmt.Sprintf(`%s %s %s %s \d+ %s\[\d+\]: .* %s`, v.projectName, v.shootName, v.shootUID, expectedNodeHostName, logEntry.program, logEntry.message))
		if logEntry.shouldBeForwarded {
			forwardedLogMatchers = append(forwardedLogMatchers, matchRegexp)
		} else {
			notForwardedLogMatchers = append(notForwardedLogMatchers, matchRegexp)
		}
	}
	return forwardedLogMatchers, notForwardedLogMatchers
}
