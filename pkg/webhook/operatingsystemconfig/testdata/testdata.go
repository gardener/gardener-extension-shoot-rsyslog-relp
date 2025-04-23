// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	_ "embed"
	"encoding/base64"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"k8s.io/utils/ptr"
)

var (
	//go:embed 60-audit.conf
	rsyslogConfig []byte
	//go:embed 60-audit-with-tls.conf
	rsyslogConfigWithTLS []byte
	//go:embed rsyslog-config-simple.conf.tpl
	rsyslogConfigSimple []byte

	//go:embed configure-rsyslog.sh
	confiugreRsyslogScript []byte
	//go:embed process-rsyslog-pstats.sh
	processRsyslogPstatsScript []byte

	//go:embed 00-base-config.rules
	baseConfigRules []byte
	//go:embed 10-privilege-escalation.rules
	privilegeEscalationRules []byte
	//go:embed 11-privileged-special.rules
	privilegeSpecialRules []byte
	//go:embed 12-system-integrity.rules
	systemIntegrityRules []byte
)

// GetAuditRulesFiles returns default Audit rules files
func GetAuditRulesFiles(useExpectedContent bool) []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/00-base-config.rules",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(GetBasedOnCondition(useExpectedContent, baseConfigRules, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/10-privilege-escalation.rules",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(GetBasedOnCondition(useExpectedContent, privilegeEscalationRules, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/11-privileged-special.rules",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(GetBasedOnCondition(useExpectedContent, privilegeSpecialRules, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/audit/rules.d/12-system-integrity.rules",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(GetBasedOnCondition(useExpectedContent, systemIntegrityRules, []byte("oldContent"))),
				},
			},
		},
	}
}

// GetRsyslogFiles returns default Rsyslog files
func GetRsyslogFiles(rsyslogConfig []byte, useExpectedContent bool) []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        "/var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     base64.StdEncoding.EncodeToString(GetBasedOnCondition(useExpectedContent, rsyslogConfig, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     base64.StdEncoding.EncodeToString(GetBasedOnCondition(useExpectedContent, confiugreRsyslogScript, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/process-rsyslog-pstats.sh",
			Permissions: ptr.To(uint32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     base64.StdEncoding.EncodeToString(GetBasedOnCondition(useExpectedContent, processRsyslogPstatsScript, []byte("oldContent"))),
				},
			},
		},
		{
			Path:        "/etc/systemd/system/rsyslog.service.d/10-shoot-rsyslog-relp-memory-limits.conf",
			Permissions: ptr.To(uint32(0644)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Data: GetBasedOnCondition(useExpectedContent, `[Service]
MemoryMin=15M
MemoryHigh=150M
MemoryMax=300M
MemorySwapMax=0`, "old"),
				},
			},
		},
	}
}

// GetRsyslogTLSFiles returns default Rsyslog TLS files
func GetRsyslogTLSFiles(useExpectedContent bool) []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        "/var/lib/rsyslog-relp-configurator/tls/ca.crt",
			Permissions: ptr.To(uint32(0600)),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    GetBasedOnCondition(useExpectedContent, "ref-rsyslog-tls", "ref-rsyslog-tls-old"),
					DataKey: "ca",
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/tls/tls.crt",
			Permissions: ptr.To(uint32(0600)),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    GetBasedOnCondition(useExpectedContent, "ref-rsyslog-tls", "ref-rsyslog-tls-old"),
					DataKey: "crt",
				},
			},
		},
		{
			Path:        "/var/lib/rsyslog-relp-configurator/tls/tls.key",
			Permissions: ptr.To(uint32(0600)),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    GetBasedOnCondition(useExpectedContent, "ref-rsyslog-tls", "ref-rsyslog-tls-old"),
					DataKey: "key",
				},
			},
		},
	}
}

// GetRsyslogConfiguratorUnit returns the Rsyslog configuration unit
func GetRsyslogConfiguratorUnit(useExpectedContent bool) extensionsv1alpha1.Unit {
	return extensionsv1alpha1.Unit{
		Name:    "rsyslog-configurator.service",
		Command: ptr.To(extensionsv1alpha1.CommandStart),
		Enable:  ptr.To(true),
		Content: ptr.To(GetBasedOnCondition(useExpectedContent, `[Unit]
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

// GetBasedOnCondition returns one of two values based on a condition
func GetBasedOnCondition[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

// GetSimpleRsyslogConfig returns a simple rsyslog config with only a target and a port set
func GetSimpleRsyslogConfig() []byte {
	return rsyslogConfigSimple
}

// GetRsyslogConfigWithTLS returns an rsyslog config with TLS enabled
func GetRsyslogConfigWithTLS() []byte {
	return rsyslogConfigWithTLS
}

// GetTestingRsyslogConfig returns a custom rsyslog config for testing optional additions
func GetTestingRsyslogConfig() []byte {
	return rsyslogConfig
}
