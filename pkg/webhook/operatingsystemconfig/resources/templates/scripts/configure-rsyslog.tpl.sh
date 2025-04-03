#!/bin/bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

auditd_metrics_file="{{ .nodeExporterTextfileCollectorDir }}/rsyslog_auditd.prom"

function remove_auditd_config() {
  if [[ -d {{ .pathAuditRulesBackupDir }} ]]; then
    if [[ -f {{ .pathSyslogAuditPlugin }} ]]; then
      sed -i "s/^active\\>.*/active = no/i" {{ .pathSyslogAuditPlugin }}
    fi
    if [[ -f {{ .audispSyslogPluginPath }} ]]; then
      sed -i "s/^active\\>.*/active = no/i" {{ .audispSyslogPluginPath }}
    fi

    if [[ -d {{ .pathAuditRulesDir }} ]]; then
      rm -rf {{ .pathAuditRulesDir }}
    fi
    cp -fa {{ .pathAuditRulesBackupDir }} {{ .pathAuditRulesDir }}
    ## The original audit rules might be erroneus so we ignore any errors here.
    augenrules --load || true
    systemctl restart auditd

    if [[ -f "${auditd_metrics_file}" ]]; then
      rm -f "${auditd_metrics_file}"
    fi
    rm -rf {{ .pathAuditRulesBackupDir }}
  fi
}

function configure_auditd() {
  if [[ ! -d {{ .pathAuditRulesFromOSCDir }} ]] || [ -z "$( ls -A '{{ .pathAuditRulesFromOSCDir }}' )" ] ; then
    remove_auditd_config
    return 0
  fi

  if [[ ! -d {{ .pathAuditRulesBackupDir }} ]] && [[ -d {{ .pathAuditRulesDir }} ]]; then
    mv {{ .pathAuditRulesDir }} {{ .pathAuditRulesBackupDir }}
  fi

  restart_auditd=false

  if [[ ! -d {{ .pathAuditRulesDir }} ]]; then
    mkdir -p {{ .pathAuditRulesDir }}
  fi
  if ! diff -rq {{ .pathAuditRulesFromOSCDir }} {{ .pathAuditRulesDir }} ; then
    rm -rf {{ .pathAuditRulesDir }}/*
    cp -fL {{ .pathAuditRulesFromOSCDir }}/* {{ .pathAuditRulesDir }}/

    restart_auditd=true
  fi

  # TODO(plkokanov): remove the additional check whether $auditd_metrics_file exists after v0.9.0 is released.
  # This check is temporarily necessary for nodes on which the `configure-rsyslog.sh` script already ran and
  # the audit rules were configured, but the $auditd_metrics_file was not created because its parent dir was missing.
  # Reference: https://github.com/gardener/gardener-extension-shoot-rsyslog-relp/pull/256
  if [ "${restart_auditd}" ] || [[ ! -f "${auditd_metrics_file}" ]]; then
    if [[ ! -d {{ .nodeExporterTextfileCollectorDir }} ]]; then
      mkdir -p "{{ .nodeExporterTextfileCollectorDir }}"
    fi

    augenrules_load_metric="# HELP rsyslog_augenrules_load_success shows whether the 'augenrules --load' command was executed successfully or not.\n# TYPE rsyslog_augenrules_load_success gauge\nrsyslog_augenrules_load_success"
    error=$(augenrules --load 2>&1 > /dev/null)
    # Writing to the output ".prom" file has to be done with an atomic operation. This is why we first write to a temporary file
    # and then we move/rename the temporary file to the actual output file.
    if [[ -n "$error" ]]; then
      logger -p error "Error loading audit rules: $error"
      echo -e "${augenrules_load_metric} 0" > "${auditd_metrics_file}.tmp"
    else
      echo -e "${augenrules_load_metric} 1" > "${auditd_metrics_file}.tmp"
    fi
    mv "${auditd_metrics_file}.tmp" "${auditd_metrics_file}"
  fi

  path_syslog_audit_plugin={{ .pathSyslogAuditPlugin }}
  if [[ -f {{ .audispSyslogPluginPath }} ]]; then
    path_syslog_audit_plugin={{ .audispSyslogPluginPath }}
  fi
  if [[ -f "$path_syslog_audit_plugin" ]] && \
      grep -m 1 -qie  "^active\\>" "$path_syslog_audit_plugin" && \
      ! grep -m 1 -qie "^active\\> = yes" "$path_syslog_audit_plugin" ; then
    sed -i "s/^active\\>.*/active = yes/i" "$path_syslog_audit_plugin"
    restart_auditd=true
  fi

  if ! systemctl is-active --quiet auditd.service ; then
    # Ensure that the auditd service is running.
    systemctl start auditd.service
  elif [ "${restart_auditd}" = true ]; then
    systemctl restart auditd.service
  fi

  # If the `systemd-journald-audit.socket` socket exists and is enabled, then journald also fetches audit logs from it.
  # To avoid duplication we disable it and only rely on the syslog audit plugin.
  if systemctl list-unit-files systemd-journald-audit.socket > /dev/null ; then
    if systemctl is-enabled --quiet systemd-journald-audit.socket ; then
      systemctl disable systemd-journald-audit.socket
    fi
    if systemctl is-active --quiet systemd-journald-audit.socket ; then
      systemctl stop systemd-journald-audit.socket
      systemctl restart systemd-journald
    fi
  fi
}

function configure_rsyslog() {
  # Enable the rsyslog service so that necessary symlinks can be created under /etc/systemd/system (e.g. /etc/systemd/system/syslog.service)
  if ! systemctl is-enabled --quiet rsyslog.service ; then
    systemctl enable rsyslog.service
  fi

  if [[ ! -d {{ .rsyslogRelpQueueSpoolDir }} ]]; then
    mkdir -p {{ .rsyslogRelpQueueSpoolDir }}
  fi

  restart_rsyslog=false

  if [[ ! -f {{ .pathRsyslogAuditConf }} ]] || ! diff -rq {{ .pathRsyslogAuditConfFromOSC }} {{ .pathRsyslogAuditConf }} ; then
    cp -fL {{ .pathRsyslogAuditConfFromOSC }} {{ .pathRsyslogAuditConf }}
    restart_rsyslog=true
  fi

  if [[ -d {{ .pathRsyslogTLSFromOSCDir }} ]] && [[ -n "$(ls -A "{{ .pathRsyslogTLSFromOSCDir }}" )" ]]; then
    if [[ ! -d {{ .pathRsyslogTLSDir }} ]]; then
      mkdir -p {{ .pathRsyslogTLSDir }}
    fi
    if ! diff -rq {{ .pathRsyslogTLSFromOSCDir }} {{ .pathRsyslogTLSDir }} ; then
      rm -rf {{ .pathRsyslogTLSDir }}/*
      cp -fL {{ .pathRsyslogTLSFromOSCDir }}/* {{ .pathRsyslogTLSDir }}/
      restart_rsyslog=true
    fi
  elif [[ -d {{ .pathRsyslogTLSDir }} ]]; then
    rm -rf {{ .pathRsyslogTLSDir }}
  fi

  if ! systemctl is-active --quiet rsyslog.service ; then
    # Ensure that the rsyslog service is running.
    systemctl start rsyslog.service
  elif [ "${restart_rsyslog}" = true ]; then
    systemctl restart rsyslog.service
  fi
}

if systemctl list-unit-files auditd.service > /dev/null; then
  echo "Configuring auditd.service ..."
  configure_auditd
else
  echo "auditd.service is not installed, skipping configuration"
fi

if systemctl list-unit-files rsyslog.service > /dev/null; then
  echo "Configuring rsyslog.service ..."
  configure_rsyslog
else
  echo "rsyslog.service and syslog.service are not installed, skipping configuration"
fi