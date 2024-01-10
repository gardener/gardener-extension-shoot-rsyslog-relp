// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rsyslogrelpconfigcleaner

import (
	"context"
	"time"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
	// PSPDisabled marks whether the PodSecurityPolicy admission plugin is disabled.
	PSPDisabled bool
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
	var (
		objects            []client.Object
		serviceAccountName = "default"
	)

	if !r.values.PSPDisabled {
		serviceAccountName = "rsyslog-relp-configuration-cleaner"

		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
			AutomountServiceAccountToken: pointer.Bool(false),
		}
		podSecurityPolicy := &policyv1beta1.PodSecurityPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.kube-system.rsyslog-relp-configuration-cleaner",
				Annotations: map[string]string{
					v1beta1constants.AnnotationSeccompDefaultProfile:  v1beta1constants.AnnotationSeccompAllowedProfilesRuntimeDefaultValue,
					v1beta1constants.AnnotationSeccompAllowedProfiles: v1beta1constants.AnnotationSeccompAllowedProfilesRuntimeDefaultValue,
				},
			},
			Spec: policyv1beta1.PodSecurityPolicySpec{
				HostPID:                true,
				ReadOnlyRootFilesystem: true,
				RunAsUser: policyv1beta1.RunAsUserStrategyOptions{
					Rule: policyv1beta1.RunAsUserStrategyRunAsAny,
				},
				SELinux: policyv1beta1.SELinuxStrategyOptions{
					Rule: policyv1beta1.SELinuxStrategyRunAsAny,
				},
				SupplementalGroups: policyv1beta1.SupplementalGroupsStrategyOptions{
					Rule: policyv1beta1.SupplementalGroupsStrategyRunAsAny,
				},
				FSGroup: policyv1beta1.FSGroupStrategyOptions{
					Rule: policyv1beta1.FSGroupStrategyRunAsAny,
				},
				Volumes: []policyv1beta1.FSType{
					policyv1beta1.HostPath,
				},
				AllowedHostPaths: []policyv1beta1.AllowedHostPath{
					{
						PathPrefix: "/",
					},
				},
			},
		}
		clusterRolePSP := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:psp:kube-system:rsyslog-relp-configuration-cleaner",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"policy", "extensions"},
					ResourceNames: []string{podSecurityPolicy.Name},
					Resources:     []string{"podsecuritypolicies"},
					Verbs:         []string{"use"},
				},
			},
		}
		roleBindingPSP := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:psp:rsyslog-relp-configuration-cleaner",
				Namespace: metav1.NamespaceSystem,
				Annotations: map[string]string{
					resourcesv1alpha1.DeleteOnInvalidUpdate: "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRolePSP.Name,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccount.Name,
				Namespace: serviceAccount.Namespace,
			}},
		}

		objects = append(objects,
			serviceAccount,
			podSecurityPolicy,
			clusterRolePSP,
			roleBindingPSP,
		)
	}

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
					AutomountServiceAccountToken: pointer.Bool(false),
					ServiceAccountName:           serviceAccountName,
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

	objects = append(objects, daemonSet)

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)
	return registry.AddAllAndSerialize(objects...)
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
fi`,
	}
}
