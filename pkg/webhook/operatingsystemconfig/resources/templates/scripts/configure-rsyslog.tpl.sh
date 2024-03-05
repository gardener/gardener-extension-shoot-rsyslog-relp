#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function configure_auditd() {
  if [[ ! -d {{ .pathAuditRulesBackupDir }} ]] && [[ -d {{ .pathAuditRulesDir }} ]]; then
    mv {{ .pathAuditRulesDir }} {{ .pathAuditRulesBackupDir }}
  fi

  if [[ ! -d {{ .pathAuditRulesDir }} ]]; then
    mkdir -p {{ .pathAuditRulesDir }}
  fi
  if ! diff -rq {{ .pathAuditRulesFromOSCDir }} {{ .pathAuditRulesDir }} ; then
    rm -rf {{ .pathAuditRulesDir }}/*
    cp -L {{ .pathAuditRulesFromOSCDir }}/* {{ .pathAuditRulesDir }}/
    if [[ -f {{ .pathSyslogAuditPlugin }} ]]; then
      sed -i 's/no/yes/g' {{ .pathSyslogAuditPlugin }}
    fi
    augenrules --load
    systemctl restart auditd
  fi
}

function configure_rsyslog() {
  # Enable the rsyslog service so that necessary symlinks can be created under /etc/systemd/system (e.g. /etc/systemd/system/syslog.service)
  if ! systemctl is-enabled --quiet rsyslog.service ; then
    systemctl enable rsyslog.service
  fi

  if [[ ! -f {{ .pathRsyslogAuditConf }} ]] || ! diff -rq {{ .pathRsyslogAuditConfFromOSC }} {{ .pathRsyslogAuditConf }} ; then
    cp -fL {{ .pathRsyslogAuditConfFromOSC }} {{ .pathRsyslogAuditConf }}
    systemctl restart rsyslog
  elif ! systemctl is-active --quiet rsyslog.service ; then
    # Ensure that the rsyslog service is running.
    systemctl start rsyslog.service
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