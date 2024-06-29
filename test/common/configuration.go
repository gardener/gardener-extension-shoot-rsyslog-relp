// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"encoding/json"
	"slices"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	rsyslogv1alpha1 "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
)

// AddOrUpdateRsyslogRelpExtension adds or updates the shooot-rsyslog-relp extension with the given options to the given shoot.
func AddOrUpdateRsyslogRelpExtension(shoot *gardencorev1beta1.Shoot, opts ...func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig)) {
	defaultProviderConfig := &rsyslogv1alpha1.RsyslogRelpConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rsyslogv1alpha1.SchemeGroupVersion.String(),
			Kind:       "RsyslogRelpConfig",
		},
		Target: "10.2.64.54",
		Port:   80,
		LoggingRules: []rsyslogv1alpha1.LoggingRule{
			{
				ProgramNames: []string{"test-program"},
				Severity:     1,
			},
		},
	}

	for _, opt := range opts {
		opt(defaultProviderConfig)
	}

	providerConfigJSON, err := json.Marshal(&defaultProviderConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	extension := gardencorev1beta1.Extension{
		Type: "shoot-rsyslog-relp",
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigJSON,
		},
	}

	i := slices.IndexFunc(shoot.Spec.Extensions, func(ext gardencorev1beta1.Extension) bool {
		return ext.Type == "shoot-rsyslog-relp"
	})
	if i == -1 {
		shoot.Spec.Extensions = append(shoot.Spec.Extensions, extension)
	} else {
		shoot.Spec.Extensions[i] = extension
	}
}

// HasRsyslogRelpExtension returns whether the shoot has an extension of type shoot-rsyslog-relp.
func HasRsyslogRelpExtension(shoot *gardencorev1beta1.Shoot) bool {
	return slices.ContainsFunc(shoot.Spec.Extensions, func(ext gardencorev1beta1.Extension) bool {
		return ext.Type == "shoot-rsyslog-relp"
	})
}

// RemoveRsyslogRelpExtension removes the shoot-rsyslog-relp extension from the given shoot.
func RemoveRsyslogRelpExtension(shoot *gardencorev1beta1.Shoot) {
	shoot.Spec.Extensions = slices.DeleteFunc(shoot.Spec.Extensions, func(ext gardencorev1beta1.Extension) bool {
		return ext.Type == "shoot-rsyslog-relp"
	})
}

// AddOrUpdateRsyslogTLSSecretResource adds or updates the rsyslog relp tls secret resource reference to the given shoot.
func AddOrUpdateRsyslogTLSSecretResource(shoot *gardencorev1beta1.Shoot, secretRefName string) {
	resource := gardencorev1beta1.NamedResourceReference{
		Name: secretRefName,
		ResourceRef: autoscalingv1.CrossVersionObjectReference{
			Kind:       "Secret",
			APIVersion: "v1",
			Name:       "rsyslog-relp-tls",
		},
	}

	i := slices.IndexFunc(shoot.Spec.Resources, func(resource gardencorev1beta1.NamedResourceReference) bool {
		return resource.Name == secretRefName
	})

	if i == -1 {
		shoot.Spec.Resources = append(shoot.Spec.Resources, resource)
	} else {
		shoot.Spec.Resources[i] = resource
	}
}

// RemoveRsyslogTLSSecretResource removes the rsyslog relp tls secret resource reference from the given shoot.
func RemoveRsyslogTLSSecretResource(shoot *gardencorev1beta1.Shoot, secretRefName string) {
	shoot.Spec.Resources = slices.DeleteFunc(shoot.Spec.Resources, func(resource gardencorev1beta1.NamedResourceReference) bool {
		return resource.Name == secretRefName
	})
}

// HasRsyslogTLSSecretResource returns whether the shoot has an named resource reference with the given name.
func HasRsyslogTLSSecretResource(shoot *gardencorev1beta1.Shoot, secretRefName string) bool {
	return slices.ContainsFunc(shoot.Spec.Resources, func(resource gardencorev1beta1.NamedResourceReference) bool {
		return resource.Name == secretRefName
	})
}

// WithPort returns a function which sets the port of the rsyslog relp configuration to the given port.
func WithPort(port int) func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
	return func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
		rsyslogRelpConfig.Port = port
	}
}

// WithTarget returns a function which sets the target of the rsyslog relp configuration to the given target.
func WithTarget(target string) func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
	return func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
		rsyslogRelpConfig.Target = target
	}
}

// AppendLoggingRule appends the given loggingRule to the logging rules of the rsyslog relp configuration.
func AppendLoggingRule(loggingRule rsyslogv1alpha1.LoggingRule) func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
	return func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
		rsyslogRelpConfig.LoggingRules = append(rsyslogRelpConfig.LoggingRules, loggingRule)
	}
}

// WithTLSWithSecretRefNameAndTLSLib returns a function which enables TLS for the rsyslog relp configuration and sets
// the tls.secretRefName to the given secretRefName and tls.tlsLib to the given tlsLib.
func WithTLSWithSecretRefNameAndTLSLib(secretRefName, tlsLib string) func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
	return func(rsyslogRelpConfig *rsyslogv1alpha1.RsyslogRelpConfig) {
		var (
			authModeName  = rsyslogv1alpha1.AuthMode("name")
			tlsLibOpenSSL = rsyslogv1alpha1.TLSLib(tlsLib)
		)

		rsyslogRelpConfig.TLS = &rsyslogv1alpha1.TLS{
			Enabled:             true,
			SecretReferenceName: &secretRefName,
			PermittedPeer:       []string{"rsyslog-server"},
			AuthMode:            &authModeName,
			TLSLib:              &tlsLibOpenSSL,
		}
	}
}
