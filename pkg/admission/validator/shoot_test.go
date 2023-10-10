// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator_test

import (
	"context"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/admission/validator"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/install"
)

var _ = Describe("Shoot", func() {
	Describe("#Validate", func() {
		var (
			shoot            *core.Shoot
			shootValidator   extensionswebhook.Validator
			ctx              = context.Background()
			fakeGardenClient client.Client
		)

		BeforeEach(func() {
			install.Install(kubernetes.GardenScheme)
			fakeGardenClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.GardenScheme).Build()
			decoder := serializer.NewCodecFactory(kubernetes.GardenScheme, serializer.EnableStrict).UniversalDecoder()

			shootValidator = NewShootValidator(fakeGardenClient, decoder)

			shoot = &core.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Spec: core.ShootSpec{
					Extensions: []core.Extension{
						{
							Type: "shoot-rsyslog-relp",
						},
						{
							Type: "some-other-extension",
						},
					},
				},
			}
		})

		It("should not do anything because extension is disabled", func() {
			shoot.Spec.Extensions[0].Disabled = pointer.Bool(true)
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
		})

		It("should return an error when extension is enabled and ProviderConfig is not set", func() {
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(
				MatchError(
					ContainSubstring("Rsyslog relp configuration is required when using gardener-extension-shoot-rsyslog-relp"),
				),
			)
		})

		It("should return an error when extension is enabled and ProviderConfig is not the correct kind", func() {
			shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: Bar`),
			}
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(
				MatchError(runtime.NewNotRegisteredErrForKind(
					kubernetes.GardenScheme.Name(),
					schema.GroupVersionKind{
						Group:   "rsyslog-relp.extensions.gardener.cloud",
						Version: "v1alpha1",
						Kind:    "Bar",
					}),
				),
			)
		})

		It("should return an error when extension is enabled and target is not set in ProviderConfig", func() {
			shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
port: 10250
loggingRules:
- severity: 0
  programNames: ["kubelet", "audisp-syslog"]`),
			}
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeRequired),
					"Field":    Equal("target"),
					"BadValue": Equal(""),
					"Detail":   Equal("target must not be empty"),
				})),
			))
		})

		It("should return an error when extension is enabled and port is not set in ProviderConfig", func() {
			shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
target: "localhost"
port: -1
loggingRules:
- severity: 0
  programNames: ["kubelet", "audisp-syslog"]`),
			}
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("port"),
					"BadValue": Equal(-1),
					"Detail":   Equal("port cannot be less than 0"),
				})),
			))
		})

		It("should return an error when there are no loggingRules set in ProviderConfig", func() {
			shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{
				Raw: []byte(`
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
target: "localhost"
port: 10250`),
			}
			Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeRequired),
					"Field":    Equal("loggingRules"),
					"BadValue": Equal(""),
					"Detail":   Equal("at least one logging rule is required"),
				})),
			))
		})

		Context("when required values (port, target and loggingRules) are already set", func() {
			var extensionSpec string

			BeforeEach(func() {
				extensionSpec = `
apiVersion: rsyslog-relp.extensions.gardener.cloud/v1alpha1
kind: RsyslogRelpConfig
target: "localhost"
port: 10250
loggingRules:
- severity: 0
  programNames: ["kubelet", "audisp-syslog"]`
				shoot.Spec.Extensions[0].ProviderConfig = &runtime.RawExtension{Raw: []byte(extensionSpec)}
			})

			It("should not return error when all optional settings are present", func() {
				shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
timeout: 60
rebindInterval: 1000
resumeRetryCount: 10
reportSuspensionContinuation: true`)...)

				Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
			})

			Context("when TLS is enabled", func() {
				BeforeEach(func() {
					shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
tls:
  enabled: true
  secretReferenceName: rsyslog-secret`)...)
					shoot.Spec.Resources = []core.NamedResourceReference{
						{
							Name: "rsyslog-secret",
							ResourceRef: autoscalingv1.CrossVersionObjectReference{
								Kind:       "Secret",
								Name:       "rsyslog-secret",
								APIVersion: "v1",
							},
						},
					}
				})

				It("should return error if referenced secret does not exist", func() {
					Expect(shootValidator.Validate(ctx, shoot, nil)).To(MatchError(ContainSubstring("referenced secret bar/rsyslog-secret does not exist")))
				})

				It("should return error if referenced secret is not valid", func() {
					referencedSecret := &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rsyslog-secret",
							Namespace: "bar",
						},
						StringData: map[string]string{
							"key": "some secret data",
						},
					}

					Expect(fakeGardenClient.Create(ctx, referencedSecret)).To(Succeed())
					Expect(shootValidator.Validate(ctx, shoot, nil)).To(MatchError("secret bar/rsyslog-secret is missing ca value"))
				})

				Context("when referenced secret exists and is valid", func() {
					BeforeEach(func() {
						referencedSecret := &corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "rsyslog-secret",
								Namespace: "bar",
							},
							Data: map[string][]byte{
								"ca":  []byte("data"),
								"crt": []byte("data"),
								"key": []byte("data"),
							},
						}

						Expect(fakeGardenClient.Create(ctx, referencedSecret)).To(Succeed())
					})

					It("should not return error if authMode is set to name", func() {
						shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
  authMode: "name"
  `)...)
						Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
					})

					It("should return error if authMode is set to fingerprint", func() {
						shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
  authMode: "fingerprint"
  `)...)
						Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
					})

					It("should return error when aithMode is neither name nor fingerprint", func() {
						authModeInvalid := rsyslog.AuthMode("foo")
						shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
  authMode: "foo"
  `)...)

						Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
							PointTo(MatchFields(IgnoreExtras, Fields{
								"Type":     Equal(field.ErrorTypeNotSupported),
								"Field":    Equal("tls.authMode"),
								"BadValue": Equal(&authModeInvalid),
								"Detail":   Equal("supported values: \"fingerprint\", \"name\""),
							})),
						))
					})

					It("should not return error if permittedPeer is set correctly", func() {
						shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
  permittedPeer:
  - "localhost"
  `)...)

						Expect(shootValidator.Validate(ctx, shoot, nil)).To(Succeed())
					})

					It("should return error if permittedPeer contains an empty element", func() {
						shoot.Spec.Extensions[0].ProviderConfig.Raw = append(shoot.Spec.Extensions[0].ProviderConfig.Raw, []byte(`
  permittedPeer:
  - "localhost"
  - ""
  `)...)
						Expect(shootValidator.Validate(ctx, shoot, nil)).To(ConsistOf(
							PointTo(MatchFields(IgnoreExtras, Fields{
								"Type":     Equal(field.ErrorTypeRequired),
								"Field":    Equal("tls.permittedPeer[1]"),
								"BadValue": Equal(""),
								"Detail":   Equal("value cannot be empty"),
							})),
						))
					})
				})
			})
		})
	})
})
