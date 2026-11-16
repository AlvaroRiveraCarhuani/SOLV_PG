# Checklist Pre-Piloto de Seguridad y Operaciones

> **Estado global al 2026-11-16:** `NO CERTIFICADO`
> **Motivo:** los controles de aplicación están implementados en el repositorio, pero faltan las verificaciones de operación, los accesos institucionales y las pruebas de recuperación que este documento exige.
> **Regla de uso:** el piloto no inicia hasta que todos los ítems marcados como `BLOQUEANTE` estén en `CERRADO`. Un control presente en el código no se considera certificado en el servidor.

---

## 1. Cómo leer este checklist

| Estado | Significado |
| :--- | :--- |
| `Verificado en repositorio` | El control existe en el código o en la configuración versionada y se revisó su implementación. No sustituye la prueba en el servidor del piloto. |
| `Parcial` | Existe el artefacto, pero falta la evidencia de ejecución, activación o aceptación por el responsable. |
| `Pendiente` | No hay evidencia de que el control esté operativo. |
| `No verificable desde el repositorio` | Depende de GitHub, del servidor, del proveedor o de una decisión institucional. |
| `CERRADO` | El control fue ejecutado, verificado y tiene evidencia adjunta con fecha y responsable. |
| `BLOQUEANTE` | Su ausencia impide iniciar el piloto. |

Cada ítem debe registrar: **responsable**, **fecha**, **evidencia** y **resultado**. Una casilla marcada sin evidencia no cierra el ítem.

---

## 2. Tabla de control

| ID | Control | Cómo se verifica | Evidencia requerida | Responsable | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **C-01** | Protección de la rama `main` | Revisar en GitHub que `main` exige PR, al menos una aprobación humana y pipeline verde antes de fusionar. | Captura de la regla de protección o enlace a la configuración del repositorio. | Administrador del repositorio | `No verificable desde el repositorio` · `BLOQUEANTE` |
| **C-02** | Último pipeline de `main` en verde | Abrir la ejecución más reciente del workflow de CI/CD sobre `main` y confirmar que terminó correctamente. | Enlace a la ejecución, fecha, commit y conclusión. | Responsable de CI/CD | `No verificable desde el repositorio` · `BLOQUEANTE` |
| **C-03** | Respaldo automático activo y primer dump | Ejecutar `systemctl list-timers solv-backup.timer`, confirmar `solv-backup.service` y verificar un archivo en `backups/AAAA-MM/` con tamaño distinto de cero. | Salida de `systemctl`, ruta del dump, tamaño y suma SHA-256. | Operaciones | `Parcial` · `BLOQUEANTE` |
| **C-04** | Prueba de restauración | Ejecutar `scripts/restore.sh` sobre una base temporal, contar las tablas principales y registrar el resultado. No usar `--promote` en la prueba. | Salida del script, base temporal, número de tablas y fecha. | Operaciones | `Pendiente` · `BLOQUEANTE` |
| **C-05** | Métricas y alertas operativas | Confirmar que Prometheus consulta `/metrics` y que las reglas de `docs/OBSERVABILITY/alerts.md` están cargadas y notifican. | Consulta de `up`, reglas cargadas y una notificación de prueba. | Operaciones | `Parcial` · `BLOQUEANTE` |
| **C-06** | Rate limiting en endpoints clave | Ejecutar seis intentos rápidos contra `/api/v1/auth/login` y confirmar HTTP `429`; repetir contra envíos y eventos de teclado. | Respuestas HTTP con código y marca temporal. | Seguridad | `Parcial` · `BLOQUEANTE` |
| **C-07** | Cabeceras de seguridad | Solicitar una página y una respuesta de API; confirmar `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy` y `Referrer-Policy`. | Salida de `curl -I` o captura de red. | Seguridad | `Verificado en repositorio` |
| **C-08** | CORS restringido al dominio del piloto | Confirmar que `CORS_ALLOWED_ORIGINS` contiene solo el origen del piloto y que un origen ajeno recibe `403`. | Variables del entorno y respuesta de un origen no autorizado. | Operaciones | `Parcial` · `BLOQUEANTE` |
| **C-09** | Expiración de tokens | Confirmar que la sesión emitida no supera las 24 horas. El código aplica un límite duro de 24 horas incluso si `JWT_TTL_HOURS` es mayor. | Token decodificado con `exp` y `iat`, o prueba de expiración. | Seguridad | `Verificado en repositorio` |
| **C-10** | Informe de capacidad y recursos del piloto | Revisar `docs/CAPACITY/load-test-report.md` y comparar CPU, RAM, swap y runners con el servidor asignado al piloto. | Informe firmado, ficha del servidor y diferencias aceptadas. | Operaciones | `Parcial` · `BLOQUEANTE` |
| **C-11** | Runbook conocido por el equipo on-call | Realizar una lectura guiada de `docs/OPERATIONS/incident-runbook.md`, incluida la rotación de `JWT_SECRET`, y registrar asistencia. | Acta de lectura, fecha y nombres del equipo on-call. | Operaciones | `Parcial` · `BLOQUEANTE` |
| **C-12** | Cuenta de administración con acceso fuerte | Verificar cuenta de producción, SSO/2FA activo, credenciales no compartidas y acceso de emergencia documentado. | Captura del método de acceso y registro del responsable. | Seguridad | `Pendiente` · `BLOQUEANTE` |
| **C-13** | Base legal y consentimiento institucional | Confirmar autorización para cargar nombres, correos y matrículas de estudiantes reales, con finalidad, retención y responsable. | Documento firmado por la institución. | Dirección académica | `Pendiente` · `BLOQUEANTE` |

---

## 3. Evidencia de repositorio revisada

Revisión de código y documentación al **2026-11-16**. Esta tabla no certifica el servidor: deja trazabilidad de dónde está implementado cada control.

| Control | Referencia | Observación |
| :--- | :--- | :--- |
| Cabeceras de seguridad | `backend/cmd/api/main.go:575-579`; `backend/internal/delivery/http/middleware/security_headers.go` | Middleware aplicado globalmente; incluye HSTS cuando la petición es TLS. |
| CORS | `backend/internal/delivery/http/middleware/cors.go` | Allowlist exacta; sin variable configurada solo admite orígenes locales. No existe comodín. |
| Rate limiting | `backend/internal/delivery/http/router.go:185-189`, `:468`, `:560-568`; `backend/internal/delivery/http/middleware/rate_limiter.go` | Login, envíos, eventos de teclado y arranque de entornos tienen middleware dedicado. |
| Métricas | `backend/internal/delivery/http/router.go:579`; `backend/internal/delivery/http/metrics_access.go` | `/metrics` exige `METRICS_TOKEN` o acceso desde loopback. |
| Alertas | `docs/OBSERVABILITY/alerts.md` | El documento define cinco reglas: `HighSubmissionFailureRate`, `DockerContainerSaturation`, `EvaluatorQueueBacklog`, `HighHTTPLatency` y `ReconciledEnvironmentStates`. El checklist anterior decía cuatro; se corrige el conteo, no se elimina ninguna. |
| Expiración de tokens | `backend/internal/core/services/auth_service.go:321-327` | Límite duro de 24 horas; un valor mayor en `JWT_TTL_HOURS` se recorta a 24. |
| Respaldo | `infra/systemd/solv-backup.service`; `infra/systemd/solv-backup.timer`; `infra/scripts/backup.sh`; `scripts/install.sh:127-152` | Timer diario a las 03:00 con persistencia; el instalador habilita el timer. Falta confirmar la ejecución real y el primer dump. |
| Restauración | `scripts/restore.sh` | Restaura por defecto en una base temporal y exige `--promote --yes` para reemplazar la base viva. No se ejecutó en esta revisión. |
| Capacidad | `docs/CAPACITY/load-test-report.md` | Informe del 2026-10-07 sobre un equipo de 4 hilos y 8 GB; no certifica el servidor asignado al piloto. |
| Runbook | `docs/OPERATIONS/incident-runbook.md` | Cubre caída del sitio, backend, PostgreSQL, migraciones, disco, RAM, entornos, certificado, seguridad y pérdida de trabajo. Incluye rotación de `JWT_SECRET` y límite conocido de revocación por usuario. |

---

## 4. Bloqueos para iniciar el piloto

No se inicia el piloto mientras exista cualquiera de estos bloqueos:

1. `main` sin protección de rama o sin pipeline verde verificable.
2. Respaldo sin timer activo o sin un dump reciente comprobado.
3. Restauración no probada en una base temporal.
4. `/metrics` sin scrape confirmado o alertas sin notificación de prueba.
5. Rate limiting sin prueba de `429` en login, envíos y eventos de teclado.
6. `CORS_ALLOWED_ORIGINS` sin el origen real del piloto o con un origen adicional no autorizado.
7. Capacidad del servidor sin comparación firmada contra el informe.
8. Equipo on-call sin lectura del runbook.
9. Cuenta de administración sin SSO/2FA o con credenciales compartidas.
10. Consentimiento institucional ausente para datos de estudiantes reales.

---

## 5. Cierre y certificación

| Campo | Registro |
| :--- | :--- |
| **Estado global** | `NO CERTIFICADO` |
| **Fecha de revisión** | 2026-11-16 |
| **Revisado por** | Equipo técnico del piloto |
| **Pendientes bloqueantes** | C-01, C-02, C-03, C-04, C-05, C-06, C-08, C-10, C-11, C-12, C-13 |
| **Controles verificados en repositorio** | C-07, C-09 |
| **Próxima revisión** | Se fija al cerrar el primer bloqueo operativo |

**Fecha de certificación:** ____________
**Responsable del piloto:** ____________
**Firma de aprobación:** ____________

La certificación solo es válida cuando todos los ítems `BLOQUEANTE` están en `CERRADO` y la evidencia adjunta permite reproducir la verificación sin depender de la memoria de quien la ejecutó.
