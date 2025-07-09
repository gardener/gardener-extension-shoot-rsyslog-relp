// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// SetDefaults_RsyslogRelpConfig sets defaults for the rsyslog relp config.
func SetDefaults_RsyslogRelpConfig(obj *RsyslogRelpConfig) {
	if obj.AuditConfig == nil {
		obj.AuditConfig = &AuditConfig{
			Enabled: true,
		}
	}
}
