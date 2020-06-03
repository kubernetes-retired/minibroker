{{/* vim: set filetype=mustache: */}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "minibroker.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Define the standard labels that will be applied to all objects in this chart.
*/}}
{{- define "minibroker.labels" -}}
app: {{ include "minibroker.fullname" . | quote }}
chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | quote }}
release: {{ .Release.Name | quote }}
heritage: {{ .Release.Service | quote }}
{{- end -}}
