// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rsyslogrelpconfigcleaner

import (
	"context"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

const managedResourceName = "extension-" + constants.ServiceName + "-configuration-cleaner"

// Values is a set of configuration values for the rsyslog relp config cleaner.
type Values struct {
	// AlpineImage is the alpine container image.
	AlpineImage string
	// PauseContainerImage is the pause container image.
	PauseContainerImage string
}

// New creates a new instance of DeployWaiter for rsyslog relp config cleaner.
func New(
	client client.Client,
	namespace string,
	values Values,
) component.DeployWaiter {
	return &rsyslogRelpConfigCleaner{
		client:    client,
		namespace: namespace,
		values:    values,
	}
}

type rsyslogRelpConfigCleaner struct {
	client    client.Client
	namespace string
	values    Values
}

func (r *rsyslogRelpConfigCleaner) Deploy(ctx context.Context) error {
	data, err := r.computeResourcesData()
	if err != nil {
		return err
	}

	return managedresources.CreateForShoot(ctx, r.client, r.namespace, managedResourceName, constants.Origin, false, data)
}

func (r *rsyslogRelpConfigCleaner) Destroy(ctx context.Context) error {
	return managedresources.Delete(ctx, r.client, r.namespace, managedResourceName, false)
}

// TimeoutWaitForManagedResource is the timeout used while waiting for the ManagedResources to become healthy
// or deleted.
var TimeoutWaitForManagedResource = 2 * time.Minute

func (r *rsyslogRelpConfigCleaner) Wait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitForManagedResource)
	defer cancel()

	return managedresources.WaitUntilHealthy(timeoutCtx, r.client, r.namespace, managedResourceName)
}

// TimeoutWaitCleanupForManagedResource is the timeout used while waiting for the ManagedResource to be deleted.
var TimeoutWaitCleanupForManagedResource = 2 * time.Minute

// WaitCleanup implements component.DeployWaiter.
func (r *rsyslogRelpConfigCleaner) WaitCleanup(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, TimeoutWaitCleanupForManagedResource)
	defer cancel()

	return managedresources.WaitUntilDeleted(timeoutCtx, r.client, r.namespace, managedResourceName)
}

func (r *rsyslogRelpConfigCleaner) computeResourcesData() (map[string][]byte, error) {
	mountPropagationHostToContainer := corev1.MountPropagationHostToContainer

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rsyslog-relp-configuration-cleaner",
			Namespace: metav1.NamespaceSystem,
			Labels:    getLabels(),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getLabels(),
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr.To(false),
					PriorityClassName:            v1beta1constants.PriorityClassNameShootSystem700,
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "rsyslog-relp-configuration-cleaner",
							Image:           r.values.AlpineImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         computeCommand(),
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
									MountPropagation: &mountPropagationHostToContainer,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "pause-container",
							Image:           r.values.PauseContainerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
							},
						},
					},
					HostPID: true,
					Volumes: []corev1.Volume{
						{
							Name: "host-root-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
	}

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)
	return registry.AddAllAndSerialize(daemonSet)
}

func getLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "rsyslog-relp-configuration-cleaner",
		"app.kubernetes.io/instance": "rsyslog-relp-configuration-cleaner",
	}
}

func computeCommand() []string {
	return []string{
		"sh",
		"-c",
		`if [[ -f /host/etc/systemd/system/rsyslog-configurator.service ]]; then
  chroot /host /bin/bash -c 'systemctl disable rsyslog-configurator; systemctl stop rsyslog-configurator; rm -f /etc/systemd/system/rsyslog-configurator.service'
fi

if [[ -d /host` + constants.RsyslogRelpQueueSpoolDir + ` ]]; then
  rm -rf /host` + constants.RsyslogRelpQueueSpoolDir + `
fi

if [[ -f /host` + constants.AuditSyslogPluginPath + ` ]]; then
  sed -i "s/^active\\>.*/active = no/i" /host` + constants.AuditSyslogPluginPath + `
fi
if [[ -f /host` + constants.AudispSyslogPluginPath + ` ]]; then
  sed -i "s/^active\\>.*/active = no/i" /host` + constants.AudispSyslogPluginPath + `
fi

chroot /host /bin/bash -c 'if systemctl list-unit-files systemd-journald-audit.socket > /dev/null; then \
  systemctl enable systemd-journald-audit.socket; \
  systemctl start systemd-journald-audit.socket; \
  systemctl restart systemd-journald; \
fi'

if [[ -d /host` + constants.AuditRulesBackupDir + ` ]]; then
  if [[ -d /host` + constants.AuditRulesDir + ` ]]; then
    rm -rf /host` + constants.AuditRulesDir + `
  fi
  mv /host` + constants.AuditRulesBackupDir + ` /host` + constants.AuditRulesDir + `
  chroot /host /bin/bash -c 'if systemctl list-unit-files auditd.service > /dev/null; then augenrules --load; systemctl restart auditd; fi'
fi

if [[ -f /host` + constants.RsyslogConfigPath + ` ]]; then
  rm -f /host` + constants.RsyslogConfigPath + `
  chroot /host /bin/bash -c 'if systemctl list-unit-files rsyslog.service > /dev/null; then systemctl restart rsyslog; fi'
fi

if [[ -d /host` + constants.RsyslogTLSDir + ` ]]; then
  rm -rf /host` + constants.RsyslogTLSDir + `
fi

if [[ -d /host` + constants.RsyslogOSCDir + ` ]]; then
  rm -rf /host` + constants.RsyslogOSCDir + `
fi`,
	}
}
