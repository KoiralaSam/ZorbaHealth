{{- define "infra.name" -}}
{{ .Chart.Name }}
{{- end }}

{{- define "infra.labels" -}}
app: {{ .name }}
{{- end }}

{{- define "zorba.userNodeAffinity" -}}
{{- if and .Values.global .Values.global.nodeAffinityUserMode }}
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.azure.com/mode
              operator: In
              values:
                - user
{{- end }}
{{- end }}
