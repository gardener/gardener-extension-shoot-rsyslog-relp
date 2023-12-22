// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle_test

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	versionutils "github.com/gardener/gardener/pkg/utils/version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

var _ = Describe("Lifecycle controller tests", func() {
	var (
		rsyslogConfigurationCleanerDaemonsetYaml = func(pspDisabled bool) string {
			return `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: rsyslog-relp-configuration-cleaner
  namespace: kube-system
  labels:
    app.kubernetes.io/name: rsyslog-relp-configuration-cleaner
    app.kubernetes.io/instance: rsyslog-relp-configuration-cleaner
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: rsyslog-relp-configuration-cleaner
      app.kubernetes.io/instance: rsyslog-relp-configuration-cleaner
  template:
    metadata:
      labels:
        app.kubernetes.io/name: rsyslog-relp-configuration-cleaner
        app.kubernetes.io/instance: rsyslog-relp-configuration-cleaner
    spec:
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      priorityClassName: gardener-shoot-system-700
      containers:
      - name: pause-container
        image: registry.k8s.io/pause:3.9
        imagePullPolicy: IfNotPresent
      initContainers:
      - name: rsyslog-configuration-cleaner
        image: europe-docker.pkg.dev/gardener-project/releases/3rd/alpine:3.18.4
        imagePullPolicy: IfNotPresent
        command:
        - "sh"
        - "-c"
        - |
          if [[ -f /host/etc/systemd/system/rsyslog-configurator.service ]]; then
            chroot /host /bin/bash -c 'systemctl disable rsyslog-configurator; systemctl stop rsyslog-configurator; rm -f /etc/systemd/system/rsyslog-configurator.service'
          fi

          if [[ -f /host/etc/audit/plugins.d/syslog.conf ]]; then
            sed -i 's/yes/no/g' /host/etc/audit/plugins.d/syslog.conf
          fi

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

          if [[ -d /host/var/lib/rsyslog-relp-configurator ]]; then
            rm -rf /host/var/lib/rsyslog-relp-configurator
          fi
        resources:
          requests:
            memory: 8Mi
            cpu: 2m
          limits:
            memory: 32Mi
        volumeMounts:
        - name: host-root-volume
          mountPath: /host
          readOnly: false` + stringBasedOnCondition(!pspDisabled, `
      serviceAccountName: rsyslog-relp-configuration-cleaner`, ``) + `
      hostPID: true
      volumes:
      - name: host-root-volume
        hostPath:
          path: /`
		}

		rsyslogRelpPSPYaml = func(pspDisabled bool) string {
			return stringBasedOnCondition(
				pspDisabled,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0`,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
---
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  annotations:
    seccomp.security.alpha.kubernetes.io/defaultProfileName: 'runtime/default'
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: 'runtime/default'
  name: gardener.kube-system.rsyslog-relp-configuration-cleaner
spec:
  hostPID: true
  volumes:
  - hostPath
  allowedHostPaths:
  - pathPrefix: /
  readOnlyRootFilesystem: true
  runAsUser:
    rule: RunAsAny
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  fsGroup:
    rule: RunAsAny`)
		}

		rsyslogRelpPSPClusterRoleYaml = func(pspDisabled bool) string {
			return stringBasedOnCondition(
				pspDisabled,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0`,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gardener.cloud:psp:kube-system:rsyslog-relp-configuration-cleaner
rules:
- apiGroups:
  - policy
  - extensions
  resourceNames:
  - gardener.kube-system.rsyslog-relp-configuration-cleaner
  resources:
  - podsecuritypolicies
  verbs:
  - use`)
		}

		rsyslogRelpPSPServiceAccountYaml = func(pspDisabled bool) string {
			return stringBasedOnCondition(
				pspDisabled,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0`,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rsyslog-relp-configuration-cleaner
  namespace: kube-system
  labels:
    app.kubernetes.io/name: rsyslog-relp-configuration-cleaner
    app.kubernetes.io/instance: rsyslog-relp-configuration-cleaner
automountServiceAccountToken: false`)
		}

		rsyslogRelpPSPRoleBindingYaml = func(pspDisabled bool) string {
			return stringBasedOnCondition(
				pspDisabled,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0`,
				`# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: gardener.cloud:psp:rsyslog-relp-configuration-cleaner
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gardener.cloud:psp:kube-system:rsyslog-relp-configuration-cleaner
subjects:
- kind: ServiceAccount
  name: rsyslog-relp-configuration-cleaner
  namespace: kube-system`)
		}

		cluster  *extensionsv1alpha1.Cluster
		shoot    *gardencorev1beta1.Shoot
		shootUID types.UID

		extensionProviderConfig *rsyslog.RsyslogRelpConfig
		extensionResource       *extensionsv1alpha1.Extension
	)

	BeforeEach(func() {
		shootName = "shoot-" + utils.ComputeSHA256Hex([]byte(uuid.NewUUID()))[:8]
		projectName = "test-" + utils.ComputeSHA256Hex([]byte(uuid.NewUUID()))[:5]
		shootUID = uuid.NewUUID()
		shootTechnicalID = fmt.Sprintf("shoot--%s--%s", projectName, shootName)

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
						ResourceRef: v1.CrossVersionObjectReference{
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
					Severity:     5,
					ProgramNames: []string{"systemd", "audisp-syslog"},
				},
				{
					Severity:     7,
					ProgramNames: []string{"kubelet"},
				},
				{
					Severity: 2,
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

	var test = func() {
		It("should properly reconcile the extension resource", func() {
			DeferCleanup(test.WithVars(
				&managedresources.IntervalWait, time.Millisecond,
			))
			pspDisabled := versionutils.ConstraintK8sGreaterEqual125.Check(semver.MustParse(shoot.Spec.Kubernetes.Version))

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
					Name:      "extension-shoot-rsyslog-relp-configuration-cleaner-shoot",
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
				g.Expect(string(configCleanerResourceSecret.Data["rsyslog-relp-configuration-cleaner_templates_daemonset.yaml"])).To(Equal(rsyslogConfigurationCleanerDaemonsetYaml(pspDisabled)))
				g.Expect(string(configCleanerResourceSecret.Data["rsyslog-relp-configuration-cleaner_templates_clusterrole-psp.yaml"])).To(Equal(rsyslogRelpPSPClusterRoleYaml(pspDisabled)))
				g.Expect(string(configCleanerResourceSecret.Data["rsyslog-relp-configuration-cleaner_templates_psp.yaml"])).To(Equal(rsyslogRelpPSPYaml(pspDisabled)))
				g.Expect(string(configCleanerResourceSecret.Data["rsyslog-relp-configuration-cleaner_templates_rolebinding-psp.yaml"])).To(Equal(rsyslogRelpPSPRoleBindingYaml(pspDisabled)))
				g.Expect(string(configCleanerResourceSecret.Data["rsyslog-relp-configuration-cleaner_templates_serviceaccount.yaml"])).To(Equal(rsyslogRelpPSPServiceAccountYaml(pspDisabled)))
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
	}

	Context("when PSP is disabled", func() {
		test()
	})

	Context("when PSP is enabled", func() {
		BeforeEach(func() {
			shoot.Spec.Kubernetes.Version = "1.24.8"
		})
		test()
	})
})

func stringBasedOnCondition(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}
