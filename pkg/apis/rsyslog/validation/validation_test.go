// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/validation"
)

var _ = Describe("Validation", func() {
	var path = field.NewPath("")

	Describe("#ValidateRsyslogRelpConfig", func() {
		const (
			relpTarget     = "rsyslog.relp.server"
			relpTargetPort = 10250
		)

		var (
			authModeName        rsyslog.AuthMode = "name"
			authModeFingerPrint rsyslog.AuthMode = "fingerprint"
			authModeInvalid     rsyslog.AuthMode = "invalid"

			tlsLibOpenSSL rsyslog.TLSLib = "openssl"
			tlsLibGnuTLS  rsyslog.TLSLib = "gnutls"
			tlsLibInvalid rsyslog.TLSLib = "invalid"

			loggingRules = []rsyslog.LoggingRule{
				{
					ProgramNames: []string{"kubelet"},
					Severity:     ptr.To(0),
				},
			}
		)

		It("should allow setting target, port and loggingRules for relp backend server", func() {
			config := rsyslog.RsyslogRelpConfig{
				Target:       relpTarget,
				Port:         relpTargetPort,
				LoggingRules: loggingRules,
			}
			errorList := validation.ValidateRsyslogRelpConfig(&config, path)
			Expect(errorList).To(BeEmpty())
		})

		It("should not allow setting empty target", func() {
			config := rsyslog.RsyslogRelpConfig{
				Target:       "",
				Port:         relpTargetPort,
				LoggingRules: loggingRules,
			}

			matcher := ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeRequired),
					"Field":    Equal("target"),
					"BadValue": Equal(""),
					"Detail":   Equal("target must not be empty"),
				})),
			)

			errorList := validation.ValidateRsyslogRelpConfig(&config, field.NewPath(""))
			Expect(errorList).To(matcher)
		})

		It("should not allow setting port less than 0", func() {
			config := rsyslog.RsyslogRelpConfig{
				Target:       relpTarget,
				Port:         -1,
				LoggingRules: loggingRules,
			}

			matcher := ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("port"),
					"BadValue": Equal(-1),
					"Detail":   Equal("port cannot be less than 0"),
				})),
			)

			errorList := validation.ValidateRsyslogRelpConfig(&config, path)
			Expect(errorList).To(matcher)
		})

		It("should not allow empty loggingRules", func() {
			config := rsyslog.RsyslogRelpConfig{
				Target: relpTarget,
				Port:   relpTargetPort,
			}

			matcher := ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeRequired),
					"Field":    Equal("loggingRules"),
					"BadValue": Equal(""),
					"Detail":   Equal("at least one logging rule is required"),
				})),
			)

			errorList := validation.ValidateRsyslogRelpConfig(&config, path)
			Expect(errorList).To(matcher)
		})

		It("should not allow empty no field of a logging rule to be set", func() {
			config := rsyslog.RsyslogRelpConfig{
				Target:       relpTarget,
				Port:         relpTargetPort,
				LoggingRules: []rsyslog.LoggingRule{{}},
			}

			matcher := ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeRequired),
					"Field":    Equal("loggingRules[0]"),
					"BadValue": Equal(""),
					"Detail":   Equal("at least one field of the logging rule is required"),
				})),
			)

			errorList := validation.ValidateRsyslogRelpConfig(&config, path)
			Expect(errorList).To(matcher)
		})

		Context("Configuration", func() {
			DescribeTable("General Configuration",
				func(config rsyslog.RsyslogRelpConfig, matcher gomegatypes.GomegaMatcher) {
					errorList := validation.ValidateRsyslogRelpConfig(&config, path)
					Expect(errorList).To(matcher)
				},

				Entry("should allow config when all setting are correct",
					rsyslog.RsyslogRelpConfig{Target: relpTarget, Port: relpTargetPort, TLS: &rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secretRef"), PermittedPeer: []string{"per"}, AuthMode: &authModeName}, LoggingRules: loggingRules, RebindInterval: ptr.To(1000), Timeout: ptr.To(90), ResumeRetryCount: ptr.To(10), ReportSuspensionContinuation: ptr.To(true)},
					BeEmpty(),
				),
			)

			DescribeTable("TLS Configuration",
				func(tlsConfig rsyslog.TLS, matcher gomegatypes.GomegaMatcher) {
					rsyslogRelpConfig := &rsyslog.RsyslogRelpConfig{
						Target:       relpTarget,
						Port:         relpTargetPort,
						LoggingRules: loggingRules,
						TLS:          &tlsConfig,
					}
					errorList := validation.ValidateRsyslogRelpConfig(rsyslogRelpConfig, path.Child("tls"))
					Expect(errorList).To(matcher)
				},

				Entry("should allow config when TLS is disabled and TLS settings are not set",
					rsyslog.TLS{Enabled: false},
					BeEmpty(),
				),

				Entry("should allow config when TLS is enabled and secretReferenceName is set",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name")},
					BeEmpty(),
				),

				Entry("should forbid config when TLS is enabled and secretReferenceName is not set",
					rsyslog.TLS{Enabled: true},
					ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":     Equal(field.ErrorTypeRequired),
							"Field":    Equal("tls.secretReferenceName"),
							"BadValue": Equal(""),
							"Detail":   Equal("secretReferenceName must not be empty when tls is enabled"),
						})),
					),
				),

				Entry("should allow config when TLS authMode is name",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), AuthMode: &authModeName},
					BeEmpty(),
				),

				Entry("should allow config when TLS authMode is fingerprint",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), AuthMode: &authModeFingerPrint},
					BeEmpty(),
				),

				Entry("should forbid config when TLS authMode is invalid",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), AuthMode: &authModeInvalid},
					ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":     Equal(field.ErrorTypeNotSupported),
							"Field":    Equal("tls.authMode"),
							"BadValue": Equal(&authModeInvalid),
							"Detail":   Equal(`supported values: "fingerprint", "name"`),
						})),
					),
				),

				Entry("should allow config when tls lib is openssl",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), TLSLib: &tlsLibOpenSSL},
					BeEmpty(),
				),

				Entry("should allow config when tls lib is gnutls",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), TLSLib: &tlsLibGnuTLS},
					BeEmpty(),
				),

				Entry("should forbid config when tls lib is invalid",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), TLSLib: &tlsLibInvalid},
					ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":     Equal(field.ErrorTypeNotSupported),
							"Field":    Equal("tls.tlsLib"),
							"BadValue": Equal(&tlsLibInvalid),
							"Detail":   Equal(`supported values: "gnutls", "openssl"`),
						})),
					),
				),

				Entry("should allow config when permittedPeer is specified",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), PermittedPeer: []string{"peer1", "peer2"}},
					BeEmpty(),
				),

				Entry("should forbid config if any permittedPeer is empty",
					rsyslog.TLS{Enabled: true, SecretReferenceName: ptr.To("secret-name"), PermittedPeer: []string{"peer1", ""}},
					ConsistOf(
						PointTo(MatchFields(IgnoreExtras, Fields{
							"Type":     Equal(field.ErrorTypeRequired),
							"Field":    Equal("tls.permittedPeer[1]"),
							"BadValue": Equal(""),
							"Detail":   Equal("value cannot be empty"),
						})),
					),
				),
			)
		})
	})

	Describe("#ValidateAuditd", func() {
		It("should not allow setting empty audit rules", func() {
			config := rsyslog.Auditd{
				AuditRules: "",
			}

			matcher := ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("auditRules"),
					"BadValue": Equal(""),
					"Detail":   Equal("auditRules must not be empty"),
				})),
			)

			errorList := validation.ValidateAuditd(&config)
			Expect(errorList).To(matcher)
		})

		It("should allow correct auditd configuration", func() {
			config := rsyslog.Auditd{
				AuditRules: "-a exit,always -F arch=b64 -S setuid -S setreuid -S setgid -S setregid -F auid>0 -F auid!=-1 -F key=privilege_escalation",
			}

			errorList := validation.ValidateAuditd(&config)
			Expect(errorList).To(BeEmpty())
		})
	})
})
