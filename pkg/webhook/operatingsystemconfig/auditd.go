// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"k8s.io/utils/pointer"
)

const (
	baseConfigRulesPath          = "/var/lib/rsyslog-relp-configurator/audit/rules.d/00-base-config.rules"
	privilegeEscalationRulesPath = "/var/lib/rsyslog-relp-configurator/audit/rules.d/10-privilege-escalation.rules"
	privilegeSpecialRulesPath    = "/var/lib/rsyslog-relp-configurator/audit/rules.d/11-privileged-special.rules"
	systemIntegrityRulesPath     = "/var/lib/rsyslog-relp-configurator/audit/rules.d/12-system-integrity.rules"

	baseConfigRules = `## First rule - delete all
-D
## Increase the buffers to survive stress events.
## Make this bigger for busy systems
-b 8192
## This determine how long to wait in burst of events
--backlog_wait_time 60000
## Set failure mode to syslog
-f 1`
	privilegeEscalationRules = `-a exit,always -F arch=b64 -S setuid -S setreuid -S setgid -S setregid -F auid>0 -F auid!=-1 -F key=privilege_escalation
-a exit,always -F arch=b64 -S execve -S execveat -F euid=0 -F auid>0 -F auid!=-1 -F key=privilege_escalation`

	privilegeSpecialRules = `-a exit,always -F arch=b64 -S mount -S mount_setattr -S umount2 -S mknod -S mknodat -S chroot -F auid!=-1 -F key=privileged_special`

	systemIntegrityRules = `-a exit,always -F dir=/boot -F perm=wa -F key=system_integrity
-a exit,always -F dir=/etc -F perm=wa -F key=system_integrity
-a exit,always -F dir=/bin -F perm=wa -F key=system_integrity
-a exit,always -F dir=/sbin -F perm=wa -F key=system_integrity
-a exit,always -F dir=/lib -F perm=wa -F key=system_integrity
-a exit,always -F dir=/lib64 -F perm=wa -F key=system_integrity
-a exit,always -F dir=/usr -F perm=wa -F key=system_integrity
-a exit,always -F dir=/opt -F perm=wa -F key=system_integrity
-a exit,always -F dir=/root -F perm=wa -F key=system_integrity`
)

func getAuditdFiles() []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        baseConfigRulesPath,
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64([]byte(baseConfigRules)),
				},
			},
		},
		{
			Path:        privilegeEscalationRulesPath,
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64([]byte(privilegeEscalationRules)),
				},
			},
		},
		{
			Path:        privilegeSpecialRulesPath,
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64([]byte(privilegeSpecialRules)),
				},
			},
		},
		{
			Path:        systemIntegrityRulesPath,
			Permissions: pointer.Int32(0744),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64([]byte(systemIntegrityRules)),
				},
			},
		},
	}
}
