// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"bytes"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerutils "github.com/gardener/gardener/pkg/utils"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/utils"
)

const (
	rsyslogOSCDir = "/var/lib/rsyslog-relp-configurator"

	rsyslogTLSDir        = "/etc/ssl/rsyslog"
	rsyslogTLSFromOSCDir = rsyslogOSCDir + "/tls"

	rsyslogConfigPath              = "/etc/rsyslog.d/60-audit.conf"
	rsyslogConfigFromOSCPath       = rsyslogOSCDir + "/rsyslog.d/60-audit.conf"
	configureRsyslogScriptPath     = rsyslogOSCDir + "/configure-rsyslog.sh"
	processRsyslogPstatsScriptPath = rsyslogOSCDir + "/process-rsyslog-pstats.sh"

	rsyslogRelpQueueSpoolDir = "/var/log/rsyslog"

	auditRulesDir          = "/etc/audit/rules.d"
	auditRulesBackupDir    = "/etc/audit/rules.d.original"
	auditSyslogPluginPath  = "/etc/audit/plugins.d/syslog.conf"
	audispSyslogPluginPath = "/etc/audisp/plugins.d/syslog.conf"
	auditRulesFromOSCDir   = rsyslogOSCDir + "/audit/rules.d"

	nodeExporterTextfileCollectorDir = "/var/lib/node-exporter/textfile-collector"
)

var (
	//go:embed resources/templates/60-audit.conf.tpl
	rsyslogAuditConfigTemplateContent string
	rsyslogAuditConfigTemplate        *template.Template

	//go:embed resources/templates/scripts/configure-rsyslog.tpl.sh
	configureRsyslogScriptTemplateContent string
	configureRsyslogScript                bytes.Buffer

	//go:embed resources/templates/scripts/process-rsyslog-pstats.tpl.sh
	processRsyslogPstatsScriptTemplateContent string
	processRsyslogPstatsScript                bytes.Buffer
)

func init() {
	var err error
	rsyslogAuditConfigTemplate, err = template.
		New("60-auditd.conf").
		Funcs(sprig.TxtFuncMap()).
		Parse(rsyslogAuditConfigTemplateContent)
	if err != nil {
		panic(err)
	}

	configureRsyslogScriptTemplate, err := template.
		New("configure-rsyslog.sh").
		Funcs(sprig.TxtFuncMap()).
		Parse(configureRsyslogScriptTemplateContent)
	if err != nil {
		panic(err)
	}

	if err := configureRsyslogScriptTemplate.Execute(&configureRsyslogScript, map[string]interface{}{
		"rsyslogRelpQueueSpoolDir":    rsyslogRelpQueueSpoolDir,
		"pathRsyslogTLSDir":           rsyslogTLSDir,
		"pathRsyslogTLSFromOSCDir":    rsyslogTLSFromOSCDir,
		"pathAuditRulesDir":           auditRulesDir,
		"pathAuditRulesBackupDir":     auditRulesBackupDir,
		"pathAuditRulesFromOSCDir":    auditRulesFromOSCDir,
		"pathSyslogAuditPlugin":       auditSyslogPluginPath,
		"audispSyslogPluginPath":      audispSyslogPluginPath,
		"pathRsyslogAuditConf":        rsyslogConfigPath,
		"pathRsyslogAuditConfFromOSC": rsyslogConfigFromOSCPath,
	}); err != nil {
		panic(err)
	}

	processRsyslogPstatsScriptTemplate, err := template.
		New("process-rsyslog-pstats.sh").
		Funcs(sprig.TxtFuncMap()).
		Parse(processRsyslogPstatsScriptTemplateContent)
	if err != nil {
		panic(err)
	}

	if err := processRsyslogPstatsScriptTemplate.Execute(&processRsyslogPstatsScript, map[string]interface{}{
		"rsyslogPstatsTextfileDir": nodeExporterTextfileCollectorDir,
	}); err != nil {
		panic(err)
	}
}

func getRsyslogFiles(rsyslogRelpConfig *rsyslog.RsyslogRelpConfig, cluster *extensionscontroller.Cluster) ([]extensionsv1alpha1.File, error) {
	var rsyslogFiles []extensionsv1alpha1.File

	rsyslogValues := getRsyslogValues(rsyslogRelpConfig, cluster)

	if rsyslogRelpConfig.TLS != nil && rsyslogRelpConfig.TLS.Enabled {
		rsyslogValues["tls"] = getRsyslogTLSValues(rsyslogRelpConfig)
		rsyslogTLSFiles, err := getRsyslogTLSFiles(cluster, *rsyslogRelpConfig.TLS.SecretReferenceName)
		if err != nil {
			return nil, err
		}
		rsyslogFiles = append(rsyslogFiles, rsyslogTLSFiles...)
	}

	var config bytes.Buffer
	if err := rsyslogAuditConfigTemplate.Execute(&config, rsyslogValues); err != nil {
		return nil, err
	}

	rsyslogFiles = append(rsyslogFiles, []extensionsv1alpha1.File{
		{
			Path:        rsyslogConfigFromOSCPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(config.Bytes()),
				},
			},
		},
		{
			Path:        configureRsyslogScriptPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(configureRsyslogScript.Bytes()),
				},
			},
		},
		{
			Path:        processRsyslogPstatsScriptPath,
			Permissions: ptr.To(int32(0744)),
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: "b64",
					Data:     gardenerutils.EncodeBase64(processRsyslogPstatsScript.Bytes()),
				},
			},
		},
	}...)

	return rsyslogFiles, nil
}

func getRsyslogValues(rsyslogRelpConfig *rsyslog.RsyslogRelpConfig, cluster *extensionscontroller.Cluster) map[string]interface{} {
	projectName := utils.ProjectName(cluster.ObjectMeta.Name, cluster.Shoot.Name)

	var reportSuspensionContinuation *string
	if rsyslogRelpConfig.ReportSuspensionContinuation != nil {
		if *rsyslogRelpConfig.ReportSuspensionContinuation {
			reportSuspensionContinuation = ptr.To("on")
		} else {
			reportSuspensionContinuation = ptr.To("off")
		}
	}

	filters := computeLogFilters(rsyslogRelpConfig.LoggingRules)

	return map[string]interface{}{
		"target":                       rsyslogRelpConfig.Target,
		"port":                         rsyslogRelpConfig.Port,
		"rsyslogRelpQueueSpoolDir":     rsyslogRelpQueueSpoolDir,
		"projectName":                  projectName,
		"shootName":                    cluster.Shoot.Name,
		"shootUID":                     cluster.Shoot.UID,
		"filters":                      filters,
		"rebindInterval":               rsyslogRelpConfig.RebindInterval,
		"timeout":                      rsyslogRelpConfig.Timeout,
		"resumeRetryCount":             rsyslogRelpConfig.ResumeRetryCount,
		"reportSuspensionContinuation": reportSuspensionContinuation,
	}
}

func getRsyslogTLSValues(rsyslogRelpConfig *rsyslog.RsyslogRelpConfig) map[string]interface{} {
	var permittedPeers []string
	for _, permittedPeer := range rsyslogRelpConfig.TLS.PermittedPeer {
		permittedPeers = append(permittedPeers, strconv.Quote(permittedPeer))
	}

	var authMode string
	if rsyslogRelpConfig.TLS.AuthMode != nil {
		authMode = string(*rsyslogRelpConfig.TLS.AuthMode)
	}

	var tlsLib string
	if rsyslogRelpConfig.TLS.TLSLib != nil {
		tlsLib = string(*rsyslogRelpConfig.TLS.TLSLib)
	}

	return map[string]interface{}{
		"caPath":        rsyslogTLSDir + "/ca.crt",
		"certPath":      rsyslogTLSDir + "/tls.crt",
		"keyPath":       rsyslogTLSDir + "/tls.key",
		"enabled":       rsyslogRelpConfig.TLS.Enabled,
		"permittedPeer": strings.Join(permittedPeers, ","),
		"authMode":      authMode,
		"tlsLib":        tlsLib,
	}
}

func getRsyslogTLSFiles(cluster *extensionscontroller.Cluster, secretRefName string) ([]extensionsv1alpha1.File, error) {
	ref := v1beta1helper.GetResourceByName(cluster.Shoot.Spec.Resources, secretRefName)
	if ref == nil || ref.ResourceRef.Kind != "Secret" {
		return nil, fmt.Errorf("failed to find referenced resource with name %s and kind Secret", secretRefName)
	}

	refSecretName := v1beta1constants.ReferencedResourcesPrefix + ref.ResourceRef.Name
	return []extensionsv1alpha1.File{
		{
			Path:        rsyslogTLSFromOSCDir + "/ca.crt",
			Permissions: ptr.To(int32(0600)),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    refSecretName,
					DataKey: constants.RsyslogCertifcateAuthorityKey,
				},
			},
		},
		{
			Path:        rsyslogTLSFromOSCDir + "/tls.crt",
			Permissions: ptr.To(int32(0600)),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    refSecretName,
					DataKey: constants.RsyslogClientCertificateKey,
				},
			},
		},
		{
			Path:        rsyslogTLSFromOSCDir + "/tls.key",
			Permissions: ptr.To(int32(0600)),
			Content: extensionsv1alpha1.FileContent{
				SecretRef: &extensionsv1alpha1.FileContentSecretRef{
					Name:    refSecretName,
					DataKey: constants.RsyslogPrivateKeyKey,
				},
			},
		},
	}, nil
}

func getRsyslogConfiguratorUnit() extensionsv1alpha1.Unit {
	return extensionsv1alpha1.Unit{
		Name:    "rsyslog-configurator.service",
		Command: ptr.To(extensionsv1alpha1.CommandStart),
		Enable:  ptr.To(true),
		Content: ptr.To(`[Unit]
Description=rsyslog configurator daemon
Documentation=https://github.com/gardener/gardener-extension-shoot-rsyslog-relp
[Service]
Type=simple
Restart=always
RestartSec=15
ExecStart=` + configureRsyslogScriptPath + `
[Install]
WantedBy=multi-user.target`),
	}
}

func computeLogFilters(loggingRules []rsyslog.LoggingRule) []string {
	var filters []string
	for _, rule := range loggingRules {
		var programNames []string
		for _, programName := range rule.ProgramNames {
			programNames = append(programNames, strconv.Quote(programName))
		}
		if len(programNames) > 0 {
			filters = append(filters, fmt.Sprintf("$programname == [%s] and $syslogseverity <= %d", strings.Join(programNames, ","), rule.Severity))
		} else {
			filters = append(filters, fmt.Sprintf("$syslogseverity <= %d", rule.Severity))
		}
	}

	return filters
}
