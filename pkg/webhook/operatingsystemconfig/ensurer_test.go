// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"context"
	_ "embed"
	"encoding/base64"
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
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/webhook/operatingsystemconfig"
)

var (
	//go:embed testdata/60-audit.conf
	rsyslogConfig []byte
	//go:embed testdata/60-audit-with-tls.conf
	rsyslogConfigWithTLS []byte

	//go:embed testdata/configure-rsyslog.sh
	confiugreRsyslogScript []byte
	//go:embed testdata/process-rsyslog-pstats.sh
	processRsyslogPstatsScript []byte

	//go:embed testdata/00-base-config.rules
	baseConfigRules []byte
	//go:embed testdata/10-privilege-escalation.rules
	privilegeEscalationRules []byte
	//go:embed testdata/11-privileged-special.rules
	privilegeSpecialRules []byte
	//go:embed testdata/12-system-integrity.rules
	systemIntegrityRules []byte
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
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())

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
					Version: "1.27.2",
				},
			},
		}

		extensionProviderConfig = &rsyslog.RsyslogRelpConfig{
			Target: "localhost",
			Port:   10250,
			LoggingRules: []rsyslog.LoggingRule{
				{
					Severity:     5,
					ProgramNames: []string{"systemd", "audisp-syslog"},
				},
				{
					Severity:     7,
					ProgramNames: []string{"kubelet"},
				},
				{
					Severity: 2,
				},
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
			expectedFiles = append([]extensionsv1alpha1.File{oldFile}, getAuditRulesFiles(true)...)
		})

		Context("when tls is not enabled", func() {
			BeforeEach(func() {
				expectedFiles = append(expectedFiles, getRsyslogFiles(rsyslogConfig, true)...)
			})

			It("should add additional files to the current ones", func() {
				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})

			It("should modify already existing rsyslog configuration files", func() {
				files = append(files, getRsyslogFiles(rsyslogConfig, false)...)
				files = append(files, getAuditRulesFiles(false)...)

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
					SecretReferenceName: pointer.String("rsyslog-tls"),
					AuthMode:            &authModeName,
					TLSLib:              &tlsLibOpenSSL,
					PermittedPeer:       []string{"rsyslog-server.foo", "rsyslog-server.foo.bar"},
				}

				expectedFiles = append(expectedFiles, getRsyslogFiles(rsyslogConfigWithTLS, true)...)
				expectedFiles = append(expectedFiles, getRsyslogTLSFiles(true)...)
			})

			It("should add additional files to the current ones", func() {
				Expect(ensurer.EnsureAdditionalFiles(ctx, gctx, &files, nil)).To(Succeed())
				Expect(files).To(ConsistOf(expectedFiles))
			})

			It("should modify already existing rsyslog configuration files", func() {
				files = append(files, getRsyslogFiles(rsyslogConfigWithTLS, false)...)
				files = append(files, getAuditRulesFiles(false)...)
				files = append(files, getRsyslogTLSFiles(false)...)

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
			expectedUnits = append([]extensionsv1alpha1.Unit{oldUnit}, getRsyslogConfiguratorUnit(true))
		})

		It("should add additional units to the current ones", func() {
			Expect(ensurer.EnsureAdditionalUnits(ctx, nil, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(expectedUnits))
		})

		It("should modify existing units", func() {
			units = append(units, getRsyslogConfiguratorUnit(false))

			Expect(ensurer.EnsureAdditionalUnits(ctx, nil, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(expectedUnits))
		})
	})
})

func getAuditRulesFiles(useExpectedContent bool) []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/00-base-config.rules",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(getBasedOnCondition(useExpectedContent, baseConfigRules, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/10-privilege-escalation.rules",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(getBasedOnCondition(useExpectedContent, privilegeEscalationRules, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/11-privileged-special.rules",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(getBasedOnCondition(useExpectedContent, privilegeSpecialRules, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/12-system-integrity.rules",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(getBasedOnCondition(useExpectedContent, systemIntegrityRules, []byte("oldContent"))),
				},
			},
		},
	}
}

func getRsyslogFiles(rsyslogConfig []byte, useExpectedContent bool) []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        "/var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     base64.StdEncoding.EncodeToString(getBasedOnCondition(useExpectedContent, rsyslogConfig, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     base64.StdEncoding.EncodeToString(getBasedOnCondition(useExpectedContent, confiugreRsyslogScript, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/process-rsyslog-pstats.sh",
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     base64.StdEncoding.EncodeToString(getBasedOnCondition(useExpectedContent, processRsyslogPstatsScript, []byte("oldContent"))),
				},
			},
		},
	}
}

func getRsyslogTLSFiles(useExpectedContent bool) []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        "/var/lib/rsyslog-relp-configurator/tls/ca.crt",
			Permissions: pointer.Int32(0600),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    getBasedOnCondition(useExpectedContent, "ref-rsyslog-tls", "ref-rsyslog-tls-old"),
					DataKey: "ca",
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/tls/tls.crt",
			Permissions: pointer.Int32(0600),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    getBasedOnCondition(useExpectedContent, "ref-rsyslog-tls", "ref-rsyslog-tls-old"),
					DataKey: "crt",
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/tls/tls.key",
			Permissions: pointer.Int32(0600),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    getBasedOnCondition(useExpectedContent, "ref-rsyslog-tls", "ref-rsyslog-tls-old"),
					DataKey: "key",
				},
			},
		},
	}
}

func getBasedOnCondition[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func getRsyslogConfiguratorUnit(useExpectedContent bool) extensionsv1alpha1.Unit {
	return extensionsv1alpha1.Unit{
		Name:    "rsyslog-configurator.service",
		Command: ptr.To(extensionsv1alpha1.CommandStart),
		Enable:  pointer.Bool(true),
		Content: pointer.String(getBasedOnCondition(useExpectedContent, `[Unit]
Description=rsyslog configurator daemon
Documentation=https://github.com/gardener/gardener-extension-shoot-rsyslog-relp
[Service]
Type=simple
Restart=always
RestartSec=15
ExecStart=/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh
[Install]
WantedBy=multi-user.target`, `old`)),
	}
}
