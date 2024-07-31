// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/validation"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

// ProjectName calculates the name of the project given the shoot namespace and shoot name.
func ProjectName(namespace, shootName string) string {
	var projectName = strings.TrimPrefix(namespace, "shoot-")
	projectName = strings.TrimSuffix(projectName, "-"+shootName)
	projectName = strings.Trim(projectName, "-")
	return projectName
}

// ValidateRsyslogRelpSecret validates the contents of a rsyslog relp secret.
func ValidateRsyslogRelpSecret(secret *corev1.Secret) error {
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

// ValidateAuditConfigMap validates the contents of a configmap containing audit config.
func ValidateAuditConfigMap(decoder runtime.Decoder, configMap *corev1.ConfigMap) error {
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

	_, _, err := decoder.Decode([]byte(auditdConfigString), nil, auditdConfig)
	if err != nil {
		return fmt.Errorf("could not decode 'data.%s' field of configMap %s: %w", constants.AuditdConfigMapDataKey, configMapKey.String(), err)
	}
	if err := validation.ValidateAuditd(auditdConfig, nil).ToAggregate(); err != nil {
		return err
	}

	return nil
}
