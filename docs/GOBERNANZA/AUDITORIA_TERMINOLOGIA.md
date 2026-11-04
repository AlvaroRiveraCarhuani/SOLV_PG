# Auditoría de Terminología y Lenguaje de Dominio — SOLV

> **Tipo:** auditoría de solo lectura. **Fecha de corte:** 2026-11-03.
> **Estado:** hallazgos y propuestas **pendientes de aprobación**. Este documento no renombra nada: describe lo que el repositorio dice hoy.
> **Fuentes:** código de `backend/` y `frontend/`, migraciones SQL, `docs/`, `AGENTS.md` y `.agents/skills/`. No se ejecutó la aplicación: el backend y PostgreSQL no estaban disponibles durante la revisión, así que los nombres de tablas se leen de `backend/migrations/`.

---

## 1. Resumen ejecutivo

SOLV tiene un **núcleo técnico coherente** y una **capa visible fragmentada**. En el código, las entidades están bien separadas y los identificadores en inglés representan conceptos estables (`exercises`, `subjects`, `environments`, `submissions`, `lab_templates`, `template_models`, `class_sessions`). En la interfaz, esos mismos conceptos aparecen con dos, tres o cuatro nombres distintos según el rol y la pantalla.

Los cuatro focos de confusión, por orden de impacto:

1. **Ejercicio / práctica / laboratorio / actividad.** La misma entidad es «Banco de Ejercicios» en el menú del docente y «Catálogo de Laboratorios» en el título de esa misma página, con subtítulo «Prácticas reutilizables» y botón «Nuevo Laboratorio».
2. **Curso / materia.** El menú del administrador dice «Cursos», la pantalla se titula «Cursos y Periodos Académicos» y el formulario de alta pide «Nombre de la Materia». El menú del estudiante dice «Mis Materias». En base de datos es `subjects`.
3. **Plantilla / modelo / entorno.** El administrador tiene «Plantillas Docker» en el menú y «Plantillas de Entornos» en la página; el catálogo de modelos se llama «Biblioteca de Modelos y Categorías»; el docente ve «Catálogo de Entornos». Son tres objetos relacionados (plantilla técnica, modelo de catálogo y su agrupación), pero los nombres no dejan ver la relación.
4. **Environment / laboratorio / instancia / entorno de práctica.** El estudiante navega «Laboratorios», el IDE muestra «Entorno del alumno», el administrador ve «instancias encontradas» y el aviso del estudiante habla de «entorno de práctica propio».

Hay además **dos discrepancias con riesgo funcional**, no solo de estilo:

- El frontend envía `JUEZ_VIRTUAL` como `target_environment` mientras el dominio define `JUEZ_EFIMERO`. El backend tolera ambos alias en varios puntos (`template_composer_service.go:47`, `admin_governance_templates.go:90`), lo que confirma que la convivencia existe, pero ningún ADR declara el alias.
- El contrato de API documentado define **5 veredictos** (`docs/API/CONTRATOS.md:162`) y el dominio implementa **9** (`exercise.go:68-79`). Lo mismo pasa con los estados de environment: el dominio tiene 8 y `AGENTS.md:48` documenta 5.

No existe glosario ni vocabulario de dominio vinculante. Las únicas reglas de lenguaje en `AGENTS.md` son de **estilo** (delatores de IA, etiquetas de coordinación prohibidas en commits) y de **semántica de estado**, no de nombres de conceptos. Eso deja abierta la puerta a que cualquier cambio —humano o agente— bautice de nuevo lo que ya tiene nombre.

---

## 2. Conceptos que ya están estandarizados

| Concepto | Denominación estable | Evidencia |
| :--- | :--- | :--- |
| Roles y actores | `student`, `teacher`, `admin` en código y guards; «Estudiante», «Docente», «Administrador» en la interfaz | `AGENTS.md:7`; `frontend/src/app/core/guards/{auth,admin,teacher}.guard.ts` comparan esos tres valores; menús `admin-layout.component.ts`, `teacher-layout.component.ts`, `sidebar.component.ts` |
| Entrega | «Entrega» visible; `submission` en código y tabla | `student-history.component.html:3` («Historial de entregas»); `domain/academic.go:35` (`Submission`); tabla `submissions`; endpoints `/api/v1/submissions`, `/api/v1/teacher/submissions/` |
| Casos de prueba y su visibilidad | `example` / `public` / `hidden` | `domain/exercise.go:47-49`; `docs/ARQUITECTURA/ADR/ADR-039-extraccion-relacional-casos-prueba-visibilidad-tres-estados.md` |
| Modo de examen congelado | `frozen` como estado buscado, no como fallo | `domain/environment.go:27-30`; `domain/exam_freeze.go:1-20`; etiqueta visible «Examen cerrado» en `status-badge.component.ts` |
| Estados del aula | `open` / `closed` | `domain/class_session.go:8-12`; tabla `class_sessions` |
| Estados de módulo curricular | `locked` / `unlocked` / `in_progress` / `completed` | `domain/course_module.go:12-16` |

Estos conceptos no necesitan intervención. Son consistentes entre base de datos, backend, API y al menos una etiqueta visible.

---

## 3. Conceptos parcialmente estandarizados

### 3.1 Plantilla de entorno

**Nombres encontrados**

| Capa | Nombre | Ubicación |
| :--- | :--- | :--- |
| Menú admin | «Plantillas Docker» | `frontend/src/app/features/admin/layout/admin-layout.component.ts:14` |
| Página admin | «Plantillas de Entornos» | `frontend/src/app/features/admin/templates/admin-templates.component.html:18` |
| Menú docente | «Catálogo de Entornos» | `frontend/src/app/features/teacher/layout/teacher-layout.component.ts:28` |
| Página docente | «Catálogo de Entornos» | `frontend/src/app/features/teacher/templates/teacher-templates.component.html:5` |
| Backend | `LabTemplate`, tabla `lab_templates` | `domain/environment.go` (tipo `LabTemplate`); `migrations/00001_baseline.sql:75` |
| Documentación | `docker_templates` | `ADR-030-catalogo-plantillas-docker-aprobacion.md` |
| Frontend | `docker_template*` en modelos y servicios | 12 apariciones en `frontend/src` |

**Problema.** El nombre visible cambia entre el menú y la página del mismo rol («Plantillas Docker» → «Plantillas de Entornos»), y la documentación histórica sigue hablando de `docker_templates`, una tabla que no existe en el esquema actual. Quien lee un ADR y luego busca la tabla no encuentra correspondencia.

**Impacto.** Medio. No rompe nada, pero obliga a traducir mentalmente en cada lectura.

### 3.2 Modelo de catálogo

**Nombres encontrados.** «Biblioteca de Modelos y Categorías» (`model-library.component.html:15`), menú «Modelos y Categorías» (`admin-layout.component.ts:16`), tabla `template_models` (`00001_baseline.sql:152-169`).

**Problema.** «Modelo» también se usa en otros dos sentidos en la plataforma: modelo de lenguaje en la configuración de IA (`admin-config-ia.component.html`, campo «modelo seleccionado») y modelo de datos en la documentación de base de datos. La relación real —un modelo de catálogo nace de una plantilla técnica, `template_models.source_template_id` → `lab_templates.id` (`00001_baseline.sql:156`)— no aparece en ninguna etiqueta visible.

**Impacto.** Medio. Un administrador nuevo no puede deducir que «modelo» y «plantilla» son objetos distintos y encadenados.

### 3.3 Juez de evaluación

**Nombres encontrados**

| Capa | Nombre | Ubicación |
| :--- | :--- | :--- |
| Dominio | `JUEZ_EFIMERO` | `domain/exercise.go:20`; 117 apariciones en `backend/` |
| Dominio (alias) | `JUEZ_VIRTUAL` | `template_composer_service.go:47`; `admin_governance_templates.go:90` |
| Frontend | `JUEZ_VIRTUAL` como valor de `target_environment` y etiqueta «Entorno de Evaluación» | `model-library.component.ts:159` |
| UI admin | «Evaluador Automático» | `template-edit-modal.component.html:64` |
| Documentación | «Juez Virtual» (89 apariciones) y «Evaluador Automático» (25) | `docs/` |
| Glosario del manual | «Juez Evaluador (Sandbox)» | `docs/UI/ADMIN/MANUAL_ADMIN.md:14` |

**Problema.** Cuatro nombres para el mismo evaluador y un alias técnico (`JUEZ_VIRTUAL`) que no está declarado en ningún ADR. El manual del administrador define un nombre («Juez Evaluador») que no se usa en ninguna otra parte del sistema.

**Impacto.** Alto en lo técnico: cualquier comparación estricta contra `JUEZ_EFIMERO` puede no reconocer un registro guardado como `JUEZ_VIRTUAL`. Hoy el backend lo compensa en cuatro puntos concretos, lo que indica que el problema ya se sintió.

### 3.4 Entorno de Desarrollo / Entorno de Evaluación

**Nombres encontrados.** El par visible está bien establecido (`model-library.component.ts:158-159`, `template-review-modal.component.html:21-25`), pero convive con tres códigos: `IDE_PERSISTENTE`, `JUEZ_EFIMERO`/`JUEZ_VIRTUAL` y `AMBOS`.

**Problema.** `AMBOS` se etiqueta «Entorno Dual» (`template-review-modal.component.html:21`) mientras el dominio `EnvironmentType` (`exercise.go:17-22`) solo declara dos valores y no conoce la modalidad dual. La misma frase «Entorno de Desarrollo» designa el tipo de ejecución y, en el modal de plantilla, la modalidad completa. Además, el modal escribe «Entorno Dual» con mayúscula y el resto de la interfaz usa minúscula para los nombres de tipo.

**Impacto.** Medio. Afecta la lectura de las plantillas y de los ejercicios que las consumen.

### 3.5 Estados del entorno

**Nombres encontrados.** Ocho códigos en dominio: `pending`, `running`, `hibernated`, `oom_killed`, `failed`, `unknown`, `orphaned`, `frozen` (`domain/environment.go:12-30`). Cinco documentados en `AGENTS.md:48`. Etiquetas visibles del badge compartido: «Activo», «Iniciando», «En Pausa», «Límite de Memoria», «Error», «Sin Verificar», «Huérfano», «Examen cerrado» (`status-badge.component.ts:43-62`).

**Problema.** Los tres estados agregados después (`unknown`, `orphaned`, `frozen`) no figuran en la tabla de semántica de `AGENTS.md`, que sigue siendo la referencia para diseñar la interfaz. Además, fuera del badge conviven etiquetas alternativas: «En reposo (Hibernado)» en el filtro de `course-environments-modal.component.ts:52`, y «Pausada»/«Pausadas» para el estado de una plantilla en `admin-templates.component.html:81` y `:457`, que comparte idea con «En Pausa» pero la nombra distinto.

**Impacto.** Medio. El estado se muestra bien donde importa (badge), pero la documentación de referencia está incompleta.

### 3.6 Veredictos del juez

**Nombres encontrados.** Nueve en dominio: `AC`, `WA`, `TLE`, `RE`, `CE`, `MLE`, `VE`, `AST_VIOLATION`, `AST_BLOCKED` (`exercise.go:68-79`). Cinco en el contrato de API (`docs/API/CONTRATOS.md:162`), cinco en `AGENTS.md:49` y cinco en el glosario del manual (`PLANTILLAS_DOCKER.md:429-434`).

**Problema.** El contrato que consumen frontend y documentación está incompleto respecto del dominio. `AST_VIOLATION` y `AST_BLOCKED` conviven en el código con significados distintos (regex histórico y Semgrep), pero solo uno aparece en la documentación.

**Impacto.** Alto para integraciones: un cliente que valide contra el contrato documentado rechaza veredictos válidos.

### 3.7 Solicitud de entorno

**Nombres encontrados.** Menú «Solicitudes docentes» (`admin-layout.component.ts:15`), página «Solicitudes de entornos» (`admin-environment-requests.component.html:4`), dominio `EnvironmentRequest` con estados en español (`environment_request.go:12-17`) y usos `practice`/`grade` en inglés (`environment_request.go:20-22`).

**Problema.** El menú nombra al solicitante; la página nombra el objeto. Los estados internos están en español y mayúsculas (`EN_REVISION`, `APROBADA`, `NECESITA_CAMBIO`, `NO_VIABLE`) mientras el resto de enums del dominio usa inglés en minúsculas.

**Impacto.** Medio. La inconsistencia de idioma en valores ya obliga a mirar el código para saber qué llega por API.

### 3.8 Entorno de revisión de una entrega

**Nombres encontrados.** Tipo `ReviewEnvironment` (`domain/review_environment.go`), tabla `review_test_environments`, endpoint `POST /api/v1/teacher/submissions/{id}/test-environment` (`teacher-grading.service.ts:141`) y método `startReviewEnvironment` (`:138`). La tabla mezcla los dos nombres.

**Problema.** Tres nombres técnicos para el mismo objeto. No hay etiqueta visible que lo nombre; el docente lo descubre por la acción, no por un término.

**Impacto.** Medio-bajo hoy (la función se usa poco), alto si se documenta o se expone en la API pública.

### 3.9 Componentes técnicos del catálogo

**Nombres encontrados.** Tabla `template_components`; en el router conviven tres prefijos para el mismo recurso —`/api/v1/admin/template-components` (líneas 251-254 y 259), `/api/v1/admin/tech-components/{id}` (255-256) y `/api/v1/admin/components/{id}` (257-258)—, todos contra `UpdateTemplateComponent`; columna `tools` en `template_models` (`00001_baseline.sql:161`); en la interfaz solo existen «Herramientas» (24 apariciones) y «Tecnologías» (30), nunca «componente técnico».

**Problema.** El nombre del backend no tiene equivalente visible, y los dos nombres visibles no son intercambiables entre sí (una «tecnología» agrupa y una «herramienta» es la pieza).

**Impacto.** Medio. Quien lee la API y luego la pantalla no encuentra el término.

### 3.10 Mapa de vistas y recuentos de la documentación

**Nombres encontrados.** `docs/UI/MAPA_DE_VISTAS.md` describe rutas `/docente/cursos/:id` (línea 139), `/docente/laboratorios/nuevo` (145), `/docente/juez` (194) y `/estudiante/juez/:ejercicioId` (68); las rutas reales son `/teacher/...` y `/student/...` (`app.routes.ts`, `teacher.routes.ts`). El mismo documento llama «hojas de trabajo» a los ejercicios (línea 66). `docs/README.md:12` habla de «14 Slices» y `:48` de «Slices 1 al 16»; `:19` anuncia «29 Registros de Decisiones (ADR-000 al ADR-028)» cuando la carpeta supera el ADR-060.

**Problema.** Las definiciones oficiales de vistas y de arquitectura no siguen vigentes, y con ellas el vocabulario que usan.

**Impacto.** Medio. La documentación de referencia es la que más leen las personas nuevas y los agentes.

---

## 4. Conceptos sin denominación establecida

| Concepto | Cómo se nombra hoy | Evidencia | Por qué no alcanza |
| :--- | :--- | :--- | :--- |
| Instancia concreta de un laboratorio | «instancia» | `course-environments-modal.component.html:140` («instancias encontradas»), `student-courses-modal.component.html` usa `environment_id` | «Instancia» no aparece en ninguna otra capa; en el dominio es un `PracticeEnvironment` |
| Entorno de práctica del estudiante | «entorno de práctica» | `student-environments.component.html:7` | Es el mismo objeto que el menú llama «Laboratorios» y el IDE llama «Environment» |
| Duración de vida de un laboratorio | «sesión» | `student-environments.component.html:7` («entre sesiones»), `course-environments-modal.component.html:127` («sesiones de laboratorio») | «Sesión» también es el aula y también el inicio de sesión |
| Actividad como genérico | «actividad» | `student-evaluations.component.html:6` («actividades publicadas»), `teacher-templates.component.html:284` («crear una nueva actividad con este entorno») | No se sabe si es un ejercicio, una práctica o un examen |
| Modalidad que combina desarrollo y evaluación | «Entorno Dual» / «AMBOS» | `template-review-modal.component.html:21`; `admin_governance_templates.go:189` | El dominio no la declara como tipo; solo se acepta como valor de texto |
| Pieza del catálogo técnico | «herramienta» / «tecnología» | 24 y 30 apariciones respectivas en templates | No hay término único ni definición de la diferencia |

---

## 5. Ambigüedades y contradicciones

1. **Curso y materia, en la misma pantalla.** «Cursos y Periodos Académicos» es el título (`admin/courses/admin-courses.component.html:17`) y el formulario pide «Nombre de la Materia» (`course-create-modal.component.html:16`). En base de datos es `subjects`; el helper `GetTeacherCourses` arma `TeacherCourseItem` desde `subjects` (`teacher_invitation_repository.go:296-312`). Además, `CourseModule.SubjectID` y `CourseCurricularMap.CourseID` nombran la misma relación con dos palabras (`course_module.go:27` y `:67`), y `classroom_course_id` designa un curso real de Google Classroom.
2. **Ejercicio, práctica y laboratorio.** «Banco de Ejercicios» (menú) contra «Catálogo de Laboratorios» (título), «Prácticas reutilizables» (subtítulo) y «Nuevo Laboratorio» (botón) en la misma vista (`teacher-layout.component.ts:27`; `teacher-exercises.component.html:5-20`). El backend y la API dicen `exercises`.
3. **Plantilla, modelo y entorno.** Tres etiquetas para el catálogo según el rol y la pantalla (§3.1, §3.2), sin que la relación entre plantilla técnica y modelo de catálogo sea visible.
4. **Environment, laboratorio, instancia y entorno de práctica.** Cuatro nombres para el mismo entorno de ejecución (§4). El código usa `environment` incluso en texto visible (`student-exercise-solver.component.html:174`, `:198`, `:224`).
5. **Sesión.** Cuatro significados en la interfaz: inicio de sesión (`login.component.html:58`), aula (`class_sessions`, menú «Aula»), vida del laboratorio («entre sesiones») y examen («eventos de pegado en la sesión», `keystroke-replay-modal.component.html:67`).
6. **Juez y evaluador.** «Juez Virtual», «Evaluador Automático», «Juez Evaluador» y el alias `JUEZ_VIRTUAL` (§3.3).
7. **Evaluaciones.** El estudiante ve «Evaluaciones y prácticas» (`student-evaluations.component.html:3`); el docente tiene un menú «Por calificar» (`teacher-layout.component.ts:21`) y además rutas `por-calificar` y `evaluaciones` (`teacher.routes.ts:97` y `:102`), cuya página se titula «Centro Global de Evaluaciones» (`teacher-evaluations.component.html:5`). Tres nombres para el mismo flujo de calificación.
8. **Entorno de Desarrollo.** Designa el tipo de ejecución (`IDE_PERSISTENTE`) y, en «Entorno Dual (Entorno de Desarrollo + Evaluador Automático)», la modalidad completa de una plantilla.
9. **Idioma de los valores de dominio.** `class_session_participants` usa `sin_empezar`, `trabajando`, `trabado`, `entregado`, `ausente` (`class_session.go:19-30`) mientras `class_sessions` usa `open`/`closed` (`:8-12`) y los módulos usan `locked`/`unlocked`/`in_progress`/`completed` (`course_module.go:12-16`). Los estados de solicitud están en español en mayúsculas y los veredictos en inglés en mayúsculas.

---

## 6. Correspondencia entre frontend, backend y documentación

| Concepto | Visible | Modelo TypeScript | Go / tabla / endpoint | Documentación | Correspondencia |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Ejercicio | «Ejercicio», «Laboratorio», «Práctica», «Actividad» | `Exercise` | `Exercise` / `exercises` / `/api/v1/exercises` | «Laboratorio», «hojas de trabajo» | Parcial |
| Materia | «Materia», «Curso» | `Course`-prefijo (`CourseModule`), `subject` en payloads | `Subject` / `subjects` / `/api/v1/courses/...` | «Curso», «materia» | Parcial |
| Plantilla técnica | «Plantilla Docker», «Plantilla de Entornos», «Entorno» | `Template` | `LabTemplate` / `lab_templates` | `docker_templates` | Parcial |
| Modelo de catálogo | «Modelo» | `TemplateModel` | `template_models` | «modelo» | Aceptable, sin relación visible con plantilla |
| Entorno de ejecución | «Laboratorio», «Environment», «Instancia», «Entorno de práctica» | `Environment` | `PracticeEnvironment` / `environments` / `/api/v1/environments/...` | «laboratorio», «environment» | Parcial |
| Aula | «Aula» | `ClassSession` | `ClassSession` / `class_sessions` / `/api/v1/teacher/sessions` | «Aula» | Buena en UI; la API usa `sessions` |
| Entrega | «Entrega» | `Submission` | `Submission` / `submissions` | «entrega» | Buena |
| Evaluador | «Juez Virtual», «Evaluador Automático» | `JUEZ_VIRTUAL` | `JUEZ_EFIMERO` (+ alias) | «Juez Virtual», «Juez Evaluador» | Contradictoria |
| Tipo de entorno | «Entorno de Desarrollo», «Entorno de Evaluación», «Entorno Dual» | `IDE_PERSISTENTE`, `JUEZ_VIRTUAL`, `AMBOS` | `IDE_PERSISTENTE`, `JUEZ_EFIMERO`, `AMBOS` | «Juez Virtual» / «IDE persistente» | Parcial |
| Estados del entorno | «Activo», «En Pausa», «Examen cerrado»… | `EnvironmentStatus` | 8 estados | 5 en `AGENTS.md` | Incompleta |
| Veredictos | `AC`, `WA`, `TLE`… | unión de 5 | 9 estados | 5 | Incompleta |
| Componente técnico | «Herramienta», «Tecnología» | `TemplateComponent` | `template_components` / `tech-components` | «componente», «tecnología» | Débil |
| Entorno de revisión | sin etiqueta | `ReviewEnvironmentAccess` | `ReviewEnvironment` / `review_test_environments` / `test-environment` | «entorno de prueba» | Débil |

---

## 7. Propuestas de nombres canónicos (pendientes de aprobación)

Ninguna propuesta se aplicó. Cada fila distingue los tres niveles: **visible** (lo que lee la persona), **conceptual** (cómo se habla del objeto en diseño y documentación) y **técnico** (identificador de código, que puede seguir en inglés).

### 7.1 Tabla de propuestas

| Concepto | Visible propuesto | Conceptual | Técnico (sin cambios) | Evitar como nombre de cosa | Impacto |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Ejercicio | **Ejercicio** | Ejercicio | `exercise` | «Laboratorio», «Práctica», «Actividad» en títulos y botones | Alto: 4 etiquetas de menú y página, más copia docente |
| Materia | **Materia** | Materia | `subject` | «Curso» para la materia; reservarlo para Classroom | Medio-alto: menú admin, títulos y modales |
| Plantilla técnica | **Plantilla** | Plantilla de entorno | `lab_template` | «Docker» como parte del nombre; `docker_templates` en docs | Medio: menú, página y ADR |
| Modelo de catálogo | **Modelo del catálogo** | Modelo | `template_model` | «Modelo» a secas (choca con modelo de IA) | Bajo-medio: títulos y filtros |
| Entorno de ejecución | **Laboratorio** | Entorno de ejecución | `environment` | «Environment» e «Instancia» en texto visible | Medio: 3 textos del IDE y 1 del admin |
| Aula | **Aula** | Sesión de clase | `class_session` | «Sesión» sola para el aula | Bajo: ya es el uso dominante |
| Evaluador | **Juez Virtual** | Juez virtual | `JUEZ_EFIMERO` (alias `JUEZ_VIRTUAL` documentado o retirado) | «Evaluador Automático», «Juez Evaluador» | Medio: 2 modales y glosario |
| Tipos de entorno | **Entorno de Desarrollo** / **Entorno de Evaluación** / **Entorno dual** | Tipo de entorno | `IDE_PERSISTENTE` / `JUEZ_EFIMERO` / `AMBOS` | «Entorno Dual» con mayúscula inconsistente | Bajo: ya está casi estable |
| Estados | Las 8 del badge | Estado de laboratorio | los 8 códigos | «En reposo (Hibernado)» y sinónimos fuera del badge | Bajo |
| Componente técnico | **Componente** | Componente técnico | `template_component` | «Herramienta» y «Tecnología» como sinónimos | Medio: catálogo y solicitudes |
| Entorno de revisión | **Entorno de revisión** | Entorno de revisión | `review_environment` | «test environment» en contratos nuevos | Bajo hoy |

### 7.2 Justificación breve

- **Ejercicio.** Es el nombre del dominio, de la tabla, de la API y del menú del docente. «Práctica» puede seguir en prosa explicativa («la práctica de la sesión»), pero no en títulos, botones ni elementos de navegación. Impacto alto porque la página del docente se contradice a sí misma; el cambio es de etiquetas, no de datos.
- **Materia.** El estudiante ya dice «Mis Materias» y el formulario de alta pide «Nombre de la Materia»; la base de datos dice `subjects`. Dejar «Curso» para el curso de Google Classroom evita que dos objetos distintos compartan palabra.
- **Plantilla.** El nombre del menú y el de la página deben coincidir. «Docker» describe la implementación, no el objeto, y ya está en la documentación de infraestructura. `docker_templates` en los ADR conviene anotarlo como nombre histórico.
- **Modelo del catálogo.** Aclarar la relación con la plantilla en la propia pantalla («modelo derivado de la plantilla X») resuelve la ambigüedad sin renombrar la tabla.
- **Laboratorio.** Es la palabra que ya usan el menú del estudiante y casi toda la copia docente. «Environment» queda como nombre técnico y no debería aparecer en el IDE ni en avisos.
- **Juez Virtual.** Es el nombre con más presencia en la documentación (89 apariciones) y el de los ADR. Unificar la etiqueta visible en «Juez Virtual» y declarar el alias `JUEZ_VIRTUAL` —o retirarlo en una migración— cierra el riesgo funcional.
- **Componente.** El backend ya tiene un nombre (`template_components`) y el catálogo lo usa en el flujo de solicitudes. «Tecnología» puede quedar para la agrupación por categoría, si se define esa diferencia.

---

## 8. Priorización

### Alta

| Hallazgo | Por qué |
| :--- | :--- |
| Ejercicio / práctica / laboratorio / actividad | La misma pantalla se contradice con su menú; afecta qué cree estar creando el docente |
| Curso / materia / `subject` | Nombra la entidad académica central; aparece en administración, docencia y estudiante con dos palabras |
| `JUEZ_VIRTUAL` frente a `JUEZ_EFIMERO` | Alias no declarado con comparaciones estrictas en varias capas |
| Veredictos: 5 documentados, 9 implementados | El contrato de API queda incompleto para cualquier cliente |
| Plantilla / modelo / entorno | Tres nombres para el catálogo que administra el docente y el administrador |

### Media

| Hallazgo | Por qué |
| :--- | :--- |
| Environment / laboratorio / instancia / entorno de práctica | Consistente en código, disperso en la interfaz |
| «Sesión» con cuatro significados | Confunde aula, login, vida del laboratorio y examen |
| Estados nuevos fuera de `AGENTS.md` | La referencia de diseño no cubre `unknown`, `orphaned` y `frozen` |
| Componente técnico / herramienta / tecnología | Sin término único entre API y pantalla |
| Solicitud de entorno: menú contra página | Dos nombres para la misma bandeja |
| Entorno de revisión con tres nombres técnicos | Deuda para cuando se documente o se publique |
| Mapa de vistas con rutas inexistentes | La documentación de referencia induce a error |

### Baja

| Hallazgo | Por qué |
| :--- | :--- |
| «Entorno Dual» con mayúsculas inconsistentes | Cosmético |
| Rutas `/cursos` y `/courses` con alias de redirección | Ya resuelto con redirecciones; no afecta al usuario |
| Recuentos de slices y ADR en `docs/README.md` | Desactualización de índice, sin efecto funcional |
| «En reposo (Hibernado)» en un filtro | Etiqueta aislada |

### Sin intervención

Roles, entrega, casos de prueba y su visibilidad, estados del aula, estados de módulo, códigos de estado del laboratorio y semántica de veredictos. Son consistentes y cualquier cambio sería cosmético.

---

## 9. Recomendaciones para evitar inconsistencias introducidas por agentes

**Situación actual.** `AGENTS.md` fija roles, semántica de estados, vocabulario prohibido (delatores de IA) y etiquetas de coordinación que no deben aparecer en el código. Ninguna de esas reglas dice cómo se llaman los conceptos del dominio. `.agents/skills/solv-context/SKILL.md` describe decisiones (D1–D6) y menciona `environments` con discriminador `type` (línea 18), pero no fija vocabulario visible. `.agents/skills/solv-design-system/SKILL.md` gobierna primitivas de interfaz, no sustantivos del dominio.

**Consecuencia observable.** Las contradicciones de la sección 5 no provienen de una decisión: provienen de que cada pantalla eligió su palabra. Un agente que agregue una vista nueva tiene hoy la misma libertad y repetirá el patrón.

**Recomendaciones (ninguna aplicada; requieren aprobación):**

1. **Vocabulario de dominio en `AGENTS.md`.** Una tabla corta con los términos aprobados y, para cada uno, el nombre visible, el conceptual y el técnico. No hace falta un glosario completo: alcanza con los conceptos de la sección 7.
2. **Regla de reutilización.** «Antes de escribir una etiqueta nueva, usá la que ya existe en el menú o en el título de la pantalla; si el concepto no está en la tabla, preguntá antes de inventarlo.»
3. **Regla de capas.** Los nombres técnicos pueden seguir en inglés; los visibles no mezclan idiomas ni introducen sinónimos nuevos.
4. **Lista de sinónimos vetados.** La columna «Evitar» de la sección 7.1, sin necesidad de un gate automático.
5. **Revisión de documentación al cerrar un cambio de interfaz.** Cuando se renombra una pantalla o una ruta, `docs/UI/MAPA_DE_VISTAS.md` se actualiza en el mismo cambio. Hoy ese archivo describe rutas que ya no existen.
6. **Contratos completos.** Cuando el dominio agrega un estado o un veredicto, `docs/API/CONTRATOS.md` y `AGENTS.md` se actualizan en el mismo cambio. Es la misma disciplina que ya se aplica a las ADR.

Un gate terminológico automático quedó fuera del alcance de esta auditoría y no se implementó.

---

## 10. Tabla resumen

| Concepto | Estado actual | Problema | Recomendación | Prioridad |
| :--- | :--- | :--- | :--- | :--- |
| Ejercicio | Parcial | Menú «Banco de Ejercicios» contra título «Catálogo de Laboratorios»; «práctica» y «actividad» en la copia | Visible «Ejercicio»; «práctica» solo en prosa | Alta |
| Materia | Parcial | «Cursos» en el menú admin, «Nombre de la Materia» en el formulario, `subjects` en la base | Visible «Materia»; «curso» para Classroom | Alta |
| Plantilla | Parcial | «Plantillas Docker» en menú y «Plantillas de Entornos» en la página; `docker_templates` en ADR | Unificar etiqueta y anotar el nombre histórico | Alta |
| Modelo de catálogo | Parcial | «Modelo» también significa modelo de IA; relación con la plantilla invisible | «Modelo del catálogo» y mostrar la plantilla de origen | Media |
| Juez | Parcial | «Juez Virtual», «Evaluador Automático», «Juez Evaluador» y alias `JUEZ_VIRTUAL` | Visible «Juez Virtual»; declarar o retirar el alias | Alta |
| Tipos de entorno | Parcial | Dos códigos, tres valores con `AMBOS`; «Entorno Dual» inconsistente | Mantener el par visible y nombrar la modalidad «Entorno dual» | Baja |
| Estados del entorno | Parcial | 8 estados, 5 documentados; etiquetas alternativas fuera del badge | Completar `AGENTS.md` y reutilizar el badge | Media |
| Veredictos | Parcial | 9 en dominio, 5 en contrato y en `AGENTS.md` | Completar contrato y semántica | Alta |
| Environment / laboratorio | Parcial | «Laboratorios», «Environment», «instancias», «entorno de práctica» | Visible «Laboratorio»; `environment` solo técnico | Media |
| Aula / sesión | Parcial | «Sesión» significa login, aula, vida del laboratorio y examen | «Aula» para la clase; evitar «sesión» sola | Media |
| Entrega | Estandarizado | — | Ninguna | — |
| Roles | Estandarizado | — | Ninguna | — |
| Casos de prueba | Estandarizado | — | Ninguna | — |
| Componente técnico | Sin estandarizar | «Herramienta» y «Tecnología» sin definición; API con tres rutas | Visible «Componente»; documentar la diferencia | Media |
| Entorno de revisión | Sin estandarizar | `ReviewEnvironment`, `test-environment`, `review_test_environments` | «Entorno de revisión» como nombre conceptual | Baja |
| Solicitud de entorno | Parcial | Menú «Solicitudes docentes», página «Solicitudes de entornos» | Alinear menú y página | Media |
| Evaluaciones del docente | Ambiguo | «Por calificar», «Centro Global de Evaluaciones» y ruta `evaluaciones` | Definir si son dos vistas o una | Media |
| Mapa de vistas | Parcial | Rutas `/docente`, `/estudiante` que ya no existen | Actualizar junto al cambio de rutas | Media |
| `LabInstance` / `LabService` | Sin correspondencia | ADR-019 declara eliminada la tabla y el repositorio, pero el tipo y dos servicios siguen en el código | Marcar como legado y retirar en un cambio aparte | Media |

---

## 11. Alcance, método y límites

- **Qué se revisó:** menús y títulos de las tres interfaces; modelos y servicios del frontend; dominio, servicios, handlers, router y migraciones del backend; ADR, guías de UI, manual del administrador, `AGENTS.md` y skills de agentes.
- **Cómo:** lectura directa de archivos y búsquedas de texto sobre el repositorio. Cada hallazgo cita archivo y línea.
- **Límites:** no se ejecutó la aplicación ni se consultó la base en vivo (backend y PostgreSQL no estaban disponibles), así que los valores de enum y los nombres de tabla provienen de migraciones y código. No se hicieron sesiones con usuarios reales: la auditoría refleja lo que el repositorio declara, no lo que una persona interpreta frente a la pantalla. La memoria del proyecto de grado se entrega como PDF y no se auditó su texto; los guiones y notas de `proyecto-grado/` son material de defensa, no documentación normativa, y no se usaron como fuente de definiciones.
- **Qué no se hizo, por pedido explícito:** renombrar variables, clases, componentes, endpoints, tablas o etiquetas; crear un glosario definitivo; implementar un gate; modificar skills o `AGENTS.md`; hacer commit.

**La decisión sobre qué estandarizar y con qué nombres queda pendiente de aprobación.**
