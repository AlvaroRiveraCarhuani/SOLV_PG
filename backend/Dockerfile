# Imagen del backend SOLV.
#
# Contexto de build: la raíz del repositorio (`context: .`), porque el binario
# necesita el código de backend/ y el directorio de migraciones.
#
#   docker build -f backend/Dockerfile -t solv-backend:2.0.0 .
#
# La versión, el commit y la fecha de compilación entran por -ldflags y son los
# que publica GET /api/v1/version.

FROM golang:1.25-bookworm AS build

WORKDIR /src

# Las dependencias primero: mientras go.mod y go.sum no cambien, esta capa se
# reutiliza y el build no vuelve a descargar el árbol completo.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

ENV CGO_ENABLED=0 GOOS=linux

RUN go build -trimpath \
      -ldflags "-s -w \
        -X solv-backend/internal/version.Version=${VERSION} \
        -X solv-backend/internal/version.Commit=${COMMIT} \
        -X solv-backend/internal/version.BuildDate=${BUILD_DATE}" \
      -o /out/solv-api ./cmd/api

# Comando de operación para dar de alta instituciones sin SQL (ADR-081).
RUN go build -trimpath -ldflags "-s -w" -o /out/solv-tenant ./cmd/tenant

FROM debian:12-slim AS runtime

# curl sostenta el HEALTHCHECK y tzdata mantiene las fechas del calendario
# académico en la zona del servidor. ca-certificates es necesario para OAuth,
# proveedores de IA, backups remotos y desec.io.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir -p /var/lib/solv/backups

WORKDIR /app

COPY --from=build /out/solv-api /app/solv-api
COPY --from=build /out/solv-tenant /usr/local/bin/solv-tenant
COPY --from=build /src/migrations /app/migrations

# El proceso corre como root dentro del contenedor a propósito: necesita
# escribir en /var/run/docker.sock para aprovisionar laboratorios, y ese socket
# pertenece a root en el host. La contención real está en los cgroups y en las
# reglas DOCKER-USER, no en el usuario del contenedor de la API.
ENV MIGRATIONS_DIR=/app/migrations \
    BACKUP_DIR=/var/lib/solv/backups \
    PORT=3000

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
  CMD curl -fsS http://127.0.0.1:3000/api/v1/version || exit 1

ENTRYPOINT ["/app/solv-api"]
