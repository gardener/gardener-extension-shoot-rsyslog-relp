// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	monitoringutils "github.com/gardener/gardener/pkg/component/observability/monitoring/utils"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
	"github.com/gardener/gardener/pkg/utils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

func deployMonitoringConfig(ctx context.Context, c client.Client, namespace string) error {
	// TODO(plkokanov): remove this in a future release.
	// Refer to https://github.com/gardener/gardener-extension-shoot-rsyslog-relp/issues/89 for more details.
	if err := c.DeleteAllOf(ctx, &corev1.ConfigMap{},
		client.InNamespace(namespace),
		client.MatchingLabels{
			"component": constants.ServiceName,
			"extensions.gardener.cloud/configuration": "monitoring",
			references.LabelKeyGarbageCollectable:     references.LabelValueGarbageCollectable,
		}); err != nil {
		return fmt.Errorf("could not delete immutable monitoring configmaps for component %q in namespace %q: %w", constants.ServiceName, namespace, err)
	}

	// TODO(rfranzke): Delete this if-condition after August 2024.
	if c.Get(ctx, client.ObjectKey{Name: "prometheus-shoot", Namespace: namespace}, &appsv1.StatefulSet{}) == nil {
		if err := kutil.DeleteObject(ctx, c, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-prometheus", constants.ServiceName), Namespace: namespace}}); err != nil {
			return fmt.Errorf("failed deleting %s ConfigMap: %w", fmt.Sprintf("%s-prometheus", constants.ServiceName), err)
		}

		configMapDashboards := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-dashboards", constants.ServiceName), Namespace: namespace}}
		if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, configMapDashboards, func() error {
			metav1.SetMetaDataLabel(&configMapDashboards.ObjectMeta, "component", constants.ServiceName)
			metav1.SetMetaDataLabel(&configMapDashboards.ObjectMeta, "dashboard.monitoring.gardener.cloud/shoot", "true")
			configMapDashboards.Data = map[string]string{"rsyslog-relp-dashboard.json": dashboardConfig}
			return nil
		}); err != nil {
			return err
		}

		prometheusRule := &monitoringv1.PrometheusRule{ObjectMeta: monitoringutils.ConfigObjectMeta(constants.ServiceName, namespace, "shoot")}
		if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, prometheusRule, func() error {
			metav1.SetMetaDataLabel(&prometheusRule.ObjectMeta, "component", constants.ServiceName)
			metav1.SetMetaDataLabel(&prometheusRule.ObjectMeta, "prometheus", "shoot")
			prometheusRule.Spec = monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{{
					Name: "rsyslog-relp.rules",
					Rules: []monitoringv1.Rule{
						{
							Alert: "RsyslogTooManyRelpActionFailures",
							Expr:  intstr.FromString(`sum(rate(rsyslog_pstat_failed{origin="core.action",name="rsyslg-relp"}[5m])) / sum(rate(rsyslog_pstat_processed{origin="core.action",name="rsyslog-relp"}[5m])) > bool 0.02 == 1`),
							For:   ptr.To(monitoringv1.Duration("15m")),
							Labels: map[string]string{
								"service":    "rsyslog-relp",
								"severity":   "warning",
								"type":       "shoot",
								"visibility": "operator",
							},
							Annotations: map[string]string{
								"description": "The rsyslog relp cumulative failure rate in processing action events is greater than 2%.",
								"summary":     "Rsyslog relp has too many failed attempts to process action events",
							},
						},
						{
							Alert: "RsyslogRelpActionProcessingRateIsZero",
							Expr:  intstr.FromString(`rate(rsyslog_pstat_processed{origin="core.action",name="rsyslog-relp"}[5m]) == 0`),
							For:   ptr.To(monitoringv1.Duration("15m")),
							Labels: map[string]string{
								"service":    "rsyslog-relp",
								"severity":   "warning",
								"type":       "seed",
								"visibility": "operator",
							},
							Annotations: map[string]string{
								"description": "The rsyslog relp action processing rate is 0 meaning that there is most likely something wrong with the rsyslog service.",
								"summary":     "Rsyslog relp action processing rate is 0",
							},
						},
					},
				}},
			}
			return nil
		}); err != nil {
			return err
		}

		scrapeConfig := &monitoringv1alpha1.ScrapeConfig{ObjectMeta: monitoringutils.ConfigObjectMeta(constants.ServiceName, namespace, "shoot")}
		if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, scrapeConfig, func() error {
			metav1.SetMetaDataLabel(&scrapeConfig.ObjectMeta, "component", constants.ServiceName)
			metav1.SetMetaDataLabel(&scrapeConfig.ObjectMeta, "prometheus", "shoot")
			scrapeConfig.Spec = monitoringv1alpha1.ScrapeConfigSpec{
				HonorLabels:   ptr.To(false),
				ScrapeTimeout: ptr.To(monitoringv1.Duration("30s")),
				Scheme:        ptr.To("HTTPS"),
				// This is needed because the kubelets' certificates are not are generated for a specific pod IP
				TLSConfig: &monitoringv1.SafeTLSConfig{InsecureSkipVerify: true},
				Authorization: &monitoringv1.SafeAuthorization{Credentials: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "shoot-access-prometheus-shoot"},
					Key:                  "token",
				}},
				KubernetesSDConfigs: []monitoringv1alpha1.KubernetesSDConfig{{
					APIServer:  ptr.To("https://kube-apiserver"),
					Role:       "endpoints",
					Namespaces: &monitoringv1alpha1.NamespaceDiscovery{Names: []string{metav1.NamespaceSystem}},
					Authorization: &monitoringv1.SafeAuthorization{Credentials: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "shoot-access-prometheus-shoot"},
						Key:                  "token",
					}},
					// This is needed because we do not fetch the correct cluster CA bundle right now
					TLSConfig:       &monitoringv1.SafeTLSConfig{InsecureSkipVerify: true},
					FollowRedirects: ptr.To(true),
				}},
				RelabelConfigs: []*monitoringv1.RelabelConfig{
					{
						Action:      "replace",
						Replacement: "rsyslog-metrics",
						TargetLabel: "job",
					},
					{
						TargetLabel: "type",
						Replacement: "shoot",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_service_name", "__meta_kubernetes_endpoint_port_name"},
						Action:       "keep",
						Regex:        serviceName + `;` + portNameMetrics,
					},
					{
						Action: "labelmap",
						Regex:  `__meta_kubernetes_service_label_(.+)`,
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_pod_name"},
						TargetLabel:  "pod",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_pod_node_name"},
						TargetLabel:  "node",
					},
					{
						TargetLabel: "__address__",
						Replacement: "kube-apiserver:443",
					},
					{
						SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_pod_name", "__meta_kubernetes_pod_container_port_number"},
						Regex:        `(.+);(.+)`,
						TargetLabel:  "__metrics_path__",
						Replacement:  "/api/v1/namespaces/kube-system/pods/${1}:${2}/proxy/metrics",
					},
				},
				MetricRelabelConfigs: monitoringutils.StandardMetricRelabelConfig("rsyslog_.+"),
			}
			return nil
		}); err != nil {
			return err
		}

		return nil
	}

	// TODO(rfranzke): Delete this and the monitoring_config.go file after August 2024.
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

	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, monitoring, func() error {
		metav1.SetMetaDataLabel(&monitoring.ObjectMeta, "component", constants.ServiceName)
		metav1.SetMetaDataLabel(&monitoring.ObjectMeta, "extensions.gardener.cloud/configuration", "monitoring")
		return nil
	})

	return err
}
