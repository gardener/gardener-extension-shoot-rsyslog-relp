#!/bin/bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

function configure_auditd() {
  if [[ ! -d {{ .pathAuditRulesBackupDir }} ]] && [[ -d {{ .pathAuditRulesDir }} ]]; then
    mv {{ .pathAuditRulesDir }} {{ .pathAuditRulesBackupDir }}
  fi

  restart_auditd=false

  if [[ ! -d {{ .pathAuditRulesDir }} ]]; then
    mkdir -p {{ .pathAuditRulesDir }}
  fi
  if ! diff -rq {{ .pathAuditRulesFromOSCDir }} {{ .pathAuditRulesDir }} ; then
    rm -rf {{ .pathAuditRulesDir }}/*
    cp -L {{ .pathAuditRulesFromOSCDir }}/* {{ .pathAuditRulesDir }}/
    augenrules --load
    restart_auditd=true
  fi

  if [[ -f {{ .pathSyslogAuditPlugin }} ]] && \
      grep -m 1 -qie  "^active\\>" "{{ .pathSyslogAuditPlugin }}" && \
      ! grep -m 1 -qie "^active\\> = yes" "{{ .pathSyslogAuditPlugin }}" ; then
    sed -i "s/^active\\>.*/active = yes/gi" {{ .pathSyslogAuditPlugin }}
    export restart_auditd=true
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

  restart_rsyslog=false

  if [[ ! -f {{ .pathRsyslogAuditConf }} ]] || ! diff -rq {{ .pathRsyslogAuditConfFromOSC }} {{ .pathRsyslogAuditConf }} ; then
    cp -fL {{ .pathRsyslogAuditConfFromOSC }} {{ .pathRsyslogAuditConf }}
    restart_rsyslog=true
  fi

  if [[ -d {{ .pathRsyslogTLSFromOSCDir }} ]] && [[ -n "$(ls -A "{{ .pathRsyslogTLSFromOSCDir }}" )" ]]; then
    if [[ ! -d {{ .pathRsyslogTLSDir }} ]]; then
      mkdir -m 0600 {{ .pathRsyslogTLSDir }}
    fi
    if ! diff -rq {{ .pathRsyslogTLSFromOSCDir }} {{ .pathRsyslogTLSDir }} ; then
      rm -rf {{ .pathRsyslogTLSDir }}/*
      cp -L {{ .pathRsyslogTLSFromOSCDir }}/* {{ .pathRsyslogTLSDir }}/
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