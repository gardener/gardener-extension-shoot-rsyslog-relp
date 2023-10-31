// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	testutils "github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	e2e "github.com/gardener/gardener/test/e2e/gardener"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Shoot Rsyslog Relp Extension Tests", func() {
	test := func(f *framework.ShootCreationFramework, shootMutateFn func(shoot *gardencorev1beta1.Shoot) error) {
		It("Create Shoot, enable shoot-rsyslog-relp extension then disable it and delete Shoot", Offset(1), func() {
			ctx, cancel := context.WithTimeout(parentCtx, 20*time.Minute)
			DeferCleanup(cancel)

			By("Create Shoot")
			Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
			f.Verify()

			ctx, cancel = context.WithTimeout(parentCtx, 1*time.Minute)
			DeferCleanup(cancel)

			By("Create NetworkPolicy to allow traffic from Shoot nodes to the rsyslog-relp echo server")
			Expect(createNetworkPolicyForEchoServer(ctx, f.ShootFramework.SeedClient, f.ShootFramework.ShootSeedNamespace())).To(Succeed())

			By("Install rsyslog-relp unit on Shoot nodes")
			Expect(execInShootNode(ctx, f.ShootFramework.SeedClient, f.Logger, f.ShootFramework.ShootSeedNamespace(), "apt-get update && apt-get install -y rsyslog-relp")).To(Succeed())

			By("Enable the shoot-rsyslog-relp extension")
			ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
			DeferCleanup(cancel)
			Expect(f.UpdateShoot(ctx, f.Shoot, shootMutateFn)).To(Succeed())

			By("Verify shoot-rsyslog-relp works")
			ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
			DeferCleanup(cancel)

			verifier, err := newVerifier(ctx, f.Logger, f.ShootFramework.SeedClient, f.ShootFramework.ShootSeedNamespace(), "local", f.Shoot.Name, string(f.Shoot.UID))
			Expect(err).NotTo(HaveOccurred())

			verifier.verifyThatLogsAreSentToEchoServer(ctx, "test-program", "1", "this should get sent to echo server")
			verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "other-program", "1", "this should not get sent to echo server")
			verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "3", "this should not get sent to echo server")

			By("Disable the shoot-rsyslog-relp extension")
			ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
			DeferCleanup(cancel)
			Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
				for i, extension := range shoot.Spec.Extensions {
					if extension.Type == "shoot-rsyslog-relp" {
						shoot.Spec.Extensions = append(shoot.Spec.Extensions[:i], shoot.Spec.Extensions[i+1:]...)
					}
				}
				return nil
			})).To(Succeed())

			By("Verify that shoot-rsyslog-relp extension is disabled")
			ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
			DeferCleanup(cancel)
			verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "1", "this should not get sent to echo server")

			By("Delete Shoot")
			ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
			DeferCleanup(cancel)
			Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
		})
	}

	Context("shoot-rsyslog-relp extension with tls disabled", Label("tls-disabled"), func() {
		f := defaultShootCreationFramework()
		f.Shoot = e2e.DefaultShoot("e2e-rslog-relp")

		enableExtensionFunc := func(shoot *gardencorev1beta1.Shoot) error {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, shootRsyslogRelpExtension())
			return nil
		}

		test(f, enableExtensionFunc)
	})

	Context("shoot-rsyslog-relp extension with tls enabled", Label("tls-enabled"), func() {
		var createdResources []client.Object
		f := defaultShootCreationFramework()
		f.Shoot = e2e.DefaultShoot("e2e-rslog-relp")

		enableExtensionFunc := func(shoot *gardencorev1beta1.Shoot) error {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, shootRsyslogRelpExtension(withPort(443), withTLSWithSecretRefName("rsyslog-relp-tls")))
			shoot.Spec.Resources = append(shoot.Spec.Resources, gardencorev1beta1.NamedResourceReference{
				Name: "rsyslog-relp-tls",
				ResourceRef: autoscalingv1.CrossVersionObjectReference{
					Kind:       "Secret",
					APIVersion: "v1",
					Name:       "rsyslog-relp-tls",
				},
			})
			return nil
		}

		BeforeEach(func() {
			ctx, cancel := context.WithTimeout(parentCtx, 1*time.Minute)
			DeferCleanup(cancel)

			By("Create rsyslog-relp tls Secret")
			var err error
			createdResources, err = testutils.EnsureTestResources(ctx, f.GardenClient.Client(), f.ProjectNamespace, "testdata/tls")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			ctx, cancel := context.WithTimeout(parentCtx, 1*time.Minute)
			DeferCleanup(cancel)

			for _, resource := range createdResources {
				Expect(f.GardenClient.Client().Delete(ctx, resource)).To(Or(Succeed(), BeNotFoundError()))
			}
		})

		test(f, enableExtensionFunc)
	})
})
