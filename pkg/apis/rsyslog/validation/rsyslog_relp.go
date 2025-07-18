// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"regexp"
	"strconv"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

// ValidateRsyslogRelpConfig validates the passed configuration instance.
func ValidateRsyslogRelpConfig(config *rsyslog.RsyslogRelpConfig, _ *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateTarget(config.Target, field.NewPath("target"))...)
	allErrs = append(allErrs, validatePort(config.Port, field.NewPath("port"))...)
	allErrs = append(allErrs, validateTLS(config.TLS, field.NewPath("tls"))...)
	allErrs = append(allErrs, validateLoggingRules(config.LoggingRules, field.NewPath("loggingRules"))...)

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

var (
	availableAuthModes = sets.New(
		string(rsyslog.AuthModeName),
		string(rsyslog.AuthModeFingerPrint),
	)
	availableTLSLibs = sets.New(
		string(rsyslog.TLSLibOpenSSL),
		string(rsyslog.TLSLibGnuTLS),
	)
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

	if tls.TLSLib != nil && !availableTLSLibs.Has(string(*tls.TLSLib)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("tlsLib"), tls.TLSLib, sets.List(availableTLSLibs)))
	}

	for i, permittedPeer := range tls.PermittedPeer {
		if permittedPeer == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("permittedPeer").Index(i), "value cannot be empty"))
		}
	}

	return allErrs
}

func validateLoggingRules(loggingRules []rsyslog.LoggingRule, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(loggingRules) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "at least one logging rule is required"))
	} else {
		for index, rule := range loggingRules {
			if len(rule.ProgramNames) == 0 && rule.Severity == nil && rule.MessageContent == nil {
				allErrs = append(allErrs, field.Required(fldPath.Index(index), "at least one of .programNames, .messageContent, or .severity is required"))
			}
			if rule.MessageContent != nil {
				if rule.MessageContent.Regex == nil && rule.MessageContent.Exclude == nil {
					allErrs = append(allErrs, field.Required(fldPath.Index(index).Child("messageContent"), "either .regex or .exclude has to be provided"))
				}
				if err := validateRegex(rule.MessageContent.Regex); err != nil {
					allErrs = append(allErrs, field.Required(fldPath.Index(index).Child("messageContent").Child("regex"), fmt.Sprintf("not a valid POSIX ERE regular expression: %v", err)))
				}
				if err := validateRegex(rule.MessageContent.Exclude); err != nil {
					allErrs = append(allErrs, field.Required(fldPath.Index(index).Child("messageContent").Child("exclude"), fmt.Sprintf("not a valid POSIX ERE regular expression: %v", err)))
				}
			}
		}
	}

	return allErrs
}

func validateRegex(regex *string) error {
	if regex != nil {
		quotedRegex := strconv.Quote(*regex)
		_, err := regexp.CompilePOSIX(quotedRegex)
		return err
	}

	return nil
}
