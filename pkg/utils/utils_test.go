// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/utils"
)

var _ = Describe("Utils", func() {
	Describe("#ProjectName", func() {
		It("should return project name correctly", func() {
			Expect(ProjectName("shoot-foo-bar", "bar")).To(Equal("foo"))
		})
	})

	DescribeTable("#ValidateRsyslogRelpSecret",
		func(caData, crtData, keyData, extraData []byte, matcher types.GomegaMatcher) {
			var data = map[string][]byte{}

			if len(caData) > 0 {
				data["ca"] = caData
			}
			if len(crtData) > 0 {
				data["crt"] = caData
			}
			if len(keyData) > 0 {
				data["key"] = caData
			}
			if len(extraData) > 0 {
				data["extra"] = extraData
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rsyslog-secret",
					Namespace: "foo",
				},
				Data: data,
			}

			Expect(ValidateRsyslogRelpSecret(secret)).To(matcher)
		},
		Entry(
			"should return error if secret does not contain 'ca' data entry",
			nil, nil, nil, nil,
			MatchError(ContainSubstring("secret foo/rsyslog-secret is missing ca value")),
		),
		Entry(
			"should return error if secret does not contain 'crt' data entry",
			[]byte("caData"), nil, nil, nil,
			MatchError(ContainSubstring("secret foo/rsyslog-secret is missing crt value")),
		),
		Entry(
			"should return error if secret does not contain 'key' data entry",
			[]byte("caData"), []byte("crtData"), nil, nil,
			MatchError(ContainSubstring("secret foo/rsyslog-secret is missing key value")),
		),
		Entry(
			"should not return error if secret is valid",
			[]byte("caData"), []byte("crtData"), []byte("keyData"), nil,
			Succeed(),
		),
		Entry(
			"should return error if secret does not contain 'tls' data entry",
			[]byte("caData"), []byte("crtData"), []byte("tlsData"), []byte("extraData"),
			MatchError(ContainSubstring("secret foo/rsyslog-secret should have only three data entries")),
		),
	)
})
