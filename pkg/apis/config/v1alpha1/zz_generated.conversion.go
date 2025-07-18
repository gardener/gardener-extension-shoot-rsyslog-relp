//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	config "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*Configuration)(nil), (*config.Configuration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Configuration_To_config_Configuration(a.(*Configuration), b.(*config.Configuration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.Configuration)(nil), (*Configuration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_Configuration_To_v1alpha1_Configuration(a.(*config.Configuration), b.(*Configuration), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_Configuration_To_config_Configuration(in *Configuration, out *config.Configuration, s conversion.Scope) error {
	return nil
}

// Convert_v1alpha1_Configuration_To_config_Configuration is an autogenerated conversion function.
func Convert_v1alpha1_Configuration_To_config_Configuration(in *Configuration, out *config.Configuration, s conversion.Scope) error {
	return autoConvert_v1alpha1_Configuration_To_config_Configuration(in, out, s)
}

func autoConvert_config_Configuration_To_v1alpha1_Configuration(in *config.Configuration, out *Configuration, s conversion.Scope) error {
	return nil
}

// Convert_config_Configuration_To_v1alpha1_Configuration is an autogenerated conversion function.
func Convert_config_Configuration_To_v1alpha1_Configuration(in *config.Configuration, out *Configuration, s conversion.Scope) error {
	return autoConvert_config_Configuration_To_v1alpha1_Configuration(in, out, s)
}
