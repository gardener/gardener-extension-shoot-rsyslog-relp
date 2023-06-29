// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/charts"
	apisconfig "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/imagevector"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/utils"
)

const (
	// ActuatorName is the name of the rsyslog relp actuator.
	ActuatorName = constants.ServiceName + "-actuator"

	releaseName     = "rsyslog-relp-configurator"
	deletionTimeout = time.Minute * 2
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(config apisconfig.Configuration, chartRendererFactory extensionscontroller.ChartRendererFactory) extension.Actuator {
	return &actuator{
		config:               config,
		chartRendererFactory: chartRendererFactory,
	}
}

type actuator struct {
	chartRendererFactory extensionscontroller.ChartRendererFactory

	client    client.Client
	clientset kubernetes.Interface
	decoder   runtime.Decoder
	config    apisconfig.Configuration
}

// InjectConfig injects the rest config to this actuator.
func (a *actuator) InjectConfig(config *rest.Config) error {
	var err error

	a.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("could not create Kubernetes client: %w", err)
	}

	return nil
}

// InjectClient injects the controller runtime client into the reconciler.
func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

// InjectScheme injects the given scheme into the reconciler.
func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return nil
}

// Reconcile reconciles the extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	if cluster.Shoot == nil {
		return errors.New("cluster.shoot is not yet populated")
	}

	projectName := utils.ProjectName(namespace, cluster.Shoot.Name)

	shootRsyslogRelpConfig := &rsyslog.RsyslogRelpConfig{}

	if ex.Spec.ProviderConfig != nil {
		if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, shootRsyslogRelpConfig); err != nil {
			return fmt.Errorf("failed to decode provider config: %w", err)
		}
	}

	chartRenderer, err := a.chartRendererFactory.NewChartRendererForShoot(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return fmt.Errorf("could not create chart renderer for shoot '%s', %w", namespace, err)
	}

	var filters = " *.* call defaultruleset"
	if shootRsyslogRelpConfig.Filters != nil {
		filters = *shootRsyslogRelpConfig.Filters
	}

	var reportSuspensionContinuation *string
	if shootRsyslogRelpConfig.ReportSuspensionContinuation != nil {
		if *shootRsyslogRelpConfig.ReportSuspensionContinuation {
			reportSuspensionContinuation = pointer.String("on")
		} else {
			reportSuspensionContinuation = pointer.String("off")
		}
	}

	rsyslogConfigValues := map[string]interface{}{
		"target":                       shootRsyslogRelpConfig.Target,
		"port":                         shootRsyslogRelpConfig.Port,
		"projectName":                  projectName,
		"shootName":                    cluster.Shoot.Name,
		"shootUID":                     cluster.Shoot.UID,
		"filters":                      filters,
		"rebindInterval":               shootRsyslogRelpConfig.RebindInterval,
		"timeout":                      shootRsyslogRelpConfig.Timeout,
		"resumeRetryCount":             shootRsyslogRelpConfig.ResumeRetryCount,
		"reportSuspensionContinuation": reportSuspensionContinuation,
	}

	if shootRsyslogRelpConfig.TLS.Enabled {
		refSecretName, err := lookupReferencedSecret(cluster, *shootRsyslogRelpConfig.TLS.SecretReferenceName)
		if err != nil {
			return err
		}

		refSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      refSecretName,
				Namespace: ex.Namespace,
			},
		}
		if err = a.client.Get(ctx, client.ObjectKeyFromObject(refSecret), refSecret); err != nil {
			return err
		}

		if err := utils.ValidateRsyslogRelpSecret(refSecret); err != nil {
			return err
		}

		var permittedPeer string
		if len(shootRsyslogRelpConfig.TLS.PermittedPeer) > 0 {
			permittedPeer = fmt.Sprintf("[%s]", strings.Join(shootRsyslogRelpConfig.TLS.PermittedPeer, ","))
		}

		var authMode string
		if shootRsyslogRelpConfig.TLS.AuthMode != nil {
			authMode = string(*shootRsyslogRelpConfig.TLS.AuthMode)
		}

		rsyslogConfigValues["tls"] = map[string]interface{}{
			"enabled":       shootRsyslogRelpConfig.TLS.Enabled,
			"permittedPeer": permittedPeer,
			"authMode":      authMode,
			"ca":            refSecret.Data["ca"],
			"crt":           refSecret.Data["crt"],
			"key":           refSecret.Data["key"],
		}
	}

	values := map[string]interface{}{
		"rsyslogConfig": rsyslogConfigValues,
		"auditdConfig": map[string]interface{}{
			"enabled": true,
		},
		"images": map[string]interface{}{
			"alpine": imagevector.AlpineImage(),
			"pause":  imagevector.PauseContainerImage(),
		},
	}

	release, err := chartRenderer.RenderEmbeddedFS(charts.InternalChart, charts.RsyslogConfiguratorChartPath, releaseName, metav1.NamespaceSystem, values)
	if err != nil {
		return err
	}

	rsyslogRelpChart := release.AsSecretData()
	return managedresources.CreateForShoot(ctx, a.client, namespace, constants.ManagedResourceNamesShoot, "rsyslog-relp", false, rsyslogRelpChart)
}

// Delete deletes the extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	if cluster.Shoot == nil {
		return errors.New("cluster.shoot is not yet populated")
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceNamesShoot); err != nil {
		return err
	}

	timeoutCtx, cancelCtx := context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	if err := managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ManagedResourceNamesShoot); err != nil {
		return err
	}

	chartRenderer, err := a.chartRendererFactory.NewChartRendererForShoot(cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return fmt.Errorf("could not create chart renderer for shoot '%s', %w", namespace, err)
	}

	values := map[string]interface{}{
		"images": map[string]interface{}{
			"alpine": imagevector.AlpineImage(),
			"pause":  imagevector.PauseContainerImage(),
		},
	}

	release, err := chartRenderer.RenderEmbeddedFS(charts.InternalChart, charts.RsyslogConfigurationCleanerChartPath, releaseName, metav1.NamespaceSystem, values)
	if err != nil {
		return err
	}

	rsyslogRelpCleanerChart := release.AsSecretData()
	if err := managedresources.CreateForShoot(ctx, a.client, namespace, constants.ConfigCleanerManagedResourceNames, "rsyslog-relp", false, rsyslogRelpCleanerChart); err != nil {
		return err
	}

	timeoutCtx, cancelCtx = context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	if err := managedresources.WaitUntilHealthy(timeoutCtx, a.client, namespace, constants.ConfigCleanerManagedResourceNames); err != nil {
		return err
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ConfigCleanerManagedResourceNames); err != nil {
		return err
	}

	timeoutCtx, cancelCtx = context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ConfigCleanerManagedResourceNames)
}

// Restore restores the extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate migrates the extension resource.
func (a *actuator) Migrate(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	// Keep objects for shoot managed resources so that they are not deleted from the shoot during the migration
	if err := managedresources.SetKeepObjects(ctx, a.client, namespace, constants.ManagedResourceNamesShoot, true); err != nil {
		return err
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceNamesShoot); err != nil {
		return err
	}

	twoMinutes := time.Minute * 2
	timeoutCtx, cancelCtx := context.WithTimeout(ctx, twoMinutes)
	defer cancelCtx()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ManagedResourceNamesShoot)
}

func lookupReferencedSecret(cluster *extensionscontroller.Cluster, refname string) (string, error) {
	if cluster.Shoot != nil {
		for _, ref := range cluster.Shoot.Spec.Resources {
			if ref.Name == refname {
				if ref.ResourceRef.Kind != "Secret" {
					err := fmt.Errorf("invalid referenced resource, expected kind Secret, not %s: %s", ref.ResourceRef.Kind, ref.ResourceRef.Name)
					return "", err
				}
				return v1beta1constants.ReferencedResourcesPrefix + ref.ResourceRef.Name, nil
			}
		}
	}
	return "", fmt.Errorf("missing or invalid referenced resource: %s", refname)
}
