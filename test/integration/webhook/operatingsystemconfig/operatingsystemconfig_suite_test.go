// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

	rsyslogrelpcmd "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/cmd/rsyslogrelp"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Integration Webhook OperatingSystemConfig Suite")
}

const (
	testID       = "webhook-operatingsystemconfig-test"
	providerName = "test-provider"
)

var (
	ctx = context.Background()
	log logr.Logger

	restConfig *rest.Config
	testEnv    *envtest.Environment
	testClient client.Client

	cluster *extensionsv1alpha1.Cluster

	testNamespace *corev1.Namespace
)

var _ = BeforeSuite(func() {
	logf.SetLogger(logger.MustNewZapLogger(logger.DebugLevel, logger.FormatJSON, zap.WriteTo(GinkgoWriter)))
	log = logf.Log.WithName(testID)

	By("Start test environment")
	testEnv = &envtest.Environment{
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				filepath.Join("resources", "crd-extensions.gardener.cloud_extensions.yaml"),
				filepath.Join("resources", "crd-extensions.gardener.cloud_operatingsystemconfigs.yaml"),
				filepath.Join("resources", "crd-extensions.gardener.cloud_clusters.yaml"),
			},
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	restConfig, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(restConfig).NotTo(BeNil())

	DeferCleanup(func() {
		By("Stop test environment")
		Expect(testEnv.Stop()).To(Succeed())
	})

	By("Create test client")
	testClient, err = client.New(restConfig, client.Options{Scheme: kubernetes.SeedScheme})
	Expect(err).NotTo(HaveOccurred())

	By("Create test Namespace")
	testNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			// create dedicated namespace for each test run, so that we can run multiple tests concurrently for stress tests
			GenerateName: testID + "-",
			Labels: map[string]string{
				v1beta1constants.LabelExtensionPrefix + constants.ExtensionType: "true",
			},
		},
	}
	Expect(testClient.Create(ctx, testNamespace)).To(Succeed())
	log.Info("Created Namespace for test", "namespaceName", testNamespace.Name)

	DeferCleanup(func() {
		By("Delete test Namespace")
		Expect(testClient.Delete(ctx, testNamespace)).To(Or(Succeed(), BeNotFoundError()))
	})

	By("Setup manager")
	mgr, err := manager.New(restConfig, manager.Options{
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    testEnv.WebhookInstallOptions.LocalServingPort,
			Host:    testEnv.WebhookInstallOptions.LocalServingHost,
			CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		}),
		Scheme:  kubernetes.SeedScheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{testNamespace.Name: {}},
		},
		Controller: controllerconfig.Controller{
			SkipNameValidation: ptr.To(true),
		},
	})
	Expect(err).NotTo(HaveOccurred())

	By("Register webhook")
	Expect(addTestWebhookToManager(mgr)).To(Succeed())

	By("Start manager")
	mgrContext, mgrCancel := context.WithCancel(ctx)

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(mgrContext)).To(Succeed())
	}()

	DeferCleanup(func() {
		By("Stop manager")
		mgrCancel()
	})

	By("Wait for MutatingWebhookConfiguration")
	Eventually(func() error {
		return testClient.Get(ctx, client.ObjectKey{Name: "gardener-extension-shoot-rsyslog-relp"}, &admissionregistrationv1.MutatingWebhookConfiguration{})
	}).Should(Succeed())

	By("Create Cluster")

	shoot := &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: testNamespace.Name,
		},
		Spec: gardencorev1beta1.ShootSpec{
			Kubernetes: gardencorev1beta1.Kubernetes{
				Version: "1.27.0",
			},
		},
	}
	shootJSON, err := json.Marshal(shoot)
	Expect(err).To(Succeed())

	cluster = &extensionsv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace.Name,
		},
		Spec: extensionsv1alpha1.ClusterSpec{
			CloudProfile: runtime.RawExtension{Raw: []byte("{}")},
			Seed:         runtime.RawExtension{Raw: []byte("{}")},
			Shoot:        runtime.RawExtension{Raw: shootJSON},
		},
	}

	Expect(testClient.Create(ctx, cluster)).To(Succeed())
	log.Info("Created Cluster for test", "cluster", client.ObjectKeyFromObject(cluster))

	DeferCleanup(func() {
		By("Delete Cluster")
		Expect(client.IgnoreNotFound(testClient.Delete(ctx, cluster))).To(Succeed())
	})

})

func addTestWebhookToManager(mgr manager.Manager) error {
	webhookSwitches := rsyslogrelpcmd.WebhookSwitchOptions()
	webhookOptions := webhookcmd.NewAddToManagerOptions(
		"shoot-rsyslog-relp",
		"",
		nil,
		&webhookcmd.ServerOptions{
			Mode: extensionswebhook.ModeURL,
			URL:  fmt.Sprintf("%s:%d", testEnv.WebhookInstallOptions.LocalServingHost, testEnv.WebhookInstallOptions.LocalServingPort),
		},
		webhookSwitches,
	)
	if err := webhookOptions.Complete(); err != nil {
		return err
	}
	_, err := webhookOptions.Completed().AddToManager(ctx, mgr, nil)
	return err
}
