{{/*
Common labels applied to every resource.
*/}}
{{- define "cortex.labels" -}}
app.kubernetes.io/part-of: cortex
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}
