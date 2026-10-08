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
