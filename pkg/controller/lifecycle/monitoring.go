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

	dashboardConfig = `{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "-- Plutono --",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "gnetId": null,
  "graphTooltip": 1,
  "iteration": 1697800309188,
  "links": [],
  "panels": [
    {
      "collapsed": false,
      "datasource": null,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": 50,
      "panels": [],
      "title": "Input",
      "type": "row"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 24,
        "x": 0,
        "y": 1
      },
      "hiddenSeries": false,
      "id": 42,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_submitted{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages Submitted",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "collapsed": false,
      "datasource": null,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 8
      },
      "id": 39,
      "panels": [],
      "repeat": null,
      "title": "Actions",
      "type": "row"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 9
      },
      "hiddenSeries": false,
      "id": 51,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_processed{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages Processed",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": null,
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 9
      },
      "hiddenSeries": false,
      "id": 44,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": true,
        "values": true
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_failed{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "interval": "",
          "legendFormat": "{{name}}",
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages Failed",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": null,
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 16
      },
      "hiddenSeries": false,
      "id": 46,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_suspended{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "interval": "",
          "legendFormat": "{{name}}",
          "refId": "A"
        },
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_resumed{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "hide": false,
          "interval": "",
          "legendFormat": "{{name}}",
          "refId": "B"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Times Suspended",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": null,
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 16
      },
      "hiddenSeries": false,
      "id": 47,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_suspended_duration{node=~\"$Node\"}[5m])) by (name)",
          "interval": "",
          "legendFormat": "{{name}}",
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Time Spent Suspended ",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": null,
      "fieldConfig": {
        "defaults": {},
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 23
      },
      "hiddenSeries": false,
      "id": 48,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 1,
      "nullPointMode": "null",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 2,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_suspended{node=~\"$Node\"}[5m])) by (name)",
          "interval": "",
          "legendFormat": "{{name}}",
          "refId": "A"
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Times Resumed",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "individual"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        },
        {
          "format": "short",
          "label": null,
          "logBase": 1,
          "max": null,
          "min": null,
          "show": true
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "collapsed": false,
      "datasource": null,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 30
      },
      "id": 53,
      "panels": [],
      "title": "Queues",
      "type": "row"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 31
      },
      "hiddenSeries": false,
      "id": 55,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_enqueued{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages Enqueued",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 31
      },
      "hiddenSeries": false,
      "id": 57,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rsyslog_pstat_size{node=~\"$Node\"}) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages in Queue",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 38
      },
      "hiddenSeries": false,
      "id": 54,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_full{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Times Queue Was Full",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 38
      },
      "hiddenSeries": false,
      "id": 59,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": false,
        "current": true,
        "max": false,
        "min": false,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rsyslog_pstat_maxqsize{node=~\"$Node\"}) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Maximum Queue Size",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 45
      },
      "hiddenSeries": false,
      "id": 56,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_discarded_nf{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages Discarded When Queue Was Not Full",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 45
      },
      "hiddenSeries": false,
      "id": 62,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_discarded_full{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Messages Discarded When Queue Was Full",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "collapsed": false,
      "datasource": null,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 52
      },
      "id": 61,
      "panels": [],
      "title": "Resources",
      "type": "row"
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 53
      },
      "hiddenSeries": false,
      "id": 58,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_utime{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "User CPU Time",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 53
      },
      "hiddenSeries": false,
      "id": 63,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_stime{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "System CPU Time",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 60
      },
      "hiddenSeries": false,
      "id": 65,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_minflt{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Total Minor Faults",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 60
      },
      "hiddenSeries": false,
      "id": 66,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_majflt{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Total Major Faults",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 67
      },
      "hiddenSeries": false,
      "id": 68,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_oublock{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "File System Output Operations",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 67
      },
      "hiddenSeries": false,
      "id": 67,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_inblock{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "File System Input Operations",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 74
      },
      "hiddenSeries": false,
      "id": 64,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rsyslog_pstat_maxrss{node=~\"$Node\"}) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Max Resident Set Size",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 74
      },
      "hiddenSeries": false,
      "id": 71,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_nivcsw{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Number of Open Files",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 0,
        "y": 81
      },
      "hiddenSeries": false,
      "id": 69,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_nvcsw{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Voluntary Context Switches",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    },
    {
      "aliasColors": {},
      "bars": false,
      "dashLength": 10,
      "dashes": false,
      "datasource": "prometheus",
      "description": "",
      "editable": true,
      "error": false,
      "fieldConfig": {
        "defaults": {
          "links": []
        },
        "overrides": []
      },
      "fill": 1,
      "fillGradient": 0,
      "grid": {},
      "gridPos": {
        "h": 7,
        "w": 12,
        "x": 12,
        "y": 81
      },
      "hiddenSeries": false,
      "id": 70,
      "interval": null,
      "legend": {
        "alignAsTable": true,
        "avg": true,
        "current": true,
        "max": true,
        "min": true,
        "rightSide": true,
        "show": true,
        "total": false,
        "values": true
      },
      "lines": true,
      "linewidth": 2,
      "links": [],
      "nullPointMode": "connected",
      "options": {
        "alertThreshold": true
      },
      "percentage": false,
      "pluginVersion": "7.5.24",
      "pointradius": 5,
      "points": false,
      "renderer": "flot",
      "seriesOverrides": [],
      "spaceLength": 10,
      "stack": false,
      "steppedLine": false,
      "targets": [
        {
          "exemplar": true,
          "expr": "sum(rate(rsyslog_pstat_nivcsw{node=~\"$Node\"}[$__rate_interval])) by (name)",
          "format": "time_series",
          "hide": false,
          "instant": false,
          "interval": "",
          "intervalFactor": 2,
          "legendFormat": "{{name}}",
          "refId": "A",
          "step": 40
        }
      ],
      "thresholds": [],
      "timeFrom": null,
      "timeRegions": [],
      "timeShift": null,
      "title": "Involuntary Context Switches",
      "tooltip": {
        "shared": true,
        "sort": 0,
        "value_type": "cumulative"
      },
      "type": "graph",
      "xaxis": {
        "buckets": null,
        "mode": "time",
        "name": null,
        "show": true,
        "values": []
      },
      "yaxes": [
        {
          "$$hashKey": "object:211",
          "format": "none",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": true
        },
        {
          "$$hashKey": "object:212",
          "format": "pps",
          "logBase": 1,
          "max": null,
          "min": 0,
          "show": false
        }
      ],
      "yaxis": {
        "align": false,
        "alignLevel": null
      }
    }
  ],
  "refresh": "1h",
  "schemaVersion": 27,
  "style": "dark",
  "tags": [
    "image",
    "cache"
  ],
  "templating": {
    "list": [
      {
        "allValue": null,
        "current": {
          "selected": true,
          "text": [
            "All"
          ],
          "value": [
            "$__all"
          ]
        },
        "datasource": null,
        "definition": "label_values(kube_node_info{type=\"shoot\"}, node)",
        "description": null,
        "error": null,
        "hide": 0,
        "includeAll": true,
        "label": "",
        "multi": true,
        "name": "Node",
        "options": [],
        "query": {
          "query": "label_values(kube_node_info{type=\"shoot\"}, node)",
          "refId": "StandardVariableQuery"
        },
        "refresh": 2,
        "regex": "",
        "skipUrlSync": false,
        "sort": 0,
        "tagValuesQuery": "",
        "tags": [],
        "tagsQuery": "",
        "type": "query",
        "useTags": false
      }
    ]
  },
  "time": {
    "from": "now-3h",
    "to": "now"
  },
  "timepicker": {
    "refresh_intervals": [
      "10s",
      "30s",
      "1m",
      "5m",
      "15m",
      "30m",
      "1h"
    ],
    "time_options": [
      "5m",
      "15m",
      "1h",
      "3h",
      "6h",
      "12h",
      "24h",
      "2d",
      "7d",
      "14d"
    ]
  },
  "timezone": "utc",
  "title": "Rsyslog Stats",
  "uid": "extension-extension-shoot-rsyslog-relp",
  "version": 1
}`
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
