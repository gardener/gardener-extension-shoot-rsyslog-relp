// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/test/common"
)

var _ = Describe("Shoot rsyslog-relp testing", func() {
	const (
		hibernationTestTimeout        = 75 * time.Minute
		hibernationTestCleanupTimeout = 25 * time.Minute
	)

	f := framework.NewShootFramework(nil)

	f.Serial().CIt("should enable and disable the shoot-rsyslog-relp extension", func(parentCtx context.Context) {
		By("Deploy the rsyslog-relp-echo-server in Shoot cluster")
		ctx, cancel := context.WithTimeout(parentCtx, time.Minute)
		defer cancel()
		echoServerIP, err := createRsyslogRelpEchoServer(ctx, f)
		Expect(err).NotTo(HaveOccurred())

		By("Enable the shoot-rsyslog-relp extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			common.AddOrUpdateRsyslogRelpExtension(shoot, common.WithTarget(echoServerIP))
			return nil
		})).To(Succeed())

		By("Verify shoot-rsyslog-relp works")
		ctx, cancel = context.WithTimeout(parentCtx, 20*time.Minute)
		defer cancel()

		verifier := common.NewVerifier(f.Logger, f.ShootClient, f.ShootClient, f.Shoot.Spec.Provider.Type, f.Project.Name, f.Shoot.Name, string(f.Shoot.UID), false, "")

		common.ForEachNode(ctx, f.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionForNode(ctx, node.Name)
		})

		By("Hibernate Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.HibernateShoot(ctx)).To(Succeed())

		By("Reconcile Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, "gardener.cloud/operation", "reconcile")
			return nil
		})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())

		By("Wake up Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.WakeUpShoot(ctx)).To(Succeed())

		By("Verify shoot-rsyslog-relp works after Shoot is woken up")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()

		common.ForEachNode(ctx, f.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionForNode(ctx, node.Name)
		})

	}, hibernationTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
		if v1beta1helper.HibernationIsEnabled(f.Shoot) {
			By("Wake up Shoot")
			Expect(f.WakeUpShoot(ctx)).To(Succeed())
		}

		if common.HasRsyslogRelpExtension(f.Shoot) {
			By("Disable the shoot-rsyslog-relp extension")
			Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
				common.RemoveRsyslogRelpExtension(shoot)
				return nil
			})).To(Succeed())
		}

		By("Delete rsyslog-relp-echo-server from Shoot cluster")
		Expect(deleteRsyslogRelpEchoServer(ctx, f)).To(Succeed())
	}, hibernationTestCleanupTimeout))
})
