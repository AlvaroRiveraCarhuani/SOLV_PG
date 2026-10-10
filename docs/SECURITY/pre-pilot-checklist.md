# Checklist Pre-Piloto de Seguridad y Operaciones

## Objetivo
Certificar en ≤15 minutos que la plataforma SOLV cumple con todos los estándares de seguridad aplicativa, infraestructura resiliente y operabilidad antes de iniciar el piloto con usuarios reales.

---

## 1. Integridad de Código y Pipeline (CI/CD)
- [ ] **Branch Protection**: Protección de la rama `main` activa en el repositorio (requiere pipeline en verde + 1 aprobación humana).
- [ ] **Pipeline en Verde**: El último workflow del pipeline CI/CD en `main` terminó con estado exitoso.

## 2. Resiliencia y Backup (DRP)
- [ ] **Backup Automático**: Tarea cron diaria instalada en el servidor de piloto (`/etc/cron.d/solv-backup`) y primer dump verificado en disco/S3.
- [ ] **Prueba de Restore**: Restauración de prueba ejecutada con éxito en el entorno de staging (fecha: ____________).

## 3. Observabilidad y Alertas
- [ ] **Métricas Prometheus**: Endpoint `/metrics` activo y monitoreado con las 4 alertas críticas configuradas (`docs/OBSERVABILITY/alerts.md`).

## 4. Hardening y Seguridad Aplicativa
- [ ] **Rate Limiting**: Rate limiting activo en endpoints clave (`/api/v1/auth/login`, `/api/v1/submissions`, `/api/v1/submissions/{id}/keystroke-events`). Probado: 6 peticiones de login rápidas retornan HTTP 429 en la sexta.
- [ ] **Headers de Seguridad**: Headers de protección presentes en todas las respuestas HTTP (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy`, `Referrer-Policy`).
- [ ] **CORS Allowlist**: Configuración `CORS_ALLOWED_ORIGINS` explícita restringida al dominio del piloto (sin comodín `*`).
- [ ] **Expiración de Tokens**: Variable `JWT_TTL_HOURS` configurada con un valor ≤ 24 en el `.env` de producción.

## 5. Dimensionamiento y Capacidad
- [ ] **Informe de Capacidad**: Reporte `docs/CAPACITY/load-test-report.md` revisado y recursos asignados al servidor de piloto de acuerdo al dimensionamiento esperado (RAM, CPU, swap).

## 6. Procedimientos de Incidentes
- [ ] **Runbook de Incidentes**: Documento `docs/OPERATIONS/incident-runbook.md` conocido por el equipo on-call del piloto, incluyendo el procedimiento de rotación de `JWT_SECRET`.

## 7. Gobernanza y Accesos
- [ ] **Cuentas de Administración**: Cuenta de administrador de producción configurada con credenciales fuertes e integración SSO/2FA activa.
- [ ] **Cumplimiento y Consentimiento**: Confirmación de la base legal y consentimiento institucional para la carga de datos de estudiantes y matrículas reales.

---

**Fecha de Certificación**: ____________  
**Responsable del Piloto**: ____________  
**Firma de Aprobación**: ____________  
