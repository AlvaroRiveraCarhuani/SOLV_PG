# SOLV — Contexto y Reglas del Proyecto

## Qué es SOLV

Plataforma académica de orquestación de laboratorios virtuales (BaaS-ready).
On-Premise single-node hoy (servidor Asus), multi-tenant ready por diseño.
Roles: student, teacher, admin.

## Skills Obligatorias para Frontend y Desarrollo
Para cualquier sesión que modifique el frontend o arquitectura, deben consultarse y aplicarse las siguientes skills de `.agents/skills/`:
- `solv-context`: Decisiones inamovibles D1–D6 y reglas core del sistema.
- `angular-architecture`: Estándares de Angular 22 standalone, zoneless y Signals.
- `solv-design-system`: Tokens de diseño, tipografía, paleta semántica e inventario UI.
- `gobernanza`: Control de cambios, trazabilidad de ADRs y estándar de commits humanos.

## Stack (NO cambiar)

- Backend: Go 1.26 + net/http.ServeMux (SIN Gin) + sqlx + PostgreSQL 18
- Frontend: Angular 22 + Material + Lucide + SCSS
- Infra: Docker Engine v27 + Traefik v3 + OpenVSCode Server + Semgrep
- DNS/TLS: desec.io + Let's Encrypt Wildcard (ACME DNS-01). NUNCA dominios .local

## Convenciones de código

- Arquitectura hexagonal: core/domain, core/services, infrastructure/, delivery/http
- Migraciones idempotentes en postgres.go
- Tests de integración en backend/tests/integration/
- Commits Conventional Commits con cuerpo: Qué/Por qué/Cómo/Impacto

## PROHIBIDO

- Hardcodear dominios o "UAB" (todo vía config)
- Exponer test_cases de exercises al rol student
- Usar iptables normal para aislar el bridge (usar cadena DOCKER-USER o bind localhost)
- Crear tablas/componentes que dupliquen existentes
- Usar imágenes :latest (versiones fijadas)
- Usar fondos oscuros, azules o slate (#0F172A, #2563EB) en toasts/alertas de éxito (siempre verde semántico #15803D)
- Escribir literales de font-size, font-family o colores hex (#...) en SCSS de componentes: tipografía y familias únicamente con tokens var(--text-*), var(--font-sans|mono); colores únicamente vía tokens (--tenant-*, --color-*, --border-color, transparent, currentColor). Gate de stylelint `color-no-hex` activo en CI. Ejecutar `npm run lint:styles` antes de commitear.

## Semántica de estados (UI y API)

running=success · pending=warning · hibernated=neutral · failed/oom_killed=error
Veredictos juez: AC=verde · WA=rojo · TLE=ámbar · RE=naranja · AST_BLOCKED=rojo intenso

## Vocabulario PROHIBIDO (Evitar delatores de IA)

- **NO usar** -> **Usar en su lugar**:
  - `suite de pruebas` -> `conjunto de tests, los tests`
  - `veredicto PASS` -> `todos los tests pasan, tests en verde, pasando`
  - `cobertura automatizada` -> `tests que cubren, probado con`
  - `evidencias empíricas` -> `resultados, logs, salida`
  - `implementación exitosa` -> `implementado, listo, funcionando`
  - `se completó exitosamente` -> `quedó listo, terminó, se hizo`
  - `holgadamente` -> `bien, sin problemas`
  - `de forma idempotente` -> `idempotente`
  - `garantizando` -> `para asegurar, de modo que`
  - `mitigar` -> `evitar, reducir`
  - `robustecer` -> `endurecer, asegurar`
  - `orquestar` (contexto no técnico) -> `gestionar, manejar`
  - `paradigm` -> `enfoque, modelo`
  - `sinérgico / sinergia` -> (eliminar, no usar)
  - `robusto` -> `sólido, estable`
  - `escalar` -> `crecer, soportar más`
  - `resiliente` -> `tolerante a fallos`

## Estilo de commits

- Tono directo, primera persona del plural o impersonal técnico: "Se agregó X", "Agregamos X", "Agregado X".
- Frases cortas. Evitar subordinadas largas.
- Números concretos mejor que adjetivos (ej. "5 tests pasan").
- Sin superlativos ni marketing ("excelente", "óptimo", "poderoso", "elegante").
- Cuerpo obligatorio (1-2 líneas): Qué / Por qué / Cómo / Impacto.
- PROHIBIDO en commits: menciones a slices o cajones (ej. "Slice F1", "Cajón C"), "CRIT-XX" y cualquier vocabulario de coordinación interna o de orquestación.

## Etiquetas de Coordinación (Vocabulario Interno)

Las siguientes etiquetas son de planificación interna y NUNCA deben aparecer en commits, PRs, código fuente o documentación versionada:
- Cajón A, B, C, D
- Slice F1, F2, A1-A4
- Adendas 1-6
- D1-D5, INV-*, PU-*, MO-*, ST-*, PB-*, TE-*

Estas etiquetas solo existen en conversaciones de coordinación entre humanos. El código y los commits usan lenguaje de dominio (plantillas, categorías, cgroups, borradores) sin referencias a la planificación.
