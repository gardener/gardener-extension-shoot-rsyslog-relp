// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/utils"
)

var _ = Describe("Utils", func() {
	Describe("#ProjectName", func() {
		It("should return project name correctly", func() {
			Expect(ProjectName("shoot-foo-bar", "bar")).To(Equal("foo"))
		})
	})
})
