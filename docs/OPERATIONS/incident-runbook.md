# Runbook de incidentes — SOLV

> **Alcance:** instalación on-premise de un solo nodo, con la pila de `compose.yml`.
> **Para quién:** quien esté frente al servidor. No asume que seas el autor del sistema.
> **Regla general:** primero se mira, después se toca. Cada comando de este documento existe en el repositorio y se puede ejecutar tal cual.

---

## 0. Reglas que no se rompen

1. **No borres volúmenes.** `docker volume rm`, `docker system prune --volumes` y `docker compose down -v` destruyen el trabajo de los alumnos. No hay respaldo que lo devuelva al instante, solo la restauración completa.
2. **No uses `goose down` en producción.** La vuelta atrás soportada es *imagen anterior + respaldo previo*. Un `down` sobre una base con datos transformados pierde datos con pasos extra.
3. **No reinicies para probar.** Un reinicio borra el estado que explica la falla. Leé `docker compose logs` primero.
4. **Todo respaldo se prueba.** Un archivo que nunca se restauró no es un respaldo: `scripts/restore.sh` sin `--promote` restaura en una base temporal y lo verifica.
5. **Las acciones de emergencia se auditan.** Exigen frase exacta y motivo de al menos 10 caracteres; quedan en `audit_logs` como `EMERGENCY_*`.

---

## 1. Mapa del sistema

Ejecutá los comandos desde el directorio del proyecto (`cd /opt/solv` o donde esté clonado).

| Componente | Nombre | Puerto local | Notas |
|---|---|---|---|
| Proxy y TLS | `solv_traefik` | 80, 443 | Enruta por etiquetas de Docker. Certificado wildcard de desec.io. |
| API | `solv_backend` | 3000 | Go. Aplica migraciones al arrancar. Monta el socket de Docker. |
| Interfaz | `solv_frontend` | — | Nginx sirviendo Angular. Solo alcanzable por Traefik. |
| Base de datos | `solv_db` | 5432 | PostgreSQL 18. Es la fuente de verdad. |
| Archivos de avance | `solv_seaweedfs` | 8333 | S3 para snapshots. |
| Métricas | `solv_prometheus`, `solv_grafana` | 9090, 3001 | Solo en loopback. |

| Recurso | Ruta |
|---|---|
| Variables y secretos | `.env` |
| Respaldos | `backups/AAAA-MM/solv_db_*.sql.gz`, `solv_sat_postgres_*.sql.gz`, `solv_workspaces_vols_*.tar.gz` |
| Log del respaldo | `/var/log/solv-backup.log` |
| Certificado | `letsencrypt/acme.json` (permisos 600, o Traefik no arranca) |
| Unidades del host | `solv-firewall.service`, `solv-firewall.timer`, `solv-backup.service`, `solv-backup.timer` |

---

## 2. Primeros 60 segundos

```bash
docker compose ps
curl -fsS http://127.0.0.1:3000/api/v1/version
docker compose logs --tail=100 backend
df -h / ; free -m
```

Cómo leer el resultado:

| Resultado | Qué significa |
|---|---|
| `ps` muestra `backend` en `restarting` o `exited` | El proceso no arranca. Ver §4.2. |
| `/api/v1/version` responde y `database` es `ok` | La API y la base están vivas: el problema está en el proxy o en el frontend. |
| `/api/v1/version` responde con `database: "unavailable"` | La API vive, PostgreSQL no. Ver §4.3. |
| `/api/v1/version` no responde | El backend está caído o el puerto está ocupado. Ver §4.2. |
| `df -h` por encima del 85 % | Disco. Ver §4.5. |
| `free -m` con `available` por debajo de 500 MB | RAM. Ver §4.6. |

El endpoint de versión es la referencia: dice qué tag corre, qué commit, y hasta qué migración llegó la base. Anotá esos tres datos antes de cualquier cambio.

---

## 3. Diagnóstico por síntoma

| Síntoma | Primer comando | Causa más probable |
|---|---|---|
| El sitio no abre, error 502 | `docker compose ps` | Backend caído o Traefik sin routers (§4.1) |
| El sitio abre pero las pantallas quedan vacías | `docker compose logs --tail=50 backend` | API respondiendo 401/403 o error de base |
| Los alumnos no pueden abrir su entorno | `docker compose logs --tail=100 backend \| grep -i admission` | Techo de contenedores o RAM del host (§4.6) |
| Todo va lento y el disco está al 100 % | `docker system df` | Imágenes y capas acumuladas (§4.5) |
| El certificado venció | `docker compose logs --tail=100 traefik \| grep -i acme` | Token de desec o `acme.json` con permisos malos (§4.8) |
| Un alumno perdió su trabajo | `docker volume ls -q \| grep solv_vol_` | Volumen desmontado o contenedor eliminado (§4.10) |
| Recién actualizaste y no arranca | `docker compose logs --tail=80 backend` | Migración fallida (§4.4) |

---

## 4. Escenarios

### 4.1 El sitio no abre (502/504)

**Diagnóstico**

```bash
docker compose ps
curl -fsS http://127.0.0.1:3000/api/v1/version
docker compose logs --tail=50 traefik
```

**Acción**

- Si el backend responde en `127.0.0.1:3000` pero el sitio no: Traefik no ve los routers.
  ```bash
  docker inspect solv_backend --format '{{json .Config.Labels}}' | head -c 800
  ```
  Si faltan las etiquetas `traefik.http.routers.solv-api.*`, el contenedor se levantó sin el compose correcto: `docker compose up -d`.
- Si el backend no responde: ir a §4.2.

**Verificación:** `curl -I https://<tu-host>` devuelve 200 y el navegador carga el login.

---

### 4.2 El backend no responde o reinicia solo

**Diagnóstico**

```bash
docker compose ps backend
docker compose logs --tail=200 backend
```

Buscá la primera línea de error, no la última. Las causas típicas:

| Log | Causa | Acción |
|---|---|---|
| `DATABASE_URL environment variable is required` | `.env` ausente o sin la clave | Revisar `.env`, recrear: `docker compose up -d backend` |
| `failed to connect to Postgres` | Base caída o credenciales cambiadas | §4.3 |
| `failed to run database migrations` | Migración rota | §4.4 |
| `Auth service requires GOOGLE_CLIENT_ID...` | Faltan credenciales de OAuth o `JWT_SECRET` | Completar `.env` y recrear |
| El proceso no deja rastro y el contenedor sale con 0 | El puerto 3000 está ocupado | `ss -ltnp \| grep 3000` |

**Acción**

```bash
docker compose up -d backend        # recrea con el .env actual
docker compose logs -f --tail=50 backend
```

**Verificación:** `/api/v1/version` responde 200 con `database: "ok"`.

**Si no alcanza:** reiniciar el servicio no cambia nada si el problema es el host. Revisá `dmesg -T | tail -50` por OOM del kernel.

---

### 4.3 PostgreSQL caído o datos corruptos

**Diagnóstico**

```bash
docker compose ps postgres
docker compose logs --tail=100 postgres
docker compose exec -T postgres psql -U solv_user -d solv_db -tAc "SELECT 1"
```

`pg_isready` también sirve:

```bash
docker compose exec -T postgres pg_isready -U solv_user -d solv_db
```

**Acción, base caída pero volumen sano**

```bash
docker compose restart postgres
```

**Acción, datos corruptos o pérdida**

1. Elegí el respaldo más reciente y verificá que exista:
   ```bash
   ls -lh backups/$(date +%Y-%m)/ | tail -5
   ```
2. Restaurá sin promover, para ver el contenido antes de tocar la base viva:
   ```bash
   bash scripts/restore.sh backups/2026-11/solv_db_20261113_030000.sql.gz
   ```
   El script verifica el gzip, restaura en `solv_restore_<fecha>` y cuenta `tenants`, `users`, `exercises` y `submissions`.
3. Si los números son razonables, promové:
   ```bash
   bash scripts/restore.sh backups/2026-11/solv_db_20261113_030000.sql.gz --promote --yes
   ```
   Detiene el backend, renombra la base actual a `solv_db_pre_restore_<fecha>`, pone la restaurada y vuelve a arrancar.
4. Confirmá con `/api/v1/version` y borrá la base anterior solo cuando la operación esté validada.

**Verificación:** el personal entra con sus cuentas y las materias aparecen.

**Si no alcanza:** un respaldo que no restaura se descarta y se prueba el anterior. Nunca restaures dos veces sobre la misma base sin haber confirmado el resultado de la primera.

---

### 4.4 Una actualización falló (migración)

**Diagnóstico**

```bash
docker compose logs --tail=100 backend | grep -i migration
curl -fsS http://127.0.0.1:3000/api/v1/version
```

El backend aplica las migraciones al arrancar y **sale** si una falla, así que el síntoma es un contenedor que no levanta después de cambiar `SOLV_VERSION`.

**Acción**

Si la actualización se hizo con `scripts/upgrade.sh`, la vuelta atrás ya ocurrió: el script vuelve a la imagen anterior, restaura el respaldo previo y termina con código 1 informando qué versión quedó. Revisá su salida antes de tocar nada.

Si la actualización se hizo a mano:

1. Volver a la imagen anterior:
   ```bash
   SOLV_VERSION=<versión-anterior> docker compose up -d backend
   ```
2. Si la base quedó a medio camino (el log muestra la migración aplicada pero el arranque sigue fallando), restaurar el respaldo previo a la actualización y quedarse en la versión anterior:
   ```bash
   SOLV_COMPOSE_CMD=true bash scripts/restore.sh <respaldo.sql.gz> --promote --yes
   SOLV_VERSION=<versión-anterior> docker compose up -d backend frontend
   ```

**Verificación:** `/api/v1/version` responde y `migration_version` coincide con lo esperado para ese tag.

**Por qué no `goose down`:** una migración que transformó o borró datos no se revierte con un `down`. Ver §0.2.

---

### 4.5 Disco lleno

**Diagnóstico**

```bash
df -h /
docker system df
du -sh backups/* | sort -h | tail -5
```

**Acción**

1. Liberar capas e imágenes huérfanas sin tocar volúmenes, desde el panel (Auditoría → Emergencias → *Purgar imágenes y volúmenes huérfanos*) o por API con la frase `PURGAR CAPAS HUERFANAS`.
2. Si el panel no está disponible:
   ```bash
   docker image prune -f          # solo imágenes sin usar
   ```
   **Nunca** `docker system prune --volumes`.
3. Revisar la retención de respaldos: `infra/scripts/backup.sh` elimina los mayores a 180 días, pero un respaldo manual no entra en esa política.

**Verificación:** `df -h /` por debajo del 80 % y `docker system df` sin capas colgadas.

---

### 4.6 RAM saturada o contenedores desbocados

**Diagnóstico**

```bash
free -m
docker stats --no-stream --format 'table {{.Name}}\t{{.MemUsage}}' | head -20
docker compose logs --tail=50 backend | grep -i "admission\|OOMKilled"
```

**Acción, inmediata y reversible**

Panel → Auditoría → Emergencias → *Hibernar todos los entornos* (frase `HIBERNAR TODOS LOS ENTORNOS`). Pausa sin destruir el estado: los alumnos reanudan al volver a entrar.

**Acción, si hay contenedores colgados**

*Limpiar zombies Docker* (`LIMPIAR ZOMBIES DOCKER`).

**Acción, si el host sigue sin responder**

Reducir el techo de concurrencia por institución: `max_containers` en `PUT /api/v1/admin/server/policies`. El valor por defecto se deriva de la RAM medida del servidor y el campo `recommendation` de la misma ruta dice cuánto sostiene hoy (ver `docs/OPERATIONS/capacidad.md`).

**Verificación:** `free -m` recupera memoria disponible y los alumnos pueden abrir entornos nuevos.

**Lo que no se hace:** `docker kill` a mano sobre contenedores de alumnos. Deja el registro inconsistente; el reconciliador lo resolverá en el siguiente ciclo, pero el `docker kill` manual no actualiza la base.

**Resolver una incidencia concreta**

Panel → Auditoría → Incidencias (o `POST /api/v1/admin/environments/{id}/resolve`) para un entorno en `failed` u `oom_killed`. La acción reinicia el contador de OOM, borra la marca del último OOM y deja el entorno en `hibernated`. Si el entorno estaba en otro estado, lo informa sin cambiarlo: la respuesta trae `environment_status` con lo que quedó en la base.

Alcance: sólo entornos de la propia institución. Un id ajeno responde `404`, igual que uno inexistente, para no confirmar que existe. Un fallo de base responde `500` en lugar de un mensaje de éxito.

**Verificación:** el entorno queda en `hibernated`, con `oom_strike_count = 0` y `last_oom_killed_at` vacío; un entorno que estaba `running` conserva ese estado y sólo pierde los strikes.

---

### 4.7 Los entornos no arrancan (Docker o imágenes)

**Diagnóstico**

```bash
docker compose ps
docker info --format '{{.ServerVersion}}'
docker compose logs --tail=80 backend | grep -i "pull\|image\|docker"
docker compose exec -T backend ls -l /var/run/docker.sock
```

| Síntoma en el log | Causa | Acción |
|---|---|---|
| `permission denied` sobre el socket | El contenedor no monta el socket o cambió el grupo | `docker compose up -d backend` |
| `toomanyrequests` | Límite de Docker Hub | Precalentar con `POST /api/v1/admin/templates/warm-all` (panel: Plantillas → preparar) |
| `no space left on device` | Disco | §4.5 |
| `context deadline exceeded` al crear | El daemon está saturado | §4.6 |

**Verificación:** abrir un entorno de práctica desde una cuenta de alumno.

---

### 4.8 El certificado no renueva

**Diagnóstico**

```bash
docker compose logs --tail=200 traefik | grep -i "acme\|desec\|certificate"
ls -l letsencrypt/acme.json
```

**Acción**

- Permisos distintos de `600`: `chmod 600 letsencrypt/acme.json && docker compose restart traefik`.
- Token de desec vencido o revocado: renovarlo en desec.io, actualizar `DESEC_TOKEN` en `.env` y `docker compose up -d traefik`.
- El certificado wildcard necesita que el registro DNS `_acme-challenge` sea resoluble. Verificá con `dig TXT _acme-challenge.<tu-host>`.

**Verificación:** `curl -vI https://<tu-host>` muestra una fecha de expiración nueva.

---

### 4.9 Incidente de seguridad

**Contención (primeros minutos)**

1. Rotar el secreto de sesión: todas las sesiones quedan invalidadas.
   ```bash
   # generar uno nuevo
   openssl rand -hex 32
   # reemplazar JWT_SECRET en .env y recrear el backend
   docker compose up -d backend
   ```
   **No hay periodo de gracia**: el backend no admite un secreto anterior, así que todo el mundo vuelve a iniciar sesión. Hacelo en una ventana sin examen en curso.
2. Revocar administradores sospechosos desde el panel (Configuración → Administradores) o con `POST /api/v1/admin/administrators/{id}/revoke`.
3. Si el servidor está comprometido, aislarlo de la red antes que nada y preservar evidencia:
   ```bash
   docker compose logs --no-color > /tmp/solv-logs-$(date +%s).txt
   journalctl -u solv-firewall --no-pager > /tmp/solv-firewall-$(date +%s).log
   ```
4. Rotar también `POSTGRES_PASSWORD` (requiere `ALTER USER` dentro del contenedor y recrear el backend), el secreto de OAuth de Google y `DESEC_TOKEN`.

**Después:** revisar `audit_logs` del periodo, y confirmar que las reglas `DOCKER-USER` siguen aplicadas (`iptables -S DOCKER-USER`).

**Límite conocido:** no existe un endpoint para revocar las sesiones de un usuario puntual. La palanca disponible es rotar `JWT_SECRET`.

---

### 4.10 Un alumno perdió su trabajo

**Diagnóstico**

```bash
docker volume ls -q | grep solv_vol_
docker compose logs --tail=100 backend | grep -i "<id-del-entorno>"
```

**Acción**

1. Pedirle que vuelva a entrar: si el contenedor fue eliminado y el volumen sigue, el entorno se recrea sobre el mismo volumen.
2. Si el volumen ya no existe, el último recurso es el tar del respaldo: `backups/AAAA-MM/solv_workspaces_vols_*.tar.gz`.
   Restaurá **en un volumen nuevo** y revisá el contenido antes de reasociarlo. Sobrescribir un volumen en uso destruye lo que quede.

**Verificación:** el alumno ve sus archivos al reabrir el entorno.

---

## 5. Consola de emergencia

Panel → **Auditoría → Emergencias**. Cada acción pide la frase exacta y un motivo; queda en `audit_logs`.

| Acción (`POST /api/v1/admin/emergency/<id>`) | Frase | Efecto | Cuándo | Cuándo no |
|---|---|---|---|---|
| `hibernate_all_environments` | `HIBERNAR TODOS LOS ENTORNOS` | Pausa sin destruir | RAM alta, clase terminada | Examen en curso sin aviso |
| `terminate_all_environments` | `TERMINAR TODOS LOS ENTORNOS` | Destruye contenedores | Entornos corruptos en masa | Como primera medida |
| `kill_zombies` | `LIMPIAR ZOMBIES DOCKER` | Borra contenedores huérfanos | Estados inconsistentes | Con laboratorios activos legítimos |
| `docker_prune` | `PURGAR CAPAS HUERFANAS` | Libera disco | §4.5 | Si no verificaste qué se borra |
| `reset_pools` | `REINICIAR POOLS` | Recicla conexiones a la base | Errores de pool o circuit breaker abierto | Como sustituto de reiniciar Postgres |

Por API, con una sesión de administrador:

```bash
curl -fsS -X POST http://127.0.0.1:3000/api/v1/admin/emergency/hibernate_all_environments \
  -H "Content-Type: application/json" \
  -H "Cookie: solv_session=$SOLV_SESSION" \
  -d '{"confirmation_phrase":"HIBERNAR TODOS LOS ENTORNOS","reason":"RAM al 94% durante examen"}'
```

Códigos esperados: `200` ejecutada, `422 invalid_confirmation_phrase` (frase mal escrita), `503 executor_unavailable` (la instancia no tiene ese ejecutor), `403` (no sos administrador).

---

## 6. Rotación de secretos

| Secreto | Dónde vive | Cómo se rota | Efecto |
|---|---|---|---|
| `JWT_SECRET` | `.env` | `openssl rand -hex 32`, editar `.env`, `docker compose up -d backend` | Todas las sesiones se invalidan |
| `POSTGRES_PASSWORD` | `.env` + base | `ALTER USER solv_user WITH PASSWORD '...'` dentro del contenedor, editar `.env`, `docker compose up -d backend postgres` | Corte breve de la API |
| Google OAuth | `.env` | Nuevo secreto en Google Cloud Console, editar `.env`, recrear backend | Los logins en curso fallan |
| `DESEC_TOKEN` | `.env` | Nuevo token en desec.io, editar `.env`, `docker compose up -d traefik` | La próxima renovación usa el token nuevo |

Después de cualquier rotación: `/api/v1/version` responde, un login funciona y un alumno abre un entorno.

---

## 7. Escalamiento: una sola persona

Esta instalación no tiene un equipo de guardia. El documento describe lo que hay, no lo que sería deseable.

| Nivel | Quién | Qué puede decidir solo | Qué no |
|---|---|---|---|
| **0 — Automático** | El sistema y el vigilante externo | Hibernar por inactividad, reconciliar estados, respaldar a las 03:00, aplicar reglas de firewall, y avisar por correo cuando el sitio deja de responder (`docs/OPERATIONS/EXTERNAL_UPTIME.md`) | Nada que requiera criterio |
| **1 — Administrador de la institución** | El rol `admin` del panel | Hibernar todos, purgar capas, reiniciar pools, poner el sitio en mantenimiento | Restaurar la base, rotar secretos, cambiar de versión |
| **2 — Mantenedor del sistema** | Autor del proyecto | Todo lo anterior, más restauración, actualización y rotación de secretos | Cambiar el hardware o el proveedor de red |
| **3 — Proveedor** | Hosting, red, energía, desec.io | Lo que esté fuera del servidor | Nada dentro de la aplicación |

**Reglas de la casa:**

- El nivel 1 no espera al nivel 2 para hibernar. Es reversible y evita que la clase se caiga.
- El nivel 2 no restaura una base sin avisar al nivel 1: la restauración descarta lo escrito después del respaldo.
- Si el nivel 2 no responde, el sistema sigue operando en degradado: se puede hibernar, poner en mantenimiento y esperar. **No se actualiza ni se restaura sin esa persona.**
- Canal y horario: definirlos una vez y escribirlos acá. Si no están escritos, no existen.

**Poner el sitio en mantenimiento** (mientras se resuelve): interruptor del panel de administración. Si el panel no carga:

```bash
docker compose exec -T postgres psql -U solv_user -d solv_db -c \
  "UPDATE tenants SET maintenance_mode = true, maintenance_reason = 'Incidente en curso' WHERE maintenance_mode = false;"
```

Para salir: el mismo `UPDATE` con `false`.

---

## 8. Cierre del incidente

- [ ] `/api/v1/version` responde con `database: "ok"`.
- [ ] Un login de cada rol funciona (admin, docente, alumno).
- [ ] Un alumno abre un entorno y ve sus archivos.
- [ ] `systemctl list-timers 'solv-*'` muestra los dos timers activos.
- [ ] `iptables -S DOCKER-USER` conserva los bloqueos.
- [ ] El respaldo del día siguiente se generó; si el incidente tocó la base, se probó una restauración.
- [ ] Se anotó qué se cambió, y si el cambio invalida algo de este documento, se actualizó el documento.

---

## 9. Límites de este documento

- **Un solo nodo.** Si el servidor físico muere, no hay conmutación: se recupera sobre el mismo hardware o uno nuevo siguiendo `docs/DEPLOYMENT.md`.
- **No cubre el aprovisionamiento inicial**, que está en `docs/DEPLOYMENT.md` §8.
- **No cubre el host fuera de Docker**: kernel, discos, red o energía son del proveedor o del área de sistemas de la institución.
- **Los procedimientos se probaron sobre la pila descrita**, no sobre cada variante de hardware. Si un comando no coincide con lo que ves, el documento está mal y hay que corregirlo en ese momento.
