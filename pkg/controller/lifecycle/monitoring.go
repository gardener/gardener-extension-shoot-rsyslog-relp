// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"fmt"

	monitoringutils "github.com/gardener/gardener/pkg/component/observability/monitoring/utils"
	"github.com/gardener/gardener/pkg/controllerutils"
	kubernetesutils "github.com/gardener/gardener/pkg/utils/kubernetes"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/apis/rsyslog"
	"github.com/gardener/gardener-extension-shoot-rsyslog-relp/pkg/constants"
)

const (
	// Metrics for the rsyslog running on the nodes are fetched via the node-exporter k8s service.
	serviceName     = "node-exporter"
	portNameMetrics = "metrics"
)

func deployMonitoringConfig(ctx context.Context, c client.Client, namespace string, auditConfig *rsyslog.AuditConfig) error {
	configMapDashboards := emptyConfigMapDashboards(namespace)
	if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, configMapDashboards, func() error {
		metav1.SetMetaDataLabel(&configMapDashboards.ObjectMeta, "component", constants.ServiceName)
		metav1.SetMetaDataLabel(&configMapDashboards.ObjectMeta, "dashboard.monitoring.gardener.cloud/shoot", "true")
		configMapDashboards.Data = map[string]string{"rsyslog-relp-dashboard.json": dashboardConfig}
		return nil
	}); err != nil {
		return err
	}

	alertingRules := []monitoringv1.Rule{
		{
			Alert: "RsyslogTooManyRelpActionFailures",
			Expr:  intstr.FromString(`sum(rate(rsyslog_pstat_failed{origin="core.action",name="rsyslg-relp"}[5m])) / sum(rate(rsyslog_pstat_processed{origin="core.action",name="rsyslog-relp"}[5m])) > bool 0.02 == 1`),
			For:   ptr.To(monitoringv1.Duration("15m")),
			Labels: map[string]string{
				"service":    "rsyslog-relp",
				"severity":   "warning",
				"type":       "shoot",
				"visibility": "all",
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
				"type":       "shoot",
				"visibility": "all",
			},
			Annotations: map[string]string{
				"description": "The rsyslog relp action processing rate is 0 meaning that there is most likely something wrong with the rsyslog service.",
				"summary":     "Rsyslog relp action processing rate is 0",
			},
		},
	}

	if auditConfig == nil || auditConfig.Enabled {
		alertingRules = append(alertingRules, monitoringv1.Rule{
			Alert: "RsyslogRelpAuditRulesNotLoadedSuccessfully",
			Expr:  intstr.FromString(`absent(rsyslog_augenrules_load_success == 1)`),
			For:   ptr.To(monitoringv1.Duration("15m")),
			Labels: map[string]string{
				"service":    "rsyslog-relp",
				"severity":   "warning",
				"type":       "shoot",
				"visibility": "all",
			},
			Annotations: map[string]string{
				"description": "The rsyslog augenrules load success is 0 meaning that there was an error when calling 'augenrules --load' on the Shoot nodes",
				"summary":     "Rsyslog augenrules load success is 0",
			},
		})
	}

	prometheusRule := emptyPrometheusRule(namespace)
	if _, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, prometheusRule, func() error {
		metav1.SetMetaDataLabel(&prometheusRule.ObjectMeta, "component", constants.ServiceName)
		metav1.SetMetaDataLabel(&prometheusRule.ObjectMeta, "prometheus", "shoot")
		prometheusRule.Spec = monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:  "rsyslog-relp.rules",
				Rules: alertingRules,
			}},
		}
		return nil
	}); err != nil {
		return err
	}

	scrapeConfig := emptyScrapeConfig(namespace)
	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, c, scrapeConfig, func() error {
		metav1.SetMetaDataLabel(&scrapeConfig.ObjectMeta, "component", constants.ServiceName)
		metav1.SetMetaDataLabel(&scrapeConfig.ObjectMeta, "prometheus", "shoot")
		scrapeConfig.Spec = monitoringv1alpha1.ScrapeConfigSpec{
			HonorLabels:   ptr.To(false),
			ScrapeTimeout: ptr.To(monitoringv1.Duration("30s")),
			Scheme:        ptr.To("HTTPS"),
			// This is needed because the kubelets' certificates are not are generated for a specific pod IP
			TLSConfig: &monitoringv1.SafeTLSConfig{InsecureSkipVerify: ptr.To(true)},
			Authorization: &monitoringv1.SafeAuthorization{Credentials: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "shoot-access-prometheus-shoot"},
				Key:                  "token",
			}},
			KubernetesSDConfigs: []monitoringv1alpha1.KubernetesSDConfig{{
				APIServer:  ptr.To("https://kube-apiserver"),
				Role:       "Endpoints",
				Namespaces: &monitoringv1alpha1.NamespaceDiscovery{Names: []string{metav1.NamespaceSystem}},
				Authorization: &monitoringv1.SafeAuthorization{Credentials: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "shoot-access-prometheus-shoot"},
					Key:                  "token",
				}},
				// This is needed because we do not fetch the correct cluster CA bundle right now
				TLSConfig:       &monitoringv1.SafeTLSConfig{InsecureSkipVerify: ptr.To(true)},
				FollowRedirects: ptr.To(true),
			}},
			RelabelConfigs: []monitoringv1.RelabelConfig{
				{
					Action:      "replace",
					Replacement: ptr.To("rsyslog-metrics"),
					TargetLabel: "job",
				},
				{
					TargetLabel: "type",
					Replacement: ptr.To("shoot"),
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
					Replacement: ptr.To("kube-apiserver:443"),
				},
				{
					SourceLabels: []monitoringv1.LabelName{"__meta_kubernetes_pod_name", "__meta_kubernetes_pod_container_port_number"},
					Regex:        `(.+);(.+)`,
					TargetLabel:  "__metrics_path__",
					Replacement:  ptr.To("/api/v1/namespaces/kube-system/pods/${1}:${2}/proxy/metrics"),
				},
			},
			MetricRelabelConfigs: monitoringutils.StandardMetricRelabelConfig("rsyslog_.+"),
		}
		return nil
	})

	return err
}

func deleteMonitoringConfig(ctx context.Context, client client.Client, namespace string) error {
	return kubernetesutils.DeleteObjects(ctx, client,
		emptyConfigMapDashboards(namespace),
		emptyPrometheusRule(namespace),
		emptyScrapeConfig(namespace),
	)
}

func emptyConfigMapDashboards(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-dashboards", constants.ServiceName), Namespace: namespace}}
}

func emptyPrometheusRule(namespace string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{ObjectMeta: monitoringutils.ConfigObjectMeta(constants.ServiceName, namespace, "shoot")}
}

func emptyScrapeConfig(namespace string) *monitoringv1alpha1.ScrapeConfig {
	return &monitoringv1alpha1.ScrapeConfig{ObjectMeta: monitoringutils.ConfigObjectMeta(constants.ServiceName, namespace, "shoot")}
}
