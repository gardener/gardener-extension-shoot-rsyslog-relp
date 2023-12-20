#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

function configure_auditd() {
  if [[ ! -d /etc/audit/rules.d.original ]] && [[ -d /etc/audit/rules.d ]]; then
    mv /etc/audit/rules.d /etc/audit/rules.d.original
  fi

  if [[ ! -d /etc/audit/rules.d ]]; then
    mkdir -p /etc/audit/rules.d
  fi
  if ! diff -rq /var/lib/rsyslog-relp-configurator/audit/rules.d /etc/audit/rules.d ; then
    rm -rf /etc/audit/rules.d/*
    cp -L /var/lib/rsyslog-relp-configurator/audit/rules.d/* /etc/audit/rules.d/
    if [[ -f /etc/audit/plugins.d/syslog.conf ]]; then
      sed -i 's/no/yes/g' /etc/audit/plugins.d/syslog.conf
    fi
    augenrules --load
    systemctl restart auditd
  fi
}

function configure_rsyslog() {
  if [[ ! -f /etc/rsyslog.d/60-audit.conf ]] || ! diff -rq /var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf /etc/rsyslog.d/60-audit.conf ; then
    cp -fL /var/lib/rsyslog-relp-configurator/rsyslog.d/60-audit.conf /etc/rsyslog.d/60-audit.conf
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

# TODO(plkokanov): due to an issue on gardenlinux, syslog.service is missing: https://github.com/gardenlinux/gardenlinux/issues/1864.
# This means that for gardenlinux we have to skip the check if syslog.service exists for now.
# As rsyslog.service comes preinstalled on gardenlinux, this should not lead to configuration problems.
if grep -q gardenlinux /etc/os-release || \
  systemctl list-unit-files syslog.service > /dev/null && \
  systemctl list-unit-files rsyslog.service > /dev/null; then
  echo "Configuring rsyslog.service ..."
  configure_rsyslog
else
  echo "rsyslog.service and syslog.service are not installed, skipping configuration"
fi