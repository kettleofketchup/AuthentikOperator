{{/*
TLS volume for CA certificate (Secret or ConfigMap source).
Only renders when caSecretRef or caConfigMapRef is set.
*/}}
{{- define "authentik-operator.tls-volumes" -}}
{{- if .Values.authentik.tls.caSecretRef }}
- name: authentik-ca-cert
  secret:
    secretName: {{ .Values.authentik.tls.caSecretRef }}
    items:
      - key: {{ .Values.authentik.tls.caSecretKey }}
        path: ca.crt
{{- else if .Values.authentik.tls.caConfigMapRef }}
- name: authentik-ca-cert
  configMap:
    name: {{ .Values.authentik.tls.caConfigMapRef }}
    items:
      - key: {{ .Values.authentik.tls.caConfigMapKey }}
        path: ca.crt
{{- end }}
{{- end }}

{{/*
TLS volume mount for CA certificate.
Only renders when caSecretRef or caConfigMapRef is set.
*/}}
{{- define "authentik-operator.tls-volumemounts" -}}
{{- if or .Values.authentik.tls.caSecretRef .Values.authentik.tls.caConfigMapRef }}
- name: authentik-ca-cert
  mountPath: /etc/ssl/authentik
  readOnly: true
{{- end }}
{{- end }}

{{/*
TLS-related container args.
Renders --authentik-ca-cert-path when a CA source is configured,
and --authentik-insecure-skip-verify when enabled.
*/}}
{{- define "authentik-operator.tls-args" -}}
{{- if or .Values.authentik.tls.caSecretRef .Values.authentik.tls.caConfigMapRef }}
- --authentik-ca-cert-path=/etc/ssl/authentik/ca.crt
{{- end }}
{{- if .Values.authentik.tls.insecureSkipVerify }}
- --authentik-insecure-skip-verify
{{- end }}
{{- end }}
