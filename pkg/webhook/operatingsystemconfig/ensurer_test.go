// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"context"
	"testing"

	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config/v1alpha1"
	. "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/webhook/operatingsystemconfig"
)

func TestOperatingSystemConfigWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperatingSystemConfig Webhook Suite")
}

var _ = Describe("Ensurer", func() {
	var (
		logger = logr.Discard()
		ctx    = context.Background()

		decoder    runtime.Decoder
		fakeClient client.Client
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(extensionsv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())

		decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
		fakeClient = fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	})

	Describe("#EnsureAdditionalFiles", func() {
		var (
			ensurer genericmutator.Ensurer

			oldFile = extensionsv1alpha1.File{
				Path: "/var/lib/foo.sh",
			}
			files []extensionsv1alpha1.File
		)

		BeforeEach(func() {
			ensurer = NewEnsurer(fakeClient, decoder, logger)
			files = []extensionsv1alpha1.File{oldFile}
		})

		It("should add additional files to the current ones", func() {
			Expect(ensurer.EnsureAdditionalFiles(ctx, nil, &files, nil)).To(Succeed())
			Expect(files).To(ConsistOf(oldFile))
		})
	})

	Describe("#EnsureAdditionalUnits", func() {
		var (
			ensurer genericmutator.Ensurer

			oldUnit = extensionsv1alpha1.Unit{
				Name: "foo.service",
			}
			units []extensionsv1alpha1.Unit
		)

		BeforeEach(func() {
			units = []extensionsv1alpha1.Unit{oldUnit}
			ensurer = NewEnsurer(fakeClient, decoder, logger)
		})

		It("should add additional units to the current ones", func() {
			Expect(ensurer.EnsureAdditionalUnits(ctx, nil, &units, nil)).To(Succeed())
			Expect(units).To(ConsistOf(oldUnit))
		})
	})
})
