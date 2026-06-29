{{- define "zorba.userNodeAffinity" -}}
{{- if .Values.global.nodeAffinityUserMode }}
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
