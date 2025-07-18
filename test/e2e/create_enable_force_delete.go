// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/test/common"
)

var _ = Describe("Shoot Rsyslog Relp Extension Tests", func() {
	f := defaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("e2e-rslog-fd")
	common.AddOrUpdateRsyslogRelpExtension(f.Shoot, common.WithAuditConfig(&v1alpha1.AuditConfig{Enabled: false}))

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
		common.ForEachNode(ctx, f.ShootFramework.ShootClient, func(ctx context.Context, node *corev1.Node) {
			installRsyslogRelp(ctx, f.Logger, f.ShootFramework.ShootClient, node.Name)
		})

		By("Verify shoot-rsyslog-relp works")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		DeferCleanup(cancel)

		verifier := common.NewVerifier(f.Logger, f.ShootFramework.ShootClient, f.ShootFramework.SeedClient, f.Shoot.Spec.Provider.Type, f.ShootFramework.Project.Name, f.Shoot.Name, string(f.Shoot.UID), false, "")

		common.ForEachNode(ctx, f.ShootFramework.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionForNode(ctx, node.Name)
		})

		By("Force Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		DeferCleanup(cancel)
		Expect(f.ForceDeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
