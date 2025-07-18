# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

template(name="SyslogForwarderTemplate" type="list") {
  constant(value=" ")
  constant(value="{{.ProjectName}}")
  constant(value=" ")
  constant(value="foo")
  constant(value=" ")
  constant(value="")
  constant(value=" ")
  property(name="hostname")
  constant(value=" ")
  property(name="pri")
  constant(value=" ")
  property(name="syslogtag")
  constant(value=" ")
  property(name="timestamp" dateFormat="rfc3339")
  constant(value=" ")
  property(name="procid")
  constant(value=" ")
  property(name="msgid")
  constant(value=" ")
  property(name="msg")
  constant(value=" ")
}

module(
  load="omrelp"
)

module(load="omprog")
module(
  load="impstats"
  interval="60"
  format="json"
  resetCounters="off"
  ruleset="process_stats"
  bracketing="on"
)

input(type="imuxsock" Socket="/run/systemd/journal/syslog")

ruleset(name="process_stats") {
  action(
    type="omprog"
    name="to_pstats_processor"
    binary="/var/lib/rsyslog-relp-configurator/process-rsyslog-pstats.sh"
  )
}

ruleset(name="relp_action_ruleset") {
  action(
    name="rsyslog-relp"
    type="omrelp"
    target="{{.Target}}"
    port="{{.Port}}"
    queue.type="linkedlist"
    queue.size="100000"
    queue.filename="rsyslog-relp-queue"
    queue.saveOnShutdown="on"
    queue.spoolDirectory="/var/log/rsyslog"
    queue.maxDiskSpace="48m"
    Template="SyslogForwarderTemplate"
  )
}
