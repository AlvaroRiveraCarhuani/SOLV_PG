# Mapa de tareas (`odd/`)

Índice de todas las fichas de [tasks/](tasks/). Cada ficha define objetivo, specs y decisiones. Este documento solo ordena el trabajo: no reemplaza a las fichas.

El estado es el que declara cada ficha. Cuando una ficha no declara estado, se indica como **sin estado declarado** y no se supone avance.

## 1. Área docente

Son las fichas que cambian la experiencia del docente. Se relacionan entre sí; el orden de lectura es el de la columna "Orden".

| Orden | Ficha | Objetivo corto | Estado declarado | Depende de / relacionada con |
|---|---|---|---|---|
| 1 | [docente-inicio-y-navegacion.md](tasks/docente-inicio-y-navegacion.md) | Pantalla "Hoy" de despacho y navegación docente por tarea | Planificada (sin marca de avance) | Diseño en `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` §2.4, §3.1, §4 |
| 2 | [docente-hoy-canvas.md](tasks/docente-hoy-canvas.md) | Canvas de materias y columna de pendientes | Planificada, en tres tramos | ADR-057; reemplaza la grilla de widgets de la ficha 1 |
| 3 | [docente-consistencia-panel.md](tasks/docente-consistencia-panel.md) | Panel, cola y banco cuentan lo mismo; sin botones muertos | Planificada | Sin dependencias duras |
| 4 | [docente-revision-y-libreta.md](tasks/docente-revision-y-libreta.md) | "Por calificar" declarado y libreta que no cuenta lo no corregido como 0 | Planificada; S7 sin marcar (resto sin verificar) | Forecast ~1200 líneas; requiere estrategia de entrega antes de empezar |
| 5 | [docente-catalogo-entornos.md](tasks/docente-catalogo-entornos.md) | Elegir entorno por lo que incluye (lenguaje, framework, BD) | Planificada; S1–S3 sin marcar | ADR-054 y `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md`; base del entorno por práctica |
| 6 | [docente-aula-y-sesiones.md](tasks/docente-aula-y-sesiones.md) | Aula en vivo por sesión de clase | **Etapa 2 entregada (2026-10-30)**; quedan §Pendientes | ADR-056; base de la ficha 7 |
| 7 | [docente-avance-voluntario.md](tasks/docente-avance-voluntario.md) | Avance voluntario del docente que el alumno decide aplicar | **Circuito completo (2026-10-31)** | ADR-058; ADR-056; ADR-005 §5 |
| 8 | [examen-congelamiento.md](tasks/examen-congelamiento.md) | Congelar el volumen del alumno al cerrar el examen | **Circuito completo (2026-10-31)** | ADR-005 §5; bloquea la emisión de avances en examen (ficha 7) |

## 2. Entornos, laboratorios y backend de IDE

| Ficha | Objetivo corto | Estado declarado | Relación |
|---|---|---|---|
| [persistent-ide-backend-oa1.md](tasks/persistent-ide-backend-oa1.md) | Reparar R1–R6 del IDE persistente: identidad, plantilla por ejercicio, arranque, entrega, `author_id` | Sin estado declarado (specs S1–S6) | Base del arranque por ejercicio (`exercise_id`) que usará el entorno por práctica |
| [catalog-guided-ecosystem-ux.md](tasks/catalog-guided-ecosystem-ux.md) | Perfiles de ecosistema oficiales (C/C++, Python, Java, etc.) y gestores de paquetes compatibles | Sin estado declarado | Relacionada con el catálogo docente (ficha 5) |
| [catalog-managed-databases-scale-to-zero.md](tasks/catalog-managed-databases-scale-to-zero.md) | Motores de BD gobernados por catálogo y arranque/parada automática | Sin estado declarado | Relacionada con ficha 5 |
| [template-studio-minimal-layout-and-context-inspection.md](tasks/template-studio-minimal-layout-and-context-inspection.md) | Quitar banners redundantes e inspector contextual en Template Studio | Sin estado declarado | Ninguna |
| [virtual-judge-e2e-full.md](tasks/virtual-judge-e2e-full.md) | Pruebas end-to-end del juez virtual (Fase 0 + J01–J07) | Sin estado declarado | Requiere `student-exercise-solver` (Fase 0) |
| [e2e-core-lifecycle-playwright.md](tasks/e2e-core-lifecycle-playwright.md) | Automatización Playwright del ciclo principal admin → docente → alumno | Sin estado declarado | Cubre el flujo de la ficha 5 y el de la ficha 7 |

## 3. IA, administración y notificaciones

| Ficha | Objetivo corto | Estado declarado | Relación |
|---|---|---|---|
| [ai-copilot-catalog-grounding-and-missing-flow.md](tasks/ai-copilot-catalog-grounding-and-missing-flow.md) | Copiloto IA con datos reales del catálogo y creación de tecnologías faltantes | Sin estado declarado | Depende del catálogo (ficha 5) |
| [ai-copilot-resilience-byok-and-ux.md](tasks/ai-copilot-resilience-byok-and-ux.md) | Copiloto autoadaptativo, autorreparación de entornos, BYOK docente, UX | Sin estado declarado | Puede compartir cambios con la ficha anterior |
| [admin-governance-and-access-control.md](tasks/admin-governance-and-access-control.md) | Gestión de administradores y control de acceso en configuración | Sin estado declarado | Ninguna |
| [academic-email-notifications-and-reminders.md](tasks/academic-email-notifications-and-reminders.md) | Correos académicos multi-tenant: evaluaciones, recordatorios a 12 h, bienvenida docente | Sin estado declarado | Ninguna |
| [host-telemetry-truthfulness.md](tasks/host-telemetry-truthfulness.md) | Cifras medidas en el dashboard de salud, sin estimaciones ocultas | Sin estado declarado; forecast ~1400 líneas | Requiere estrategia de entrega antes de empezar |

## 4. Calidad y arquitectura

| Ficha | Objetivo corto | Estado declarado | Relación |
|---|---|---|---|
| [lint-harness.md](tasks/lint-harness.md) | golangci-lint y ESLint/angular-eslint en CI y en el flujo local | Sin estado declarado (S1–S5) | Ninguna |
| [modular-architecture-refactoring.md](tasks/modular-architecture-refactoring.md) | Dividir los 5 archivos monolíticos de backend y componentes Angular | Sin estado declarado | Se solapa con la ficha siguiente |
| [modular-architecture-completar-refactor.md](tasks/modular-architecture-completar-refactor.md) | Completar la refactorización pendiente (archivos >520 líneas, Docker, componentes Angular) | Sin estado declarado | **Continuación de la anterior**: revisar antes de empezar para no repetir trabajo |

## Orden sugerido

1. Terminar la línea docente en este orden: ficha 1 → 2 → 3 → 4 → 5, y después la ficha 6 pendiente.
2. Entorno por práctica (sección 5), que depende del catálogo (ficha 5) y del arranque por ejercicio (`persistent-ide-backend-oa1`).
3. Calidad: `lint-harness` puede ir en cualquier momento; el refactor se planifica una sola vez tras revisar la continuación.
4. Revisión por modalidad (sección 6): cerrar primero el preview en vivo, que ya está implementado; después los comentarios por archivo y la auditoría de similitud, que dependen de decisiones abiertas.

## 5. Flujo de laboratorio docente (por definir)

Ficha: [docente-flujo-laboratorio.md](tasks/docente-flujo-laboratorio.md) (en curso; fila, fichas, prueba de entorno, apertura del IDE por práctica y retiro de `Mi entorno` hechos). Estas son las decisiones que debe cerrar.

- **Entorno por práctica: confirmado.** Cada práctica tiene su propio entorno; no es uno por materia. Un laboratorio puede usar una tecnología y otro, otra. La vista del docente es por práctica (`proyecto-grado/Observaciones/INICIO.md`). Consecuencia: el avance voluntario (ficha 7) y ADR-058 deben revisarse, porque hoy asumen un entorno por sesión de aula. Lo mismo para el volumen de avance.
- **Probar entorno: entorno separado de los alumnos.** Se acordó que el docente tenga un entorno propio para probar la práctica y no reutilizar el arranque de alumnos con `exercise_id`. Así no se mezcla con las entregas.
- **Regla de Fuzzing en la pestaña de la materia.** Sin decidir: unificarla antes de la tarea de entorno o dentro de ella. No depende de las decisiones anteriores.

## 6. Revisión por modalidad (fase 4 y siguientes)

Fichas derivadas de las decisiones del 2026-11-01 sobre revisión docente. La fase 4 está implementada en el árbol de trabajo, sin commit; las demás están planificadas y bloqueadas por decisiones explícitas en cada ficha.

| Orden | Ficha | Objetivo corto | Estado declarado | Depende de / relacionada con |
|---|---|---|---|---|
| 1 | [docente-preview-entrega-en-vivo.md](tasks/docente-preview-entrega-en-vivo.md) | Ejecutar la entrega IDE en un contenedor desechable y abrir su app desde SpeedGrader | **Implementada (2026-11-01), sin commit**; migración 00044 aplicada (2026-11-02); e2e pendiente | ADR-061; ADR-011; ADR-059 |
| 2 | [docente-comentarios-por-archivo.md](tasks/docente-comentarios-por-archivo.md) | Anclar cada comentario a `file_path` + línea local y dejar de duplicar `code` en IDE | Planificada; bloqueada por D1–D3 | ADR-060; fase 3 de `REVISION_POR_MODALIDAD.md` |
| 3 | [docente-auditoria-similitud-en-practica.md](tasks/docente-auditoria-similitud-en-practica.md) | Mover el anti-plagio a la práctica, restar el código base y devolver rangos de línea | Planificada; bloqueada por D1–D4; dos entregas | Motor de plagio; J07 y flujo 12 desactualizados |
| 4 | [docente-aula-abrir-ide.md](tasks/docente-aula-abrir-ide.md) | Abrir el IDE de la práctica desde la cabecera del Aula | Planificada; bloqueada por D1 | ADR-059; `TeacherEnvironmentService.openPracticeIDE` |
| 5 | [docente-vocabulario-residual.md](tasks/docente-vocabulario-residual.md) | Barrer «Juez Virtual» y «Laboratorio IDE» del copy visible | Planificada; bloqueada por D1 (alcance admin) | `Corregir.md`; `Reestructuracion Entorno Docente y Vocabulario.md` |

Fuera de alcance de esta sección: el **Banco de Ejercicios** como biblioteca maestra (rediseño de modelo, no de copy) y la extensión **SOLV Companion**, que no está en el repositorio. Ambas quedan como fichas futuras.

## Pendientes transversales

- Revisar `docs/ARQUITECTURA/ADR/ADR-058-avance-voluntario-del-docente-en-aula.md` cuando se cierre el entorno por práctica.
- Actualizar este mapa cuando una ficha cambie de estado.
