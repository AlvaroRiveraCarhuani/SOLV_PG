# Levantar SOLV en Fedora para probarlo en el navegador

Esta guía cubre el entorno local de una sola máquina:

- **Frontend Angular:** `http://localhost:4200`
- **Backend Go:** `http://localhost:3000`
- **PostgreSQL 18:** `127.0.0.1:5432`
- **SeaweedFS:** `127.0.0.1:8333` (solo necesario para avances y snapshots)

El navegador se abre en `http://localhost:4200`. El servidor de Angular
reenvía `/api`, `/auth` y `/ws` al backend de `:3000` usando
`frontend/proxy.conf.json`.

---

## 1. Requisitos en Fedora

Instalar o verificar:

```bash
git --version
go version          # 1.25 o superior; el go.mod declara 1.25.0
node --version      # Node 22 LTS
npm --version       # npm 10.x
docker --version
docker compose version
lsof -v             # scripts/dev.sh lo usa para detectar puertos
ss -ltn             # iproute
```

Si falta Docker Engine:

```bash
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager addrepo \
  --from-repofile=https://download.docker.com/linux/fedora/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io \
  docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker "$USER"
```

Después de `usermod`, cerrar sesión y volver a entrar para que el usuario
pueda usar el socket de Docker sin `sudo`. Verificar con:

```bash
docker ps
```

Si `go version` o `node --version` no alcanzan las versiones pedidas, instalar
el toolchain oficial de Go y Node 22 LTS antes de continuar. Los paquetes de
Fedora pueden venir atrasados respecto al `go.mod` y a Angular 22.

Comprobar que los puertos estén libres:

```bash
ss -ltnp | grep -E ':(3000|4200|5432|8333)\b'
```

Si alguno está ocupado, revisar qué proceso o contenedor lo usa antes de
levantar otro servicio.

---

## 2. Configurar `.env`

El archivo `.env` está en la raíz, está ignorado por Git y no se debe commitear.

```bash
cd SOLV_PG
cp example.env .env
```

Editar `.env` con estos valores mínimos para local:

```dotenv
DATABASE_URL=postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable
PORT=3000

JWT_SECRET=<valor-largo-y-aleatorio>
COOKIE_DOMAIN=

# Orígenes permitidos por CORS. Agregar la IP o el nombre desde el que se abre
# el navegador; por ejemplo, la IP de Tailscale del servidor.
ALLOWED_ORIGINS=http://localhost:4200,http://127.0.0.1:4200

SEAWEEDFS_ENDPOINT=http://127.0.0.1:8333
SEAWEEDFS_BUCKET=solv-advances
SEAWEEDFS_ACCESS_KEY=<clave-local>
SEAWEEDFS_SECRET_KEY=<secreto-local>
```

Generar los valores aleatorios sin escribirlos en el chat ni en el repositorio:

```bash
openssl rand -hex 32   # JWT_SECRET
openssl rand -hex 16   # SEAWEEDFS_ACCESS_KEY
openssl rand -hex 32   # SEAWEEDFS_SECRET_KEY
```

`COOKIE_DOMAIN` se deja vacío para localhost. El backend ya ignora el dominio
de cookie cuando el host es `localhost` o `127.0.0.1`.

Si se va a usar el login con Google, completar además:

```dotenv
GOOGLE_CLIENT_ID=<client-id>
GOOGLE_CLIENT_SECRET=<client-secret>
GOOGLE_REDIRECT_URL=http://localhost:3000/auth/google/callback
```

En Google Cloud Console, el redirect URI autorizado debe coincidir exactamente
con `http://localhost:3000/auth/google/callback`.

`DESEC_TOKEN` y `ACME_EMAIL` solo hacen falta si se levanta Traefik con el
compose completo. Para probar en localhost no se necesitan.

---

## 3. Levantar la infraestructura

Para el navegador alcanza con PostgreSQL y SeaweedFS:

```bash
docker compose up -d postgres seaweedfs
docker compose ps
```

No hace falta levantar `traefik`, `prometheus` ni `grafana` para probar la
aplicación en `localhost`. Traefik ocupa los puertos 80/443 y su configuración
de TLS apunta al dominio institucional.

> El compose exige `SEAWEEDFS_ACCESS_KEY` y `SEAWEEDFS_SECRET_KEY` en `.env`
> aunque solo se levante PostgreSQL, porque Compose interpola todo el archivo.

El backend ejecuta las migraciones de Goose al arrancar. No hay que correr un
comando de migración aparte para el entorno local.

---

## 4. Levantar backend y frontend

### Opción A: lanzador del proyecto

```bash
docker compose up -d postgres seaweedfs
./scripts/dev.sh start
```

El script compila el backend, lo deja en segundo plano, levanta Angular en
`:4200` y muestra la URL. Para ver el estado:

```bash
./scripts/dev.sh status
```

Para detenerlo:

```bash
./scripts/dev.sh stop
```

`dev.sh` solo arranca PostgreSQL; SeaweedFS se levanta aparte como en el paso
anterior. Los logs quedan en `scripts/.run/logs/`.

### Opción B: dos terminales

Terminal 1, backend:

```bash
cd SOLV_PG
set -a
source .env
set +a
cd backend
go run ./cmd/api
```

Terminal 2, frontend:

```bash
cd SOLV_PG/frontend
npm ci
npm start -- --host 0.0.0.0 --port 4200
```

Abrir `http://localhost:4200`.

---

## 5. Verificar que está arriba

```bash
curl -fsS http://localhost:3000/api/v1/config/public
curl -fsS -o /dev/null -w '%{http_code}\n' http://localhost:4200/
curl -fsS -o /dev/null -w '%{http_code}\n' http://localhost:4200/api/v1/config/public
```

Los tres deben responder `200`. Sin sesión, `GET /api/v1/users/me` debe
responder `401`:

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:3000/api/v1/users/me
```

---

## 6. Entrar al navegador

### Opción A: Google OAuth

1. Abrir `http://localhost:4200/login`.
2. Presionar **Continuar con Google Workspace**.
3. El backend debe tener `GOOGLE_REDIRECT_URL` apuntando a
   `http://localhost:3000/auth/google/callback` y ese URI debe estar
   autorizado en el cliente de Google.

### Opción B: sesión local sin Google

Útil para probar la UI sin configurar OAuth.

1. Sembrar los actores sintéticos en la base local:

   ```bash
   docker ps --format '{{.Names}}' | grep postgres
   docker exec -i solv_db psql -U solv_user -d solv_db \
     < backend/seeds/e2e_actors.sql
   ```

   Si el contenedor se llama distinto, reemplazar `solv_db` por el nombre que
   devuelve `docker ps`.

2. Generar el token de un rol:

   ```bash
   cd SOLV_PG
   node scripts/dev-session.mjs admin
   # también: teacher o student
   ```

   El script lee `JWT_SECRET` de `.env`, imprime el token y trata de copiarlo
   al portapapeles.

3. En el navegador, abrir `http://localhost:4200`, entrar a DevTools (F12) →
   **Almacenamiento** → **Cookies** → `http://localhost:4200` y agregar:

   | Campo | Valor |
   | :--- | :--- |
   | Nombre | `solv_session` |
   | Valor | el token generado |
   | Ruta | `/` |

   Recargar con `Ctrl+Shift+R`. La cookie debe quedar asociada a `localhost`,
   no a `.solv.dedyn.io`.

4. Verificar que el topbar muestra el rol y que
   `GET /api/v1/users/me` responde `200`.

> El seed actualiza el tenant
> `00000000-0000-0000-0000-000000000001` con el nombre `SOLV E2E`. Usarlo solo
> en una base local de pruebas.
>
> Al momento de esta guía, `solv_db` no tenía actores `@e2e.solv.test`
> sembrados. Por eso el paso 1 es necesario antes de usar `dev-session.mjs`.

---

## 7. Problemas frecuentes

| Síntoma | Causa probable | Acción |
| :--- | :--- | :--- |
| `connect ECONNREFUSED 127.0.0.1:3000` en el navegador | El backend no está escuchando en `:3000`. | Revisar la terminal o `scripts/.run/logs/backend.log`; comprobar `curl http://localhost:3000/api/v1/config/public`. |
| `port is already allocated` al levantar PostgreSQL | Ya hay otro PostgreSQL en `:5432`. | Ver `ss -ltnp` y `docker ps`; usar el contenedor existente o cambiar el puerto en `compose.yml`. |
| `dev.sh status` marca PostgreSQL detenido | Puede quedar un contenedor viejo con otro nombre ocupando `5432`; `dev.sh` busca `solv_db`. | Revisar `docker ps` y usar el contenedor que crea el compose (`solv_db`). |
| `permission denied` al usar Docker | El usuario no está en el grupo `docker`. | `sudo usermod -aG docker "$USER"` y volver a iniciar sesión. |
| El login con Google vuelve a `/login` | Redirect URI distinto o cookie de otro dominio. | Revisar `GOOGLE_REDIRECT_URL`, el URI autorizado y que `COOKIE_DOMAIN` esté vacío. |
| `dev-session.mjs` genera token pero `/users/me` da 401 | Los actores E2E no están en la base. | Ejecutar `backend/seeds/e2e_actors.sql` en la base que usa el backend. |
| El navegador remoto recibe `403 Forbidden` | El origen no está en `ALLOWED_ORIGINS`. | Agregar `http://IP:4200` a `ALLOWED_ORIGINS` y reiniciar el backend. |
| El botón de Google vuelve a `localhost` desde otra máquina | `GOOGLE_REDIRECT_URL` apunta a `localhost`. | Usar un túnel SSH o cambiar el redirect y autorizarlo en Google Cloud. |

---

## 8. Qué se verificó

Comprobaciones realizadas en el host de trabajo actual:

- `bash -n scripts/dev.sh` y `node --check scripts/dev-session.mjs`: sin errores.
- `GET http://127.0.0.1:3000/api/v1/config/public`: `200`.
- `GET http://127.0.0.1:4200/`: `200`.
- `GET http://127.0.0.1:4200/api/v1/config/public` (proxy de Angular): `200`.
- `GET http://127.0.0.1:4200/login`: `200`.
- `GET http://127.0.0.1:3000/api/v1/users/me` sin sesión: `401`.
- `./scripts/dev.sh status`: PostgreSQL (`solv_db`), backend y frontend activos.
- `./scripts/dev.sh start`: idempotente, termina con código 0 y no duplica
  servicios.
- CORS por el proxy desde `http://192.168.1.50:4200` y
  `http://100.117.207.81:4200`: `200`; el preflight `OPTIONS` remoto también.
- `solv_db` publica `127.0.0.1:5432`; `solv_postgres` quedó detenido como
  respaldo y su volumen no se borró.
- Datos migrados: 46 tablas, `users=1`, `tenants=1`, `subjects=1`,
  `exercises=2`, goose en la versión 53.
- Actores `@e2e.solv.test` en `solv_db`: `0`.

El host donde se ejecutaron estas comprobaciones reporta kernel Linux 6.8, no
Fedora. Por eso los comandos de ejecución y los servicios están verificados,
pero la instalación de paquetes con `dnf` no se ejecutó en Fedora. Las rutas,
puertos y variables son las mismas.

---

## 9. Migración local de PostgreSQL (2026-11-10)

En este host se unificó el control plane en el contenedor `solv_db` del
compose. Antes, el backend usaba `solv_postgres`, un contenedor creado fuera
del compose.

- Se generaron dumps de `solv_db` y `solv_test` en
  `backup/pre-migracion-20261110-191136/`.
- Los conteos antes y después coinciden: 46 tablas, un usuario, un tenant, una
  materia y dos ejercicios.
- `solv_postgres` quedó detenido, no eliminado. Su volumen
  `solv_postgres_data` sigue disponible para volver atrás.
- `solv_db` usa el volumen `solv_pg_pg_data` y es el que atiende `5432`.

Para volver atrás, detener `solv_db`, arrancar `solv_postgres` y levantar el
backend con `DATABASE_URL` apuntando al mismo puerto. No borrar ninguno de los
dos volúmenes hasta validar la migración.

La última falla de `dev.sh` fue un `connection refused` contra
`127.0.0.1:5432`: `solv_db` había quedado creado sin publicar el puerto. Se
recreó con Compose, se restauraron los datos y ahora `dev.sh start` arranca
sin error.

---

## 10. Acceso desde tu Fedora

El frontend escucha en `0.0.0.0:4200` y el backend en `*:3000`. En este host
`firewalld` está inactivo y Tailscale muestra el nodo `fedora` activo.

Abrir en el navegador de Fedora, según la red que uses:

| Red | URL |
| :--- | :--- |
| Tailscale | `http://100.117.207.81:4200` |
| LAN | `http://192.168.1.50:4200` |
| MagicDNS | `http://asus-server:4200` o `http://asus-server.tail14ebbe.ts.net:4200` |

No uses `http://localhost:4200` en Fedora: ese `localhost` es tu Fedora, no el
servidor. `127.0.0.1` tiene el mismo problema.

Los orígenes anteriores están en `ALLOWED_ORIGINS`. Si la IP del servidor
cambia, agregar el nuevo `http://IP:4200` y reiniciar el backend:

```bash
cd /home/alrivera/SOLV_PG
# Editar ALLOWED_ORIGINS en .env y agregar el origen nuevo.
systemctl --user restart solv-backend
```

### Login desde Fedora

El login con Google está configurado con
`GOOGLE_REDIRECT_URL=http://localhost:3000/auth/google/callback`. Si abrís el
sistema por IP, el botón de Google redirige al `localhost` de tu Fedora y no
vuelve al servidor. Tenés dos opciones:

1. **Túnel SSH** desde Fedora, que conserva `localhost`:

   ```bash
   ssh -L 4200:localhost:4200 -L 3000:localhost:3000 <usuario>@100.117.207.81
   ```

   Después abrir `http://localhost:4200` en Fedora.

2. **Cambiar el redirect** a `http://100.117.207.81:3000/auth/google/callback`,
   autorizar ese URI exacto en Google Cloud, dejar `COOKIE_DOMAIN=` vacío y
   reiniciar el backend. `COOKIE_DOMAIN` ya está vacío en este host.

Para una prueba rápida sin Google, usar `scripts/dev-session.mjs` y agregar la
cookie `solv_session` para el host que estés usando (`100.117.207.81` o
`192.168.1.50`), no para `localhost`.

> Estas comprobaciones se hicieron desde el host y por Tailscale. No se abrió
> un navegador en la Fedora; si la URL no carga, revisar primero que Tailscale
> esté conectado y que el navegador no esté usando un proxy que bloquee la IP.
