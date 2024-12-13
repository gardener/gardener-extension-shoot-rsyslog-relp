// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/validation"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

// shoot validates shoots
type shoot struct {
	apiReader client.Reader
	decoder   runtime.Decoder
}

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(apiReader client.Reader, decoder runtime.Decoder) extensionswebhook.Validator {
	return &shoot{
		apiReader: apiReader,
		decoder:   decoder,
	}
}

// Validate validates the given shoot object.
func (s *shoot) Validate(ctx context.Context, new, _ client.Object) error {
	shoot, ok := new.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	var ext *core.Extension
	var fldPath *field.Path
	for i, ex := range shoot.Spec.Extensions {
		if ex.Type == constants.ExtensionType {
			ext = ex.DeepCopy()
			fldPath = field.NewPath("spec", "extensions").Index(i)
			break
		}
	}

	if !isExtensionEnabled(ext) {
		return nil
	}

	providerConfigPath := fldPath.Child("providerConfig")
	if ext.ProviderConfig == nil {
		return field.Required(providerConfigPath, "Rsyslog relp configuration is required when using gardener-extension-shoot-rsyslog-relp")
	}

	rsyslogRelpConfig := &rsyslog.RsyslogRelpConfig{}
	if err := runtime.DecodeInto(s.decoder, ext.ProviderConfig.Raw, rsyslogRelpConfig); err != nil {
		return fmt.Errorf("could not decode rsyslog relp configuration: %w", err)
	}

	if err := validation.ValidateRsyslogRelpConfig(rsyslogRelpConfig, providerConfigPath).ToAggregate(); err != nil {
		return err
	}

	if rsyslogRelpConfig.TLS != nil && rsyslogRelpConfig.TLS.Enabled {
		secretName, err := getReferencedResourceName(shoot, "Secret", *rsyslogRelpConfig.TLS.SecretReferenceName)
		if err != nil {
			return err
		}

		// validate the secret
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: shoot.Namespace,
			},
		}

		secretKey := client.ObjectKeyFromObject(secret)
		if err := s.apiReader.Get(ctx, secretKey, secret); err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("referenced secret %s does not exist", secretKey.String())
			}

			return fmt.Errorf("failed to get referenced secret %s with error: %w", secretKey.String(), err)
		}

		if err := validateRsyslogRelpSecret(secret); err != nil {
			return err
		}
	}

	if rsyslogRelpConfig.AuditConfig != nil && rsyslogRelpConfig.AuditConfig.Enabled && rsyslogRelpConfig.AuditConfig.ConfigMapReferenceName != nil {
		configMapName, err := getReferencedResourceName(shoot, "ConfigMap", *rsyslogRelpConfig.AuditConfig.ConfigMapReferenceName)
		if err != nil {
			return err
		}

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: shoot.Namespace,
			},
		}

		configMapKey := client.ObjectKeyFromObject(configMap)
		if err := s.apiReader.Get(ctx, configMapKey, configMap); err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("referenced configMap %s does not exist", configMapKey.String())
			}

			return fmt.Errorf("failed to get referenced configMap %s with error: %w", configMapKey.String(), err)
		}

		if err := validateAuditConfigMap(s.decoder, configMap); err != nil {
			return err
		}
	}

	return nil
}

// validateRsyslogRelpSecret validates the content of an rsyslog relp secret.
func validateRsyslogRelpSecret(secret *corev1.Secret) error {
	key := client.ObjectKeyFromObject(secret)
	if _, ok := secret.Data[constants.RsyslogCertifcateAuthorityKey]; !ok {
		return fmt.Errorf("secret %s is missing %s value", key.String(), constants.RsyslogCertifcateAuthorityKey)
	}
	if _, ok := secret.Data[constants.RsyslogClientCertificateKey]; !ok {
		return fmt.Errorf("secret %s is missing %s value", key.String(), constants.RsyslogClientCertificateKey)
	}
	if _, ok := secret.Data[constants.RsyslogPrivateKeyKey]; !ok {
		return fmt.Errorf("secret %s is missing %s value", key.String(), constants.RsyslogPrivateKeyKey)
	}
	if !ptr.Deref(secret.Immutable, false) {
		return fmt.Errorf("secret %s must be immutable", key.String())
	}
	if len(secret.Data) != 3 {
		return fmt.Errorf("secret %s should have only three data entries", key.String())
	}

	return nil
}

// validateAuditConfigMap validates the content of a configmap containing audit config.
func validateAuditConfigMap(decoder runtime.Decoder, configMap *corev1.ConfigMap) error {
	configMapKey := client.ObjectKeyFromObject(configMap)
	if !ptr.Deref(configMap.Immutable, false) {
		return fmt.Errorf("configMap %s must be immutable", configMapKey.String())
	}

	auditdConfigString, ok := configMap.Data[constants.AuditdConfigMapDataKey]
	if !ok {
		return fmt.Errorf("missing 'data.%s' field in configMap %s", constants.AuditdConfigMapDataKey, configMapKey.String())
	}
	if len(auditdConfigString) == 0 {
		return fmt.Errorf("empty auditd config. Provide non-empty auditd config in configMap %s", configMapKey.String())
	}

	auditdConfig := &rsyslog.Auditd{}

	if err := runtime.DecodeInto(decoder, []byte(auditdConfigString), auditdConfig); err != nil {
		return fmt.Errorf("could not decode 'data.%s' field of configMap %s: %w", constants.AuditdConfigMapDataKey, configMapKey.String(), err)
	}
	if err := validation.ValidateAuditd(auditdConfig).ToAggregate(); err != nil {
		return err
	}

	return nil
}

// isExtensionEnabled checks whether the passed extension is enabled or not.
func isExtensionEnabled(ext *core.Extension) bool {
	if ext == nil {
		return false
	}
	if ext.Disabled != nil {
		return !*ext.Disabled
	}
	return true
}

func getReferencedResourceName(shoot *core.Shoot, resourceKind, resourceName string) (string, error) {
	if shoot != nil {
		for _, ref := range shoot.Spec.Resources {
			if ref.Name == resourceName {
				if ref.ResourceRef.Kind != resourceKind {
					return "", fmt.Errorf("invalid referenced resource, expected kind %s, not %s: %s", resourceKind, ref.ResourceRef.Kind, ref.ResourceRef.Name)
				}
				return ref.ResourceRef.Name, nil
			}
		}
	}
	return "", fmt.Errorf("missing or invalid referenced resource: %s", resourceName)
}
