# Submódulo 14.6 — Configuración y Mantenimiento (`/admin/configuracion`)

**Estado:** Implementado (vista completa con las 3 pestañas del wireframe oficial)
**Fecha de cierre:** 2026-09-27
**Mapa de ejecución:** `especificaciones/mapa-implementacion-configuracion-14.6.md`
**Acta de implementación:** `especificaciones/acta-implementacion-14.6-configuracion.md`

---

## 1. ADRs del Submódulo

* [ADR-014: Estrategia Operativa, Observabilidad y Red](../../ARQUITECTURA/ADR/ADR-014-estrategia-operativa-observabilidad-red.md) *(políticas QoS del host)*
* [ADR-024: Esquema Académico Multi-Tenant Unificado](../../ARQUITECTURA/ADR/ADR-024-esquema-academico-multitenant.md) *(períodos y cursos)*
* [ADR-029: Períodos Académicos y Archivado de Cursos](../../ARQUITECTURA/ADR/ADR-029-periodos-academicos-archivado-cursos.md) *(hardening: archivado formal irreversible, migración 00007)*
* [ADR-031: Modo Mantenimiento Global](../../ARQUITECTURA/ADR/ADR-031-modo-mantenimiento-global.md) *(integración UI del switch con bypass administrativo)*
* [ADR-035: Backups Configurables con Retención](../../ARQUITECTURA/ADR/ADR-035-backups-configurables-retencion.md) *(consumo de la estrategia e historial existentes)*
* [ADR-038: Tipografía White-Label por Tenant](../../ARQUITECTURA/ADR/ADR-038-tipografia-white-label-tenant.md) *(nuevo: catálogo curado + URL custom validada)*

## 2. Alcance Implementado

| Pestaña | Funcionalidad | Backend |
| :--- | :--- | :--- |
| **Períodos Académicos** | KPIs del ciclo lectivo, tabla con ciclo de vida (Activo/Próximo/Archivado), alta con activación inmediata como unidad, edición, borrado con bloqueo por materias asociadas (409), archivo fuerte por tipeo del código **formal e irreversible** (is_archived + sellado :ro transaccional) | CRUD preexistente + `POST /{id}/archive` nuevo (migración 00007) + corrección del sweep que sellaba períodos futuros |
| **Identidad (White-Label)** | Split-screen: logo por subida real (multipart, max 2 MB) o URL, nombre, color HEX con contraste WCAG AA en vivo, **tipografía por catálogo curado (ADR-038)** con specimen y vía avanzada de URL custom validada, correo de soporte, preview en vivo del Juez Virtual | `PUT /admin/branding` (merge parcial + fuentes), `POST /tenants/logo` nuevo, servicio público del imagotipo |
| **Servidor y Respaldos** | Políticas QoS con radio-pills del catálogo (RAM 256/512/1024, inactividad 10/15/30, concurrencia 1-500), switch de Modo Mantenimiento con motivo obligatorio y ventana opcional, estrategia de respaldos + historial con Verificación SHA-256 y Descarga | `GET/PUT /admin/server/policies` nuevo con recarga por ciclo en el worker QoS; mantenimiento (ADR-031) y backups (ADR-035, Slice 16) preexistentes |

## 3. Decisiones Técnicas Relevantes

1. **Archivado formal e irreversible (ADR-029):** el diseño documentado exigía
   congelamiento permanente; la implementación histórica solo tenía `is_active`.
   Se alineó la BD al spec (migración 00007 con `is_archived`, `archived_at`,
   `archived_by`), con inmutabilidad blindada en servicio y queries (409
   `period_archived` en PUT/DELETE). El sweep de expiración fue corregido: ya no
   sella materias de períodos futuros planificados y formaliza los vencidos.
2. **Alta + activación como unidad:** `Create` del backend no desactiva otros
   períodos; el frontend compone `createAndActivate` para evitar ventanas con
   doble período activo.
3. **Tipografía por tokens (ADR-038):** `TenantService.applyTenantFonts()` inyecta
   hojas CSS de Google Fonts (display=swap, idempotente) y setea `--font-sans` /
   `--font-mono` en `:root`; la app completa consume los tokens sin tocar sus
   ~126 usos CSS. Catálogo espejo backend/front sincronizado.
4. **Guardado de identidad en 2 actos:** branding primero, tipografía después;
   un 422 tipográfico queda aislado en su banner sin perder el branding guardado.
5. **Aplicación local inmediata:** el admin que guarda ve su branding/fuente al
   instante (`applyBranding` local); los demás usuarios la reciben al expirar la
   caché de 5 minutos de `/config/public` (aviso visible en el panel).

## 4. Verificación

- Frontend: 336 tests en verde (incluye specs de las 3 pestañas: 4 del shell,
  7 de períodos, 7 de identidad, 9 de tipografía, 7 de servidor)
- Backend: `go build`, `go vet` y `go test ./internal/...` en verde (incluye
  tests de archivado formal, políticas QoS y validación tipográfica con
  alcanzabilidad real contra Google Fonts)
- Gates de calidad: `npm run build` y `npm run lint:styles` en verde
  (stylelint `color-no-hex` y gate D8 de estilos inline)
