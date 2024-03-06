// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/utils/pointer"

	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
)

var _ = Describe("RsyslogRelpConfig defaulting", func() {
	Describe("audit rules config defaulting", func() {
		It("should correctly set default values", func() {
			obj := &RsyslogRelpConfig{}
			SetObjectDefaults_RsyslogRelpConfig(obj)

			Expect(obj.AuditRulesConfig).NotTo(BeNil())
			Expect(obj.AuditRulesConfig.Enabled).To(PointTo(Equal(true)))
		})

		It("should not overwrite values if already set", func() {
			obj := &RsyslogRelpConfig{
				AuditRulesConfig: &AuditRulesConfig{
					Enabled: pointer.Bool(false),
				},
			}

			SetObjectDefaults_RsyslogRelpConfig(obj)

			Expect(obj.AuditRulesConfig).NotTo(BeNil())
			Expect(obj.AuditRulesConfig.Enabled).To(PointTo(Equal(false)))
		})
	})
})
