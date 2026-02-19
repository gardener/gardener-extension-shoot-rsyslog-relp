// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"context"
	"os"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/imagevector"
)

const (
	rsyslogRelpInstallerNamespace = "kube-system"
	rsyslogRelpInstallerName      = "rsyslog-relp-installer"
)

var (
	parentCtx context.Context
)

var _ = BeforeEach(func() {
	parentCtx = context.Background()
})

func defaultShootCreationFramework() *framework.ShootCreationFramework {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	return framework.NewShootCreationFramework(&framework.ShootCreationConfig{
		GardenerConfig: &framework.GardenerConfig{
			ProjectNamespace:   "garden-local",
			GardenerKubeconfig: kubeconfigPath,
			SkipAccessingShoot: false,
			CommonConfig:       &framework.CommonConfig{},
		},
	})
}

func deployRsyslogRelpInstaller(ctx context.Context, c kubernetes.Interface) error {
	alpineImage, err := imagevector.ImageVector().FindImage(imagevector.ImageNameAlpine)
	if err != nil {
		return err
	}

	pauseImage, err := imagevector.ImageVector().FindImage(imagevector.ImageNamePauseContainer)
	if err != nil {
		return err
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rsyslogRelpInstallerName,
			Namespace: rsyslogRelpInstallerNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "rsyslog-relp",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "rsyslog-relp",
					},
				},
				Spec: corev1.PodSpec{
					HostPID:     true,
					HostNetwork: true,
					InitContainers: []corev1.Container{{
						Name:    "rsyslog-relp",
						Image:   alpineImage.String(),
						Command: []string{"/bin/sh", "-c"},
						Args:    []string{"chroot /hostroot apt-get update && chroot /hostroot apt-get install -y rsyslog-relp"},
						SecurityContext: &corev1.SecurityContext{
							Privileged: ptr.To(true),
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "root-volume",
							MountPath: "/hostroot",
						}},
					}},
					Containers: []corev1.Container{{
						Name:  "pause",
						Image: pauseImage.String(),
					}},
					Volumes: []corev1.Volume{{
						Name: "root-volume",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/",
							},
						},
					}},
				},
			},
		},
	}

	return c.Client().Create(ctx, ds)
}

func isRsyslogRelpInstallerReady(ctx context.Context, c kubernetes.Interface) (bool, error) {
	ds := &appsv1.DaemonSet{}
	if err := c.Client().Get(ctx, client.ObjectKey{Namespace: rsyslogRelpInstallerNamespace, Name: rsyslogRelpInstallerName}, ds); err != nil {
		return false, err
	}

	return ds.Status.NumberReady == ds.Status.DesiredNumberScheduled, nil
}

func createNetworkPolicyForEchoServer(ctx context.Context, c kubernetes.Interface, namespace string) error {
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-machine-to-rsyslog-relp-echo-server",
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "machine",
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name":     "rsyslog-relp-echo-server",
							"app.kubernetes.io/instance": "rsyslog-relp-echo-server",
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "rsyslog-relp-echo-server",
						},
					},
				}},
			}},
		},
	}

	return client.IgnoreAlreadyExists(c.Client().Create(ctx, networkPolicy))
}
