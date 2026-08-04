# Manual Operativo - SLICE 7: OpenVSCode Server y SemgrepWorker

## 1. Verificación de la Instanciación de OpenVSCode Server (Puerto 3000)

Para verificar que los entornos interactivos levantan la imagen oficial de OpenVSCode Server en el puerto 3000:

```bash
# Instanciar un entorno desde la API
curl -X POST http://localhost:3000/api/v1/workspaces/start \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"student_id": "stu-123", "subject_id": "sub-456"}'

# Inspeccionar etiquetas de Traefik v3 en Docker Engine
docker inspect solv-workspace-<UUID> --format '{{json .Config.Labels}}' | jq

# Salida esperada:
# "traefik.http.services.<UUID>.loadbalancer.server.port": "3000"
```

---

## 2. Ejecución Manual de la Auditoría AST Semántica (`SemgrepWorker`)

Para auditar el código fuente del estudiante de forma ad-hoc desde el backend:

```bash
# Invocar el worker de auditoría desde el código Go o test de integración
# El worker ejecutará efímeramente:
docker run --rm -v solv_workspace_stu-123_sub-456:/src:ro semgrep/semgrep:latest semgrep scan --json --config auto /src
```

---

## 3. Consulta de Resultados AST en PostgreSQL

Verificar el JSONB guardado en la tabla de workspaces o lab_instances:

```bash
psql -h 127.0.0.1 -U solv_user -d solv_db -c "SELECT id, status, semgrep_audit FROM workspaces WHERE semgrep_audit != '{}'::jsonb;"
```
