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
	v.setNodeName(nodeName)

	v.verifyThatRsyslogIsActiveAndConfigured(ctx)
	v.verifyThatLogsAreSentToEchoServer(ctx, "test-program", "1", "this should get sent to echo server")
	v.verifyThatLogsAreNotSentToEchoServer(ctx, "other-program", "1", "this should not get sent to echo server")
	v.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "3", "this should not get sent to echo server")

	v.cleanPodExecutor(ctx)
	v.setNodeName("")
}

// VerifyExtensionDisabledForNode verifies whether the configuration done by the shoot-rsyslog-relp extension
// has been properly cleaned up from the given node after the extension is disabled.
func (v *Verifier) VerifyExtensionDisabledForNode(ctx context.Context, nodeName string) {
	v.setPodExecutor(framework.NewRootPodExecutor(v.log, v.client, &nodeName, "kube-system"))
	v.setNodeName(nodeName)

	v.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "1", "this should not get sent to echo server")

	v.cleanPodExecutor(ctx)
	v.setNodeName("")
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

func (v *Verifier) verifyThatRsyslogIsActiveAndConfigured(ctx context.Context) {
	EventuallyWithOffset(2, func(g Gomega) {
		response, _ := ExecCommand(ctx, v.log, v.rootPodExecutor, "systemctl is-active rsyslog.service &>/dev/null && echo 'active' || echo 'not active'")
		g.Expect(string(response)).To(Equal("active\n"), fmt.Sprintf("Expected the rsyslog.service unit to be active on node %s", v.nodeName))

		response, _ = ExecCommand(ctx, v.log, v.rootPodExecutor, "test -f /etc/rsyslog.d/60-audit.conf && echo 'configured' || echo 'not configured'")
		g.Expect(string(response)).To(Equal("configured\n"), fmt.Sprintf("Expected the /etc/rsyslog.d/60-audit.conf file to exist on node %s", v.nodeName))
	}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
}

func (v *Verifier) verifyThatLogsAreSentToEchoServer(ctx context.Context, programName, severity, logMessage string) {
	EventuallyWithOffset(2, func(g Gomega) {
		logLines, err := v.generateAndGetLogs(ctx, programName, severity, logMessage)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logLines).To(ContainElement(MatchRegexp(v.constructRegex(programName, logMessage))))
	}).WithTimeout(1*time.Minute).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected to successfully generate logs for node %s and logs to be present in rsyslog-relp-echo-server", v.nodeName))
}

func (v *Verifier) verifyThatLogsAreNotSentToEchoServer(ctx context.Context, programName, severity, logMessage string) {
	ConsistentlyWithOffset(2, func(g Gomega) {
		logLines, err := v.generateAndGetLogs(ctx, programName, severity, logMessage)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logLines).NotTo(ContainElement(MatchRegexp(v.constructRegex(programName, logMessage))))
	}).WithTimeout(30*time.Second).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected to successfully generate logs for node %s and logs to NOT be present in rsyslog-relp-echo-server", v.nodeName))
}

func (v *Verifier) generateAndGetLogs(ctx context.Context, programName, severity, logMessage string) ([]string, error) {
	command := fmt.Sprintf("sh -c 'echo %s | systemd-cat -t %s -p %s'", logMessage, programName, severity)
	timeBeforeLogGeneration := metav1.Now()
	if _, err := ExecCommand(ctx, v.log, v.rootPodExecutor, command); err != nil {
		return nil, err
	}
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

func (v *Verifier) constructRegex(programName, logMessage string) string {
	var expectedNodeHostName string

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
		expectedNodeHostName = "(?i:" + expectedNodeHostName + ")"
	default:
		expectedNodeHostName = v.nodeName
	}

	return fmt.Sprintf(`%s %s %s %s \d+ %s\[\d+\]: .* %s`, v.projectName, v.shootName, v.shootUID, expectedNodeHostName, programName, logMessage)
}
