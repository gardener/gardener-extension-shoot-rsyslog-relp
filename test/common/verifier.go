// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const fileForAuditEvent = "/etc/newfile"

// Verifier is a struct that can be used to verify whether the shoot-rsyslog-relp extension is working as expected.
type Verifier struct {
	log                logr.Logger
	client             kubernetes.Interface
	clientForLogs      kubernetes.Interface
	providerType       string
	nodeName           string
	projectName        string
	shootName          string
	shootUID           string
	rootPodExecutor    framework.RootPodExecutor
	testAuditLogging   bool
	expectedAuditRules string
}

// LogEntry is a struct used in the verification of logs, based on the logging rules
type LogEntry struct {
	// Program name for entry.
	Program string
	// Severity for entry.
	Severity string
	// Message for entry.
	Message string
	// ShouldBeForwarded determines whether or not the log should be forwarded to echo server.
	ShouldBeForwarded bool
}

// NewVerifier creates a new Verifier.
func NewVerifier(log logr.Logger,
	client, clientForLogs kubernetes.Interface,
	providerType, projectName, shootName, shootUID string,
	testAuditLogging bool,
	expectedAuditRules string) *Verifier {
	return &Verifier{
		log:                log,
		client:             client,
		clientForLogs:      clientForLogs,
		providerType:       providerType,
		projectName:        projectName,
		shootName:          shootName,
		shootUID:           shootUID,
		testAuditLogging:   testAuditLogging,
		expectedAuditRules: expectedAuditRules,
	}
}

// VerifyExtensionForNode verifies whether the shoot-rsyslog-relp extension has properly configured
// the rsyslog service running on the given node.
func (v *Verifier) VerifyExtensionForNode(ctx context.Context, nodeName string, logEntries ...LogEntry) {
	v.setPodExecutor(framework.NewRootPodExecutor(v.log, v.client, &nodeName, "kube-system"))
	defer v.cleanPodExecutor(ctx)

	v.setNodeName(nodeName)
	defer v.setNodeName("")

	v.verifyThatRsyslogIsActiveAndConfigured(ctx)

	if v.testAuditLogging {
		v.verifyThatAuditRulesAreInstalled(ctx)
	}

	logEntries = append(logEntries,
		LogEntry{Program: "test-program", Severity: "1", Message: "this should get sent to echo server", ShouldBeForwarded: true},
		LogEntry{Program: "test-program", Severity: "3", Message: "this should not get sent to echo server", ShouldBeForwarded: false},
		LogEntry{Program: "other-program", Severity: "1", Message: "this should not get sent to echo server", ShouldBeForwarded: false},
	)

	v.verifyLogsAreForwardedToEchoServer(ctx, logEntries...)
}

// VerifyExtensionDisabledForNode verifies whether the configuration done by the shoot-rsyslog-relp extension
// has been properly cleaned up from the given node after the extension is disabled.
func (v *Verifier) VerifyExtensionDisabledForNode(ctx context.Context, nodeName string) {
	v.setPodExecutor(framework.NewRootPodExecutor(v.log, v.client, &nodeName, "kube-system"))
	defer v.cleanPodExecutor(ctx)

	v.setNodeName(nodeName)
	defer v.setNodeName("")

	v.verifyThatLogsAreNotForwardedToEchoServer(ctx,
		LogEntry{Program: "test-program", Severity: "1", Message: "this should not get sent to echo server"},
	)
}

func (v *Verifier) verifyThatAuditRulesAreInstalled(ctx context.Context) {
	EventuallyWithOffset(2, func(g Gomega) {
		response, _ := ExecCommand(ctx, v.log, v.rootPodExecutor, "cat", "/etc/audit/audit.rules")
		g.Expect(string(response)).To(Equal(v.expectedAuditRules), fmt.Sprintf("Expected the /etc/audit/audit.rules file to contain correct audit rules on node %s", v.nodeName))
	}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
}

func (v *Verifier) verifyThatRsyslogIsActiveAndConfigured(ctx context.Context) {
	EventuallyWithOffset(2, func(g Gomega) {
		response, _ := ExecCommand(ctx, v.log, v.rootPodExecutor, "sh", "-c", "[ -f /etc/rsyslog.d/60-audit.conf ] && systemctl is-active rsyslog.service 1>/dev/null 2>&1 && echo configured || echo not configured")
		g.Expect(string(response)).To(Equal("configured\n"), fmt.Sprintf("Expected the /etc/rsyslog.d/60-audit.conf file to exist and the rsyslog service to be active on node %s", v.nodeName))
	}).WithTimeout(1 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
}

func (v *Verifier) verifyLogsAreForwardedToEchoServer(ctx context.Context, logEntries ...LogEntry) {
	timeBeforeLogGeneration := metav1.Now()
	EventuallyWithOffset(2, func() error {
		return v.generateLogs(ctx, logEntries)
	}).WithTimeout(30*time.Second).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected to successfully generate logs for node %s", v.nodeName))

	forwardedLogMatchers, notForwardedLogMatchers := v.constructLogMatchers(logEntries)
	if len(forwardedLogMatchers) > 0 {
		EventuallyWithOffset(2, func(g Gomega) {
			logLines, err := v.getLogs(ctx, timeBeforeLogGeneration)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(logLines).To(ContainElements(forwardedLogMatchers...))
		}).WithTimeout(1*time.Minute).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected logs for node %s to be present in rsyslog-relp-echo-server", v.nodeName))
	}

	if len(notForwardedLogMatchers) > 0 {
		By("Wait 30 seconds before checking for logs")
		timer := time.NewTimer(30 * time.Second)
		select {
		case <-ctx.Done():
			Fail("context deadline exceeded while waiting to check for logs")
		case <-timer.C:
		}

		By("Verify that there are no logs")
		EventuallyWithOffset(2, func(g Gomega) {
			logLines, err := v.getLogs(ctx, timeBeforeLogGeneration)
			g.Expect(err).NotTo(HaveOccurred())
			// Iterate over all matchers to ensure that none of them are contained in the log lines.
			for _, notSentLogsMatcher := range notForwardedLogMatchers {
				g.Expect(logLines).NotTo(ContainElement(notSentLogsMatcher))
			}
		}).WithTimeout(30*time.Second).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected logs for node %s to NOT be present in rsyslog-relp-echo-server", v.nodeName))
	}
}

func (v *Verifier) verifyThatLogsAreNotForwardedToEchoServer(ctx context.Context, logEntries ...LogEntry) {
	timeBeforeLogGeneration := metav1.Now()
	EventuallyWithOffset(2, func() error {
		return v.generateLogs(ctx, logEntries)
	}).WithTimeout(30*time.Second).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected to successfully generate logs for node %s", v.nodeName))

	By("Wait 30 seconds before checking for logs")
	timer := time.NewTimer(30 * time.Second)
	select {
	case <-ctx.Done():
		Fail("context deadline exceeded while waiting to check for logs")
	case <-timer.C:
	}

	notForwardedLogs := make([]types.GomegaMatcher, 0, len(logEntries))
	for _, log := range logEntries {
		notForwardedLogs = append(notForwardedLogs, ContainSubstring(log.Message))
	}

	By("Verify that there are no logs")
	EventuallyWithOffset(2, func(g Gomega) {
		logLines, err := v.getLogs(ctx, timeBeforeLogGeneration)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logLines).To(Or(BeEmpty(), Not(ContainElements(notForwardedLogs))))
	}).WithTimeout(30*time.Second).WithPolling(10*time.Second).WithContext(ctx).Should(Succeed(), fmt.Sprintf("Expected to successfully generate logs for node %s and logs to NOT be present in rsyslog-relp-echo-server", v.nodeName))
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

func (v *Verifier) generateLogs(ctx context.Context, logEntries []LogEntry) error {
	command := ""
	for _, logEntry := range logEntries {
		command += "echo " + logEntry.Message + " | systemd-cat -t " + logEntry.Program + " -p " + logEntry.Severity + "; "
	}
	if v.testAuditLogging {
		// Create a file under /etc directory so that an audit event is generated.
		command += "echo some-content > " + fileForAuditEvent + "; rm -f " + fileForAuditEvent
	}
	command = strings.TrimSuffix(command, "; ")

	if _, err := ExecCommand(ctx, v.log, v.rootPodExecutor, "sh", "-c", command); err != nil {
		return err
	}
	return nil
}

func (v *Verifier) getLogs(ctx context.Context, timeBeforeLogGeneration metav1.Time) ([]string, error) {
	echoServerPodIf, echoServerPodName, err := GetEchoServerPodInterfaceAndName(ctx, v.clientForLogs)
	if err != nil {
		return nil, err
	}
	logs, err := kubernetesutils.GetPodLogs(ctx, echoServerPodIf, echoServerPodName, &corev1.PodLogOptions{SinceTime: &timeBeforeLogGeneration})
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

func (v *Verifier) constructLogMatchers(logEntries []LogEntry) ([]interface{}, []interface{}) {
	var (
		expectedNodeHostName    string
		forwardedLogMatchers    []interface{}
		notForwardedLogMatchers []interface{}
	)

	switch v.providerType {
	case "aws":
		// On aws the nodes are named by adding the regional domain name after the instance, e.g.:
		// `ip-xxx-xxx-xxx-xxx.ec2.<region>.internal`. However the rsyslog `hostname` property only returns the
		// instance name - `ip-xxx-xxx-xxx-xxx`.
		expectedNodeHostName = strings.SplitN(v.nodeName, ".", 2)[0]
	case "alicloud":
		// On alicloud the name of a node is made of lower case characters, e.g. `izgw846obiag360olq8sdaz`.
		// However, the rsyslog `hostname` property can also contain upper case characters, e.g. `iZgw846obiag360olq8sdaZ`.
		// This is why we use the (?i:...) - to turn on case-insensitive mode for the hostname matching.
		expectedNodeHostName = "(?i:" + v.nodeName + ")"
	default:
		expectedNodeHostName = v.nodeName
	}

	for _, logEntry := range logEntries {
		matchRegexp := MatchRegexp(fmt.Sprintf(`%s %s %s %s \d+ %s\[\d+\]: .* %s`, v.projectName, v.shootName, v.shootUID, expectedNodeHostName, logEntry.Program, logEntry.Message))
		if logEntry.ShouldBeForwarded {
			forwardedLogMatchers = append(forwardedLogMatchers, matchRegexp)
		} else {
			notForwardedLogMatchers = append(notForwardedLogMatchers, matchRegexp)
		}
	}

	if v.testAuditLogging {
		forwardedLogMatchers = append(forwardedLogMatchers, ContainSubstring(fileForAuditEvent))
	}

	return forwardedLogMatchers, notForwardedLogMatchers
}
