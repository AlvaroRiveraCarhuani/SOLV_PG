# Manual Operativo — Slice 11: Script de Backups y Trazabilidad

## 1. Operación del Script de Respaldos (`infra/scripts/backup.sh`)

### Permisos y Ejecución Manual
Para ejecutar el script de respaldo manualmente:

```bash
chmod +x infra/scripts/backup.sh
./infra/scripts/backup.sh
```

El script genera automáticamente:
- Dump de base de datos comprimido: `./backups/YYYY-MM/solv_db_YYYYMMDD_HHMMSS.sql.gz`
- Empaquetado de volúmenes de estudiantes: `./backups/YYYY-MM/solv_workspaces_vols_YYYYMMDD_HHMMSS.tar.gz`
- Registro de actividad en `/var/log/solv-backup.log` (o `./backups/solv-backup.log`).

---

## 2. Configuración en Cron (Tarea Programada)

Para ejecutar el respaldo automáticamente todas las noches a las 02:00 AM:

1. Abrir la edición de crontab con permisos de superusuario:
   ```bash
   sudo crontab -e
   ```
2. Agregar la siguiente línea:
   ```cron
   0 2 * * * /home/alvarorivera/Documentos/Desarrollo/SOLV_PG/infra/scripts/backup.sh > /dev/null 2>&1
   ```

---

## 3. Inspección de Logs de Auditoría (`audit_logs`)

Para consultar las acciones administrativas y docentes registradas en la base de datos:

```sql
SELECT id, tenant_id, actor_id, action, resource_type, status_code, ip_address, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT 20;
```
