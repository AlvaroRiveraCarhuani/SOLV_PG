# Imagen del frontend SOLV: Angular compilado y servido por Nginx.
#
# Contexto de build: la raíz del repositorio (`context: .`), para poder copiar
# frontend/ y la configuración de Nginx desde infra/.
#
#   docker build -f frontend/Dockerfile -t solv-frontend:2.0.0 .

FROM node:22-bookworm-slim AS build

WORKDIR /app

ENV CI=true \
    NG_CLI_ANALYTICS=false

# El lockfile manda: npm ci falla si package.json y package-lock.json no
# coinciden, que es exactamente lo que se quiere en una imagen reproducible.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY frontend/ ./
RUN npm run build -- --configuration production

FROM nginx:1.27-alpine AS runtime

COPY infra/nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/frontend/browser /usr/share/nginx/html

EXPOSE 80
