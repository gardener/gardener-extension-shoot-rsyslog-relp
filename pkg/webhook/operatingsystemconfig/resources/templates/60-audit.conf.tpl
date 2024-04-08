# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

template(name="SyslogForwarderTemplate" type="list") {
  constant(value=" ")
  constant(value="{{ .projectName }}")
  constant(value=" ")
  constant(value="{{ .shootName }}")
  constant(value=" ")
  constant(value="{{ .shootUID }}")
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
  {{- if .tls.tlsLib }}
  tls.tlslib="{{ .tls.tlsLib }}"
  {{- end }}
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
    target="{{ .target }}"
    port="{{ .port }}"
    Template="SyslogForwarderTemplate"
    {{- if .rebindInterval }}
    rebindInterval="{{ .rebindInterval }}"
    {{- end }}
    {{- if .timeout }}
    timeout="{{ .timeout }}"
    {{- end }}
    {{- if .resumeRetryCount }}
    action.resumeRetryCount="{{ .resumeRetryCount }}"
    {{- end }}
    {{- if .reportSuspensionContinuation }}
    action.reportSuspensionContinuation="{{ .reportSuspensionContinuation }}"
    {{- end }}
    {{- if .tls.enabled }}
    tls="on"
    tls.caCert="{{ .tls.caPath }}"
    tls.myCert="{{ .tls.certPath }}"
    tls.myPrivKey="{{ .tls.keyPath }}"
    {{- end }}
    {{- if .tls.authMode }}
    tls.authmode="{{ .tls.authMode }}"
    {{- end }}
    {{- if .tls.permittedPeer }}
    tls.permittedpeer=[{{ .tls.permittedPeer }}]
    {{- end }}
  )
}{{ printf "\n" }}

{{- range .filters }}
if {{ . }} then {
  call relp_action_ruleset
  stop
}
{{- end}}

input(type="imuxsock" Socket="/run/systemd/journal/syslog")