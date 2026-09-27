# Submódulo 14.7 — Auditoría y Emergencias (`/admin/auditoria`)

**Estado:** Implementado (vista completa con las 2 pestañas del wireframe oficial)
**Fecha de cierre:** 2026-09-27
**Mapa de ejecución:** `especificaciones/mapa-implementacion-auditoria-14.7.md`

---

## 1. ADRs del Submódulo

* [ADR-027: Operabilidad B2B y Audit Logs](../../ARQUITECTURA/ADR/ADR-027-operabilidad-b2b-audit-logs.md) *(worker pool asíncrono, tabla audit_logs, enriquecimiento semántico)*
* [ADR-032: Acciones de Emergencia del Administrador](../../ARQUITECTURA/ADR/ADR-032-acciones-emergencia-administrador.md) *(catálogo completo con doble confirmación y anexo de implementación real)*

## 2. Alcance Implementado

| Pestaña | Funcionalidad | Backend |
| :--- | :--- | :--- |
| **Registro de Auditoría** | Tabla cronológica con enriquecimiento semántico de verbos (POST Creación / PUT Actualización / DELETE Eliminación), status coloreado por combinación verbo + código, actores con email institucional resuelto, búsqueda y filtro por acción, paginación server-side y off-canvas drawer con la cronología aislada del actor | `GET /admin/audit-logs` con `actor_email` (JOIN a users), `GET /admin/audit-logs/actors/{actorId}/timeline` nuevo (tope 200) |
| **Emergencias** | Centro de control con las 5 acciones del catálogo ADR-032 clasificadas por impacto, franja de estado del host con banner crítico >= 90% RAM, modal de doble confirmación (frase exacta + motivo >= 10 caracteres) e historial de ejecuciones `EMERGENCY_*` | `POST /admin/emergency/{action}` completo: `docker_prune` y `reset_pools` nuevos, auditoría obligatoria con motivo e impacto, 503 si no hay executor |

## 3. Decisiones Técnicas Relevantes

1. **Identidad institucional resuelta en backend:** el listado y el timeline
   proyectan `actor_email` vía LEFT JOIN a `users` con fallback al propio
   `actor_id` para actores huérfanos; la UI nunca muestra UUIDs crudos.
2. **Auditoría obligatoria de emergencias:** toda ejecución aceptada persiste
   su evento `EMERGENCY_*` con motivo e impacto; un rechazo por frase
   inválida no registra evento. Un fallo de auditoría no revierte la acción
   (el efecto operativo ya ocurrió) pero se reporta en el log del sistema.
3. **Pruning selectivo:** `docker_prune` elimina contenedores SOLV detenidos
   (por label `solv.managed` o prefijo de nombre) e imágenes dangling; jamás
   toca volúmenes nombrados con trabajo de estudiantes.
4. **Executors inyectables:** `AdminGovernanceService` recibe el pruner y el
   pool resetter por inyección, manteniendo el core desacoplado del daemon
   Docker y del pool de conexiones; sin executor la API responde 503
   `executor_unavailable` en lugar de simular éxito.
5. **`reap-stale` como política continua:** el cierre de sesiones inactivas
   dejó de ser una acción manual y pasó al worker QoS con `inactivity_minutes`
   configurable (submódulo 14.6); documentado en el anexo de ADR-032.

## 4. Verificación

- Frontend: 352 tests en verde (6 del shell, 6 de Registro con drawer, 6 de
  Emergencias con doble confirmación y banner crítico)
- Backend: `go build`, `go vet` y `go test ./internal/...` en verde; suite de
  integración en `solv_test` incluye el timeline por actor y el catálogo
  ADR-032 completo (5 acciones, rechazo sin auditoría, acción desconocida)
- Gates de calidad: `npm run build` y `npm run lint:styles` en verde
  (0 componentes con estilos inline, 0 hex hardcodeado)
