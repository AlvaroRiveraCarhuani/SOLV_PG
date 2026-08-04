# Diseño Técnico - SLICE 7: OpenVSCode Server y SemgrepWorker

## 1. Arquitectura de Enrutamiento e Instanciación OpenVSCode Server

```
                          ┌───────────────────────────┐
                          │   Traefik v3 Proxy Ingress│ (Puerto 443 HTTPS)
                          └─────────────┬─────────────┘
                                        │
                                        │ Router: Host(`<uuid>.solv.local`)
                                        ▼
                          ┌───────────────────────────┐
                          │   OpenVSCode Container    │ (Puerto 3000)
                          │ gitpod/openvscode-server  │ Flag: --without-connection-token
                          └─────────────┬─────────────┘
                                        │
                                        ▼
                          ┌───────────────────────────┐
                          │  Volumen Nombrado Docker  │ Mapeo: /home/workspace:rw
                          │ solv_workspace_<stu>_<sub│
                          └───────────────────────────┘
```

---

## 2. Diagrama de Secuencia del Motor de Auditoría Semántica (`SemgrepWorker`)

```
 ┌──────────┐         ┌───────────────┐        ┌─────────────┐        ┌────────────┐
 │ Backend  │         │ SemgrepWorker │        │ Docker Engine│        │ PostgreSQL │
 └────┬─────┘         └───────┬───────┘        └──────┬──────┘        └─────┬──────┘
      │                       │                       │                     │
      │ AuditWorkspace(ws,vol)│                       │                     │
      ├──────────────────────>│                       │                     │
      │                       │ ContainerCreate (:ro) │                     │
      │                       ├──────────────────────>│                     │
      │                       │                       │                     │
      │                       │ ContainerStart        │                     │
      │                       ├──────────────────────>│                     │
      │                       │ semgrep scan --json   │                     │
      │                       │                       │                     │
      │                       │ ContainerLogs (stdout)│                     │
      │                       │<──────────────────────┤                     │
      │                       │                       │                     │
      │                       │ ContainerRemove (force)                     │
      │                       ├──────────────────────>│                     │
      │                       │                       │                     │
      │                       │ SaveSemgrepAudit (JSONB)                    │
      │                       ├────────────────────────────────────────────>│
      │                       │                       │                     │
```

---

## 3. Esquema de Migración en PostgreSQL (`semgrep_audit`)

```sql
ALTER TABLE workspaces 
ADD COLUMN IF NOT EXISTS semgrep_audit JSONB DEFAULT '{}'::jsonb;

ALTER TABLE lab_instances 
ADD COLUMN IF NOT EXISTS semgrep_audit JSONB DEFAULT '{}'::jsonb;
```

### Ejemplo del Objeto Guardado en `semgrep_audit`:
```json
{
  "errors": [],
  "results": [
    {
      "check_id": "rules.python.security.forbidden-import",
      "path": "/src/solution.py",
      "start": {"line": 1, "col": 1},
      "extra": {
        "message": "Uso de módulo os detectado en el análisis AST",
        "severity": "WARNING"
      }
    }
  ],
  "version": "1.80.0"
}
```
