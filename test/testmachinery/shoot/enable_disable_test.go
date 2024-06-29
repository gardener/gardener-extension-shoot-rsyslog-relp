// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	testutils "github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rsyslogv1alpha1 "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog/v1alpha1"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/test/common"
)

const (
	defaultTestTimeout        = 50 * time.Minute
	defaultTestCleanupTimeout = 10 * time.Minute
)

var _ = Describe("Shoot rsyslog-relp testing", func() {
	var (
		f            = framework.NewShootFramework(nil)
		echoServerIP string
	)

	test := func(parentCtx context.Context, shootMutateFn func(shoot *gardencorev1beta1.Shoot) error) {
		By("Enable the shoot-rsyslog-relp extension")
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, shootMutateFn)).To(Succeed())

		By("Verify shoot-rsyslog-relp works")
		ctx, cancel = context.WithTimeout(parentCtx, 20*time.Minute)
		defer cancel()
		echoServerPodIf, echoServerPodName, err := common.GetEchoServerPodInterfaceAndName(ctx, f.ShootClient)
		Expect(err).NotTo(HaveOccurred())
		verifier := common.NewVerifier(f.Logger, f.ShootClient, echoServerPodIf, echoServerPodName, f.Shoot.Spec.Provider.Type, f.Project.Name, f.Shoot.Name, string(f.Shoot.UID), true)

		common.ForEachNode(ctx, f.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionForNode(ctx, node.Name)
		})

		By("Disable the shoot-rsyslog-relp extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			common.RemoveRsyslogRelpExtension(shoot)
			return nil
		})).To(Succeed())

		By("Verify that shoot-rsyslog-relp extension is disabled")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		DeferCleanup(cancel)
		common.ForEachNode(ctx, f.ShootClient, func(ctx context.Context, node *corev1.Node) {
			verifier.VerifyExtensionDisabledForNode(ctx, node.Name)
		})
	}

	framework.CBeforeEach(func(ctx context.Context) {
		By("Deploy rsyslog-relp-echo-server in Shoot cluster")
		var err error
		echoServerIP, err = createRsyslogRelpEchoServer(ctx, f)
		Expect(err).NotTo(HaveOccurred())
	}, time.Minute)

	framework.CAfterEach(func(ctx context.Context) {
		By("Delete rsyslog-relp-echo-server from Shoot cluster")
		Expect(deleteRsyslogRelpEchoServer(ctx, f))
	}, time.Minute)

	Context("shoot-rsyslog-relp extension with tls disabled", Label("tls-disabled"), func() {
		f.Serial().Beta().CIt("should enable and disable the shoot-rsyslog-relp extension", func(parentCtx context.Context) {
			test(parentCtx, func(shoot *gardencorev1beta1.Shoot) error {
				common.AddOrUpdateRsyslogRelpExtension(
					shoot,
					common.WithTarget(echoServerIP),
					common.AppendLoggingRule(rsyslogv1alpha1.LoggingRule{ProgramNames: []string{"audisp-syslog", "audispd"}, Severity: 7}),
				)
				return nil
			})
		}, defaultTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
			if common.HasRsyslogRelpExtension(f.Shoot) {
				By("Disable the shoot-rsyslog-relp extension")
				Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
					common.RemoveRsyslogRelpExtension(shoot)
					return nil
				})).To(Succeed())
			}
		}, defaultTestCleanupTimeout))

	})

	Context("shoot-rsyslog-relp extension with tls and openssl enabled", Label("tls-openssl-enabled"), func() {
		const secretReferenceName = "rsyslog-relp-tls"
		var createdResources []client.Object

		f.Serial().Beta().CIt("should enable and disable the shoot-rsyslog-relp extension", func(parentCtx context.Context) {
			By("Create rsyslog-relp-tls Secret")
			ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
			defer cancel()

			var err error
			createdResources, err = testutils.EnsureTestResources(ctx, f.GardenClient.Client(), f.ProjectNamespace, "../../common/testdata/tls")
			Expect(err).NotTo(HaveOccurred())

			test(parentCtx, func(shoot *gardencorev1beta1.Shoot) error {
				common.AddOrUpdateRsyslogRelpExtension(
					shoot,
					common.WithPort(443),
					common.WithTLSWithSecretRefNameAndTLSLib(secretReferenceName, "openssl"),
					common.WithTarget(echoServerIP),
					common.AppendLoggingRule(rsyslogv1alpha1.LoggingRule{ProgramNames: []string{"audisp-syslog", "audispd"}, Severity: 7}),
				)
				common.AddOrUpdateRsyslogTLSSecretResource(shoot, secretReferenceName)
				return nil
			})
		}, defaultTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
			if common.HasRsyslogRelpExtension(f.Shoot) || common.HasRsyslogTLSSecretResource(f.Shoot, secretReferenceName) {
				By("Disable the shoot-rsyslog-relp extension and remove rsyslog-relp-tls named resource reference")
				Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
					common.RemoveRsyslogRelpExtension(shoot)
					common.RemoveRsyslogTLSSecretResource(shoot, secretReferenceName)
					return nil
				})).To(Succeed())
			}

			By("Delete resources created for test")
			for _, resource := range createdResources {
				Expect(f.GardenClient.Client().Delete(ctx, resource)).To(Or(Succeed(), BeNotFoundError()))
			}
		}, defaultTestCleanupTimeout))
	})
})
