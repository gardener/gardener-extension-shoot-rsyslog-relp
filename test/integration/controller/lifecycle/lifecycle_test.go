// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

var _ = Describe("Lifecycle controller tests", func() {
	var (
		rsyslogConfigurationCleanerDaemonset = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app.kubernetes.io/instance": "rsyslog-relp-configuration-cleaner",
					"app.kubernetes.io/name":     "rsyslog-relp-configuration-cleaner",
				},
				Name:      "rsyslog-relp-configuration-cleaner",
				Namespace: "kube-system",
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/instance": "rsyslog-relp-configuration-cleaner",
						"app.kubernetes.io/name":     "rsyslog-relp-configuration-cleaner",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.kubernetes.io/instance": "rsyslog-relp-configuration-cleaner",
							"app.kubernetes.io/name":     "rsyslog-relp-configuration-cleaner",
						},
					},
					Spec: corev1.PodSpec{
						AutomountServiceAccountToken: ptr.To(false),
						Containers: []corev1.Container{
							{
								Image:           "registry.k8s.io/pause:3.10",
								ImagePullPolicy: corev1.PullIfNotPresent,
								Name:            "pause-container",
								SecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
								},
							},
						},
						InitContainers: []corev1.Container{
							{
								Command: []string{
									"sh",
									"-c",
									`if [[ -f /host/etc/systemd/system/rsyslog-configurator.service ]]; then
  chroot /host /bin/bash -c 'systemctl disable rsyslog-configurator; systemctl stop rsyslog-configurator; rm -f /etc/systemd/system/rsyslog-configurator.service'
fi

if [[ -d /host/var/log/rsyslog ]]; then
  rm -rf /host/var/log/rsyslog
fi

if [[ -f /host/etc/audit/plugins.d/syslog.conf ]]; then
  sed -i "s/^active\\>.*/active = no/i" /host/etc/audit/plugins.d/syslog.conf
fi
if [[ -f /host/etc/audisp/plugins.d/syslog.conf ]]; then
  sed -i "s/^active\\>.*/active = no/i" /host/etc/audisp/plugins.d/syslog.conf
fi

chroot /host /bin/bash -c 'if systemctl list-unit-files systemd-journald-audit.socket > /dev/null; then \
  systemctl enable systemd-journald-audit.socket; \
  systemctl start systemd-journald-audit.socket; \
  systemctl restart systemd-journald; \
fi'

if [[ -d /host/etc/audit/rules.d.original ]]; then
  if [[ -d /host/etc/audit/rules.d ]]; then
    rm -rf /host/etc/audit/rules.d
  fi
  mv /host/etc/audit/rules.d.original /host/etc/audit/rules.d
  chroot /host /bin/bash -c 'if systemctl list-unit-files auditd.service > /dev/null; then augenrules --load; systemctl restart auditd; fi'
fi

if [[ -f /host/etc/rsyslog.d/60-audit.conf ]]; then
  rm -f /host/etc/rsyslog.d/60-audit.conf
  chroot /host /bin/bash -c 'if systemctl list-unit-files rsyslog.service > /dev/null; then systemctl restart rsyslog; fi'
fi

if [[ -d /host/etc/ssl/rsyslog ]]; then
  rm -rf /host/etc/ssl/rsyslog
fi

if [[ -d /host/var/lib/rsyslog-relp-configurator ]]; then
  rm -rf /host/var/lib/rsyslog-relp-configurator
fi`,
								},
								Image:           "europe-docker.pkg.dev/gardener-project/releases/3rd/alpine:3.21.3",
								ImagePullPolicy: corev1.PullIfNotPresent,
								Name:            "rsyslog-relp-configuration-cleaner",
								SecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: ptr.To(false),
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("2m"),
										corev1.ResourceMemory: resource.MustParse("8Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:             "host-root-volume",
										MountPath:        "/host",
										MountPropagation: ptr.To(corev1.MountPropagationHostToContainer),
									},
								},
							},
						},
						HostPID:           true,
						PriorityClassName: "gardener-shoot-system-700",
						SecurityContext: &corev1.PodSecurityContext{
							SeccompProfile: &corev1.SeccompProfile{
								Type: corev1.SeccompProfileTypeRuntimeDefault,
							},
						},
						Volumes: []corev1.Volume{
							{
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/",
									},
								},
								Name: "host-root-volume",
							},
						},
					},
				},
			},
		}

		consistOf func(...client.Object) types.GomegaMatcher
		cluster   *extensionsv1alpha1.Cluster
		shoot     *gardencorev1beta1.Shoot
		shootUID  apimachinerytypes.UID

		extensionProviderConfig *rsyslog.RsyslogRelpConfig
		extensionResource       *extensionsv1alpha1.Extension
	)

	BeforeEach(func() {
		shootName = "shoot-" + utils.ComputeSHA256Hex([]byte(uuid.NewUUID()))[:8]
		projectName = "test-" + utils.ComputeSHA256Hex([]byte(uuid.NewUUID()))[:5]
		shootUID = uuid.NewUUID()
		shootTechnicalID = fmt.Sprintf("shoot--%s--%s", projectName, shootName)

		consistOf = NewManagedResourceConsistOfObjectsMatcher(testClient)

		By("Create test Namespace")
		shootSeedNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootTechnicalID,
			},
		}
		Expect(testClient.Create(ctx, shootSeedNamespace)).To(Succeed())
		log.Info("Created Namespace for test", "namespaceName", shootSeedNamespace.Name)

		DeferCleanup(func() {
			By("Delete test Namespace")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, shootSeedNamespace))).To(Succeed())
		})

		shoot = &gardencorev1beta1.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shootName,
				Namespace: fmt.Sprintf("garden-%s", projectName),
				UID:       shootUID,
			},
			Spec: gardencorev1beta1.ShootSpec{
				Provider: gardencorev1beta1.Provider{
					Workers: []gardencorev1beta1.Worker{{Name: "worker"}},
				},
				Kubernetes: gardencorev1beta1.Kubernetes{
					Version: "1.27.2",
				},
				Resources: []gardencorev1beta1.NamedResourceReference{
					{
						Name: "rsyslog-tls",
						ResourceRef: autoscalingv1.CrossVersionObjectReference{
							Kind: "Secret",
							Name: "rsyslog-tls",
						},
					},
				},
			},
		}

		extensionProviderConfig = &rsyslog.RsyslogRelpConfig{
			Target: "localhost",
			Port:   10250,
			LoggingRules: []rsyslog.LoggingRule{
				{
					Severity:       ptr.To(5),
					ProgramNames:   []string{"systemd", "audisp-syslog"},
					MessageContent: &rsyslog.MessageContent{Regex: ptr.To("testing"), Exclude: ptr.To("not")},
				},
				{
					Severity:     ptr.To(7),
					ProgramNames: []string{"kubelet"},
				},
				{
					Severity: ptr.To(2),
				},
			},
		}
	})

	JustBeforeEach(func() {
		By("Create Cluster")
		cluster = &extensionsv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootTechnicalID,
			},
			Spec: extensionsv1alpha1.ClusterSpec{
				Shoot: runtime.RawExtension{
					Object: shoot,
				},
				Seed: runtime.RawExtension{
					Object: &gardencorev1beta1.Seed{},
				},
				CloudProfile: runtime.RawExtension{
					Object: &gardencorev1beta1.CloudProfile{},
				},
			},
		}

		Expect(testClient.Create(ctx, cluster)).To(Succeed())
		log.Info("Created cluster for test", "cluster", client.ObjectKeyFromObject(cluster))

		By("Ensure manager cache observes cluster creation")
		Eventually(func() error {
			return mgrClient.Get(ctx, client.ObjectKeyFromObject(cluster), &extensionsv1alpha1.Cluster{})
		}).Should(Succeed())

		DeferCleanup(func() {
			By("Delete Cluster")
			Expect(client.IgnoreNotFound(testClient.Delete(ctx, cluster))).To(Succeed())
		})

		extensionResource = &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shoot-rsyslog-relp",
				Namespace: shootSeedNamespace.Name,
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					ProviderConfig: &runtime.RawExtension{
						Object: extensionProviderConfig,
					},
					Type: "shoot-rsyslog-relp",
				},
			},
		}

		By("Create shoot-rsyslog-relp Extension Resource")
		Expect(testClient.Create(ctx, extensionResource)).To(Succeed())
		log.Info("Created shoot-rsyslog-tls extension resource", "extension", client.ObjectKeyFromObject(extensionResource))

		DeferCleanup(func() {
			By("Delete shoot-rsyslog-relp Extension Resource")
			Expect(testClient.Delete(ctx, extensionResource)).To(Or(Succeed(), BeNotFoundError()))
		})
	})

	It("should properly reconcile the extension resource", func() {
		DeferCleanup(test.WithVars(
			&managedresources.IntervalWait, time.Millisecond,
		))

		By("Verify that extension resource is reconciled successfully")
		Eventually(func(g Gomega) {
			g.Expect(mgrClient.Get(ctx, client.ObjectKeyFromObject(extensionResource), extensionResource)).To(Succeed())
			g.Expect(extensionResource.Status.LastOperation).To(Not(BeNil()))
			g.Expect(extensionResource.Status.LastOperation.State).To(Equal(gardencorev1beta1.LastOperationStateSucceeded))
		}).Should(Succeed())

		By("Delete shoot-rsyslog-relp Extension Resource")
		Expect(testClient.Delete(ctx, extensionResource)).To(Succeed())

		configCleanerManagedResource := &resourcesv1alpha1.ManagedResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extension-shoot-rsyslog-relp-configuration-cleaner",
				Namespace: shootSeedNamespace.Name,
			},
		}
		configCleanerResourceSecret := &corev1.Secret{}

		By("Verify that managed resource used for configuration cleanup gets created")
		Eventually(func(g Gomega) {
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerManagedResource), configCleanerManagedResource)).To(Succeed())

			configCleanerResourceSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configCleanerManagedResource.Spec.SecretRefs[0].Name,
					Namespace: configCleanerManagedResource.Namespace,
				},
			}

			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerResourceSecret), configCleanerResourceSecret)).To(Succeed())
			g.Expect(configCleanerResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
			g.Expect(configCleanerManagedResource).To(consistOf(rsyslogConfigurationCleanerDaemonset))
		}).Should(Succeed())

		By("Ensure that managed resource used for configuration cleanup does not get deleted immediately")
		Consistently(func(g Gomega) {
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerManagedResource), configCleanerManagedResource)).To(Succeed())
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerResourceSecret), configCleanerResourceSecret)).To(Succeed())
		}).Should(Succeed())

		By("Set managed resource used for configuration cleanup to healthy")
		patch := client.MergeFrom(configCleanerManagedResource.DeepCopy())
		configCleanerManagedResource.Status.Conditions = append(configCleanerManagedResource.Status.Conditions, []gardencorev1beta1.Condition{
			{
				Type:               resourcesv1alpha1.ResourcesApplied,
				Status:             gardencorev1beta1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				LastUpdateTime:     metav1.Now(),
			},
			{
				Type:               resourcesv1alpha1.ResourcesHealthy,
				Status:             gardencorev1beta1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				LastUpdateTime:     metav1.Now(),
			},
		}...)
		configCleanerManagedResource.Status.ObservedGeneration = 1
		Expect(testClient.Status().Patch(ctx, configCleanerManagedResource, patch)).To(Succeed())

		By("Verify that managed resource used for configuration cleanup gets deleted")
		Eventually(func(g Gomega) {
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerManagedResource), configCleanerManagedResource)).To(BeNotFoundError())
			g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerResourceSecret), configCleanerResourceSecret)).To(BeNotFoundError())
		}).Should(Succeed())
	})

})
