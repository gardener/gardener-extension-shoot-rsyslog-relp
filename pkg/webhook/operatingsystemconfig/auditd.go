// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"context"
	_ "embed"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

const (
	baseConfigRulesPath          = "/var/lib/rsyslog-relp-configurator/audit/rules.d/00-base-config.rules"
	privilegeEscalationRulesPath = "/var/lib/rsyslog-relp-configurator/audit/rules.d/10-privilege-escalation.rules"
	privilegeSpecialRulesPath    = "/var/lib/rsyslog-relp-configurator/audit/rules.d/11-privileged-special.rules"
	systemIntegrityRulesPath     = "/var/lib/rsyslog-relp-configurator/audit/rules.d/12-system-integrity.rules"
)

var (
	//go:embed resources/auditrules/00-base-config.rules
	baseConfigRules []byte
	//go:embed resources/auditrules/10-privilege-escalation.rules
	privilegeEscalationRules []byte
	//go:embed resources/auditrules/11-privileged-special.rules
	privilegeSpecialRules []byte
	//go:embed resources/auditrules/12-system-integrity.rules
	systemIntegrityRules []byte
)

func getAuditdFiles(ctx context.Context, c client.Client, decoder runtime.Decoder, namespace string, rsyslogRelpConfig *rsyslog.RsyslogRelpConfig, cluster *extensionscontroller.Cluster) ([]extensionsv1alpha1.File, error) {
	if rsyslogRelpConfig.AuditConfig != nil && rsyslogRelpConfig.AuditConfig.ConfigMapReferenceName != nil {
		return getAuditConfigFromConfigMap(ctx, c, decoder, cluster, namespace, *rsyslogRelpConfig.AuditConfig.ConfigMapReferenceName)
	}

	return getDefaultAuditRules(), nil
}

func getAuditConfigFromConfigMap(ctx context.Context, c client.Client, decoder runtime.Decoder, cluster *extensionscontroller.Cluster, namespace, configMapRefName string) ([]extensionsv1alpha1.File, error) {
	ref := v1beta1helper.GetResourceByName(cluster.Shoot.Spec.Resources, configMapRefName)
	if ref == nil || ref.ResourceRef.Kind != "ConfigMap" {
		return nil, fmt.Errorf("failed to find referenced resource with name %s and kind ConfigMap", configMapRefName)
	}

	refConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ref.ResourceRef.Name,
			Namespace: namespace,
		},
	}
	if err := extensionscontroller.GetObjectByReference(ctx, c, &ref.ResourceRef, namespace, refConfigMap); err != nil {
		return nil, fmt.Errorf("failed to read referenced configMap %s%s for reference %s", v1beta1constants.ReferencedResourcesPrefix, ref.ResourceRef.Name, configMapRefName)
	}

	auditConfigString, ok := refConfigMap.Data[constants.AuditdConfigMapDataKey]
	if !ok {
		return nil, fmt.Errorf("missing 'data.%s' field in configMap %s%s", constants.AuditdConfigMapDataKey, v1beta1constants.ReferencedResourcesPrefix, ref.ResourceRef.Name)
	}
	auditdConfig, err := parseAuditConfig(decoder, auditConfigString)
	if err != nil {
		return nil, err
	}

	return []extensionsv1alpha1.File{{
		Path:        fmt.Sprintf("%s/%s", auditRulesFromOSCDir, "00_shoot_rsyslog_relp.rules"),
		Permissions: ptr.To(int32(0644)),
		Content: extensionsv1alpha1.FileContent{
			Inline: &extensionsv1alpha1.FileContentInline{
				Encoding: "b64",
				Data:     gardenerutils.EncodeBase64(auditdConfig),
			},
		},
	}}, nil
}

func getDefaultAuditRules() []extensionsv1alpha1.File {
	return []extensionsv1alpha1.File{
		{
			Path:        baseConfigRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(baseConfigRules),
				},
			},
		},
		{
			Path:        privilegeEscalationRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(privilegeEscalationRules),
				},
			},
		},
		{
			Path:        privilegeSpecialRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(privilegeSpecialRules),
				},
			},
		},
		{
			Path:        systemIntegrityRulesPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(systemIntegrityRules),
				},
			},
		},
	}
}

func parseAuditConfig(decoder runtime.Decoder, data string) ([]byte, error) {
	auditdConfig := &rsyslog.Auditd{}
	_, _, err := decoder.Decode([]byte(data), nil, auditdConfig)
	if err != nil {
		return nil, err
	}
	return []byte(auditdConfig.AuditRules), nil
}
