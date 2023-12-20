// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import "k8s.io/utils/pointer"

// SetDefaults_RsyslogRelpConfig sets defaults for the rsyslog relp config.
func SetDefaults_RsyslogRelpConfig(obj *RsyslogRelpConfig) {
	if obj.AuditRulesConfig == nil {
		obj.AuditRulesConfig = &AuditRulesConfig{}
	}
}

// SetDefaults_AuditRulesConfig sets default for the audit rules config.
func SetDefaults_AuditRulesConfig(obj *AuditRulesConfig) {
	if obj.Enabled == nil {
		obj.Enabled = pointer.Bool(true)
	}
}
