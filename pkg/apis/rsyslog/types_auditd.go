// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rsyslog

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Auditd contains configuration for the audit daemon.
type Auditd struct {
	metav1.TypeMeta

	// AuditRules contains the audit rules that will be placed under /etc/audit/rules.d.
	AuditRules string
}