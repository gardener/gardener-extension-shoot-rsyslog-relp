// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/charts"
	apisconfig "github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/config"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/imagevector"
)

const (
	// ActuatorName is the name of the rsyslog relp actuator.
	ActuatorName = constants.ServiceName + "-actuator"

	releaseName                      = "rsyslog-relp-configurator"
	configurationCleanerReleaseName  = "rsyslog-relp-configuration-cleaner"
	deletionTimeout                  = time.Minute * 2
	nodeExporterTextfileCollectorDir = "/var/lib/node-exporter/textfile-collector"
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(client client.Client, decoder runtime.Decoder, config apisconfig.Configuration, chartRendererFactory extensionscontroller.ChartRendererFactory) extension.Actuator {
	return &actuator{
		client:               client,
		decoder:              decoder,
		config:               config,
		chartRendererFactory: chartRendererFactory,
	}
}

type actuator struct {
	chartRendererFactory extensionscontroller.ChartRendererFactory

	client  client.Client
	decoder runtime.Decoder
	config  apisconfig.Configuration
}

// Reconcile reconciles the extension resource.
func (a *actuator) Reconcile(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	if cluster.Shoot == nil {
		return errors.New("cluster.shoot is not yet populated")
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceName); err != nil {
		return err
	}

	timeoutCtx, cancelCtx := context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	if err := managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ManagedResourceName); err != nil {
		return err
	}

	return deployMonitoringConfig(ctx, a.client, namespace)
}

// Delete deletes the extension resource.
func (a *actuator) Delete(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	if cluster.Shoot == nil {
		return errors.New("cluster.shoot is not yet populated")
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceName); err != nil {
		return err
	}

	timeoutCtx, cancelCtx := context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	if err := managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ManagedResourceName); err != nil {
		return err
	}

	// If the Shoot is in deletion, then there is no need to clean up the rsyslog configuration from Nodes.
	// The Shoot deletion flows ensures that the Worker is deleted before the Extension deletion.
	// Hence, there are no Nodes, no need to clean up rsyslog configuration.
	if cluster.Shoot.DeletionTimestamp != nil {
		return nil
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
		"pspDisabled": gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
	}

	release, err := chartRenderer.RenderEmbeddedFS(charts.InternalChart, charts.RsyslogConfigurationCleanerChartPath, configurationCleanerReleaseName, metav1.NamespaceSystem, values)
	if err != nil {
		return err
	}

	rsyslogRelpCleanerChart := release.AsSecretData()
	if err := managedresources.CreateForShoot(ctx, a.client, namespace, constants.ManagedResourceNameConfigCleaner, "rsyslog-relp", false, rsyslogRelpCleanerChart); err != nil {
		return err
	}

	timeoutCtx, cancelCtx = context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	if err := managedresources.WaitUntilHealthy(timeoutCtx, a.client, namespace, constants.ManagedResourceNameConfigCleaner); err != nil {
		return err
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceNameConfigCleaner); err != nil {
		return err
	}

	timeoutCtx, cancelCtx = context.WithTimeout(ctx, deletionTimeout)
	defer cancelCtx()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ManagedResourceNameConfigCleaner)
}

// ForceDelete deletes the extension resource.
//
// We don't need to wait for the ManagedResource deletion because ManagedResources are finalized by gardenlet
// in later step in the Shoot force deletion flow.
func (a *actuator) ForceDelete(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	cluster, err := extensionscontroller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	if cluster.Shoot == nil {
		return errors.New("cluster.shoot is not yet populated")
	}

	return managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceName)
}

// Restore restores the extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate migrates the extension resource.
func (a *actuator) Migrate(ctx context.Context, _ logr.Logger, ex *extensionsv1alpha1.Extension) error {
	namespace := ex.GetNamespace()

	// Keep objects for shoot managed resources so that they are not deleted from the shoot during the migration
	if err := managedresources.SetKeepObjects(ctx, a.client, namespace, constants.ManagedResourceName, true); err != nil {
		return err
	}

	if err := managedresources.DeleteForShoot(ctx, a.client, namespace, constants.ManagedResourceName); err != nil {
		return err
	}

	twoMinutes := time.Minute * 2
	timeoutCtx, cancelCtx := context.WithTimeout(ctx, twoMinutes)
	defer cancelCtx()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, constants.ManagedResourceName)
}
