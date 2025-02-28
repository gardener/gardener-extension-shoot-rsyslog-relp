// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rsyslog

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RsyslogRelpConfig configuration resource.
type RsyslogRelpConfig struct {
	metav1.TypeMeta

	// Target is the target server to connect to via relp.
	Target string
	// Port is the TCP port to use when connecting to the target server.
	Port int
	// TLS hods the TLS config.
	TLS *TLS
	// LoggingRules contain a list of LoggingRules that are used to determine which logs are
	// sent to the target server by the the rsyslog relp action.
	LoggingRules []LoggingRule
	// RebindInterval is the rebind interval for the rsyslog relp action.
	RebindInterval *int
	// Timeout is the connection timeout for the rsyslog relp action.
	Timeout *int
	// ResumeRetryCount is the resume retry count for the rsyslog relp action.
	ResumeRetryCount *int
	// ReportSuspensionContinuation determines whether suspension continuation in the relp action
	// should be reported.
	ReportSuspensionContinuation *bool
	// AuditConfig contains configuration that can be used to setup node level auditing so that audit logs
	// can be forwarded via rsyslog to the target RELP server.
	AuditConfig *AuditConfig
}

// TLS contains options for the tls connection to the target server.
type TLS struct {
	// Enabled determines whether TLS encryption should be used for the connection
	// to the target server.
	Enabled bool
	// SecretReferenceName is the name of the reference for the secret
	// containing the certificates for the TLS connection when encryption is enabled.
	SecretReferenceName *string
	// PermittedPeer is the name of the rsyslog relp permitted peer.
	// Only peers which have been listed in this parameter may be connected to.
	PermittedPeer []string
	// AuthMode is the mode used for mutual authentication.
	// Possible values are "fingerprint" or "name".
	AuthMode *AuthMode
	// TLSLib specifies the tls library that will be used by librelp on the shoot nodes.
	// If the field is omitted, the librelp default is used.
	// Possible values are "openssl" or "gnutls".
	TLSLib *TLSLib
}

// LoggingRule contains options that determines which logs are sent to the target server.
type LoggingRule struct {
	// ProgramNames are the names of the programs for which logs are sent to the target server.
	ProgramNames []string
	// Severity determines which logs are sent to the target server based on their severity.
	Severity int
	// MessageContent contains messages, used to filter messages
	MessageContent *MessageContent
}

// AuditConfig contains options to configure the audit system.
type AuditConfig struct {
	// Enabled determines whether auditing configurations are applied to the nodes or not.
	// Will be defaulted to true, if AuditConfig is nil.
	Enabled bool
	// ConfigMapReferenceName is the name of the reference for the ConfigMap containing
	// auditing configuration to apply to shoot nodes.
	ConfigMapReferenceName *string
}

// AuthMode is the type of authentication mode that can be used for the rsyslog relp connection to the target server.
type AuthMode string

const (
	// AuthModeName specifies the rsyslog name authentication mode.
	AuthModeName AuthMode = "name"
	// AuthModeFingerPrint specifies the rsyslog fingerprint authentication mode.
	AuthModeFingerPrint AuthMode = "fingerprint"
)

// TLSLib is the tls library that is used by the librelp library on the shoot's nodes.
type TLSLib string

const (
	// TLSLibOpenSSL specifies the openssl tls library.
	TLSLibOpenSSL = "openssl"
	// TLSLibGnuTLS specifies the gnutls tls library.
	TLSLibGnuTLS = "gnutls"
)

type MessageContent struct {
	// Message that should be contained
	Regex *string
	// Message that shouldn't be contained
	Exclude *string
}
