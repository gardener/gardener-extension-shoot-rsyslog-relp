//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	rsyslog "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*AuditConfig)(nil), (*rsyslog.AuditConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_AuditConfig_To_rsyslog_AuditConfig(a.(*AuditConfig), b.(*rsyslog.AuditConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*rsyslog.AuditConfig)(nil), (*AuditConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rsyslog_AuditConfig_To_v1alpha1_AuditConfig(a.(*rsyslog.AuditConfig), b.(*AuditConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Auditd)(nil), (*rsyslog.Auditd)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Auditd_To_rsyslog_Auditd(a.(*Auditd), b.(*rsyslog.Auditd), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*rsyslog.Auditd)(nil), (*Auditd)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rsyslog_Auditd_To_v1alpha1_Auditd(a.(*rsyslog.Auditd), b.(*Auditd), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*LoggingRule)(nil), (*rsyslog.LoggingRule)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_LoggingRule_To_rsyslog_LoggingRule(a.(*LoggingRule), b.(*rsyslog.LoggingRule), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*rsyslog.LoggingRule)(nil), (*LoggingRule)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rsyslog_LoggingRule_To_v1alpha1_LoggingRule(a.(*rsyslog.LoggingRule), b.(*LoggingRule), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MessageContent)(nil), (*rsyslog.MessageContent)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MessageContent_To_rsyslog_MessageContent(a.(*MessageContent), b.(*rsyslog.MessageContent), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*rsyslog.MessageContent)(nil), (*MessageContent)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rsyslog_MessageContent_To_v1alpha1_MessageContent(a.(*rsyslog.MessageContent), b.(*MessageContent), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RsyslogRelpConfig)(nil), (*rsyslog.RsyslogRelpConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RsyslogRelpConfig_To_rsyslog_RsyslogRelpConfig(a.(*RsyslogRelpConfig), b.(*rsyslog.RsyslogRelpConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*rsyslog.RsyslogRelpConfig)(nil), (*RsyslogRelpConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rsyslog_RsyslogRelpConfig_To_v1alpha1_RsyslogRelpConfig(a.(*rsyslog.RsyslogRelpConfig), b.(*RsyslogRelpConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*TLS)(nil), (*rsyslog.TLS)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_TLS_To_rsyslog_TLS(a.(*TLS), b.(*rsyslog.TLS), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*rsyslog.TLS)(nil), (*TLS)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_rsyslog_TLS_To_v1alpha1_TLS(a.(*rsyslog.TLS), b.(*TLS), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_AuditConfig_To_rsyslog_AuditConfig(in *AuditConfig, out *rsyslog.AuditConfig, s conversion.Scope) error {
	out.Enabled = in.Enabled
	out.ConfigMapReferenceName = (*string)(unsafe.Pointer(in.ConfigMapReferenceName))
	return nil
}

// Convert_v1alpha1_AuditConfig_To_rsyslog_AuditConfig is an autogenerated conversion function.
func Convert_v1alpha1_AuditConfig_To_rsyslog_AuditConfig(in *AuditConfig, out *rsyslog.AuditConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_AuditConfig_To_rsyslog_AuditConfig(in, out, s)
}

func autoConvert_rsyslog_AuditConfig_To_v1alpha1_AuditConfig(in *rsyslog.AuditConfig, out *AuditConfig, s conversion.Scope) error {
	out.Enabled = in.Enabled
	out.ConfigMapReferenceName = (*string)(unsafe.Pointer(in.ConfigMapReferenceName))
	return nil
}

// Convert_rsyslog_AuditConfig_To_v1alpha1_AuditConfig is an autogenerated conversion function.
func Convert_rsyslog_AuditConfig_To_v1alpha1_AuditConfig(in *rsyslog.AuditConfig, out *AuditConfig, s conversion.Scope) error {
	return autoConvert_rsyslog_AuditConfig_To_v1alpha1_AuditConfig(in, out, s)
}

func autoConvert_v1alpha1_Auditd_To_rsyslog_Auditd(in *Auditd, out *rsyslog.Auditd, s conversion.Scope) error {
	out.AuditRules = in.AuditRules
	return nil
}

// Convert_v1alpha1_Auditd_To_rsyslog_Auditd is an autogenerated conversion function.
func Convert_v1alpha1_Auditd_To_rsyslog_Auditd(in *Auditd, out *rsyslog.Auditd, s conversion.Scope) error {
	return autoConvert_v1alpha1_Auditd_To_rsyslog_Auditd(in, out, s)
}

func autoConvert_rsyslog_Auditd_To_v1alpha1_Auditd(in *rsyslog.Auditd, out *Auditd, s conversion.Scope) error {
	out.AuditRules = in.AuditRules
	return nil
}

// Convert_rsyslog_Auditd_To_v1alpha1_Auditd is an autogenerated conversion function.
func Convert_rsyslog_Auditd_To_v1alpha1_Auditd(in *rsyslog.Auditd, out *Auditd, s conversion.Scope) error {
	return autoConvert_rsyslog_Auditd_To_v1alpha1_Auditd(in, out, s)
}

func autoConvert_v1alpha1_LoggingRule_To_rsyslog_LoggingRule(in *LoggingRule, out *rsyslog.LoggingRule, s conversion.Scope) error {
	out.ProgramNames = *(*[]string)(unsafe.Pointer(&in.ProgramNames))
	out.Severity = (*int)(unsafe.Pointer(in.Severity))
	out.MessageContent = (*rsyslog.MessageContent)(unsafe.Pointer(in.MessageContent))
	return nil
}

// Convert_v1alpha1_LoggingRule_To_rsyslog_LoggingRule is an autogenerated conversion function.
func Convert_v1alpha1_LoggingRule_To_rsyslog_LoggingRule(in *LoggingRule, out *rsyslog.LoggingRule, s conversion.Scope) error {
	return autoConvert_v1alpha1_LoggingRule_To_rsyslog_LoggingRule(in, out, s)
}

func autoConvert_rsyslog_LoggingRule_To_v1alpha1_LoggingRule(in *rsyslog.LoggingRule, out *LoggingRule, s conversion.Scope) error {
	out.ProgramNames = *(*[]string)(unsafe.Pointer(&in.ProgramNames))
	out.Severity = (*int)(unsafe.Pointer(in.Severity))
	out.MessageContent = (*MessageContent)(unsafe.Pointer(in.MessageContent))
	return nil
}

// Convert_rsyslog_LoggingRule_To_v1alpha1_LoggingRule is an autogenerated conversion function.
func Convert_rsyslog_LoggingRule_To_v1alpha1_LoggingRule(in *rsyslog.LoggingRule, out *LoggingRule, s conversion.Scope) error {
	return autoConvert_rsyslog_LoggingRule_To_v1alpha1_LoggingRule(in, out, s)
}

func autoConvert_v1alpha1_MessageContent_To_rsyslog_MessageContent(in *MessageContent, out *rsyslog.MessageContent, s conversion.Scope) error {
	out.Regex = (*string)(unsafe.Pointer(in.Regex))
	out.Exclude = (*string)(unsafe.Pointer(in.Exclude))
	return nil
}

// Convert_v1alpha1_MessageContent_To_rsyslog_MessageContent is an autogenerated conversion function.
func Convert_v1alpha1_MessageContent_To_rsyslog_MessageContent(in *MessageContent, out *rsyslog.MessageContent, s conversion.Scope) error {
	return autoConvert_v1alpha1_MessageContent_To_rsyslog_MessageContent(in, out, s)
}

func autoConvert_rsyslog_MessageContent_To_v1alpha1_MessageContent(in *rsyslog.MessageContent, out *MessageContent, s conversion.Scope) error {
	out.Regex = (*string)(unsafe.Pointer(in.Regex))
	out.Exclude = (*string)(unsafe.Pointer(in.Exclude))
	return nil
}

// Convert_rsyslog_MessageContent_To_v1alpha1_MessageContent is an autogenerated conversion function.
func Convert_rsyslog_MessageContent_To_v1alpha1_MessageContent(in *rsyslog.MessageContent, out *MessageContent, s conversion.Scope) error {
	return autoConvert_rsyslog_MessageContent_To_v1alpha1_MessageContent(in, out, s)
}

func autoConvert_v1alpha1_RsyslogRelpConfig_To_rsyslog_RsyslogRelpConfig(in *RsyslogRelpConfig, out *rsyslog.RsyslogRelpConfig, s conversion.Scope) error {
	out.Target = in.Target
	out.Port = in.Port
	out.LoggingRules = *(*[]rsyslog.LoggingRule)(unsafe.Pointer(&in.LoggingRules))
	out.TLS = (*rsyslog.TLS)(unsafe.Pointer(in.TLS))
	out.RebindInterval = (*int)(unsafe.Pointer(in.RebindInterval))
	out.Timeout = (*int)(unsafe.Pointer(in.Timeout))
	out.ResumeRetryCount = (*int)(unsafe.Pointer(in.ResumeRetryCount))
	out.ReportSuspensionContinuation = (*bool)(unsafe.Pointer(in.ReportSuspensionContinuation))
	out.AuditConfig = (*rsyslog.AuditConfig)(unsafe.Pointer(in.AuditConfig))
	return nil
}

// Convert_v1alpha1_RsyslogRelpConfig_To_rsyslog_RsyslogRelpConfig is an autogenerated conversion function.
func Convert_v1alpha1_RsyslogRelpConfig_To_rsyslog_RsyslogRelpConfig(in *RsyslogRelpConfig, out *rsyslog.RsyslogRelpConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_RsyslogRelpConfig_To_rsyslog_RsyslogRelpConfig(in, out, s)
}

func autoConvert_rsyslog_RsyslogRelpConfig_To_v1alpha1_RsyslogRelpConfig(in *rsyslog.RsyslogRelpConfig, out *RsyslogRelpConfig, s conversion.Scope) error {
	out.Target = in.Target
	out.Port = in.Port
	out.TLS = (*TLS)(unsafe.Pointer(in.TLS))
	out.LoggingRules = *(*[]LoggingRule)(unsafe.Pointer(&in.LoggingRules))
	out.RebindInterval = (*int)(unsafe.Pointer(in.RebindInterval))
	out.Timeout = (*int)(unsafe.Pointer(in.Timeout))
	out.ResumeRetryCount = (*int)(unsafe.Pointer(in.ResumeRetryCount))
	out.ReportSuspensionContinuation = (*bool)(unsafe.Pointer(in.ReportSuspensionContinuation))
	out.AuditConfig = (*AuditConfig)(unsafe.Pointer(in.AuditConfig))
	return nil
}

// Convert_rsyslog_RsyslogRelpConfig_To_v1alpha1_RsyslogRelpConfig is an autogenerated conversion function.
func Convert_rsyslog_RsyslogRelpConfig_To_v1alpha1_RsyslogRelpConfig(in *rsyslog.RsyslogRelpConfig, out *RsyslogRelpConfig, s conversion.Scope) error {
	return autoConvert_rsyslog_RsyslogRelpConfig_To_v1alpha1_RsyslogRelpConfig(in, out, s)
}

func autoConvert_v1alpha1_TLS_To_rsyslog_TLS(in *TLS, out *rsyslog.TLS, s conversion.Scope) error {
	out.Enabled = in.Enabled
	out.SecretReferenceName = (*string)(unsafe.Pointer(in.SecretReferenceName))
	out.PermittedPeer = *(*[]string)(unsafe.Pointer(&in.PermittedPeer))
	out.AuthMode = (*rsyslog.AuthMode)(unsafe.Pointer(in.AuthMode))
	out.TLSLib = (*rsyslog.TLSLib)(unsafe.Pointer(in.TLSLib))
	return nil
}

// Convert_v1alpha1_TLS_To_rsyslog_TLS is an autogenerated conversion function.
func Convert_v1alpha1_TLS_To_rsyslog_TLS(in *TLS, out *rsyslog.TLS, s conversion.Scope) error {
	return autoConvert_v1alpha1_TLS_To_rsyslog_TLS(in, out, s)
}

func autoConvert_rsyslog_TLS_To_v1alpha1_TLS(in *rsyslog.TLS, out *TLS, s conversion.Scope) error {
	out.Enabled = in.Enabled
	out.SecretReferenceName = (*string)(unsafe.Pointer(in.SecretReferenceName))
	out.PermittedPeer = *(*[]string)(unsafe.Pointer(&in.PermittedPeer))
	out.AuthMode = (*AuthMode)(unsafe.Pointer(in.AuthMode))
	out.TLSLib = (*TLSLib)(unsafe.Pointer(in.TLSLib))
	return nil
}

// Convert_rsyslog_TLS_To_v1alpha1_TLS is an autogenerated conversion function.
func Convert_rsyslog_TLS_To_v1alpha1_TLS(in *rsyslog.TLS, out *TLS, s conversion.Scope) error {
	return autoConvert_rsyslog_TLS_To_v1alpha1_TLS(in, out, s)
}
