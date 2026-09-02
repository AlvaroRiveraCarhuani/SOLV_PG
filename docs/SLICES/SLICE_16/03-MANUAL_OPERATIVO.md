# Manual Operativo — SLICE 16: Backups y Retención Institucional

## 1. Endpoints de Gestión de Respaldos

| Método | Ruta | Descripción | Rol Requerido |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/admin/backups` | Listar historial de respaldos ejecutados | `admin` |
| `POST` | `/api/v1/admin/backups/trigger` | Disparar respaldo manual inmediato | `admin` |
| `GET` | `/api/v1/admin/backups/config` | Consultar política de retención activa | `admin` |
| `PUT` | `/api/v1/admin/backups/config` | Modificar días de retención y destino | `admin` |

## 2. Variables de Configuración en el Host
```env
BACKUP_LOCAL_DIR=/var/solv/backups
BACKUP_RETENTION_DAYS=7
BACKUP_S3_ENABLED=false
BACKUP_S3_BUCKET=solv-institutional-backups
BACKUP_S3_ENDPOINT=https://s3.us-east-005.backblazeb2.com
BACKUP_S3_ACCESS_KEY=
BACKUP_S3_SECRET_KEY=
```
