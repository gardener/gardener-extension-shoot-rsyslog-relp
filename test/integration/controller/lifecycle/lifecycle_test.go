// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
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
	v1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
)

var _ = Describe("Lifecycle controller tests", func() {
	var (
		authModeName  rsyslog.AuthMode = "name"
		tlsLibOpenSSL rsyslog.TLSLib   = "openssl"

		rsyslogConfigurationCleanerDaemonsetYaml = `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
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
        image: registry.k8s.io/pause:3.7
        imagePullPolicy: IfNotPresent
      initContainers:
      - name: rsyslog-configuration-cleaner
        image: eu.gcr.io/gardener-project/3rd/alpine:3.15.8
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
        volumeMounts:
        - name: host-root-volume
          mountPath: /host
          readOnly: false
      hostPID: true
      volumes:
      - name: host-root-volume
        hostPath:
          path: /`

		auditdConfigMapYaml = `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
kind: ConfigMap
apiVersion: v1
metadata:
  name: rsyslog-relp-configurator-auditd-config
  namespace: kube-system
data:
  00-base-config.rules: |
    ## First rule - delete all
    -D
    ## Increase the buffers to survive stress events.
    ## Make this bigger for busy systems
    -b 8192
    ## This determine how long to wait in burst of events
    --backlog_wait_time 60000
    ## Set failure mode to syslog
    -f 1
  10-privilege-escalation.rules: |
    -a exit,always -F arch=b64 -S setuid -S setreuid -S setgid -S setregid -F auid>0 -F auid!=-1 -F key=privilege_escalation
    -a exit,always -F arch=b64 -S execve -S execveat -F euid=0 -F auid>0 -F auid!=-1 -F key=privilege_escalation
  11-privileged-special.rules: |
    -a exit,always -F arch=b64 -S mount -S mount_setattr -S umount2 -S mknod -S mknodat -S chroot -F auid!=-1 -F key=privileged_special
  12-system-integrity.rules: |
    -a exit,always -F dir=/boot -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/etc -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/bin -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/sbin -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/lib -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/lib64 -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/usr -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/opt -F perm=wa -F key=system_integrity
    -a exit,always -F dir=/root -F perm=wa -F key=system_integrity
  configured-by-rsyslog-relp-configurator: |
    # The files in this directory are managed by the shoot-rsyslog-relp extension
    # The original files were moved to /etc/audit/rules.d.original`

		rsyslogConfigMapYaml = func(tlsEnabled bool, projectName, shootName string, shootUID types.UID) string {
			return `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

apiVersion: v1
kind: ConfigMap
metadata:
  name: rsyslog-relp-configurator-config
  namespace: kube-system
data:
  rsyslog-configurator.service: |
    [Unit]
    Description=rsyslog configurator daemon
    Documentation=https://github.com/gardener/gardener-extension-shoot-rsyslog-relp
    [Service]
    Type=simple
    Restart=always
    RestartSec=15
    ExecStart=/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh
    [Install]
    WantedBy=multi-user.target

  configure-rsyslog.sh: |
    #!/bin/bash

    function configure_auditd() {
      if [[ ! -d /etc/audit/rules.d.original ]] && [[ -d /etc/audit/rules.d ]]; then
        mv /etc/audit/rules.d /etc/audit/rules.d.original
      fi

      if [[ ! -d /etc/audit/rules.d ]]; then
        mkdir -p /etc/audit/rules.d
      fi
      if ! diff -rq /var/lib/rsyslog-relp-configurator/audit/rules.d /etc/audit/rules.d ; then
        rm -rf /etc/audit/rules.d/*
        cp -L /var/lib/rsyslog-relp-configurator/audit/rules.d/* /etc/audit/rules.d/
        if [[ -f /etc/audit/plugins.d/syslog.conf ]]; then
          sed -i 's/no/yes/g' /etc/audit/plugins.d/syslog.conf
        fi
        augenrules --load
        systemctl restart auditd
      fi
    }

    function configure_rsyslog() {
      if [[ ! -f etc/rsyslog.d/60-audit.conf ]] || ! diff -rq /var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf /etc/rsyslog.d/60-audit.conf ; then
        cp -fL /var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf /etc/rsyslog.d/60-audit.conf
        systemctl restart rsyslog
      fi
    }

    if systemctl list-unit-files auditd.service > /dev/null; then
      echo "Configuring auditd.service ..."
      configure_auditd
    else
      echo "auditd.service is not installed, skipping configuration"
    fi

    if systemctl list-unit-files rsyslog.service > /dev/null; then
      echo "Configuring rsyslog.service ..."
      configure_rsyslog
    else
      echo "rsyslog.service is not installed, skipping configuration"
    fi

  60-audit.conf: |
    template(name="SyslogForwarderTemplate" type="list") {
      constant(value=" ")
      constant(value="` + projectName + `")
      constant(value=" ")
      constant(value="` + shootName + `")
      constant(value=" ")
      constant(value="` + string(shootUID) + `")
      constant(value=" ")
      property(name="hostname")
      constant(value=" ")
      property(name="pri")
      constant(value=" ")
      property(name="syslogtag")
      constant(value=" ")
      property(name="timestamp" dateFormat="rfc3339")
      constant(value=" ")
      property(name="procid")
      constant(value=" ")
      property(name="msgid")
      constant(value=" ")
      property(name="msg")
      constant(value=" ")
    }

    module(
      load="omrelp"` + stringBasedOnCondition(tlsEnabled, `
      tls.tlslib="openssl"`, "") + `
    )

    module(
      load="impstats"
      interval="600"
      severity="7"
      log.syslog="off"
      log.file="/var/log/rsyslog-stats.log"
    )

    ruleset(name="relp_action_ruleset") {
      action(
        type="omrelp"
        target="localhost"
        port="10250"
        Template="SyslogForwarderTemplate"` + stringBasedOnCondition(tlsEnabled, `
        tls="on"
        tls.caCert="/var/lib/rsyslog-relp-configurator/tls/ca.crt"
        tls.myCert="/var/lib/rsyslog-relp-configurator/tls/tls.crt"
        tls.myPrivKey="/var/lib/rsyslog-relp-configurator/tls/tls.key"
        tls.authmode="name"
        tls.permittedpeer=["rsyslog-server.foo","rsyslog-server.foo.bar"]`, "") + `
      )
    }

    if $programname == ["systemd","audisp-syslog"] and $syslogseverity <= 5 then {
      call relp_action_ruleset
      stop
    }
    if $programname == ["kubelet"] and $syslogseverity <= 7 then {
      call relp_action_ruleset
      stop
    }
    if $syslogseverity <= 2 then {
      call relp_action_ruleset
      stop
    }

    input(type="imuxsock" Socket="/run/systemd/journal/syslog")`
		}

		rsyslogTlsSecretYaml = func(tlsEnabled bool) string {
			if !tlsEnabled {
				return `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0`
			}

			return `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
apiVersion: v1
kind: Secret
metadata:
  name: rsyslog-relp-configurator-tls
  namespace: kube-system
type: Opaque
data:
  ca.crt: ` + utils.EncodeBase64([]byte("ca")) + `
  tls.crt: ` + utils.EncodeBase64([]byte("crt")) + `
  tls.key: ` + utils.EncodeBase64([]byte("key"))
		}

		rsyslogConfiguratorDaemonsetYaml = func(tlsEnabled bool, rsyslogConfigMap, auditdConfigMap, tlsSecret string) string {
			return `# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: rsyslog-relp-configurator
  namespace: kube-system
  labels:
    app.kubernetes.io/name: rsyslog-relp-configurator
    app.kubernetes.io/instance: rsyslog-relp-configurator
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: rsyslog-relp-configurator
      app.kubernetes.io/instance: rsyslog-relp-configurator
  template:
    metadata:
      annotations:` + stringBasedOnCondition(tlsEnabled, `
        checksum/rsyslog-relp-configurator-tls: `+utils.ComputeSHA256Hex([]byte(tlsSecret)), "") + `
        checksum/rsyslog-relp-configurator-config: ` + utils.ComputeSHA256Hex([]byte(rsyslogConfigMap)) + `
        checksum/rsyslog-relp-configurator-auditd-config: ` + utils.ComputeSHA256Hex([]byte(auditdConfigMap)) + `
      labels:
        app.kubernetes.io/name: rsyslog-relp-configurator
        app.kubernetes.io/instance: rsyslog-relp-configurator
    spec:
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      priorityClassName: gardener-shoot-system-700
      containers:
      - name: pause
        image: registry.k8s.io/pause:3.7
        imagePullPolicy: IfNotPresent
      initContainers:
      - name: rsyslog-relp-configurator
        image: eu.gcr.io/gardener-project/3rd/alpine:3.15.8
        imagePullPolicy: IfNotPresent
        command:
        - "sh"
        - "-c"
        - |
          mkdir -p /host/var/lib/rsyslog-relp-configurator/audit/rules.d
          cp -fL /var/lib/rsyslog-relp-configurator/audit/rules.d/* /host/var/lib/rsyslog-relp-configurator/audit/rules.d/
          mkdir -p /host/var/lib/rsyslog-relp-configurator/rsyslog.d
          cp -fL /var/lib/rsyslog-relp-configurator/config/60-audit.conf /host/var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf` + stringBasedOnCondition(tlsEnabled, `
          mkdir -p /host/var/lib/rsyslog-relp-configurator/tls
          cp -fL /var/lib/rsyslog-relp-configurator/tls/* /host/var/lib/rsyslog-relp-configurator/tls/`, "") + `
          cp -fL /var/lib/rsyslog-relp-configurator/config/configure-rsyslog.sh /host/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh
          chmod +x /host/var/lib/rsyslog-relp-configurator/configure-rsyslog.sh
          cp -fL /var/lib/rsyslog-relp-configurator/config/rsyslog-configurator.service /host/etc/systemd/system/rsyslog-configurator.service
          chroot /host /bin/bash -c "systemctl enable rsyslog-configurator; systemctl start rsyslog-configurator"
        volumeMounts:` + stringBasedOnCondition(tlsEnabled, `
        - name: rsyslog-relp-configurator-tls-volume
          mountPath: /var/lib/rsyslog-relp-configurator/tls`,
				"") + `
        - name: rsyslog-relp-configurator-config-volume
          mountPath: /var/lib/rsyslog-relp-configurator/config
        - name: auditd-config-volume
          mountPath: /var/lib/rsyslog-relp-configurator/audit/rules.d
        - name: host-root-volume
          mountPath: /host
          readOnly: false
      hostPID: true
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
      volumes:` + stringBasedOnCondition(tlsEnabled, `
      - name: rsyslog-relp-configurator-tls-volume
        secret:
          secretName: rsyslog-relp-configurator-tls`,
				"") + `
      - name: rsyslog-relp-configurator-config-volume
        configMap:
          name: rsyslog-relp-configurator-config
      - name: auditd-config-volume
        configMap:
          name: rsyslog-relp-configurator-auditd-config
      - name: host-root-volume
        hostPath:
          path: /`
		}

		cluster  *extensionsv1alpha1.Cluster
		shootUID types.UID
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

		By("Create Cluster")
		cluster = &extensionsv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootTechnicalID,
			},
			Spec: extensionsv1alpha1.ClusterSpec{
				Shoot: runtime.RawExtension{
					Object: &gardencorev1beta1.Shoot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      shootName,
							Namespace: fmt.Sprintf("garden-%s", projectName),
							UID:       shootUID,
						},
						Spec: gardencorev1beta1.ShootSpec{
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
					},
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
	})

	var test = func(tlsEnabled bool) {
		It("should properly reconcile the extension resource", func() {
			DeferCleanup(test.WithVars(
				&managedresources.IntervalWait, time.Millisecond,
			))

			extensionResource := &extensionsv1alpha1.Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot-rsyslog-relp",
					Namespace: shootSeedNamespace.Name,
				},
				Spec: extensionsv1alpha1.ExtensionSpec{
					DefaultSpec: extensionsv1alpha1.DefaultSpec{
						ProviderConfig: &runtime.RawExtension{
							Object: &rsyslog.RsyslogRelpConfig{
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
							},
						},
						Type: "shoot-rsyslog-relp",
					},
				},
			}

			if tlsEnabled {
				extensionConfig := extensionResource.Spec.ProviderConfig.Object.(*rsyslog.RsyslogRelpConfig)
				extensionConfig.TLS = &rsyslog.TLS{
					Enabled:             true,
					SecretReferenceName: pointer.String("rsyslog-tls"),
					AuthMode:            &authModeName,
					TLSLib:              &tlsLibOpenSSL,
					PermittedPeer:       []string{"rsyslog-server.foo", "rsyslog-server.foo.bar"},
				}
				rsyslogSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ref-rsyslog-tls",
						Namespace: shootSeedNamespace.Name,
					},
					Data: map[string][]byte{
						"ca":  []byte("ca"),
						"crt": []byte("crt"),
						"key": []byte("key"),
					},
				}
				By("Create rsyslog-tls secret")
				Expect(testClient.Create(ctx, rsyslogSecret)).To(Succeed())
				log.Info("Created rsyslog-tls secret", "secret", client.ObjectKeyFromObject(rsyslogSecret))

				DeferCleanup(func() {
					By("Delete rsyslog-tls Secret")
					Expect(testClient.Delete(ctx, rsyslogSecret)).To(Or(Succeed(), BeNotFoundError()))
				})
			}

			By("Create shoot-rsyslog-relp Extension Resource")
			Expect(testClient.Create(ctx, extensionResource)).To(Succeed())
			log.Info("Created shoot-rsyslog-tls extension resource", "extension", client.ObjectKeyFromObject(extensionResource))

			DeferCleanup(func() {
				By("Delete shoot-rsyslog-relp Extension Resource")
				Expect(testClient.Delete(ctx, extensionResource)).To(Or(Succeed(), BeNotFoundError()))
			})

			managedResource := &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extension-shoot-rsyslog-relp-shoot",
					Namespace: shootSeedNamespace.Name,
				},
			}
			managedResourceSecret := &corev1.Secret{}

			By("Verify that managed resource is created correctly")
			rsyslogConfigMap := rsyslogConfigMapYaml(tlsEnabled, projectName, shootName, shootUID)
			rsyslogTlsSecret := rsyslogTlsSecretYaml(tlsEnabled)
			Eventually(func(g Gomega) {
				g.Expect(mgrClient.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())

				managedResourceSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedResource.Spec.SecretRefs[0].Name,
						Namespace: managedResource.Namespace,
					},
				}

				g.Expect(mgrClient.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
				g.Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
				g.Expect(string(managedResourceSecret.Data["rsyslog-relp-configurator_templates_auditd-config.yaml"])).To(Equal(auditdConfigMapYaml))
				g.Expect(string(managedResourceSecret.Data["rsyslog-relp-configurator_templates_configmap.yaml"])).To(Equal(rsyslogConfigMap))
				g.Expect(string(managedResourceSecret.Data["rsyslog-relp-configurator_templates_tls.yaml"])).To(Equal(rsyslogTlsSecret))
				g.Expect(string(managedResourceSecret.Data["rsyslog-relp-configurator_templates_daemonset.yaml"])).To(Equal(rsyslogConfiguratorDaemonsetYaml(tlsEnabled, rsyslogConfigMap, auditdConfigMapYaml, rsyslogTlsSecret)))
			}).Should(Succeed())

			By("Delete shoot-rsyslog-relp Extension Resource")
			Expect(testClient.Delete(ctx, extensionResource)).To(Succeed())

			By("Verify that managed resource used for configuration gets deleted")
			Eventually(func(g Gomega) {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(BeNotFoundError())
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(BeNotFoundError())
			}).Should(Succeed())

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
				g.Expect(string(configCleanerResourceSecret.Data["rsyslog-relp-configuration-cleaner_templates_daemonset.yaml"])).To(Equal(rsyslogConfigurationCleanerDaemonsetYaml))
			}).Should(Succeed())

			By("Ensure that managed resource used for configuration cleanup does not get deleted immediately")
			Consistently(func(g Gomega) {
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerManagedResource), managedResource)).To(Succeed())
				g.Expect(testClient.Get(ctx, client.ObjectKeyFromObject(configCleanerResourceSecret), managedResourceSecret)).To(Succeed())
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

	Context("when TLS is not enabled", func() {
		test(false)
	})

	Context("when TLS is enabled", func() {
		test(true)
	})
})

func stringBasedOnCondition(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}
