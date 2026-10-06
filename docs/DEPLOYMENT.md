# Guía de Despliegue Técnico y Operación en Producción — SOLV

Esta guía documenta los pasos para la instalación, configuración, migración de base de datos y puesta en marcha del entorno On-Premise y producción para la plataforma **SOLV**.

---

## 1. Requisitos del Sistema

### Software Base
| Componente | Versión Requerida | Uso / Propósito |
|---|---|---|
| **OS** | Linux (Ubuntu 22.04 LTS / Debian 12 / RHEL 9) | Sistema operativo recomendado |
| **Go** | 1.26+ | Compilación del backend API y ejecutor de migraciones |
| **Node.js** | 20 LTS (npm 10.x) | Entorno de construcción del frontend Angular 22 |
| **PostgreSQL** | 18+ | Motor de base de datos relacional y storage de JSONB |
| **Docker Engine** | v27+ (con socket local `/var/run/docker.sock`) | Aislamiento y ejecución de sandboxes de evaluación |
| **Traefik** | v3+ | Proxy inverso, enrutamiento dinámico y terminación TLS |

---

## 2. Variables de Entorno (`.env`)

Crear el archivo `.env` en la raíz del proyecto o en el directorio del servidor backend:

```bash
# Servidor HTTP Backend
PORT=8080
ENV=production
LOG_LEVEL=info
TENANT_ID=default

# Persistencia PostgreSQL
DATABASE_URL=postgres://solv_user:solv_password_secret@localhost:5432/solv_db?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=15m

# Seguridad y Autenticación
JWT_SECRET=super-secret-jwt-key-change-in-production-32bytes
JWT_EXPIRATION_HOURS=24

# Integración Docker Engine
DOCKER_HOST=unix:///var/run/docker.sock
SANDBOX_MEMORY_LIMIT_MB=256
SANDBOX_TIME_LIMIT_MS=3000

# Proxy & ForwardAuth (Traefik)
ALLOWED_ORIGINS=https://solv.uab.edu.bo
```

---

## 3. Base de Datos y Migraciones

El backend utiliza **Goose** para la ejecución idempotente de migraciones ordenadas por número en la carpeta `backend/migrations/`.

### Ejecución de Migraciones
```bash
# Cambiar al directorio del backend
cd backend

# Ejecutar todas las migraciones pendientes hasta la última versión (ej. 00019)
go run ./cmd/migrate -dir ./migrations up
```

### Verificación del Estado de Migraciones
```bash
go run ./cmd/migrate -dir ./migrations status
```

---

## 4. Compilación e Instalación del Backend (Go)

```bash
cd backend

# Descargar dependencias Go
go mod download

# Ejecutar tests de integración y unidad
go test ./internal/core/services/... ./internal/infrastructure/database/...

# Compilar binario de producción
go build -o build/solv-api ./cmd/api

# Ejecutar el servicio
./build/solv-api
```

---

## 5. Compilación del Frontend (Angular 22)

```bash
cd frontend

# Registrar e instalar dependencias
npm install

# Verificar estilo y reglas del sistema de diseño
npm run lint:styles

# Compilar paquete estático de producción
CI=true NG_CLI_ANALYTICS=false npm run build
```

El artefacto resultante se generará en `frontend/dist/frontend` listo para ser servido vía Nginx o Traefik.

---

## 6. Configuración e Aislamiento del Socket de Docker

El backend de SOLV interactúa directamente con el Docker Engine del host para crear contenedores efímeros de evaluación.

```bash
# Verificar que el grupo 'docker' existe y asignar permisos al usuario de SOLV
sudo usermod -aG docker $USER

# Verificar acceso correcto al socket sin sudo
docker ps

# Asegurar que el socket /var/run/docker.sock tenga permisos correctos
sudo chmod 660 /var/run/docker.sock
```

> [!IMPORTANT]
> En la cadena `DOCKER-USER` de `iptables`, asegúrese de mantener la regla Zero-Trust que impide que los contenedores de evaluación de estudiantes accedan a la red interna del servidor o a interfaces de administración.
