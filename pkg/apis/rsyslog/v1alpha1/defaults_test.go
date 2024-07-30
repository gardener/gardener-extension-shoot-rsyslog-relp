// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/utils/ptr"

	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
)

var _ = Describe("RsyslogRelpConfig defaulting", func() {
	Describe("audit config defaulting", func() {
		It("should correctly set default values", func() {
			obj := &RsyslogRelpConfig{}
			SetObjectDefaults_RsyslogRelpConfig(obj)

			Expect(obj.AuditConfig).NotTo(BeNil())
			Expect(obj.AuditConfig.Enabled).To(PointTo(Equal(true)))
		})

		It("should not overwrite values if already set", func() {
			obj := &RsyslogRelpConfig{
				AuditConfig: &AuditConfig{
					Enabled: ptr.To(false),
				},
			}

			SetObjectDefaults_RsyslogRelpConfig(obj)

			Expect(obj.AuditConfig).NotTo(BeNil())
			Expect(obj.AuditConfig.Enabled).To(PointTo(Equal(false)))
		})
	})
})
