// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOperatingSystemConfigWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperatingSystemConfig Webhook Suite")
}
