# Incident Recovery Runbook

This document details standard operational procedures for incident response, database disaster recovery, failed migration rollbacks, and security containment within the SOLV platform environment.

---

## 1. Recovery Scenarios

### Scenario 1: Database Corruption or Data Loss
1. **Stop API Service**:
   ```bash
   sudo systemctl stop solv-api
   ```
2. **Identify Latest Valid Backup**:
   ```bash
   ls -lt /var/backups/solv/solv_db_*.dump | head -n 1
   ```
3. **Execute Automated Integrity-Verified Restore**:
   ```bash
   ./scripts/restore.sh /var/backups/solv/solv_db_YYYYMMDD_HHMMSS.dump
   ```
4. **Verify Database Integrity**:
   ```bash
   psql -U solv_user -d solv_db -c "SELECT COUNT(*) FROM exercises;"
   ```
5. **Restart API Service**:
   ```bash
   sudo systemctl start solv-api
   ```

---

### Scenario 2: Failed Migration Deployment
1. **Identify Problematic Migration**: Inspect migration files under `backend/migrations/`.
2. **Execute Automatic Migration Rollback**:
   ```bash
   cd backend && go run ./cmd/migrate -dir ./migrations -action down
   ```
3. **Verify Reversion Integrity**: Run the migration rollback verification tool:
   ```bash
   ./scripts/test-migration-rollback.sh
   ```
4. **Fallback Procedure**: If automatic rollback fails due to non-reversible schema corruption, restore from the latest clean backup following **Scenario 1**.

---

### Scenario 3: Security Breach or Intrusion Incident
1. **Isolate Server Endpoint**:
   ```bash
   sudo iptables -A INPUT -p tcp --dport 3000 -j DROP
   ```
2. **Capture Forensic Snapshot**:
   ```bash
   sudo tar -czf /tmp/forensic_snapshot_$(date +%s).tar.gz /var/log/solv /var/lib/solv
   ```
3. **Restore Clean Database State**: Perform disaster recovery from a pre-incident clean backup following **Scenario 1**.
4. **Credential & Token Rotation**:
   - Rotate JWT signing secret (`JWT_SECRET` in `.env`).
   - Change PostgreSQL password (`solv_password`).
   - Rotate deSec DNS / ACME credentials.
5. **Notify Incident Response**: Escalate to the security team with the forensic snapshot log location.

---

### Scenario 4: Image Pull Latency or Docker Hub Rate Limit (HTTP 429)
1. **Symptoms**:
   - Workspace creation times out (>30s) or student requests fail during morning peak.
   - Docker daemon logs report `toomanyrequests: You have reached your unauthenticated pull rate limit`.
2. **Immediate Remediation (Host Cache Pre-warming)**:
   - Trigger mass warming from Admin UI (*Calentar Caché* button) or via terminal on the Asus host:
   ```bash
   curl -X POST "http://localhost:3000/api/v1/admin/templates/warm-all" \
        -H "X-User-Id: <admin_uuid>" \
        -H "X-User-Role: admin"
   ```
3. **Inspect Cache Status**:
   ```bash
   curl -s -X GET "http://localhost:3000/api/v1/admin/templates/cache-status" \
        -H "X-User-Id: <admin_uuid>" \
        -H "X-User-Role: admin" | jq .
   ```
4. **Verify Local Storage Footprint**:
   ```bash
   docker system df
   ```
   Ensure SSD usage for Docker images does not exceed 80% of `/mnt/ssd`.


---

## 2. Post-Restore Validation Checklist

- [ ] Backend API health check responds `200 OK` on `http://localhost:3000/metrics` or `/api/v1/config/public`.
- [ ] Database contains valid exercise records (`SELECT COUNT(*) FROM exercises` > 0).
- [ ] Endpoint `/api/v1/exercises` returns valid JSON data payload.
- [ ] Application logs confirm zero database connection failures (`journalctl -u solv-api -n 50`).
- [ ] Prometheus metrics collection (`solv_http_requests_total`) is active and reporting.

---

## 3. Escalation Path

| Level | Role | Scope / Trigger | Action |
| :--- | :--- | :--- | :--- |
| **Level 1** | DevOps On-Call | Service outage, DB corruption, routine backup restore | Execute `restore.sh` or `backup.sh` procedures |
| **Level 2** | System Architect | Failed migration rollback, corrupt backup, schema mismatch | Debug migration scripts, perform manual DB repair |
| **Level 3** | Security Lead / CTO | Critical data breach, ransomware, unrecoverable data loss | Authorize emergency failover, credential revocation, public disclosure |

---

## 4. Automated Backup Schedule (Cron Configuration)

Daily backups are automated via system cron. The crontab configuration resides at `/etc/cron.d/solv-backup`:

```bash
# Executed daily at 03:00 AM local system time
0 3 * * * root /opt/solv/scripts/backup.sh >> /var/log/solv/backup.log 2>&1
```

---

## 5. Rotación de JWT Secret

En caso de sospecha de compromiso del secreto de firma JWT o mantenimiento de seguridad preventivo, ejecute el siguiente procedimiento de rotación:

### Procedimiento de Rotación
1. **Generar nuevo secreto aleatorio (32+ bytes base64/hex)**:
   ```bash
   NEW_JWT_SECRET=$(openssl rand -hex 32)
   echo "Nuevo secreto generado: ${NEW_JWT_SECRET}"
   ```

2. **Ventana de Gracia (Grace Period de 1 Hora)**:
   - Mantener temporalmente `JWT_OLD_SECRET=<secreto_anterior>` en el `.env` del servidor durante 60 minutos para permitir que las sesiones activas expiren o se renueven suavemente sin cerrar sesión abruptamente a los usuarios en uso activo.

3. **Actualización de Configuración**:
   ```bash
   # Actualizar .env en el servidor
   sed -i "s/^JWT_SECRET=.*/JWT_SECRET=${NEW_JWT_SECRET}/" /etc/solv/.env
   ```

4. **Reinicio del Servicio API**:
   ```bash
   sudo systemctl restart solv-api
   ```

5. **Finalización del Grace Period**:
   - Transcurrida 1 hora, eliminar `JWT_OLD_SECRET` del archivo `.env` y reiniciar el servicio nuevamente.

