// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"time"

	e2e "github.com/gardener/gardener/test/e2e/gardener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/test/common"
)

var _ = Describe("Shoot Rsyslog Relp Extension Tests", func() {
	f := defaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("e2e-rslog-hib")
	common.AddOrUpdateRsyslogRelpExtension(f.Shoot)

	It("Create Shoot with shoot-rsyslog-relp extension enabled, hibernate Shoot, reconcile Shoot, wake up Shoot, delete Shoot", Label("hibernation"), func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 20*time.Minute)
		DeferCleanup(cancel)
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		DeferCleanup(cancel)

		By("Create NetworkPolicy to allow traffic from Shoot nodes to the rsyslog-relp echo server")
		Expect(createNetworkPolicyForEchoServer(ctx, f.ShootFramework.SeedClient, f.ShootFramework.ShootSeedNamespace())).To(Succeed())

		By("Install rsyslog-relp unit on Shoot nodes")
		common.ForEachNode(ctx, f.ShootFramework.ShootClient, func(ctx context.Context, node *corev1.Node) {
			installRsyslogRelp(ctx, f.Logger, f.ShootFramework.ShootClient, node.Name)
		})

		By("Verify shoot-rsyslog-relp works")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		DeferCleanup(cancel)

		echoServerPodIf, echoServerPodName, err := common.GetEchoServerPodInterfaceAndName(ctx, f.ShootFramework.SeedClient)
		Expect(err).NotTo(HaveOccurred())
		verifier := common.NewVerifier(f.Logger, f.ShootFramework.ShootClient, echoServerPodIf, echoServerPodName, f.Shoot.Spec.Provider.Type, f.ShootFramework.Project.Name, f.Shoot.Name, string(f.Shoot.UID), false)

		common.ForEachNode(ctx, f.ShootFramework.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionForNode(ctx, node.Name)
		})

		By("Hibernate Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		DeferCleanup(cancel)
		Expect(f.HibernateShoot(ctx, f.Shoot)).To(Succeed())

		By("Wake up Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		DeferCleanup(cancel)
		Expect(f.WakeUpShoot(ctx, f.Shoot)).To(Succeed())

		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		DeferCleanup(cancel)

		By("Install rsyslog-relp unit on Shoot nodes")
		common.ForEachNode(ctx, f.ShootFramework.ShootClient, func(ctx context.Context, node *corev1.Node) {
			installRsyslogRelp(ctx, f.Logger, f.ShootFramework.ShootClient, node.Name)
		})

		By("Verify that shoot-rsyslog-relp works after wake up")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		DeferCleanup(cancel)
		common.ForEachNode(ctx, f.ShootFramework.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionForNode(ctx, node.Name)
		})

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		DeferCleanup(cancel)
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
