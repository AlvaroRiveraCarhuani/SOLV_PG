# Suite de Pruebas E2E Multi-Rol — Juez Virtual SOLV

Esta suite implementa pruebas end-to-end (E2E) con **alta fidelidad multi-rol**, imitando el comportamiento real de Administradores, Docentes y Estudiantes en tiempo real con aislamiento de sesiones JWT y monitoreo forense automatizado.

---

## 📁 Taxonomía de Flujos por Categorías de Complejidad

### 🌟 Flujo Core Integral
* **`01-core-lifecycle.spec.mjs`** — *Ciclo de Vida Multi-Rol Completo*
  * **Qué comprobamos:** Homologación y verificación en vivo de Plantilla Oficial Dual en Docker $\rightarrow$ Creación de Materia y Matriculación $\rightarrow$ Editor de Ejercicios en 3 Pasos con autosave sincronizado $\rightarrow$ Resolución del Estudiante en Python con veredicto **AC** $\rightarrow$ Apertura de la Cola de Entregas y auditoría del Docente en SpeedGrader.
  * **Roles:** `admin`, `teacher`, `student`.

---

### 🟢 Categoría 01: Flujos Fáciles / Fundacionales (`01-easy/`)
1. **`01-catalog-and-templates.spec.mjs`** — *Biblioteca de Modelos, Categorías y Plantillas de Juez*
   * **Para qué sirve:** Validar la visualización del catálogo de tecnologías oficiales y la creación de plantillas de Juez Efímero con verificación real en Docker.
   * **Qué comprobamos:** Carga de tecnologías, selección de runtime Python 3.12, comprobación de estado "Verificada" en el sidebar y publicación de la plantilla.
   * **Roles:** `admin`.
2. **`02-course-management.spec.mjs`** — *Gestión de Materias, Periodos y Matriculación*
   * **Para qué sirve:** Validar la administración de asignaturas y la coherencia de visibilidad entre roles.
   * **Qué comprobamos:** Creación de materia mediante modal, asignación de docente titular, matriculación de alumno y confirmación de visibilidad en los dashboards de Docente y Estudiante.
   * **Roles:** `admin`, `teacher`, `student`.
3. **`03-exercise-authoring-simple.spec.mjs`** — *Editor en 3 Pasos y Resolución Básica*
   * **Para qué sirve:** Validar el flujo estándar de creación de un ejercicio de Juez Efímero y su resolución feliz.
   * **Qué comprobamos:** Paso 1 (Título, Dificultad, Lenguaje, Enunciado), Paso 2 (Solución de referencia y casos de prueba), Paso 3 (Checklist sin bloqueantes y publicación) y resolución con veredicto **AC**.
   * **Roles:** `admin`, `teacher`, `student`.

---

### 🟡 Categoría 02: Flujos Medios / Lógica Pedagógica y de Negocio (`02-medium/`)
4. **`04-judge-verdicts.spec.mjs`** — *Matriz Completa de Veredictos del Juez Virtual*
   * **Para qué sirve:** Asegurar que el motor de evaluación clasifica exactamente cada resultado según el comportamiento del código.
   * **Qué comprobamos:** **WA** (Wrong Answer por cálculo incorrecto), **RE** (Runtime Error por índice fuera de rango), **TLE** (Time Limit Exceeded por sleep/bucle) y **AC** (Accepted con casos ocultos).
   * **Roles:** `admin`, `teacher`, `student`.
5. **`05-ast-rules-security.spec.mjs`** — *Barrera de Seguridad y Análisis Estático (AST)*
   * **Para qué sirve:** Garantizar que el Juez bloquea código malicioso o no autorizado antes de enviarlo a ejecución.
   * **Qué comprobamos:** Detección y bloqueo con veredicto **AST_BLOCKED** ante imports prohibidos (`os`, `sys`, `subprocess`) o funciones dinámicas (`eval()`, `exec()`), y aprobación con código limpio.
   * **Roles:** `admin`, `student`.
6. **`06-curricular-modules.spec.mjs`** — *Módulos Curriculares y Desbloqueo por Prerrequisitos*
   * **Para qué sirve:** Validar la progresión académica guiada del estudiante por módulos dependientes.
   * **Qué comprobamos:** Creación de Módulos 1 y 2 con enlace de prerrequisito, visualización de estado `locked` para el estudiante en el Módulo 2 y desbloqueo automático a `available` tras aprobar el Módulo 1.
   * **Roles:** `admin`, `student`.
7. **`07-rubrics-and-ide-projects.spec.mjs`** — *Laboratorios de IDE Persistente y Rúbricas al 100%*
   * **Para qué sirve:** Validar la configuración de laboratorios con entorno de desarrollo persistente y evaluación multicriterio.
   * **Qué comprobamos:** Creación de ejercicio tipo `project` con script de inicialización DDL/DML, asignación de rúbrica multicriterio con suma obligatoria del 100% y validación en checklist.
   * **Roles:** `admin`, `teacher`.

---

### 🔴 Categoría 03: Flujos Difíciles & Extremos (`03-hard/`)
8. **`08-concurrency-and-resource-limits.spec.mjs`** — *Límites de Tiempo Estrictos (100ms) y Recursos*
   * **Para qué sirve:** Evaluar la precisión del Juez ante límites de tiempo ultra exigentes ($O(N)$ vs $O(N^2)$).
   * **Qué comprobamos:** Detección de TLE en soluciones con retardos o ineficiencias y aprobación en menos de 150ms para soluciones optimizadas.
   * **Roles:** `admin`, `student`.
9. **`09-exam-mode-seed.spec.mjs`** — *Modo Examen con Semilla Aleatoria por Estudiante*
   * **Para qué sirve:** Garantizar la integridad académica en exámenes mediante variación de entradas.
   * **Qué comprobamos:** Creación de ejercicio en modalidad `purpose: exam` con `per_student_seed: true`, generación de casos y evaluación del examen.
   * **Roles:** `admin`, `student`.
10. **`10-bulk-import-and-analytics.spec.mjs`** — *Importación Masiva (Dry-Run) y Analíticas Docente*
    * **Para qué sirve:** Validar la gestión a escala de materias y la consulta de métricas de rendimiento.
    * **Qué comprobamos:** Consulta de analíticas de curso en tiempo real y métricas de desempeño estudiantil.
    * **Roles:** `admin`, `teacher`.

---

### ⚡ Categoría 04: Casos Borde y Escenarios Extremos (`04-edge-cases/`)
11. **`11-extreme-inputs-and-utf8.spec.mjs`** — *Caracteres UTF-8 Complejos, Emojis y Enteros de $10^{18}$*
    * **Para qué sirve:** Asegurar que el sistema no corrompe caracteres especiales, saltos de línea ni números gigantes.
    * **Qué comprobamos:** Entradas con emojis (🤖👋), caracteres con tildes y diéresis, saltos de línea múltiples y enteros de 20 dígitos procesados con veredicto **AC**.
    * **Roles:** `admin`, `student`.
12. **`12-anti-plagiarism-supervision.spec.mjs`** — *Supervisión Anti-Plagio y Detección de Similitud*
    * **Para qué sirve:** Proporcionar al docente herramientas de auditoría ante sospechas de copia de código.
    * **Qué comprobamos:** Generación y consulta del reporte de similitud de código entre entregas del curso.
    * **Roles:** `admin`, `teacher`.

---

### 🏆 Categoría 05: Badges, Calificación y SpeedGrader (`05-badges-and-grading/`)
13. **`13-speedgrader-audit-and-override.spec.mjs`** — *SpeedGrader, Comentarios por Línea y Override de Nota*
    * **Para qué sirve:** Validar la experiencia integral de calificación ágil del docente.
    * **Qué comprobamos:** Apertura de SpeedGrader, visor de código con resaltado, inserción de comentarios pedagógicos por línea y ajuste manual de nota (`override`) con justificación auditada.
    * **Roles:** `admin`, `teacher`.
14. **`14-student-gamification-and-progress.spec.mjs`** — *Progreso del Estudiante, Entregas y Estadísticas*
    * **Para qué sirve:** Comprobar el seguimiento continuo del estudiante y el reflejo de sus logros académicos.
    * **Qué comprobamos:** Registro de entregas completadas, cálculo de avance en la materia y mapa curricular con estado de resolución.
    * **Roles:** `admin`, `student`.

---

## ⚖️ Serie J — Juez Virtual de Punta a Punta (real UI, sin llamadas a la API desde el test)

Los specs `J0x` ejercitan el flujo completo del **Juez Virtual** sobre la interfaz real: el docente autoriza, el estudiante resuelve y el juez evalúa en Docker. No usan `fetch` dentro de `page.evaluate` para el flujo de negocio; la única excepción es la matrícula del estudiante (no existe UI de matrícula en la plataforma, ver `fixtures/authoring.fixture.mjs`).

| Spec | Ubicación | Qué comprobamos | Roles |
| :--- | :--- | :--- | :--- |
| `J01-template-to-judge-multilang.spec.mjs` | `01-easy/` | Admin homologa plantilla de Juez en Docker → docente publica ejercicio multi-lenguaje → estudiante resuelve en C++ y en Python (**AC**). | `admin`, `teacher`, `student` |
| `J02-teacher-authoring-pipeline.spec.mjs` | `01-easy/` | Pipeline de autoría en 3 pasos: identidad y enunciado → solución de referencia, generación de casos y **cálculo de salidas en Docker** → checklist sin bloqueantes y publicación. | `admin`, `teacher` |
| `J03-verdict-matrix-and-hidden-isolation.spec.mjs` | `02-medium/` | Matriz de veredictos **WA → RE → TLE → AC** enviada desde el editor del estudiante contra un ejercicio con **1 caso de ejemplo y 3 casos ocultos**, y **aislamiento**: ni el payload HTTP (`GET /api/v1/exercises/:id`, `POST /api/v1/evaluations`) ni el DOM del rol `student` exponen casos ocultos ni `reference_solution`, en cada veredicto. | `admin`, `teacher`, `student` |
| `J04-ast-security-barrier.spec.mjs` | `02-medium/` | Barrera de seguridad AST: el docente declara `os, subprocess, eval` como prohibidos y el estudiante que envía `import os; os.system('ls')` recibe **AST_BLOCKED** de inmediato (`execution_time_ms = 0`, sin casos evaluados, mensaje con la regla violada); la solución limpia obtiene **AC**. Un segundo ejercicio con la política "Restringir Módulos del Sistema Operativo" cubre el veredicto **AST_VIOLATION** y su presentación en la UI ("Regla de código", nunca "Sin evaluar"). Un tercer ejercicio con la regla personalizada `sorted` comprueba que la enumeración del docente decide el veredicto (**AST_BLOCKED** con `rule_id` `custom_rule:python:function:sorted`). | `admin`, `teacher`, `student` |
| `J05-curricular-unlock-and-exam-seed.spec.mjs` | `03-hard/` | Desbloqueo curricular encadenado: el docente crea Módulo 1 y Módulo 2 con prerrequisito, el estudiante ve el Módulo 2 **Bloqueado**, resuelve el ejercicio del Módulo 1 desde el editor (**AC**) y al volver al mapa el Módulo 2 queda **Disponible**. Luego dos estudiantes rinden el mismo examen con `per_student_seed` y reciben **entradas distintas** (semilla `sha256(ejercicio:estudiante)`), y ni el payload de evaluación ni el de la entrega exponen `expected_output`. | `admin`, `teacher`, `student` |
| `J06-extreme-utf8-and-bulk-import.spec.mjs` | `04-edge-cases/` | Importación masiva (dry-run y definitiva) del endpoint `/exercises/import`, publicación del ejercicio importado y lectura de las analíticas del curso; el estudiante resuelve desde el editor un caso con emojis, tildes, CJK y un entero de $10^{18}$ y obtiene **AC**. | `admin`, `teacher`, `student` |
| `J07-speedgrader-ephemeral-and-plagiarism.spec.mjs` | `05-badges-and-grading/` | Dos estudiantes envían desde la UI soluciones estructuralmente equivalentes; el docente abre SpeedGrader, ejecuta la prueba efímera (`run-ephemeral`), publica un comentario por línea, aplica un `override` de nota con motivo auditado (verificable en la revisión) y comprueba que el panel anti-plagio marca el par de entregas. | `admin`, `teacher`, `student` |
| `J08-persistent-ide-lifecycle.spec.mjs` | `02-medium/` | Ciclo completo de **IDE Persistente**: admin homologa plantilla `IDE_PERSISTENTE` con PostgreSQL en Template Studio → docente autoriza proyecto con `db_init_script` SQL y rúbrica al 100% → estudiante inicia su contenedor real (`http://localhost:<port>`, sin invocar `/api/v1/evaluations`), edita/agrega archivos y entrega el proyecto (`PENDING`) → reconexión idempotente de `_solv_exercise_migrations` → docente revisa el snapshot (`# === FILE: ... ===`) y califica con rúbrica y comentario general en SpeedGrader (`score` y `author_id` persistidos). | `admin`, `teacher`, `student` |

### Fixtures compartidos de la serie
* `fixtures/authoring.fixture.mjs` — creación de materia por UI de admin, matrícula del estudiante (API, no hay UI), escritura en el editor CodeMirror y `authorJudgeExercise()` (autoría completa de un ejercicio de juez en 3 pasos con caso de ejemplo, casos ocultos, reglas AST y publicación).
* `fixtures/auth-roles.fixture.mjs` — sesiones JWT aisladas por rol (`createAuthenticatedRoleSession`).
* `fixtures/forensic-logger.fixture.mjs` — capturas y traza forense ante cualquier fallo.
* `fixtures/api.fixture.mjs` — `apiJson()`/`apiRequest()` con reintento ante `429` (el backend limita por IP a 100 req/min con burst 20) y `openCurricularMap()`, que reintenta la carga del mapa curricular con el botón "Actualizar" cuando la petición es limitada por tasa.

### Comportamiento real verificado del motor (no supuesto)
* Las "Restricciones Específicas" que el docente registra con el botón **Prohibir** se guardan en `ast_rules.custom_rules` y **el pre-chequeo las evalúa** (gap **GOB-SEGURO-02** cerrado): la comparación es por identificador completo y descarta comentarios y literales, de modo que `is_sorted` no dispara la prohibición de `sorted` y un `sorted()` en un comentario no bloquea. La infracción se reporta como **AST_BLOCKED** con `actual_json.rule_id` (`custom_rule:<lenguaje>:<tipo>:<patrón>`).
* El veredicto **AST_BLOCKED** también lo emite el pre-chequeo Semgrep propio de la plataforma (`internal/infrastructure/semgrep/rules`), que corre **antes** que las reglas personalizadas y detiene `import os`, `import subprocess`, `eval`, `exec` y `__import__` (`execution_time_ms = 0` y ningún caso de prueba evaluado).
* La política **"Restringir Módulos del Sistema Operativo"** se aplica en el filtro regex del analizador estático y produce un veredicto distinto: **AST_VIOLATION**, que la UI presenta con la misma semántica de bloqueo que `AST_BLOCKED` ("Regla de código", icono de escudo), nunca como "Sin evaluar" (gap **GOB-SEGURO-03** cerrado).
* El rol `student` **nunca** recibe la salida esperada de los casos generados por semilla: el backend la conserva en el snapshot autoritativo `evaluation_generated_cases` y la entrega del estudiante se persiste con ese snapshot, no con el payload que envía el cliente.
* El límite de tiempo del juez se aplica al ciclo completo del contenedor (arranque + ejecución): con 1000 ms la compilación de C++ compite con el arranque de Docker y el caso muere por **TLE** de forma intermitente. Los specs que compilan fijan 5000 ms.

---

## 🚀 Comandos de Ejecución

> Todos los comandos usan `E2E_SLOW_MO=3000` (3 segundos por acción) para poder seguir el flujo con los ojos.
> Bajar el número acelera la corrida: `E2E_SLOW_MO=800`, `E2E_SLOW_MO=0` (sin pausas).

### 1. Suite Completa (modo visual, lento)
```bash
# Toda la suite en modo visual, 3000 ms por acción (default del script watch)
cd frontend && npm run test:e2e:flows:watch

# Misma suite ajustando el ritmo a mano
cd frontend && E2E_SLOW_MO=3000 npx playwright test --config=playwright.e2e.config.mjs --headed
```

### 2. Ejecutar por Categoría Completa
```bash
# Flujo Core Integral
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/01-core-lifecycle.spec.mjs --config=playwright.e2e.config.mjs --headed

# Categoría 01 (Fáciles)
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/01-easy/ --config=playwright.e2e.config.mjs --headed

# Categoría 02 (Medios)
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/02-medium/ --config=playwright.e2e.config.mjs --headed

# Categoría 03 (Difíciles)
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/03-hard/ --config=playwright.e2e.config.mjs --headed

# Categoría 04 (Casos Borde)
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/04-edge-cases/ --config=playwright.e2e.config.mjs --headed

# Categoría 05 (Badges y SpeedGrader)
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/05-badges-and-grading/ --config=playwright.e2e.config.mjs --headed
```

### 3. Ejecutar un Flujo Individual
```bash
# Core integral
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/01-core-lifecycle.spec.mjs --config=playwright.e2e.config.mjs --headed

# 01 — Catálogo, tecnologías y plantillas de juez
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/01-easy/01-catalog-and-templates.spec.mjs --config=playwright.e2e.config.mjs --headed

# 02 — Cursos, periodos académicos y matriculación
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/01-easy/02-course-management.spec.mjs --config=playwright.e2e.config.mjs --headed

# 03 — Autoría simple de ejercicios
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/01-easy/03-exercise-authoring-simple.spec.mjs --config=playwright.e2e.config.mjs --headed

# 04 — Veredictos del juez (AC, WA, RE, TLE)
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/02-medium/04-judge-verdicts.spec.mjs --config=playwright.e2e.config.mjs --headed

# 05 — Reglas AST y seguridad
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/02-medium/05-ast-rules-security.spec.mjs --config=playwright.e2e.config.mjs --headed

# 06 — Módulos curriculares y prerrequisitos
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/02-medium/06-curricular-modules.spec.mjs --config=playwright.e2e.config.mjs --headed

# 07 — Rúbricas y proyectos de IDE
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/02-medium/07-rubrics-and-ide-projects.spec.mjs --config=playwright.e2e.config.mjs --headed

# 08 — Concurrencia y límites de recursos
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/03-hard/08-concurrency-and-resource-limits.spec.mjs --config=playwright.e2e.config.mjs --headed

# 09 — Modo examen y semilla por estudiante
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/03-hard/09-exam-mode-seed.spec.mjs --config=playwright.e2e.config.mjs --headed

# 10 — Importación masiva y analítica
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/03-hard/10-bulk-import-and-analytics.spec.mjs --config=playwright.e2e.config.mjs --headed

# 11 — Entradas extremas y UTF-8
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/04-edge-cases/11-extreme-inputs-and-utf8.spec.mjs --config=playwright.e2e.config.mjs --headed

# 12 — Antiplagio y supervisión
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/04-edge-cases/12-anti-plagiarism-supervision.spec.mjs --config=playwright.e2e.config.mjs --headed

# 13 — SpeedGrader, auditoría y override
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/05-badges-and-grading/13-speedgrader-audit-and-override.spec.mjs --config=playwright.e2e.config.mjs --headed

# 14 — Gamificación y progreso del estudiante
cd frontend && E2E_SLOW_MO=3000 npx playwright test e2e/flows/05-badges-and-grading/14-student-gamification-and-progress.spec.mjs --config=playwright.e2e.config.mjs --headed

# J01 — Plantilla de Juez multi-lenguaje (C++ y Python)
cd frontend && npx playwright test e2e/flows/01-easy/J01-template-to-judge-multilang.spec.mjs --config=playwright.e2e.config.mjs

# J02 — Pipeline de autoría del docente en 3 pasos
cd frontend && npx playwright test e2e/flows/01-easy/J02-teacher-authoring-pipeline.spec.mjs --config=playwright.e2e.config.mjs

# J03 — Matriz de veredictos y aislamiento de casos ocultos
cd frontend && npx playwright test e2e/flows/02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs --config=playwright.e2e.config.mjs

# J04 — Barrera de seguridad AST
cd frontend && npx playwright test e2e/flows/02-medium/J04-ast-security-barrier.spec.mjs --config=playwright.e2e.config.mjs

# J05 — Desbloqueo curricular encadenado y examen con semilla por estudiante
cd frontend && npx playwright test e2e/flows/03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs --config=playwright.e2e.config.mjs

# J06 — Importación masiva, analíticas y UTF-8 extremo
cd frontend && npx playwright test e2e/flows/04-edge-cases/J06-extreme-utf8-and-bulk-import.spec.mjs --config=playwright.e2e.config.mjs

# J07 — SpeedGrader, prueba efímera, override auditado y anti-plagio
cd frontend && npx playwright test e2e/flows/05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs --config=playwright.e2e.config.mjs

# J08 — Ciclo de vida completo de IDE Persistente, BD satélite y Rúbrica en SpeedGrader
cd frontend && npx playwright test e2e/flows/02-medium/J08-persistent-ide-lifecycle.spec.mjs --config=playwright.e2e.config.mjs

# Serie J completa
cd frontend && npx playwright test e2e/flows/01-easy/J0*.spec.mjs e2e/flows/02-medium/J0*.spec.mjs e2e/flows/03-hard/J0*.spec.mjs e2e/flows/04-edge-cases/J0*.spec.mjs e2e/flows/05-badges-and-grading/J0*.spec.mjs --config=playwright.e2e.config.mjs
```

---

## 🔍 Diagnóstico Forense Automatizado
Cuando cualquier prueba encuentra un fallo:
* Genera **`frontend/e2e/test-results/forensic-last-failure.json`** con el stack trace, los pasos previos, peticiones HTTP fallidas y errores de consola.
* Guarda capturas de pantalla de cada ventana activa en **`frontend/e2e/test-results/failure-*.png`**.
