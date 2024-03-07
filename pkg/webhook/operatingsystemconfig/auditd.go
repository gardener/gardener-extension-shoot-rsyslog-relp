// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	_ "embed"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"k8s.io/utils/ptr"
)

const (
	baseConfigRulesPath          = "/var/lib/rsyslog-relp-configurator/audit/rules.d/00-base-config.rules"
	privilegeEscalationRulesPath = "/var/lib/rsyslog-relp-configurator/audit/rules.d/10-privilege-escalation.rules"
	privilegeSpecialRulesPath    = "/var/lib/rsyslog-relp-configurator/audit/rules.d/11-privileged-special.rules"
	systemIntegrityRulesPath     = "/var/lib/rsyslog-relp-configurator/audit/rules.d/12-system-integrity.rules"
)

var (
	//go:embed resources/auditrules/00-base-config.rules
	baseConfigRules []byte
	//go:embed resources/auditrules/10-privilege-escalation.rules
	privilegeEscalationRules []byte
	//go:embed resources/auditrules/11-privileged-special.rules
	privilegeSpecialRules []byte
	//go:embed resources/auditrules/12-system-integrity.rules
	systemIntegrityRules []byte
)

func getAuditdFiles() []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        baseConfigRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(baseConfigRules),
				},
			},
		},
		{
			Path:        privilegeEscalationRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(privilegeEscalationRules),
				},
			},
		},
		{
			Path:        privilegeSpecialRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(privilegeSpecialRules),
				},
			},
		},
		{
			Path:        systemIntegrityRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(systemIntegrityRules),
				},
			},
		},
	}
}
