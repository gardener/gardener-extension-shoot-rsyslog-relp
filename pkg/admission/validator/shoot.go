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
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/validation"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/utils"
)

// shoot validates shoots
type shoot struct {
	client  client.Client
	decoder runtime.Decoder
}

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &shoot{
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

// Validate validates the given shoot object.
func (s *shoot) Validate(ctx context.Context, new, old client.Object) error {
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
	rsyslogRelpConfig, err := decodeRsyslogRelpConfig(s.decoder, ext.ProviderConfig, providerConfigPath)
	if err != nil {
		return err
	}

	if err = validation.ValidateRsyslogRelpConfig(rsyslogRelpConfig, providerConfigPath).ToAggregate(); err != nil {
		return err
	}

	if rsyslogRelpConfig.TLS != nil && rsyslogRelpConfig.TLS.Enabled {
		secretName, err := getReferencedSecretName(shoot, *rsyslogRelpConfig.TLS.SecretReferenceName)
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
		if err := s.client.Get(ctx, secretKey, secret); err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("referenced secret %s does not exist", secretKey.String())
			}

			return fmt.Errorf("failed to get referenced secret %s with error: %w", secretKey.String(), err)
		}

		if err := utils.ValidateRsyslogRelpSecret(secret); err != nil {
			return err
		}
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

func decodeRsyslogRelpConfig(decoder runtime.Decoder, config *runtime.RawExtension, fldPath *field.Path) (*rsyslog.RsyslogRelpConfig, error) {
	if config == nil {
		return nil, field.Required(fldPath, "Rsyslog relp configuration is required when using gardener-extension-shoot-rsyslog-relp")
	}

	rsyslogRelpConfig := &rsyslog.RsyslogRelpConfig{}
	if err := runtime.DecodeInto(decoder, config.Raw, rsyslogRelpConfig); err != nil {
		return nil, err
	}

	return rsyslogRelpConfig, nil
}

func getReferencedSecretName(shoot *core.Shoot, secretReferenceName string) (string, error) {
	if shoot != nil {
		for _, ref := range shoot.Spec.Resources {
			if ref.Name == secretReferenceName {
				if ref.ResourceRef.Kind != "Secret" {
					return "", fmt.Errorf("invalid referenced resource, expected kind Secret, not %s: %s", ref.ResourceRef.Kind, ref.ResourceRef.Name)
				}
				return ref.ResourceRef.Name, nil
			}
		}
	}
	return "", fmt.Errorf("missing or invalid referenced resource: %s", secretReferenceName)
}
