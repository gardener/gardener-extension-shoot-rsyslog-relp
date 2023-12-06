// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package lifecycle

const (
	monitoringPrometheusJobName = "rsyslog-metrics"
	// Metrics for the rsyslog running on the nodes are fetched via the node-exporter k8s service.
	serviceName     = "node-exporter"
	portNameMetrics = "metrics"

	monitoringAlertingRules = `groups:
- name: rsyslog-relp.rules
  rules:
  - alert: RsyslogTooManyRelpActionFailures
    expr: sum(rate(rsyslog_pstat_failed{origin="core.action",name="rsyslg-relp"}[5m])) / sum(rate(rsyslog_pstat_processed{origin="core.action",name="rsyslog-relp"}[5m])) > bool 0.02 == 1
    for: 15m
    labels:
      service: rsyslog-relp
      severity: warning
      type: shoot
      visibility: operator
    annotations:
      description: 'The rsyslog relp cumulative failure rate in processing action events is greater than 2%.'
      summary: 'Rsyslog relp has too many failed attempts to process action events'
  - alert: RsyslogRelpActionProcessingRateIsZero
    expr: rate(rsyslog_pstat_processed{origin="core.action",name="rsyslog-relp"}[5m]) == 0
    for: 15m
    labels:
      service: rsyslog-relp
      severity: warning
      type: seed
      visibility: operator
    annotations:
      description: 'The rsyslog relp action processing rate is 0 meaning that there is most likely something wrong with the rsyslog service.'
      summary: 'Rsyslog relp action processing rate is 0'
`

	monitoringScrapeConfig = `- job_name: ` + monitoringPrometheusJobName + `
  honor_labels: false
  scrape_timeout: 30s
  scheme: https
  tls_config:
    ca_file: /etc/prometheus/seed/ca.crt
  authorization:
    type: Bearer
    credentials_file: /var/run/secrets/gardener.cloud/shoot/token/token
  follow_redirects: false
  kubernetes_sd_configs:
  - role: endpoints
    api_server: https://kube-apiserver:443
    tls_config:
      ca_file: /etc/prometheus/seed/ca.crt
    authorization:
      type: Bearer
      credentials_file: /var/run/secrets/gardener.cloud/shoot/token/token
    namespaces:
      names: [ kube-system ]
  relabel_configs:
  - target_label: type
    replacement: shoot
  - source_labels:
    - __meta_kubernetes_service_name
    - __meta_kubernetes_endpoint_port_name
    action: keep
    regex: ` + serviceName + `;` + portNameMetrics + `
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - source_labels: [ __meta_kubernetes_pod_name ]
    target_label: pod
  - source_labels: [ __meta_kubernetes_pod_node_name ]
    target_label: node
  - target_label: __address__
    replacement: kube-apiserver:443
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_container_port_number]
    regex: (.+);(.+)
    target_label: __metrics_path__
    replacement: /api/v1/namespaces/kube-system/pods/${1}:${2}/proxy/metrics
  metric_relabel_configs:
  - source_labels: [ __name__ ]
    action: keep
    regex: rsyslog_.+
`
)
