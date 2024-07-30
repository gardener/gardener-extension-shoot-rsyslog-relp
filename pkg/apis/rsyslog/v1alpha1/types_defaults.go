// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import "k8s.io/utils/ptr"

// SetDefaults_RsyslogRelpConfig sets defaults for the rsyslog relp config.
func SetDefaults_RsyslogRelpConfig(obj *RsyslogRelpConfig) {
	if obj.AuditConfig == nil {
		obj.AuditConfig = &AuditConfig{}
	}
}

// SetDefaults_AuditConfig sets default for the audit config.
func SetDefaults_AuditConfig(obj *AuditConfig) {
	if obj.Enabled == nil {
		obj.Enabled = ptr.To(true)
	}
}
