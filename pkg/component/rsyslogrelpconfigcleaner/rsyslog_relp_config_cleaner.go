// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package rsyslogrelpconfigcleaner

import (
	"bytes"
	"context"
	_ "embed"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
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

	registry := managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)
	return registry.AddAllAndSerialize(daemonSet)
}

func getLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "rsyslog-relp-configuration-cleaner",
		"app.kubernetes.io/instance": "rsyslog-relp-configuration-cleaner",
	}
}

var (
	//go:embed resources/templates/scripts/clean-rsyslog.tpl.sh
	cleanRsyslogScriptTemplateContent string
	cleanRsyslogScript                bytes.Buffer
)

const (
	rsyslogOSCDir = "/var/lib/rsyslog-relp-configurator"

	rsyslogTLSDir        = "/etc/ssl/rsyslog"
	rsyslogTLSFromOSCDir = rsyslogOSCDir + "/tls"

	rsyslogConfigPath              = "/etc/rsyslog.d/60-audit.conf"
	rsyslogConfigFromOSCPath       = rsyslogOSCDir + "/rsyslog.d/60-audit.conf"
	configureRsyslogScriptPath     = rsyslogOSCDir + "/configure-rsyslog.sh"
	processRsyslogPstatsScriptPath = rsyslogOSCDir + "/process-rsyslog-pstats.sh"

	rsyslogRelpQueueSpoolDir = "/var/log/rsyslog"

	auditRulesDir          = "/etc/audit/rules.d"
	auditRulesBackupDir    = "/etc/audit/rules.d.original"
	auditSyslogPluginPath  = "/etc/audit/plugins.d/syslog.conf"
	audispSyslogPluginPath = "/etc/audisp/plugins.d/syslog.conf"
	auditRulesFromOSCDir   = rsyslogOSCDir + "/audit/rules.d"
)

func init() {
	var err error

	cleanRsyslogScriptTemplate, err := template.
		New("clean-rsyslog.sh").
		Funcs(sprig.TxtFuncMap()).
		Parse(cleanRsyslogScriptTemplateContent)
	if err != nil {
		panic(err)
	}

	if err := cleanRsyslogScriptTemplate.Execute(&cleanRsyslogScript, map[string]interface{}{
		"rsyslogRelpQueueSpoolDir":    "/host" + rsyslogRelpQueueSpoolDir,
		"pathRsyslogTLSDir":           "/host" + rsyslogTLSDir,
		"pathRsyslogTLSFromOSCDir":    "/host" + rsyslogTLSFromOSCDir,
		"pathAuditRulesDir":           "/host" + auditRulesDir,
		"pathAuditRulesBackupDir":     "/host" + auditRulesBackupDir,
		"pathAuditRulesFromOSCDir":    "/host" + auditRulesFromOSCDir,
		"pathSyslogAuditPlugin":       "/host" + auditSyslogPluginPath,
		"audispSyslogPluginPath":      "/host" + audispSyslogPluginPath,
		"pathRsyslogAuditConf":        "/host" + rsyslogConfigPath,
		"pathRsyslogAuditConfFromOSC": "/host" + rsyslogConfigFromOSCPath,
		"pathRsyslogOSCDir":           "/host" + rsyslogOSCDir,
	}); err != nil {
		panic(err)
	}
}

func computeCommand() []string {
	return []string{
		"sh",
		"-c",
		cleanRsyslogScript.String(),
	}
}
