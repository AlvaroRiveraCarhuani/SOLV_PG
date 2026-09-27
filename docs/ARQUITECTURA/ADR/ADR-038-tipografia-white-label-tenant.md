# ADR-038: Tipografía White-Label por Tenant con Catálogo Curado

**Estado:** Aprobado
**Fecha:** 2026-09-27
**Slice:** 14 (submódulo 14.6 Configuración y Mantenimiento)
**Relacionados:** ADR-012 (arquitectura frontend), ADR-024 (esquema multitenant), ADR-029 (períodos académicos)

---

## Contexto y Problema

SOLV es multitenant: cada universidad configura su identidad (logo, nombre,
color primario) sin hex hardcodeado, provista por `/api/v1/config/public` y
aplicada en runtime vía tokens CSS. La tipografía era el último elemento de
marca hardcoded: toda la plataforma usaba Inter (UI) y JetBrains Mono (datos
de máquina), sin posibilidad de que cada institución proyectara su identidad
tipográfica.

El contrato tipográfico del design system SOLV (personas vs. máquina) y la
escala tipográfica son canon normativo: cualquier solución debía respetarlos.

## Decisión

1. **Configuración por tenant:** `tenants.config` persiste `font_sans_family`
   y `font_mono_family` con formato `cat:slug` (catálogo curado) o
   `url:https://...` (hoja CSS custom). Servidos por `GET /api/v1/config/public`.

2. **Catálogo cerrado:** 6 fuentes sans (Inter, Source Sans 3, Open Sans,
   Public Sans, Lato, Roboto; pesos 400-700) y 3 mono (JetBrains Mono, Fira
   Code, IBM Plex Mono; pesos 400-600). Catálogo espejo sincronizado en
   `backend/internal/core/services/typography_service.go` (fuente de verdad)
   y `frontend/src/app/shared/curated-fonts.ts` (selectores y carga).

3. **Vía custom validada:** solo hojas CSS de `fonts.googleapis.com` (dominio
   permitido único), con validación backend de HTTPS + dominio +
   **alcanzabilidad real** (HTTP GET, timeout 5s) antes de persistir. Rechazo
   422 accionable (`font_slug_unknown`, `font_kind_mismatch`, `font_url_invalid`,
   `font_url_host_not_allowed`, `font_url_unreachable`).

4. **Aplicación runtime:** `TenantService.applyTenantFonts()` inyecta las hojas
   CSS del tenant (idempotente, un `<link>` por URL única, `display=swap`) y
   setea `--font-sans`/`--font-mono` en `:root`. Los ~126 usos CSS existentes
   consumen los tokens sin modificación. Los stacks con fallbacks de
   `_primitives.scss` permanecen como red de seguridad ante caída de CDN.

5. **Invariantes intactos:** defaults Inter/JetBrains Mono para tenants sin
   configuración; escala tipográfica, jerarquía de cabeceras y ponderaciones
   del design system NO configurables; el contrato personas (sans) vs. máquina
   (mono) se preserva con catálogos separados por `kind`.

## Consecuencias

- **Positivas:** identidad tipográfica por institución sin tocar CSS de la app;
  riesgo de legibilidad contenido por catálogo pre-aprobado; carga tolerante a
  fallos (swap + fallbacks); validación backend impide fuentes lentas o
  inseguras.
- **Negocias:** dependencia de la disponibilidad de Google Fonts por parte del
  navegador del usuario final; fuentes fuera del catálogo requieren ser hojas
  CSS de Google Fonts (no archivos woff/ttf sueltos).
- **Neutral:** la propagación a usuarios respeta la caché de 5 minutos de
  `config/public` (stale-while-revalidate existente).

## Cumplimiento

- Backend: `typography_service.go` + validación en `UpdateBranding` (7 tests)
- Frontend: `curated-fonts.ts`, `TenantService.applyTenantFonts`, sección de
  tipografía en la pestaña Identidad del submódulo 14.6 (9 tests)
