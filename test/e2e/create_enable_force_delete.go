// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	e2e "github.com/gardener/gardener/test/e2e/gardener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Shoot Rsyslog Relp Extension Tests", func() {
	f := defaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("e2e-rslog-fd")
	f.Shoot.Spec.Extensions = []gardencorev1beta1.Extension{shootRsyslogRelpExtension()}

	It("Create Shoot with shoot-rsyslog-relp extension enabled and force delete Shoot", Label("force-delete"), func() {
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

		By("Verify shoot-rsyslog-relp works")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		DeferCleanup(cancel)

		verifier, err := newVerifier(ctx, f.Logger, f.ShootFramework.SeedClient, f.ShootFramework.ShootSeedNamespace(), "local", f.Shoot.Name, string(f.Shoot.UID))
		Expect(err).NotTo(HaveOccurred())

		verifier.verifyThatLogsAreSentToEchoServer(ctx, "test-program", "1", "this should get sent to echo server")
		verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "other-program", "1", "this should not get sent to echo server")
		verifier.verifyThatLogsAreNotSentToEchoServer(ctx, "test-program", "3", "this should not get sent to echo server")

		By("Force Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		DeferCleanup(cancel)
		Expect(f.ForceDeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
