{{- define "darts-league.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "darts-league.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "darts-league.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "darts-league.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "darts-league.labels" -}}
helm.sh/chart: {{ include "darts-league.chart" . }}
app.kubernetes.io/name: {{ include "darts-league.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.global.labels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "darts-league.selectorLabels" -}}
app.kubernetes.io/name: {{ include "darts-league.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "darts-league.frontendName" -}}
{{- printf "%s-frontend" (include "darts-league.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "darts-league.backendName" -}}
{{- printf "%s-backend" (include "darts-league.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "darts-league.postgresName" -}}
{{- printf "%s-postgres" (include "darts-league.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "darts-league.frontendLabels" -}}
{{ include "darts-league.labels" . }}
app.kubernetes.io/component: frontend
{{- end -}}

{{- define "darts-league.backendLabels" -}}
{{ include "darts-league.labels" . }}
app.kubernetes.io/component: backend
{{- end -}}

{{- define "darts-league.postgresLabels" -}}
{{ include "darts-league.labels" . }}
app.kubernetes.io/component: postgres
{{- end -}}

{{- define "darts-league.frontendSelectorLabels" -}}
{{ include "darts-league.selectorLabels" . }}
app.kubernetes.io/component: frontend
{{- end -}}

{{- define "darts-league.backendSelectorLabels" -}}
{{ include "darts-league.selectorLabels" . }}
app.kubernetes.io/component: backend
{{- end -}}

{{- define "darts-league.postgresSelectorLabels" -}}
{{ include "darts-league.selectorLabels" . }}
app.kubernetes.io/component: postgres
{{- end -}}

{{- define "darts-league.frontendImage" -}}
{{- $repository := required "frontend.image.repository is required" .Values.frontend.image.repository -}}
{{- $tag := required "frontend.image.tag is required" .Values.frontend.image.tag -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}

{{- define "darts-league.backendImage" -}}
{{- $repository := required "backend.image.repository is required" .Values.backend.image.repository -}}
{{- $tag := required "backend.image.tag is required" .Values.backend.image.tag -}}
{{- printf "%s:%s" $repository $tag -}}
{{- end -}}

{{- define "darts-league.postgresImage" -}}
{{- printf "%s:%s" .Values.postgres.image.repository .Values.postgres.image.tag -}}
{{- end -}}

{{- define "darts-league.frontendNginxConfig" -}}
server {
    listen 80;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    location /api/ {
        proxy_pass http://{{ include "darts-league.backendName" . }}:{{ .Values.backend.service.port }};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
{{- end -}}

{{- define "darts-league.backendAuthSecretName" -}}
{{- default (printf "%s-backend-auth" (include "darts-league.fullname" .)) .Values.backend.auth.existingSecret -}}
{{- end -}}

{{- define "darts-league.backendDatabaseSecretName" -}}
{{- if .Values.externalDatabase.existingSecret -}}
{{- .Values.externalDatabase.existingSecret -}}
{{- else -}}
{{- printf "%s-backend-db" (include "darts-league.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "darts-league.postgresSecretName" -}}
{{- default (printf "%s-postgres" (include "darts-league.fullname" .)) .Values.postgres.auth.existingSecret -}}
{{- end -}}

{{- define "darts-league.databaseUrl" -}}
{{- if .Values.postgres.enabled -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=disable" .Values.postgres.auth.username .Values.postgres.auth.password (include "darts-league.postgresName" .) .Values.postgres.service.port .Values.postgres.auth.database -}}
{{- else -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=%s" .Values.externalDatabase.user .Values.externalDatabase.password .Values.externalDatabase.host .Values.externalDatabase.port .Values.externalDatabase.name .Values.externalDatabase.sslmode -}}
{{- end -}}
{{- end -}}
