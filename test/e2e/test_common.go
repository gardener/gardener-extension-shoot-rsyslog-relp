// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	e2e "github.com/gardener/gardener/test/e2e/gardener"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rsyslogv1alpha1 "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
)

var (
	parentCtx context.Context
)

var _ = BeforeEach(func() {
	parentCtx = context.Background()
})

func defaultShootCreationFramework() *framework.ShootCreationFramework {
	return framework.NewShootCreationFramework(&framework.ShootCreationConfig{
		GardenerConfig: e2e.DefaultGardenConfig("garden-local"),
	})
}

type verifier struct {
	log                        logr.Logger
	client                     kubernetes.Interface
	rsyslogRelpEchoServerPodIf clientcorev1.PodInterface
	rsyslogEchoServerPodName   string
	shootSeedNamespace         string
	projectName                string
	shootName                  string
	shootUID                   string
}

func newVerifier(ctx context.Context, log logr.Logger, c kubernetes.Interface, shootSeedNamespace, projectName, shootName, shootUID string) (*verifier, error) {
	podIf := c.Kubernetes().CoreV1().Pods("rsyslog-relp-echo-server")

	pods := &corev1.PodList{}
	if err := c.Client().List(ctx, pods, client.InNamespace("rsyslog-relp-echo-server")); err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, errors.New("could not find any rsyslog-relp-echo-server pods")
	}

	return &verifier{
		log:                        log,
		client:                     c,
		rsyslogRelpEchoServerPodIf: podIf,
		rsyslogEchoServerPodName:   pods.Items[0].Name,
		shootSeedNamespace:         shootSeedNamespace,
		projectName:                projectName,
		shootName:                  shootName,
		shootUID:                   shootUID,
	}, nil
}

func (v *verifier) verifyThatLogsAreSentToEchoServer(ctx context.Context, programName, severity, logMessage string, args ...interface{}) {
	EventuallyWithOffset(1, func(g Gomega) {
		logLines, err := v.generateAndGetLogs(ctx, programName, severity, logMessage)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logLines).To(ContainElement(MatchRegexp(v.constructRegex(programName, logMessage))))
	}, args...).Should(Succeed())
}

func (v *verifier) verifyThatLogsAreNotSentToEchoServer(ctx context.Context, programName, severity, logMessage string, args ...interface{}) {
	ConsistentlyWithOffset(1, func(g Gomega) {
		logLines, err := v.generateAndGetLogs(ctx, programName, severity, logMessage)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logLines).NotTo(ContainElement(MatchRegexp(v.constructRegex(programName, logMessage))))
	}, args...).Should(Succeed())
}

func (v *verifier) generateAndGetLogs(ctx context.Context, programName, severity, logMessage string) ([]string, error) {
	timeBeforeLogGeneration := metav1.Now()
	if err := execInShootNode(ctx, v.client, v.log, v.shootSeedNamespace, fmt.Sprintf("echo '%s' | systemd-cat -t %s -p %s", logMessage, programName, severity)); err != nil {
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

func (v *verifier) constructRegex(programName, logMessage string) string {
	return fmt.Sprintf(` %s %s %s .* %s\[\d+\]: .* %s`, v.projectName, v.shootName, v.shootUID, programName, logMessage)
}

func execInShootNode(ctx context.Context, c kubernetes.Interface, log logr.Logger, namespace, command string) error {
	machineLabels := map[string]string{
		"app":              "machine",
		"machine-provider": "local",
	}
	machineLabelsSelector := labels.SelectorFromSet(labels.Set(machineLabels))

	err := retry.Until(ctx, 5*time.Second, func(ctx context.Context) (bool, error) {
		_, err := framework.PodExecByLabel(ctx, machineLabelsSelector, "node", command, namespace, c)

		if err != nil {
			log.Error(err, "Error exec'ing into pod")
			return retry.MinorError(err)
		}
		return retry.Ok()
	})

	return err
}

func createNetworkPolicyForEchoServer(ctx context.Context, c kubernetes.Interface, namespace string) error {
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-machine-to-rsyslog-relp-echo-server",
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "machine",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name":     "rsyslog-relp-echo-server",
							"app.kubernetes.io/instance": "rsyslog-relp-echo-server",
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "rsyslog-relp-echo-server",
						},
					},
				}},
			}},
		},
	}

	return c.Client().Create(ctx, networkPolicy)
}

func shootRsyslogRelpExtension(opts ...func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig)) gardencorev1beta1.Extension {
	defaultProviderConfig := &rsyslogv1alpha1.RsyslogRelpConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rsyslogv1alpha1.SchemeGroupVersion.String(),
			Kind:       "RsyslogRelpConfig",
		},
		Target: "10.2.64.54",
		Port:   80,
		LoggingRules: []rsyslogv1alpha1.LoggingRule{
			{
				ProgramNames: []string{"test-program"},
				Severity:     1,
			},
		},
	}

	for _, opt := range opts {
		opt(defaultProviderConfig)
	}

	providerConfigJSON, err := json.Marshal(&defaultProviderConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	extension := gardencorev1beta1.Extension{
		Type: "shoot-rsyslog-relp",
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigJSON,
		},
	}

	return extension
}

func withPort(port int) func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
	return func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
		rsyslogRelpConfig.Port = port
	}
}

func withTLSWithSecretRefName(secretRefName string) func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
	return func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
		var (
			authModeName  = rsyslogv1alpha1.AuthMode("name")
			tlsLibOpenSSL = rsyslogv1alpha1.TLSLib("openssl")
		)

		rsyslogRelpConfig.TLS = &rsyslogv1alpha1.TLS{
			Enabled:             true,
			SecretReferenceName: &secretRefName,
			PermittedPeer:       []string{"rsyslog-server"},
			AuthMode:            &authModeName,
			TLSLib:              &tlsLibOpenSSL,
		}
	}
}
