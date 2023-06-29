// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

// ValidateRsyslogRelpConfig validates the passed configuration instance.
func ValidateRsyslogRelpConfig(config *rsyslog.RsyslogRelpConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateTarget(config.Target, field.NewPath("target"))...)
	allErrs = append(allErrs, validatePort(config.Port, field.NewPath("port"))...)
	allErrs = append(allErrs, validateTLS(config.TLS, field.NewPath("tls"))...)
	allErrs = append(allErrs, validateFilters(config.Filters, field.NewPath("filters"))...)

	return allErrs
}

func validateTarget(target string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if target == "" {
		allErrs = append(allErrs, field.Required(fldPath, "target must not be empty"))
	}

	return allErrs
}

func validatePort(port int, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if port < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, port, "port cannot be less than 0"))
	}

	return allErrs
}

var availableAuthModes = sets.New(
	string(rsyslog.AuthModeName),
	string(rsyslog.AuthModeFingerPrint),
)

func validateTLS(tls *rsyslog.TLS, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if tls == nil {
		return allErrs
	}

	if tls.Enabled {
		if tls.SecretReferenceName == nil || *tls.SecretReferenceName == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("secretReferenceName"), "secretReferenceName must not be empty when tls is enabled"))
		}
	}

	if tls.AuthMode != nil && !availableAuthModes.Has(string(*tls.AuthMode)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("authMode"), tls.AuthMode, sets.List(availableAuthModes)))
	}

	for i, permittedPeer := range tls.PermittedPeer {
		if permittedPeer == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("permittedPeer").Index(i), "value cannot be empty"))
		}
	}

	return allErrs
}

func validateFilters(filters *string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if filters != nil && *filters == "" {
		allErrs = append(allErrs, field.Required(fldPath, "filters must not be empty"))
	}

	return allErrs
}
