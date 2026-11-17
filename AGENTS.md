# SOLV — Contexto y Reglas del Proyecto

## Qué es SOLV

Plataforma académica de orquestación de laboratorios virtuales (BaaS-ready).
On-Premise single-node hoy (servidor Asus), multi-tenant ready por diseño.
Roles: student, teacher, admin.

## Skills según intención (no cargar todas)

Cargar solo la skill que matchee la intención del trabajo; no cargar todas por defecto:

- `solv-context`: solo si el trabajo toca auth, sesiones, multi-tenancy, roles o decisiones D1–D6.
- `angular-architecture`: solo si se escribe o modifica código Angular.
- `solv-design-system`: solo si se tocan estilos, tokens, componentes UI o visual styling.
- `gobernanza`: en auditorías/lectura de convenciones; siempre antes de escribir commits, ADRs o docs de arquitectura.
- `angular-developer`: guías genéricas del framework, solo si `angular-architecture` no cubre el caso.

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
- Desarrollo indivisible de la documentación: Todo cambio importante (arquitectura, modelos, infraestructura, configuración de bases de datos o rediseño de flujos) debe acompañarse obligatoriamente de su documentación técnica correspondiente o actualización de ADR/especificación. No se considera terminado ningún desarrollo sin su documento o sección actualizada.

## PROHIBIDO

- Hardcodear dominios o "UAB" (todo vía config)
- Exponer test_cases de exercises al rol student
- Usar iptables normal para aislar el bridge (usar cadena DOCKER-USER o bind localhost)
- Crear tablas/componentes que dupliquen existentes
- Usar imágenes :latest (versiones fijadas)
- Usar fondos oscuros, azules o slate (#0F172A, #2563EB) en toasts/alertas de éxito (siempre verde semántico #15803D)
- Escribir literales de font-size, font-family o colores hex (#...) en SCSS de componentes: tipografía y familias únicamente con tokens var(--text-*), var(--font-sans|mono); colores únicamente vía tokens (--tenant-*, --color-*, --border-color, transparent, currentColor). Gate de stylelint `color-no-hex` activo en CI. Ejecutar `npm run lint:styles` antes de commitear.
- Toda fecha, timestamp, TTL o tiempo relativo se renderiza con el pipe compartido `dateText` (`@shared/pipes/date-text.pipe.ts`) dentro de un `<time>` semántico; la regla global `<time>` aplica fuente mono y números tabulares. Los valores técnicos visibles (códigos, IDs, métricas, versiones, tags, veredictos y logs) usan `[machineData]` o un contenedor semántico `<code>`, `<pre>`, `<kbd>`, `<samp>` o con la clase estática `font-mono`. Mantener las etiquetas y la prosa fuera del nodo marcado; no marcar controles, opciones ni etiquetas humanizadas de badges de estado. Los controles usan `--font-sans`; usar la fuente de UI configurada en `--font-sans`, nunca hardcodear Inter. Gate de CI `date-format-gate` bloquea `toLocale*String` fuera del formateador canónico.
- Usar el prefijo `solv-` en selectors o clases de componentes (PROHIBIDO `selector: 'solv-...'` o `class Solv...Component`). Utilizar nombres semánticos limpios en kebab-case para el selector (ej. `kpi-card`, `search-bar`, `status-tabs`, `pagination-bar`, `view-switcher`, `template-status-modal`).

## Semántica de estados (UI y API)

running=success · pending=warning · hibernated=neutral · failed/oom_killed=error
Veredictos juez: AC=verde · WA=rojo · TLE=ámbar · RE=naranja · AST_BLOCKED=rojo intenso

## Terminología de entornos (UI, documentación y agentes)

Correspondencia obligatoria entre el identificador interno y el nombre visible:

| Identificador interno (no se traduce) | Nombre visible |
| :--- | :--- |
| `JUEZ_EFIMERO` | Entorno de Evaluación |
| `IDE_PERSISTENTE` | Entorno de Desarrollo |
| `AMBOS` | Entorno de Desarrollo y Evaluación |

- Identificador, nombre visible y concepto técnico son tres capas distintas. El identificador es valor de contrato: no se renombra, no se traduce y no se escribe como etiqueta humana.
- «Juez Virtual» nombra el motor que evalúa soluciones. No se usa como nombre de modalidad en la interfaz del estudiante ni como sustituto de «Entorno de Evaluación».
- Toda etiqueta de modalidad sale de `@shared/constants/environment-types.constants.ts` a través del pipe `environmentLabel` (`@shared/pipes/environment-label.pipe.ts`) o de `environmentTypeOptions()` para filtros y selectores. PROHIBIDO escribir literales de modalidad en plantillas, componentes o utilidades.
- Un `environment_type` desconocido se muestra con el estado neutro del catálogo (`environmentBadge`); no se asume Entorno de Desarrollo ni Entorno de Evaluación.
- La política y sus excepciones están en `docs/ARQUITECTURA/ADR/ADR-067-politica-terminologica-de-modalidades.md`. El gate clasifica por forma (no por lista de palabras) y solo admite nombres antiguos en documentos históricos que referencian esa decisión.
- Gate `terminology-gate` en `npm run lint:styles` y en CI: falla ante etiquetas de modalidad fuera del catálogo, ante el catálogo incompleto y ante instrucciones de agentes desalineadas.

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

El estándar completo de commits (formato, cuerpo, footers, tono y prohibiciones) vive en la skill `gobernanza` (`.agents/skills/gobernanza/SKILL.md`). Cargarla antes de escribir cualquier commit; este archivo no duplica esas reglas.

## Etiquetas de Coordinación (Vocabulario Interno)

Las siguientes etiquetas son de planificación interna y NUNCA deben aparecer en commits, PRs, código fuente o documentación versionada:
- Cajón A, B, C, D
- Slice F1, F2, A1-A4
- Adendas 1-6
- D1-D5, INV-*, PU-*, MO-*, ST-*, PB-*, TE-*

Estas etiquetas solo existen en conversaciones de coordinación entre humanos. El código y los commits usan lenguaje de dominio (plantillas, categorías, cgroups, borradores) sin referencias a la planificación.
