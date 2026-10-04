# Especificación del módulo JUEZ de ejercicios (motor de evaluación)

Cambio: juez-ejercicios. Propuesta fuente: decisiones congeladas del encargo de esta fase (sin archivo de propuesta previo).
Fecha: 2026-10-04. Idioma del artefacto: español institucional neutro.
Estado: especificación para diseño y tareas. No contiene código de aplicación.

Ubicación canónica: openspec/specs/juez-ejercicios/spec.md (convención openspec/config.yaml; contexto SDD Engram 185 sdd-init/solv_pg con persistencia both; índice de skills .atl/skill-registry.md). Deuda de ubicación cerrada el 2026-10-04: trasladado desde docs/SPEC/JUEZ_EJERCICIOS.md sin cambios de contenido salvo este encabezado.

## 1. Estado actual verificado

Cada afirmación indica ruta y símbolo verificados con búsqueda. La lista de búsquedas utilizadas figura al final de esta sección.

| Afirmación | Ruta y símbolo |
|---|---|
| La evaluación se ramifica en dos rutas: algoritmia y base de datos. | backend/internal/core/services/evaluation_service.go: Evaluate, evaluateAlgorithm, evaluateDatabase |
| La ruta de algoritmia aplica filtro AST por expresiones regulares y luego pre-chequeo Semgrep antes de ejecutar. | backend/internal/core/services/evaluation_service.go: evaluateAlgorithm; backend/internal/core/services/ast_analyzer.go: ValidateCode |
| La ruta de base de datos genera el JSON esperado mediante ejecución previa de la solución de referencia cuando el campo se encuentra vacío, y persiste el resultado. | backend/internal/core/services/evaluation_service.go: evaluateDatabase; backend/internal/core/domain/exercise.go: DatabaseConfig.ReferenceSolution, ExpectedJSON |
| La ejecución de cada caso inyecta la entrada por stdin en un contenedor efímero sin red y con límite de memoria. | backend/internal/infrastructure/docker/strategies/lang/base_runner.go: runContainerExecution, OpenStdin, StdinOnce, AttachStdin, NetworkMode none |
| La comparación actual normaliza ambos lados con TrimSpace y decide AC o WA por igualdad exacta. | backend/internal/infrastructure/docker/strategies/lang/base_runner.go: líneas de comparación con strings.TrimSpace |
| La factoría registra estrategias por lenguaje: python, c, cpp, c++, csharp, java, javascript (clave canónica: c++ → cpp; la normalización de alias rige en perfiles y run_metrics, §3). | backend/internal/infrastructure/docker/factory.go: NewPythonStrategy, NewCStrategy, NewCppStrategy, NewCSharpStrategy, NewJavaStrategy, NewJavaScriptStrategy |
| El analizador estático cubre familias por lenguaje con expresiones regulares: python (import, from, __import__, llamada), cpp y c (include, llamada), csharp (using, llamada), java (import, llamada), javascript (require, import from, llamada) y una rama genérica. | backend/internal/core/services/ast_analyzer.go: validatePython, validateCpp, validateCSharp, validateJava, validateJS, validateGeneric |
| Existen reglas Semgrep por lenguaje, incluida la de python con identificadores solv-python-forbidden-*. | backend/internal/infrastructure/semgrep/rules/python/forbidden.yaml; resto en backend/internal/infrastructure/semgrep/rules/cpp, csharp, java, javascript |
| El editor docente existe con modal y pasos: datos generales, técnico y reglas. Los pasos exponen blockNativeSort, blockSystemModules y forceRecursion como señales del formulario. | frontend/src/app/features/teacher/courses/exercise-editor/exercise-editor-modal.component.ts: blockNativeSort, blockSystemModules, forceRecursion; frontend/src/app/features/teacher/courses/exercise-editor/steps/step-rules/step-rules.component.ts |
| Existe un motor de generación de casos de borde. | backend/internal/core/services/fuzzing_engine.go: FuzzingEngine, GenerateTestCases |
| Las migraciones versionadas crean exercises con time_limit_ms, memory_limit_mb y config JSONB, y test_cases embebidos en el seed. | backend/migrations/00001_baseline.sql: CREATE TABLE exercises, columnas time_limit_ms, memory_limit_mb, db_config; backend/internal/infrastructure/database/postgres.go |
| Las columnas entrypoint, timeout_ms y sample_input existen para plantillas de laboratorio, no para ejercicios. | backend/internal/infrastructure/storage/postgres/admin_governance_repository.go: entrypoint, timeout_ms, sample_input; backend/internal/core/domain/admin_governance.go; backend/internal/core/domain/env_test_job.go |
| La ruta de evaluación no recibe ni utiliza template_id. El candado se cumple en el estado actual. | Búsqueda de template_id en backend/internal/core/services/evaluation_service.go y backend/internal/infrastructure/docker/strategies/lang/base_runner.go sin resultados |
| Las imágenes runner por lenguaje no están pineadas con digest: gcc:latest (c, cpp), mono:latest (csharp), eclipse-temurin:21-alpine sin digest (java), frente a python:3.11-slim y node:20-alpine con tag menor. Deuda de pinning registrada como estado actual; entra a sdd-design como ítem obligatorio (política de pinning, migración de sembrados y prohibición de :latest). | backend/internal/infrastructure/docker/strategies/lang/lang_c.go, lang_cpp.go (gcc:latest); lang_csharp.go (mono:latest); lang_java.go (eclipse-temurin:21-alpine); lang_python.go (python:3.11-slim); lang_javascript.go (node:20-alpine) |

DESVÍOS detectados (el código real contradice decisiones congeladas; no se reabren las decisiones, se registran y esta sección los deja marcados):

| Código | Decisión afectada | Descripción |
|---|---|---|
| DESVÍO-01 | Evaluación sin fail-fast | backend/internal/core/services/evaluation_service.go: evaluateAlgorithm retorna al primer caso con veredicto distinto de AC (bloque con FailedTestCase). La decisión exige evaluar todos los casos. |
| DESVÍO-02 | TrimSpace limitado a exact | backend/internal/infrastructure/docker/strategies/lang/base_runner.go aplica strings.TrimSpace a toda comparación. La decisión reserva TrimSpace al comparador exact. |
| DESVÍO-03 | Dos fases y veredictos completos | El dominio declara VerdictCE pero el ejecutor solo emite AC, WA, TLE y RE. No existen fase de build, ni MLE por OOM de cgroups, ni VE, ni CE emitido. Rutas: backend/internal/core/domain/exercise.go (veredictos), backend/internal/infrastructure/docker/strategies/lang/base_runner.go (veredictos emitidos). |

Ausencias esperadas (no son desvíos, son trabajo por especificar): registro de comparadores con comparator_id, override por caso, perfiles de lenguaje servidos por configuración, trabajo dry-run asíncrono con progreso, marca stale, autoguardado de expected con confirmación, referencia de algoritmia persistida (hoy solo existe ReferenceSolution para base de datos), tabla de métricas por caso y endpoint de perfiles con auditoría.

Búsquedas utilizadas como evidencia (todas con rg desde la raíz):

1. evaluateAlgorithm|evaluateDatabase en backend con filtro *.go
2. TrimSpace|stdin|Stdin en backend con filtro *runner*.go
3. python|javascript|cpp|java|csharp en backend con filtro *factory*.go
4. ForbiddenFunctions|ForbiddenImports|blockNativeSort|blockSystemModules|forceRecursion en backend y frontend
5. Expresiones Regexp|Compile en backend con filtro *ast*.go
6. Archivos con fuzz|forbidden|ast en backend
7. entrypoint|timeout_ms|sample_input|comparator|reference_solution|template_id en backend con filtro *.go
8. fail-fast|failFast|break en evaluation_service.go y base_runner.go
9. exercises|test_cases|comparator|reference en backend/migrations/*.sql y postgres.go
10. MLE|VE|VerdictCE|build en exercise.go y evaluation_service.go

## 2. Decisiones D-EJ-01 a D-EJ-08

| ID | Fundamento | Consecuencia sobre contratos |
|---|---|---|
| D-EJ-01 Comparadores como registro de estrategias | Un único punto de comparación evita lógicas dispersas por lenguaje y permite auditoría del criterio aplicado a cada ejercicio. | El ejercicio declara comparator_id; cada caso MAY incluir override. Los contratos de datos incluyen comparator_spec JSONB por ejercicio y override por caso. El contrato de ejecución registra comparator_id y parámetros por caso. |
| D-EJ-02 Comparadores v1: exact, unordered, float, custom | Cubren igualdad textual, equivalencia sin orden, tolerancia numérica y criterio docente arbitrario sin ampliar la superficie del motor. exact: TrimSpace en ambos lados más flag trim_lines. unordered: división por líneas o tokens, ignore_duplicates, ignore_case, comparación como multiconjunto. float: comparación token a token con modos absolute, relative, hybrid según la desigualdad con atol y rtol sobre el valor esperado; valores por defecto eps 1e-6, atol 1e-9, rtol 1e-6; rangos admitidos 1e-12 a 1e-2. custom: checker python3 del docente en imagen sidecar curada, sin red, solo lectura, con timeout; rutas input, expected y output en solo lectura; veredicto AC o WA más mensaje por stdout; fallo o timeout del checker = VE. | Los payloads de ejercicio y caso exponen el comparador y sus parámetros con validación de rangos. El contrato de veredictos por caso incluye VE. Los fixtures congelados de esta especificación son obligatorios. |
| D-EJ-03 Ejecución en dos fases sin fail-fast | Separar build de run distingue errores de compilación de errores de ejecución y permite reutilizar artefactos. Evaluar todos los casos entrega diagnóstico completo al docente. | Contratos de trabajos con fase build (comando, timeout y memoria por lenguaje desde configuración servida; fallo = CE con stderr sanitizado sin rutas internas; artefacto cacheado por envío) y fase run por caso (timeout del docente dentro de rango; veredictos AC, WA, TLE, RE, MLE por OOM de cgroups, CE, VE, AST_BLOCKED). El resultado agrega todos los casos; no existe detención temprana. |
| D-EJ-04 Límites por lenguaje como configuración servida | Los límites dependen del ecosistema de cada lenguaje y deben ajustarse sin despliegue. | Valores iniciales v0: python 2000/256, javascript 2000/256, cpp 1000/128, c 1000/128, csharp 2500/256, java 3000/512 (ms/MB). Rangos 100 a 10000 ms y 64 a 1024 MB; fuera de rango = 400. Todo payload incluye memoria explícita. Un único validador de dominio aplica en todas las fronteras. |
| D-EJ-05 Referencia obligatoria y ciclo de publicación (v1: modo stdin) | La referencia permite dry-run, autoguardado asistido y recalificación con el mismo criterio. | En v1 el ciclo completo (referencia obligatoria, dry-run asíncrono, marca stale) rige para el modo stdin. El modo función hereda el mismo ciclo con su harness en EX2; sin publicación de ejercicios en modo función hasta entonces. Editar casos, límites, comparador o referencia marca stale y bloquea la publicación. El expected propuesto por autoguardado requiere confirmación docente. La referencia se persiste para recalificación. |
| D-EJ-06 Restricciones fijas y textos por lenguaje | La seguridad no es configurable por el docente; la presentación sí varía por lenguaje. | Seguridad fija no editable más blockNativeSort (ForbiddenFunctions por lenguaje) y blockSystemModules (ForbiddenImports por lenguaje) cableados. forceRecursion queda fuera de la interfaz. Los textos dinámicos se resuelven por lenguaje. |
| D-EJ-07 Candado juez: template_id ausente o ignorado | El algoritmo del juez no depende de plantillas de laboratorio. | Ningún contrato de evaluación acepta template_id. Se exige un test que afirme su ausencia o su ignorancia en el algoritmo. |
| D-EJ-08 Fixtures congelados | Fijan el comportamiento observable de los comparadores. | 0.30000000004 frente a 0.3 = AC en los tres modos float. 2.000002 frente a 2.0 = WA en absolute y AC en relative y hybrid. Permutación de líneas = AC en unordered. Misma permutación sin un duplicado = WA con ignore_duplicates en false y AC en true. |

## 3. Contratos de datos y backend

Requisitos de forma: las migraciones usan goose versionadas. Los estados y códigos de error utilizan valores máquina estables. Las validaciones de dominio devuelven 400 ante rango o formato inválido y 409 ante publicación bloqueada por stale o por referencia faltante.

### 3.1 Datos

| Elemento | Requisito |
|---|---|
| comparator_spec como columna explícita por ejercicio | El sistema MUST almacenar el comparador del ejercicio en columna dedicada comparator_spec JSONB NOT NULL con default {"id":"exact"} (identificador más parámetros), no solo dentro de config. Cada caso MAY declarar override propio que prevalece sobre el del ejercicio. |
| override por caso | Cada caso MAY declarar un comparador propio que prevalece sobre el del ejercicio. |
| reference_solution | El sistema MUST persistir la solución de referencia de algoritmia y de base de datos para dry-run y recalificación. |
| Perfiles de lenguaje | El sistema MUST servir build (comando, timeout, memoria) y run (timeout y memoria por defecto) por lenguaje desde configuración versionada con imagen fijada. La clave de lenguaje es canónica (c++ → cpp, c# → csharp); los perfiles MUST rechazar alias no canónicos. |
| run_metrics | El sistema MUST registrar por caso: lenguaje en clave canónica (c++ → cpp), digest de imagen, duration_ms y veredicto. |

### 3.2 Trabajos y veredictos

El sistema MUST modelar el dry-run como trabajo asíncrono con estados: queued, running, done y failed, con progreso por caso (índice, total, veredicto parcial). El sistema MUST modelar la evaluación con resultado por caso: índice, veredicto (AC, WA, TLE, RE, MLE, CE, VE, AST_BLOCKED), duration_ms y mensaje. El sistema MUST definir códigos de error máquina para: comparador desconocido, parámetro fuera de rango, timeout fuera de rango, memoria fuera de rango, referencia ausente, publicación bloqueada por stale y checker inválido.

Escenarios:

- GIVEN un ejercicio con comparador float en modo relative y tolerancia dentro de rango, WHEN se publica, THEN el sistema lo acepta.
- GIVEN un timeout de caso fuera del rango 100 a 10000 ms, WHEN se guarda, THEN el sistema responde 400 con código máquina.
- GIVEN un ejercicio marcado stale, WHEN se intenta publicar, THEN el sistema responde 409 con código máquina.
- GIVEN un envío con todos los casos evaluados, WHEN se consulta el resultado, THEN todos los casos presentan veredicto, incluido más de un fallo si existe.

### 3.3 Endpoints (forma de los payloads, sin código)

Creación y edición de ejercicio: incluye comparator (identificador y parámetros), casos con override opcional, límites con tiempo y memoria explícitos, y referencia según modo. Respuesta de validación: 400 con código máquina y campo afectado. Publicación: 409 cuando falta referencia o existe marca stale. Dry-run: respuesta 202 con identificador de trabajo; consulta de trabajo con estado y progreso por caso. Evaluación: respuesta con veredicto global y lista de veredictos por caso con métricas. Perfiles de lenguaje: lectura pública para formularios y escritura restringida a administración con registro de auditoría.

## 4. Contratos frontend

El sistema MUST ofrecer el asistente completo en modo stdin: paso de datos generales, paso técnico (lenguaje, límites, comparador con parámetros visibles), paso de casos y paso de referencia con editor. El modo función queda definido a nivel de contrato en esta spec, pero su asistente y publicación quedan bloqueados hasta EX2 (harness de función versionado). El selector de comparador MUST mostrar los parámetros editables del comparador elegido con sus rangos. El panel dry-run MUST mostrar progreso por caso y el badge stale cuando casos, límites, comparador o referencia cambien tras el último dry-run válido. El badge stale MUST bloquear la acción de publicar hasta un nuevo dry-run válido o confirmación docente según corresponda.

Escenarios:

- GIVEN un docente que cambia el comparador tras un dry-run válido, WHEN vuelve al panel, THEN observa el badge stale y la publicación deshabilitada.
- GIVEN un ejercicio en modo función, WHEN intenta publicar en v1, THEN el sistema lo impide por estar fuera del alcance de v1 (harness pendiente en EX2); el ciclo de referencia que hereda se rige por D-EJ-05.
- GIVEN un dry-run en curso, WHEN avanza cada caso, THEN el progreso por caso se actualiza sin recargar la vista.

Tabla de textos familia EJ (el ID del deck es el ID i18n, antes de cualquier código):

| ID i18n | Texto |
|---|---|
| EJ-CMP-TITLE | Comparador de salida |
| EJ-CMP-EXACT | Coincidencia exacta |
| EJ-CMP-UNORDERED | Sin orden |
| EJ-CMP-FLOAT | Tolerancia numérica |
| EJ-CMP-CUSTOM | Checker del docente |
| EJ-CMP-UNKNOWN | Comparador desconocido |
| EJ-CHECKER-INVALID | Checker inválido |
| EJ-CMP-PARAMS | Parámetros del comparador |
| EJ-REF-TITLE | Solución de referencia |
| EJ-REF-HINT | Se utiliza para el dry-run y la recalificación |
| EJ-DRY-RUN | Probar con referencia |
| EJ-DRY-PROGRESS | Caso en curso |
| EJ-STALE | Requiere nueva comprobación |
| EJ-PUB-BLOCKED | La publicación se encuentra bloqueada |
| EJ-PUB-REF | Falta la solución de referencia |
| EJ-ERR-RANGE | Valor fuera del rango admitido: {campo} |
| EJ-TELEMETRY | Métricas de ejecución |

EJ-ERR-RANGE MUST interpolar el campo afectado en {campo}.

## 5. Calibración y telemetría

El sistema MUST registrar run_metrics por caso con lenguaje, digest de imagen, duration_ms y veredicto. El sistema MUST calcular el p95 de duration_ms sobre veredictos AC agrupado por lenguaje en ventana configurable. El sistema SHOULD proponer recalibración cuando el p95 se desvíe más o menos 30 por ciento respecto de la línea base o cuando cambie la imagen fijada. El responsable es la administración mediante el endpoint de perfiles; toda modificación queda en auditoría con autor, valores anterior y nuevo, y motivo.

Escenarios:

- GIVEN métricas suficientes por lenguaje, WHEN se calcula el p95 sobre AC, THEN el valor se encuentra disponible para el panel de administración.
- GIVEN un cambio de imagen fijada, WHEN se registra, THEN el sistema marca la línea base como pendiente de revisión.
- GIVEN una modificación de perfil por administración, WHEN se guarda, THEN queda un asiento de auditoría.

## 6. Aceptación ejecutable

Comandos con salida esperada, a ejecutar antes del código donde aplique. Ninguno genera código de aplicación.

| N.º | Comando | Salida esperada |
|---|---|---|
| 1 | Conjunto de tests del comparador exact como función pura con TrimSpace y flag trim_lines | Todos los tests pasan |
| 2 | Conjunto de tests del comparador unordered con fixtures de permutación y duplicados | Todos los tests pasan |
| 3 | Conjunto de tests del comparador float con fixtures 0.30000000004 frente a 0.3 y 2.000002 frente a 2.0 | AC en tres modos para el primer par; WA en absolute y AC en relative y hybrid para el segundo |
| 4 | Contratos 400 ante timeout, memoria o tolerancia fuera de rango | 400 con código máquina y campo afectado |
| 5 | Búsqueda de detención temprana en el servicio de evaluación (bucle que retorna al primer fallo) | Sin resultados tras la corrección |
| 6 | Búsqueda de TrimSpace fuera del comparador exact en el ejecutor | Sin resultados tras la corrección |
| 7 | Test que afirma template_id ausente o ignorado en el algoritmo | Todos los tests pasan |
| 8 | Conjunto de tests de backend y compilación de frontend según openspec/config.yaml | Todos los tests pasan y la compilación termina |
| 9 | Extracción i18n: todos los ID del deck EJ-* de §4 existen como claves | Sin ID faltante ni sobrante respecto de la tabla §4 |
| 10 | Pinning: grep de imágenes runner sin pinear vacío, o perfiles con digest fijado por lenguaje | Cero imágenes `:latest` o sin digest en runners del juez |

## 7. Fuera de alcance

Quedan fuera de este cambio: harness de modo función versionado y publicación de ejercicios en modo función (el modo función hereda referencia, dry-run y marca stale con su harness en EX2), casos de borde con oráculo, importación CSV sólida (corresponde al incremento EX2), interfaz del checker custom más allá de su contrato, capa de acceso a datos directa, vistas del docente fuera del editor y dry-run, y plantillas de laboratorio.

## 8. Traducción de commits y regla stop para el incremento de código futuro

Los commits siguen Conventional Commits con cuerpo Qué, Por qué, Cómo e Impacto, sin etiquetas internas de planificación y sin los términos indicados en gobernanza. Ejemplos de ámbitos y mensajes (forma, no contenido cerrado): tipo feat con ámbito judge para el registro de comparadores; tipo feat con ámbito judge para la ejecución en dos fases; tipo feat con ámbito exercises para referencia obligatoria y marca stale; tipo feat con ámbito profiles para perfiles servidos y auditoría; tipo test con ámbito judge para fixtures congelados.

Regla stop: el incremento de código futuro se detiene y vuelve a especificación si aparece código real que contradiga las decisiones D-EJ-01 a D-EJ-08, si falta el test del candado template_id, si algún veredicto por caso queda sin registrar en run_metrics, o si la publicación resulta posible con marca stale o sin referencia.
