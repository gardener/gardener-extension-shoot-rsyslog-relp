// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

// ValidateAuditd validates the passed configuration instance.
func ValidateAuditd(auditd *rsyslog.Auditd) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(auditd.AuditRules) == 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("auditRules"), auditd.AuditRules, "auditRules must not be empty"))
	}

	return allErrs
}
