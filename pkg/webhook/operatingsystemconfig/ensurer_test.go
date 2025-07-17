// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"context"
	"fmt"

	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1alpha1 "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	rsysloginstall "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/install"
	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/webhook/operatingsystemconfig"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/webhook/operatingsystemconfig/webhooktest"
)

var _ = Describe("Ensurer", func() {
	var (
		logger = logr.Discard()
		ctx    = context.Background()

		decoder    runtime.Decoder
		fakeClient client.Client

		gctx    gcontext.GardenContext
		shoot   *gardencorev1beta1.Shoot
		cluster *extensions.Cluster

		extensionProviderConfig *rsyslog.RsyslogRelpConfig
		extensionResource       *extensionsv1alpha1.Extension

		shootName        = "foo"
		projectName      = "bar"
		shootUID         = types.UID("uid")
		shootTechnicalID = fmt.Sprintf("shoot--%s--%s", projectName, shootName)

		authModeName  rsyslog.AuthMode = "name"
		tlsLibOpenSSL rsyslog.TLSLib   = "openssl"
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(configv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(rsysloginstall.AddToScheme(scheme)).To(Succeed())

		decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
		fakeClient = fakeclient.NewClientBuilder().WithScheme(scheme).Build()

		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shootName,
				Namespace: fmt.Sprintf("garden-%s", projectName),
				UID:       shootUID,
			},
			Spec: gardencorev1beta1.ShootSpec{
				Provider: gardencorev1beta1.Provider{
					Workers: []gardencorev1beta1.Worker{{Name: "worker"}},
				},
				Kubernetes: gardencorev1beta1.Kubernetes{
					Version: "1.29.0",
				},
			},
		}

		extensionProviderConfig = &rsyslog.RsyslogRelpConfig{
			Target: "localhost",
			Port:   10250,
			LoggingRules: []rsyslog.LoggingRule{
				{
					Severity:     ptr.To(5),
					ProgramNames: []string{"systemd", "audisp-syslog"},
					MessageContent: &rsyslog.MessageContent{
						Regex:   ptr.To("foo"),
						Exclude: ptr.To("bar"),
					},
				},
				{
					Severity:     ptr.To(7),
					ProgramNames: []string{"kubelet"},
				},
				{
					Severity: ptr.To(2),
				},
			},
			AuditConfig: &rsyslog.AuditConfig{
				Enabled: true,
			},
		}
	})

	JustBeforeEach(func() {
		cluster = &extensions.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootTechnicalID,
			},
			Shoot: shoot,
		}
		gctx = gcontext.NewInternalGardenContext(cluster)

		extensionResource = &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-rsyslog-relp",
				Namespace: shootTechnicalID,
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					ProviderConfig: &runtime.RawExtension{
						Object: extensionProviderConfig,
					},
					Type: "shoot-rsyslog-relp",
				},
			},
		}
		Expect(fakeClient.Create(ctx, extensionResource)).To(Succeed())
	})

	Describe("#EnsureAdditionalFiles", func() {
		var (
			ensurer genericmutator.Ensurer

			oldFile = extensionsv1alpha1.File{
				Path: "/var/lib/foo.sh",
			}
			files         []extensionsv1alpha1.File
			expectedFiles []extensionsv1alpha1.File
		)

		BeforeEach(func() {
			ensurer = NewEnsurer(fakeClient, decoder, logger)
			files = []extensionsv1alpha1.File{oldFile}
			expectedFiles = append([]extensionsv1alpha1.File{oldFile}, webhooktest.GetAuditRulesFiles(true)...)
		})

		Context("when tls is not enabled", func() {
			BeforeEach(func() {
				expectedFiles = append(expectedFiles, webhooktest.GetRsyslogFiles(webhooktest.GetTestingRsyslogConfig(), true)...)
			})

			It("should add additional files to the current ones", func() {
				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})

			It("should modify already existing rsyslog configuration files", func() {
				files = append(files, webhooktest.GetRsyslogFiles(webhooktest.GetTestingRsyslogConfig(), false)...)
				files = append(files, webhooktest.GetAuditRulesFiles(false)...)

				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})
		})

		Context("when tls is enabled", func() {
			BeforeEach(func() {
				shoot.Spec.Resources = []gardencorev1beta1.NamedResourceReference{
					{
						Name: "rsyslog-tls",
						ResourceRef: v1.CrossVersionObjectReference{
							Kind: "Secret",
							Name: "rsyslog-tls",
						},
					},
				}

				rsyslogSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ref-rsyslog-tls",
						Namespace: shootTechnicalID,
					},
					Data: map[string][]byte{
						"ca":  []byte("ca"),
						"crt": []byte("crt"),
						"key": []byte("key"),
					},
				}
				Expect(fakeClient.Create(ctx, rsyslogSecret)).To(Succeed())

				extensionProviderConfig.TLS = &rsyslog.TLS{
					Enabled:             true,
					SecretReferenceName: ptr.To("rsyslog-tls"),
					AuthMode:            &authModeName,
					TLSLib:              &tlsLibOpenSSL,
					PermittedPeer:       []string{"rsyslog-server.foo", "rsyslog-server.foo.bar"},
				}

				expectedFiles = append(expectedFiles, webhooktest.GetRsyslogFiles(webhooktest.GetRsyslogConfigWithTLS(), true)...)
				expectedFiles = append(expectedFiles, webhooktest.GetRsyslogTLSFiles(true)...)
			})

			It("should add additional files to the current ones", func() {
				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})

			It("should modify already existing rsyslog configuration files", func() {
				files = append(files, webhooktest.GetRsyslogFiles(webhooktest.GetRsyslogConfigWithTLS(), false)...)
				files = append(files, webhooktest.GetAuditRulesFiles(false)...)
				files = append(files, webhooktest.GetRsyslogTLSFiles(false)...)

				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})
		})

		Context("when audit rules are specified via a configmap reference", func() {
			BeforeEach(func() {
				shoot.Spec.Resources = []gardencorev1beta1.NamedResourceReference{
					{
						Name: "audit-rules",
						ResourceRef: v1.CrossVersionObjectReference{
							Kind: "ConfigMap",
							Name: "audit-rules",
						},
					},
				}

				auditRulesConfigMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ref-audit-rules",
						Namespace: shootTechnicalID,
					},
					Data: map[string]string{
						"auditd": `apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: Auditd
auditRules: |
  custom-rule-00`,
					},
				}
				Expect(fakeClient.Create(ctx, auditRulesConfigMap)).To(Succeed())

				extensionProviderConfig.AuditConfig = &rsyslog.AuditConfig{
					Enabled:                true,
					ConfigMapReferenceName: ptr.To("audit-rules"),
				}

				expectedFiles = append([]extensionsv1alpha1.File{oldFile}, webhooktest.GetRsyslogFiles(webhooktest.GetTestingRsyslogConfig(), true)...)
				expectedFiles = append(expectedFiles, []extensionsv1alpha1.File{
					{
						Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/00_shoot_rsyslog_relp.rules",
						Permissions: ptr.To(uint32(0644)),
						Content: extensionsv1alpha1.FileContent{
							Inline: &extensionsv1alpha1.FileContentInline{
								Encoding: "b64",
								Data:     gardenerutils.EncodeBase64([]byte("custom-rule-00")),
							},
						},
					},
				}...)
			})

			It("should add additional files to current ones", func() {
				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})
		})

		Context("when modification of audit rules is disabled", func() {
			BeforeEach(func() {
				extensionProviderConfig.AuditConfig = &rsyslog.AuditConfig{
					Enabled: false,
				}
				expectedFiles = append([]extensionsv1alpha1.File{oldFile}, webhooktest.GetRsyslogFiles(webhooktest.GetTestingRsyslogConfig(), true)...)
			})

			It("should add additional files to the current ones, but not include audit rules files", func() {
				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})
		})
	})

	Describe("#EnsureAdditionalUnits", func() {
		var (
			ensurer genericmutator.Ensurer

			oldUnit = extensionsv1alpha1.Unit{
				Name: "foo.service",
			}
			units         []extensionsv1alpha1.Unit
			expectedUnits []extensionsv1alpha1.Unit
		)

		BeforeEach(func() {
			ensurer = NewEnsurer(fakeClient, decoder, logger)
			units = []extensionsv1alpha1.Unit{oldUnit}
			expectedUnits = append([]extensionsv1alpha1.Unit{oldUnit}, webhooktest.GetRsyslogConfiguratorUnit(true))
		})

		It("should add additional units to the current ones", func() {
			Expect(ensurer.EnsureAdditionalUnits(ctx, nil, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(expectedUnits))
		})

		It("should modify existing units", func() {
			units = append(units, webhooktest.GetRsyslogConfiguratorUnit(false))

			Expect(ensurer.EnsureAdditionalUnits(ctx, nil, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(expectedUnits))
		})
	})
})
