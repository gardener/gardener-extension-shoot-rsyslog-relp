// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package constants

const (
	// ExtensionType is the name of the extension type.
	ExtensionType = "shoot-rsyslog-relp"
	// ServiceName is the name of the service.
	ServiceName = "shoot-rsyslog-relp"

	// Origin is the origin used for the shoot-rsyslog-relp ManagedResources.
	Origin = "shoot-rsyslog-relp"
	// ManagedResourceName is the name used to describe the managed shoot resources.
	ManagedResourceName = "extension-" + ServiceName + "-shoot"

	// RsyslogCertifcateAuthorityKey is a key in a secret's data which holds the certificate authority used for the tls connection.
	RsyslogCertifcateAuthorityKey = "ca"
	// RsyslogClientCertificateKey is a key in a secret's data which holds the client certificate used for the tls connection.
	RsyslogClientCertificateKey = "crt"
	// RsyslogPrivateKeyKey is a key in a secret's data which holds the private key used for the tls connection.
	RsyslogPrivateKeyKey = "key"

	// AuditdConfigMapDataKey is a key in a ConfigMap's data which holds the configuration for the auditd service.
	AuditdConfigMapDataKey = "auditd"
	// RsyslogOSCDir is the path where node-agent will put rsyslog files from the OSC
	RsyslogOSCDir = "/var/lib/rsyslog-relp-configurator"

	// ConfigureRsyslogScriptPath is the path where node-agent will put the rsyslog configuration script from the OSC
	ConfigureRsyslogScriptPath = RsyslogOSCDir + "/configure-rsyslog.sh"
	// ProcessRsyslogPstatsScriptPath is the path where node-agent will put the rsyslog pstats script from the OSC
	ProcessRsyslogPstatsScriptPath = RsyslogOSCDir + "/process-rsyslog-pstats.sh"
	// RsyslogConfigFromOSCPath is the path where node-agent will put rsyslog audit config file from the OSC
	RsyslogConfigFromOSCPath = RsyslogOSCDir + "/rsyslog.d/60-audit.conf"
	// RsyslogConfigPath is the path where rsyslog audit config file will be placed
	RsyslogConfigPath = "/etc/rsyslog.d/60-audit.conf"
	// RsyslogTLSFromOSCDir is the path where node-agent will put rsyslog tls files from the OSC
	RsyslogTLSFromOSCDir = RsyslogOSCDir + "/tls"
	// RsyslogTLSDir is the path where tls files for rsyslog will be placed
	RsyslogTLSDir = "/etc/ssl/rsyslog"
	// RsyslogRelpQueueSpoolDir is the path for the rsyslog queue spool directory
	RsyslogRelpQueueSpoolDir = "/var/log/rsyslog"

	// AuditRulesFromOSCDir is the path where node-agent will put the audit rule files from the OSC
	AuditRulesFromOSCDir = RsyslogOSCDir + "/audit/rules.d"
	// AuditRulesDir is the path for where the audit rules will be places
	AuditRulesDir = "/etc/audit/rules.d"

	// AuditRulesBackupDir is the path for where the audit rules will be backed up
	AuditRulesBackupDir = "/etc/audit/rules.d.original"
	// AuditSyslogPluginPath is the path where the audit syslog plugin is expected to be
	AuditSyslogPluginPath = "/etc/audit/plugins.d/syslog.conf"
	// AudispSyslogPluginPath is the path where the audisp syslog plugin is expected to be
	AudispSyslogPluginPath = "/etc/audisp/plugins.d/syslog.conf"
)
