// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
	"github.com/gardener/gardener/pkg/utils"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

func deployMonitoringConfig(ctx context.Context, client client.Client, namespace string) error {
	monitoring := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-prometheus", constants.ServiceName),
			Namespace: namespace,
		},
		Data: map[string]string{
			v1beta1constants.PrometheusConfigMapScrapeConfig:   monitoringScrapeConfig,
			v1beta1constants.PrometheusConfigMapAlertingRules:  fmt.Sprintf("rsyslog-relp.rules.yaml: |\n  %s\n", utils.Indent(monitoringAlertingRules, 2)),
			v1beta1constants.PlutonoConfigMapOperatorDashboard: fmt.Sprintf("rsyslog-relp-dashboard.json: '%s'", dashboardConfig),
		},
	}

	utilruntime.Must(kubernetesutils.MakeUnique(monitoring))

	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, client, monitoring, func() error {
		metav1.SetMetaDataLabel(&monitoring.ObjectMeta, "component", constants.ServiceName)
		metav1.SetMetaDataLabel(&monitoring.ObjectMeta, "extensions.gardener.cloud/configuration", "monitoring")
		metav1.SetMetaDataLabel(&monitoring.ObjectMeta, references.LabelKeyGarbageCollectable, references.LabelValueGarbageCollectable)
		return nil
	})

	return err
}
