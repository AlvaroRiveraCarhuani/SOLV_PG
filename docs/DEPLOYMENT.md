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

# Almacenamiento de snapshots de avances (SeaweedFS, ADR-058)
# Sin estas cuatro primeras, emitir y aplicar avances responde 503.
SEAWEEDFS_ENDPOINT=http://127.0.0.1:8333
SEAWEEDFS_BUCKET=solv-advances
SEAWEEDFS_ACCESS_KEY=cambiar_en_.env
SEAWEEDFS_SECRET_KEY=cambiar_en_.env
ADVANCE_SNAPSHOT_MAX_BYTES=4194304
ADVANCE_SNAPSHOT_MAX_FILES=512
ADVANCE_SNAPSHOT_EXCLUDES=.vscode/,.solv_*,.git/,node_modules/,.cache/,.openvscode-server/
ADVANCE_APPLY_CONCURRENCY=4
```

Las credenciales son las mismas que usa el servicio `seaweedfs` de `compose.yml`. El backend crea el bucket al arrancar si no existe, así que no hace falta prepararlo a mano.

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

## 4. Compilación manual del Backend (Go)

> Para una instalación institucional usá el despliegue con imágenes de la
> [sección 8](#8-imágenes-versiones-y-actualización). Esta sección describe el
> camino manual, útil para desarrollo y para diagnosticar un servidor.

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

---

## 7. Firewall y Backup Automáticos

El host no depende de cron ni de reglas aplicadas a mano. `infra/systemd/`
contiene cuatro unidades y `scripts/install.sh` las instala, habilita y deja el
sistema en un estado reproducible:

| Unidad | Qué hace | Cuándo |
|---|---|---|
| `solv-firewall.service` | Aplica las reglas Zero Trust en `DOCKER-USER` | Al arrancar y a pedido |
| `solv-firewall.timer` | Reaplica las reglas | Cada 15 minutos |
| `solv-backup.service` | Dump de `solv_db`, del PostgreSQL de aula y de los volúmenes | Diario, 03:00 |
| `solv-backup.timer` | Dispara el respaldo (se recupera si el host estuvo apagado) | Diario, 03:00 |

```bash
# Instación completa: unidades, timers, directorios y pila
sudo bash scripts/install.sh

# Solo preparar el host, sin levantar contenedores
sudo bash scripts/install.sh --no-start

# Ver qué haría, sin tocar nada
bash scripts/install.sh --dry-run
```

El script es idempotente: escribe una unidad solo si su contenido cambió y
`systemctl enable --now` no duplica nada. Ejecutarlo dos veces seguidas deja el
host igual que la primera vez.

### Verificación

```bash
systemctl status solv-firewall.service
systemctl list-timers 'solv-*'

# Las reglas deben seguir presentes después de un reinicio del host
iptables -S DOCKER-USER
```

### Restaurar un respaldo

```bash
# Restaura en una base temporal y verifica contenido. No toca la base viva.
bash scripts/restore.sh backups/2026-11/solv_db_20261113_030000.sql.gz

# Reemplazar la base viva (detiene el backend y conserva la anterior)
bash scripts/restore.sh <archivo> --promote --yes
```

La restauración por defecto es no destructiva a propósito: primero se verifica
que el archivo tenga las tablas esperadas, y solo después se decide promover.
La base anterior queda como `<nombre>_pre_restore_<fecha>` hasta que se confirme
la operación.

---

## 8. Imágenes, versiones y actualización

La pila completa se levanta con `compose.yml`. El backend y el frontend son
imágenes propias con tag; Traefik, PostgreSQL, SeaweedFS, Prometheus y Grafana
conservan las versiones fijadas del archivo.

### Variables que gobiernan el despliegue

| Variable | Uso |
|---|---|
| `SOLV_HOST` | Host público del sitio. Define rutas, certificado wildcard y el dominio de los entornos de práctica. Reemplaza al literal que antes vivía en `infra/traefik/dynamic_conf.yml`. |
| `SOLV_VERSION` | Tag de `solv-backend` y `solv-frontend`. Cambiarlo es actualizar; volver al valor anterior es revertir. |
| `POSTGRES_PASSWORD` | Clave del usuario `solv_user`. Compose la usa para la base y para el `DATABASE_URL` del backend. |

Dentro de la red de compose el backend resuelve `postgres` y `seaweedfs` por
nombre de servicio. El `DATABASE_URL` y el `SEAWEEDFS_ENDPOINT` del `.env`
apuntan a `127.0.0.1` porque están escritos para ejecutar el binario en el host;
compose los sobrescribe para el contenedor.

### Construir las imágenes con su versión

```bash
# Versión, commit y fecha salen de git y del reloj del servidor
make images

# Equivalente explícito
docker compose build
```

Los tres valores se inyectan con `-ldflags` y son los que responde el endpoint
de versión. Un build local sin ellos informa `dev` / `unknown`.

### Levantar y actualizar

```bash
# Primera puesta en marcha
cp example.env .env    # completar secretos
SOLV_VERSION=2.0.0 docker compose up -d

# Actualizar a una versión publicada
docker compose pull
SOLV_VERSION=2.1.0 docker compose up -d

# Volver a la versión anterior (el esquema no se revierte solo: ver abajo)
SOLV_VERSION=2.0.0 docker compose up -d
```

Las migraciones se aplican al arrancar el backend, con el lock de PostgreSQL
`pg_advisory_lock(1337)`. La versión aplicada se publica en `/api/v1/version`.

### Actualizar con el script

`scripts/upgrade.sh` hace los pasos en el único orden que admite vuelta atrás:

```bash
# Actualizar a una versión publicada
bash scripts/upgrade.sh --version 2.1.0

# Ver el plan sin tocar nada
bash scripts/upgrade.sh --version 2.1.0 --dry-run

# Construir en el servidor en lugar de descargar la imagen
bash scripts/upgrade.sh --version 2.1.0 --build
```

| Paso | Qué ocurre | Si falla |
|---|---|---|
| 1 | Lee `/api/v1/version` y el tag del contenedor `solv_backend` | Aborta sin tocar nada: sin versión actual no hay vuelta atrás |
| 2 | `pull` (o `build`) de `backend` y `frontend` con el tag destino | Aborta; la pila sigue en la versión anterior |
| 3 | Ejecuta `infra/scripts/backup.sh` y verifica el `.sql.gz` con `gzip -t` | Aborta; no se levanta nada sin respaldo válido |
| 4 | `docker compose up -d backend frontend` con `SOLV_VERSION` nuevo | — |
| 5 | Espera `/api/v1/health` en 200 y `/api/v1/version` con el tag nuevo | Vuelve a la imagen anterior y restaura el respaldo previo |

La espera por defecto es de 180 segundos (`--timeout`). Si la versión nueva no
queda sana, el script detiene el backend, restaura el respaldo previo con
`scripts/restore.sh --promote --yes`, recrea los contenedores con el tag anterior
y verifica que `/api/v1/health` responda. Termina con código 1 y el detalle.

`--skip-backup` existe para ensayos, no para producción: sin respaldo previo la
vuelta atrás solo puede volver la imagen, no los datos. `--skip-db-rollback`
vuelve la imagen dejando la base como quedó.

El script exige que la pila esté respondiendo antes de empezar y que exista
`.env`. Con `make upgrade VERSION=2.1.0` se invoca igual.

### Endpoint de versión

`GET /api/v1/version` responde sin autenticación:

```json
{
  "data": {
    "version": "2.1.0",
    "commit": "a1b2c3d",
    "build_date": "2026-11-13T10:00:00Z",
    "go_version": "go1.25.0",
    "migration_version": 57,
    "database": "ok"
  },
  "error": "",
  "message": "Versión del sistema"
}
```

`database` vale `unavailable` cuando el proceso vive pero no alcanza a
PostgreSQL, y en ese caso `migration_version` es `null`. Es el mismo endpoint
que usa el `HEALTHCHECK` del contenedor y el target `make version`.

### Revertir una actualización

La vuelta atrás soportada es **imagen anterior + respaldo previo**, no
`goose down`. Un `down` sobre una base con datos en una migración que
transformó o borró algo no restaura el estado: pierde datos con pasos extra.

1. Restaurar el respaldo tomado antes de actualizar.
2. Levantar el tag anterior con `SOLV_VERSION`.
3. Confirmar en `/api/v1/version` que versión y migración coinciden con el respaldo.

---

## 9. Alta de institución

Una instalación nueva nace con una institución neutra (`default`, nombre
«Institución») para que el backend arranque y el panel responda. Esa fila no
identifica a nadie: el nombre, los dominios y el primer administrador se definen
con `solv-tenant` (ADR-081).

### Crear una institución

```bash
docker compose exec backend solv-tenant -action create \
  -name "Universidad Ejemplo" \
  -slug uejemplo \
  -admin-email admin@uejemplo.edu \
  -domains "uejemplo.edu,uejemplo.edu.bo" \
  -period-code 2026-I \
  -period-name "Primer Semestre 2026" \
  -period-start 2026-02-01 \
  -period-end 2026-06-30
```

Crea en una transacción la institución, su primer administrador, el periodo
académico y la materia ancla `GEN-101`. Si un paso falla, no queda una
institución a medio construir.

| Opción | Efecto |
|---|---|
| `-slug` | Identificador público: minúsculas, números y guiones. Se normaliza a minúsculas. |
| `-domains` | Dominios de correo permitidos, separados por coma. Vacío significa sin filtro por dominio: las cuentas se crean por invitación. |
| `-admin-email` | Correo del primer administrador. Debe ser único en toda la instalación. |
| `-period-*` | Periodo inicial. Sin fechas, se usa el año calendario en curso. |

Repetir el comando con el mismo slug no duplica nada: informa que la institución
ya existe y termina bien, así que el procedimiento se puede reejecutar.

### Ver las instituciones

```bash
docker compose exec backend solv-tenant -action list
```

### Completar la identidad

Después del alta, iniciar sesión con la cuenta administradora y completar en
**Configuración → Identidad**: nombre institucional, logo, color y correo de
soporte. El nombre que aparece en los correos de bienvenida a docentes sale de
ahí, no del código.

El dominio público del sitio no se configura acá: se define una sola vez en
`SOLV_HOST`, dentro del `.env`. Compose se detiene si falta.

