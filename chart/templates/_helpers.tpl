{{/*
Resource name prefix. Defaults to the release name, but truncated to 53
chars so a 10-char suffix (e.g. "-postgres-0") still fits under the
63-char DNS label limit. Override with .Values.fullnameOverride.
*/}}
{{- define "RCN.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 53 | trimSuffix "-" -}}
{{- else -}}
{{- .Release.Name | trunc 53 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Common labels applied to every object in the chart.
Helm best practice — makes selecting/removing resources via `helm` easy.
*/}}
{{- define "RCN.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: RCN
app.kubernetes.io/instance: {{ include "RCN.fullname" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels — must be stable across upgrades (subset of the labels
above). Used by Service selectors and Deployment selectors.
*/}}
{{- define "RCN.selectorLabels" -}}
app.kubernetes.io/name: RCN
app.kubernetes.io/instance: {{ include "RCN.fullname" . }}
{{- end -}}

{{/*
Resolve the Secret name. If the user provided an existing Secret name in
values, return that; otherwise default to the chart-managed Secret name.
*/}}
{{- define "RCN.secretName" -}}
{{- if .Values.secrets.existingSecret -}}
{{ .Values.secrets.existingSecret }}
{{- else -}}
{{ include "RCN.fullname" . }}-secrets
{{- end -}}
{{- end -}}
