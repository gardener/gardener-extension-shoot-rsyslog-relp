#!/bin/bash

# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

if [[ -f /host/etc/systemd/system/rsyslog-configurator.service ]]; then
  chroot /host /bin/bash -c 'systemctl disable rsyslog-configurator; systemctl stop rsyslog-configurator; rm -f /etc/systemd/system/rsyslog-configurator.service'
fi

if [[ -d {{ .rsyslogRelpQueueSpoolDir}} ]]; then
  rm -rf {{ .rsyslogRelpQueueSpoolDir}}
fi

if [[ -f {{ .pathSyslogAuditPlugin}} ]]; then
  sed -i "s/^active\\>.*/active = no/i" {{ .pathSyslogAuditPlugin}}
fi
if [[ -f {{ .audispSyslogPluginPath}} ]]; then
  sed -i "s/^active\\>.*/active = no/i" {{ .audispSyslogPluginPath}}
fi

chroot /host /bin/bash -c 'if systemctl list-unit-files systemd-journald-audit.socket > /dev/null; then \
  systemctl enable systemd-journald-audit.socket; \
  systemctl start systemd-journald-audit.socket; \
  systemctl restart systemd-journald; \
fi'

if [[ -d {{ .pathAuditRulesBackupDir}} ]]; then
  if [[ -d {{ .pathAuditRulesDir}} ]]; then
    rm -rf {{ .pathAuditRulesDir}}
  fi
  mv {{ .pathAuditRulesBackupDir}} {{ .pathAuditRulesDir}}
  chroot /host /bin/bash -c 'if systemctl list-unit-files auditd.service > /dev/null; then augenrules --load; systemctl restart auditd; fi'
fi

if [[ -f {{ .pathRsyslogAuditConf}} ]]; then
  rm -f {{ .pathRsyslogAuditConf}}
  chroot /host /bin/bash -c 'if systemctl list-unit-files rsyslog.service > /dev/null; then systemctl restart rsyslog; fi'
fi

if [[ -d {{ .pathRsyslogTLSDir}} ]]; then
  rm -rf {{ .pathRsyslogTLSDir}}
fi

if [[ -d {{ .pathRsyslogOSCDir }} ]]; then
  rm -rf {{ .pathRsyslogOSCDir }}
fi