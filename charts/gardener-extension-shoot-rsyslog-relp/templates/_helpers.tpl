{{- define "leaderelectionid" -}}
extension-shoot-rsyslog-relp-leader-election
{{- end -}}


{{-  define "image" -}}
  {{- if .Values.image.ref -}}
  {{ .Values.image.ref }}
  {{- else -}}
  {{- if hasPrefix "sha256:" .Values.image.tag }}
  {{- printf "%s@%s" .Values.image.repository .Values.image.tag }}
  {{- else }}
  {{- printf "%s:%s" .Values.image.repository .Values.image.tag }}
  {{- end }}
  {{- end -}}
{{- end }}
