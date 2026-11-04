# Propuesta de vocabulario de dominio de SOLV

> **Estado:** propuesta, pendiente de aprobación. No se aplicó ningún cambio de código, etiqueta, identificador, tabla ni endpoint.
>
> **Origen:** `docs/GOBERNANZA/AUDITORIA_TERMINOLOGIA.md` (auditoría de solo lectura). Este documento convierte sus hallazgos en definiciones, contrastes y decisiones explícitas. La auditoría original queda intacta.
>
> **Fecha:** noviembre de 2026.

---

## 0. Alcance, método y límites

### 0.1 Qué resuelve este documento

Define el vocabulario de dominio de SOLV en tres capas separadas —visible, conceptual y técnica—, contrasta cada concepto con los que se le parecen y deja por escrito qué decisiones faltan. No propone renombrar código: propone qué palabra usar para cada cosa y por qué.

### 0.2 Cómo se investigó

No se dedujeron relaciones por el nombre de las clases. Cada afirmación sale de seguir el dato:

- claves foráneas y columnas en `backend/migrations/00001_baseline.sql` y migraciones posteriores;
- consultas SQL reales en `backend/internal/infrastructure/storage/postgres/`;
- DTO, servicios de dominio y handlers de `backend/internal/core/` y `backend/internal/delivery/http/`;
- modelos, constantes, pipes y plantillas del frontend;
- ADR, contratos y manuales de `docs/`.

Cuando el código no alcanza para decidir, se marca **pendiente de decisión** y no se inventa un significado.

### 0.3 Las cuatro capas

| Capa | Qué es | Dónde vive | Idioma |
| :--- | :--- | :--- | :--- |
| **Visible** | Lo que lee una persona en la interfaz | plantillas Angular, mensajes de error de cara al usuario | español |
| **Conceptual** | Cómo se habla del objeto en diseño, ADR y documentación | `docs/`, revisiones, conversaciones | español |
| **Técnico** | Identificador de código, DTO, ruta o tipo | Go, TypeScript, endpoints | inglés |
| **Almacenado** | Valor persistido o transmitido por API | PostgreSQL, JSON | códigos en mayúsculas o inglés |

La cuarta capa se agrega porque en SOLV no siempre coincide con la técnica: `JUEZ_EFIMERO` es el valor guardado, `EnvironmentType` el tipo de código y «Entorno de Evaluación» la etiqueta visible.

### 0.4 Niveles de certeza usados

- **Confirmado:** el código o la migración lo demuestran y se cita archivo y línea.
- **Probable:** la evidencia es consistente pero no hay una ruta de ejecución que lo cierre (por ejemplo, no se pudo consultar la base).
- **Pendiente de decisión:** depende de una elección humana de producto, no de una lectura del repositorio.

### 0.5 Límites

- El backend y PostgreSQL no estaban disponibles durante el relevamiento: no se consultaron valores reales de `target_environment`, `status` ni `verdict`. Las listas de valores provienen de constantes, validaciones y restricciones `CHECK`.
- No se ejecutaron flujos con usuarios reales ni se midió qué interpreta una persona frente a cada pantalla.
- La memoria del proyecto de grado se entrega como PDF y no forma parte de esta revisión.
- No se modificaron `AGENTS.md`, skills, código ni etiquetas; no se implementó ningún gate; no se crearon alias.

---

## 1. Principios de nomenclatura

**P1. El concepto manda sobre la pantalla.** Un nombre se elige por lo que el objeto *es* en el dominio, no por la pantalla donde aparece. Si dos pantallas muestran el mismo objeto, comparten palabra.

**P2. Una palabra, un concepto; un concepto, una palabra visible.** Si «curso» nombra la materia en una pantalla y el aula en otra, deja de servir para ambas. La polisemia se resuelve con palabras distintas, no con contexto implícito.

**P3. Las tres capas no se mezclan.** El nombre técnico puede seguir en inglés (`environment`, `template`, `submission`). El visible no mezcla idiomas ni muestra códigos. El conceptual usa el español del dominio y es el que se cita en ADR y documentación.

**P4. El código técnico no se traduce ni se muestra como texto de interfaz.** `JUEZ_EFIMERO`, `AST_BLOCKED` o `oom_killed` son valores de contrato. Cuando deben verse, se muestran como dato técnico (`machineData`, `<code>`), nunca como etiqueta humana.

**P5. La modalidad califica la configuración, no la instancia.** Una plantilla, un modelo o una solicitud declaran para qué entornos sirven. Un ejercicio elige uno de los dos modos concretos. Un laboratorio en ejecución no es «dual»: está abierto en un modo.

**P6. «Entorno» nunca va solo en la interfaz.** Siempre «entorno de desarrollo», «entorno de evaluación», «entorno de revisión» o «tipo de entorno». La palabra suelta es la que hoy produce ambigüedad entre plantilla, modalidad e instancia.

**P7. El sinónimo de prosa no es nombre de cosa.** «Práctica» puede aparecer en una frase explicativa («la práctica de la sesión»); no puede ser el título de una página, una opción de menú ni el texto de un botón.

**P8. No se crean alias nuevos.** Los alias existentes (`JUEZ_VIRTUAL` para `JUEZ_EFIMERO`, `approved` para `APROBADA`) se declaran por escrito y, si se retiran, se retiran con una migración y una ventana de compatibilidad. Agregar un tercer nombre nunca es la solución.

**P9. El estado se nombra con el vocabulario del badge.** Si un estado tiene etiqueta canónica en `status-badge`, esa etiqueta es la única. No se inventan variantes por pantalla («En reposo», «Pausada», «Detenido»).

**P10. Lo que no está definido se marca, no se rellena.** Ante un concepto sin definición en el repositorio, la propuesta deja la decisión abierta en la sección 7 en lugar de fijar un significado por intuición.

---

## 2. Glosario conceptual

Cada ficha describe un concepto, no una clase. La columna de categoría distingue **entidad** (algo con identidad propia y ciclo de vida), **configuración** (una receta que se reutiliza), **categoría** (una agrupación), **instancia** (un objeto efímero creado a partir de una configuración) y **clasificación** (un valor de un vocabulario cerrado).

### 2.1 Entidades académicas y pedagógicas

#### 2.1.1 Ejercicio

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Ejercicio |
| Nombre visible recomendado | **Ejercicio** |
| Identificador técnico | `Exercise`, tabla `exercises`, rutas `/api/v1/.../exercises` |
| Categoría | Entidad |
| Definición funcional | Unidad de trabajo evaluable que un docente publica en una materia: tiene enunciado, tipo, modalidad de entorno, plantilla asociada y una forma de corrección (casos de prueba o revisión con rúbrica). Es lo que el alumno resuelve y lo que genera entregas. |
| Qué no debe confundirse | No es la **práctica** de una sesión (es el mismo objeto visto desde el aula), no es el **laboratorio** (el entorno en ejecución), no es la **plantilla** (la receta técnica) y no es una **actividad** genérica. |
| Sinónimos permitidos en prosa | «práctica» cuando el texto habla de lo que se resuelve en clase («la práctica de la sesión»); «consigna» para el enunciado |
| Evitar en interfaz | «Laboratorio», «Actividad», «Práctica» en títulos, menús, botones y filtros |
| Evidencia | `backend/internal/core/domain/exercise.go:278-306` (estructura), `:17-20` (tipos), `:25-26` (modalidades), `:71-79` (veredictos); `backend/migrations/00001_baseline.sql:205`; UI contradictoria en `frontend/src/app/features/teacher/exercises/teacher-exercises.component.html:5-6,18` |
| Certeza | Confirmado en dominio y API; **pendiente de decisión** el reemplazo de las etiquetas visibles |

#### 2.1.2 Materia

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Materia |
| Nombre visible recomendado | **Materia** |
| Identificador técnico | `Subject`, tabla `subjects`, rutas `/api/v1/teacher/courses` y `/api/v1/admin/courses` |
| Categoría | Entidad |
| Definición funcional | Asignatura que se dicta en un periodo académico, con un docente responsable, un código propio, estudiantes inscriptos y una plantilla por defecto. Es la unidad a la que pertenecen ejercicios, aulas, módulos y entornos. |
| Qué no debe confundirse | No es el **periodo académico** (el rango de fechas que la contiene), no es el **módulo curricular** (una unidad interna de su programa) y no es el **curso de Google Classroom** (un objeto externo que se referencia con un identificador). |
| Sinónimos permitidos en prosa | «asignatura» en textos introductorios |
| Evitar en interfaz | «Curso» como nombre de la materia |
| Evidencia | `backend/internal/core/domain/academic.go:8-24`; `backend/internal/infrastructure/storage/postgres/subject_repository.go:21-22`; pantalla que mezcla ambos nombres: `frontend/src/app/features/admin/courses/admin-courses.component.html:17` y `.../course-create-modal/course-create-modal.component.html:16` |
| Certeza | Confirmado; la API conserva `courses` como nombre técnico heredado |

#### 2.1.3 Periodo académico

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Periodo académico |
| Nombre visible recomendado | **Periodo académico** |
| Identificador técnico | `AcademicPeriod`, tabla `academic_periods`, rutas `/api/v1/admin/academic-periods` |
| Categoría | Entidad |
| Definición funcional | Rango de fechas con nombre y código (semestre, trimestre, año) que agrupa materias y habilita su archivo al cerrar. Una materia pertenece a un periodo. |
| Qué no debe confundirse | No es la **materia** ni el **aula**; no marca el inicio de una clase. |
| Sinónimos permitidos en prosa | «periodo» cuando el contexto ya lo nombró |
| Evitar en interfaz | «Curso» para el periodo |
| Evidencia | `backend/migrations/00001_baseline.sql:349`; `frontend/src/app/features/admin/services/admin-courses.service.ts:8-19` |
| Certeza | Confirmado |

#### 2.1.4 Módulo curricular

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Módulo curricular |
| Nombre visible recomendado | **Módulo** |
| Identificador técnico | `CourseModule`, tabla `course_modules`, rutas `/api/v1/teacher/courses/{id}/modules` |
| Categoría | Entidad |
| Definición funcional | Unidad ordenada del programa de una materia, con nota de aprobación y prerrequisitos entre módulos. Agrupa ejercicios y determina qué puede resolver el alumno según su avance. |
| Qué no debe confundirse | No es la materia: la columna se llama `subject_id` aunque el tipo se llame `CourseModule` y el mapa estudiantil exponga `course_id` con el identificador de la materia. |
| Sinónimos permitidos en prosa | «módulo del programa» |
| Evitar en interfaz | «Curso» o «materia» para el módulo |
| Evidencia | `backend/internal/core/domain/course_module.go:25-37` (módulo), `:66-71` (`CourseCurricularMap.CourseID`); rutas en `backend/internal/delivery/http/router.go:631-640` |
| Certeza | Confirmado el objeto; **probable** que el nombre `CourseModule` sea deuda histórica (no hay tabla `courses`) |

#### 2.1.5 Inscripción

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Inscripción |
| Nombre visible recomendado | **Inscripción** (o «alumnos inscriptos» en listados) |
| Identificador técnico | `Enrollment`, tabla `enrollments` |
| Categoría | Entidad de relación |
| Definición funcional | Vínculo entre un estudiante y una materia, con fecha de alta. Habilita al alumno a ver ejercicios, abrir laboratorios y entrar al aula de esa materia. |
| Qué no debe confundirse | No es la **presencia en el aula** (`sin_empezar`, `trabajando`, `ausente`), que es un estado dentro de una sesión; no es «matrícula» de un periodo completo. |
| Sinónimos permitidos en prosa | «inscripto», «cursando» |
| Evitar en interfaz | «Sesión» o «asistencia» para la inscripción |
| Evidencia | `backend/internal/core/domain/academic.go:26-33`; `class_session.go:78-86` (`Enrolled`/`Present` resueltos por el servidor) |
| Certeza | Confirmado |

#### 2.1.6 Entrega

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Entrega |
| Nombre visible recomendado | **Entrega** |
| Identificador técnico | `Submission`, tabla `submissions`, rutas `/api/v1/.../submissions` |
| Categoría | Entidad |
| Definición funcional | Envío de un alumno para un ejercicio: código o snapshot, veredicto del juez cuando corresponde, resultado de análisis y estado de revisión del docente. Es el hecho evaluable y calificable. |
| Qué no debe confundirse | No es el **veredicto** (un atributo de la entrega), no es el **estado de revisión** (otro atributo, de naturaleza distinta) y no es una **evaluación** del docente (la vista que las agrupa). |
| Sinónimos permitidos en prosa | «envío», «trabajo entregado» |
| Evitar en interfaz | «Intento» cuando se refiere a la entrega ya registrada; «evaluación» para la entrega individual |
| Evidencia | `backend/internal/core/domain/academic.go:31-53`; `backend/internal/core/domain/review_workflow.go:5-27` |
| Certeza | Confirmado |

#### 2.1.7 Aula

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Aula |
| Nombre visible recomendado | **Aula** |
| Identificador técnico | `ClassSession`, tabla `class_sessions`, rutas `/api/v1/teacher/sessions` |
| Categoría | Entidad |
| Definición funcional | Apertura acotada de una materia durante una clase: nace abierta con una duración y muere cerrada. Mientras está abierta reúne la presencia de los alumnos, el ejercicio de la sesión y los avances. |
| Qué no debe confundirse | No es la **sesión de autenticación** (el inicio de sesión con Google y el token JWT), no es la **vida del laboratorio** («entre sesiones» en la copia) y no es la **materia**: una materia puede tener un aula abierta a la vez. |
| Sinónimos permitidos en prosa | «clase» cuando se habla del encuentro; «sesión de clase» en documentación conceptual |
| Evitar en interfaz | «Sesión» sola para el aula |
| Evidencia | `backend/internal/core/domain/class_session.go:9-12` (estados), `:16-31` (estados del alumno), `:78-100` (estructura); `ErrSessionAlreadyOpen` en `:44` |
| Certeza | Confirmado |

### 2.2 Configuración de entornos

#### 2.2.1 Plantilla

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Plantilla |
| Nombre visible recomendado | **Plantilla** (con el calificador «de entorno» solo si hace falta desambiguar) |
| Identificador técnico | `LabTemplate`, tabla `lab_templates`, rutas `/api/v1/admin/templates` |
| Categoría | Configuración |
| Definición funcional | Receta ejecutable de un entorno: imagen base o composición de tecnologías, script de arranque, memoria base, modalidad declarada y estado de auditoría. Es lo que un ejercicio referencia (`exercises.template_id`) y lo que un laboratorio instancia (`environments.template_id`). |
| Qué no debe confundirse | No es el **modelo del catálogo** (una copia curada y categorizada, pensada para reutilizar), no es el **laboratorio** (la instancia en ejecución) y no es «Docker»: la imagen es un detalle de implementación. |
| Sinónimos permitidos en prosa | «plantilla de entorno» en documentación de infraestructura |
| Evitar en interfaz | «Plantilla Docker», «Entorno» a secas, «Imagen» |
| Evidencia | `backend/internal/core/domain/environment.go:173-183`; `backend/migrations/00001_baseline.sql:68-101`; estados y modalidad en la misma tabla; `TemplateSupportsEnvironment` en `backend/internal/core/domain/admin_governance.go:58-63` |
| Certeza | Confirmado |

#### 2.2.2 Modelo del catálogo

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Modelo del catálogo |
| Nombre visible recomendado | **Modelo del catálogo** (o «Modelo» dentro de la pantalla del catálogo) |
| Identificador técnico | `TemplateModelItemDTO`, tabla `template_models`, rutas `/api/v1/admin/template-models` |
| Categoría | Configuración |
| Definición funcional | Copia curada y categorizada de una plantilla aprobada, con título propio, imagen, memoria y modalidad, que el administrador publica para que sea reutilizable (contador de uso incluido). Se crea promoviendo una plantilla y conserva el vínculo con su origen. |
| Qué no debe confundirse | No es la **plantilla** que ejecuta el ejercicio, no es una **categoría** (la agrupación) y no es un **modelo de lenguaje** de IA (`ai_settings`, `ai_copilot`), que comparte la palabra en otro contexto del producto. |
| Sinónimos permitidos en prosa | «modelo oficial», «modelo preconfigurado» (así lo dice el comentario del DTO) |
| Evitar en interfaz | «Modelo» sin calificador fuera de la pantalla del catálogo |
| Evidencia | `backend/migrations/00001_baseline.sql:152-171` (`source_template_id`, `category_id`, `target_environment`); `backend/internal/core/domain/admin_governance.go:504-521`; promoción desde plantilla en `backend/internal/delivery/http/admin_templates_handler.go:325` |
| Certeza | Confirmado; **probable** que la relación con la plantilla de origen no se muestre hoy en la interfaz |

#### 2.2.3 Tecnología (componente técnico)

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Componente técnico |
| Nombre visible recomendado | **Tecnología** (ítem del catálogo); **Catálogo de Tecnologías** (la pantalla) |
| Identificador técnico | `TemplateComponent`, tabla `template_components`, rutas `/api/v1/admin/template-components` (con alias `tech-components` y `components`) |
| Categoría | Configuración |
| Definición funcional | Pieza reutilizable del catálogo que el constructor combina para armar una plantilla: declara capa (`runtime`, `addon`, `database`), modo de instalación, paquetes, binarios verificables y huella de memoria. |
| Qué no debe confundirse | No es una **categoría** (agrupa modelos) ni una **subcategoría** (`backend`, `web`, `framework`, `toolkit`, `relational`, `nosql`, que clasifica dentro de una capa). «Lenguaje» y «framework» son valores de esa subcategoría, no conceptos del catálogo. «Herramienta» designa el binario verificado (`tools_provided`), no la pieza completa. |
| Sinónimos permitidos en prosa | «pieza del catálogo», «componente» en documentación técnica |
| Evitar en interfaz | «Herramienta» para el ítem del catálogo; «Componente» y «Tecnología» en la misma pantalla |
| Evidencia | `backend/internal/core/domain/template_component.go:14-22` (capas), `:50-74` (estructura), `:55` (subcategorías); tres prefijos de ruta en `backend/internal/delivery/http/router.go:251-258`; etiquetas visibles en `frontend/.../model-library.component.html:35,116` y `.../catalog-tree.component.html:8,48` |
| Certeza | Confirmado el objeto y la etiqueta dominante; **pendiente de decisión** el nombre visible definitivo |

#### 2.2.4 Categoría de plantilla

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Categoría |
| Nombre visible recomendado | **Categoría** |
| Identificador técnico | `TemplateCategory`, tabla `template_categories` |
| Categoría | Clasificación con identidad |
| Definición funcional | Agrupación nombrada de modelos dentro de un tenant, con orden propio. Organiza el catálogo; no cambia cómo se ejecuta nada. |
| Qué no debe confundirse | No es la **capa** del componente (`runtime`/`addon`/`database`) ni su **subcategoría**; no es un **tag** libre de ejercicio. |
| Sinónimos permitidos en prosa | «grupo del catálogo» |
| Evitar en interfaz | «Capa», «Tipo» o «Familia» para la categoría |
| Evidencia | `backend/migrations/00001_baseline.sql:56-63`; uso en `template_models.category_id` (`:152-171`) |
| Certeza | Confirmado |

#### 2.2.5 Solicitud de entorno

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Solicitud de entorno |
| Nombre visible recomendado | **Solicitud de entorno** (menú y página con la misma palabra) |
| Identificador técnico | `EnvironmentRequest`, tabla `environment_requests`, rutas `/api/v1/teacher/environment-requests` |
| Categoría | Entidad de flujo |
| Definición funcional | Pedido de un docente para que el administrador cree un entorno: declara título, descripción, usos previstos, piezas del catálogo y materias interesadas. El sistema deriva la modalidad, detecta duplicados por huella y, al aprobarse, la resuelve en una plantilla. |
| Qué no debe confundirse | No es la **plantilla** resultante (se vincula con `resolved_template_id`), no es una **incidencia técnica** y no es un **pedido de acceso** a un laboratorio existente. |
| Sinónimos permitidos en prosa | «pedido de entorno» |
| Evitar en interfaz | «Solicitud docente» como nombre de la bandeja; «Solicitud de laboratorio» |
| Evidencia | `backend/internal/core/domain/environment_request.go:62-88` (estructura), `:13-18` (estados), `:21-24` (usos), `:164-182` (derivación de modalidad); `backend/migrations/00046_environment_requests.sql:39` |
| Certeza | Confirmado |

### 2.3 Ejecución y evaluación

#### 2.3.1 Entorno de práctica (instancia del alumno)

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Entorno de práctica |
| Nombre visible recomendado | **Entorno** (con «de práctica» cuando el contexto no lo aclara) |
| Identificador técnico | `PracticeEnvironment`, tabla `environments`, columna `environment_id`, rutas `/api/v1/environments` |
| Categoría | Instancia |
| Definición funcional | Entorno de ejecución concreto de un alumno para una práctica: contenedor, volumen nombrado propio, URL de acceso y estado. Nace de una plantilla, se acota a la práctica (volumen separado) y se hiberna o congela según su ciclo de vida. |
| Qué no debe confundirse | No es la **plantilla** (la receta), no es el **ejercicio** (la consigna), no es el **entorno de revisión** (lo abre el docente sobre una entrega) y no es el **juez** (ejecuta y descarta contenedores sin volumen del alumno). |
| Sinónimos permitidos en prosa | «entorno de trabajo» cuando se explica qué contiene; «laboratorio virtual» para el producto completo, nunca para una fila |
| Evitar en interfaz | «Workspace», «Environment», «Instancia», «Contenedor» |
| Evidencia | `backend/internal/core/domain/practice_environment.go` (instancia y nombre de volumen); `backend/migrations/00048_environment_vocabulary.sql` (renombre de tabla y columnas); `backend/internal/core/services/practice_environment_service.go` (arranque por práctica); `frontend/src/app/core/models/environment.model.ts`; `.../solver/student-exercise-solver.component.html` («Entorno del alumno») |
| Certeza | Confirmado y aplicado el 2026-11-04 (ADR-063) |

#### 2.3.2 Juez Virtual

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Juez Virtual (evaluador automático) |
| Nombre visible recomendado | **Juez Virtual** |
| Identificador técnico | valor almacenado `JUEZ_EFIMERO`; alias aceptado `JUEZ_VIRTUAL`; código `EnvironmentTypeJuezEfimero` |
| Categoría | Servicio del dominio |
| Definición funcional | Componente que ejecuta el código o la consulta de un alumno contra casos de prueba, aplica los filtros de seguridad y emite un veredicto por caso y uno global. Corre en contenedores efímeros, sin volumen del alumno. |
| Qué no debe confundirse | No es el **entorno de evaluación** (la modalidad declarada en una plantilla o modelo) ni el **veredicto** (el resultado). «Juez Evaluador (Sandbox)» del manual es una tercera forma de nombrarlo. |
| Sinónimos permitidos en prosa | «juez», «evaluador automático» al explicar el servicio |
| Evitar en interfaz | «Evaluador Automático» como etiqueta de pantalla; «Juez Efímero» visible |
| Evidencia | `backend/internal/core/domain/exercise.go:25-26`; alias tolerado en `backend/internal/core/services/template_composer_service.go:47`, `admin_governance_templates.go:90,189` y `admin_governance_components.go:364`; el frontend envía `JUEZ_VIRTUAL` como valor de filtro en `frontend/.../model-library.component.ts:159` |
| Certeza | Confirmado el alias; **pendiente de decisión** su retiro o declaración formal |

#### 2.3.3 Entorno de revisión

| Campo | Valor |
| :--- | :--- |
| Nombre canónico | Entorno de revisión |
| Nombre visible recomendado | **Vista previa del proyecto** (si se muestra al docente) |
| Identificador técnico | `ReviewEnvironment`, tabla `review_test_environments`, rutas `/api/v1/teacher/submissions/{id}/test-environment` |
| Categoría | Instancia |
| Definición funcional | Contenedor temporal que monta el snapshot de una entrega y ejecuta la aplicación del proyecto para que el docente la vea funcionando. Se apaga por inactividad y no conserva volumen del alumno. |
| Qué no debe confundirse | No es el **laboratorio del alumno** (persistente y con volumen propio), no es el **juez** (no evalúa casos) y no es la **entrega**. |
| Sinónimos permitidos en prosa | «preview» en documentación técnica |
| Evitar en interfaz | «Test environment», «entorno de prueba», «sandbox» |
| Evidencia | `backend/internal/core/domain/review_environment.go:8-36` (estados y tiempos), `:63` (estructura); `backend/migrations/00044_review_test_environments.sql`; `review_test_environments` convive con la ruta `test-environment` |
| Certeza | Confirmado el objeto; **pendiente de decisión** la etiqueta visible |

### 2.4 Clasificaciones y estados

No son conceptos con identidad propia: son valores de vocabularios cerrados que se definen en la sección 6.

- **Modalidad de entorno** (`target_environment`): desarrollo, evaluación o dual.
- **Estado del laboratorio** (`environments.status`): ocho valores.
- **Estado de revisión** (`submissions.review_status`): cinco valores.
- **Estado de plantilla** (`lab_templates.status`): cuatro valores canónicos más alias en inglés.
- **Tipo de ejercicio** (`exercises.type`): `algorithm`, `database`, `project`.
- **Propósito** (`exercises.purpose`): `class`, `exam`.
- **Visibilidad de caso de prueba** (`test_cases.visibility`): `example`, `public`, `hidden`.
- **Estado de módulo**: `locked`, `unlocked`, `in_progress`, `completed`.
- **Estado de aula**: `open`, `closed`.
- **Estado del alumno en el aula**: `sin_empezar`, `trabajando`, `trabado`, `entregado`, `ausente`.
- **Estado del entorno de revisión**: `starting`, `running`, `expired`, `failed`.
- **Estado del dry-run**: `queued`, `running`, `done`, `failed`.

---

## 3. Tabla de nombres visibles y técnicos

| Concepto | Visible recomendado | Conceptual | Técnico | Almacenado / API | Situación actual |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Ejercicio | Ejercicio | Ejercicio | `Exercise` | `exercises`, `/exercises` | Parcial: «Catálogo de Laboratorios», «Prácticas reutilizables» |
| Materia | Materia | Materia | `Subject` | `subjects`, `/courses` | Parcial: «Cursos» en administración |
| Periodo académico | Periodo académico | Periodo académico | `AcademicPeriod` | `academic_periods` | Estandarizado |
| Módulo curricular | Módulo | Módulo curricular | `CourseModule` | `course_modules` | Parcial: `course_id` con id de materia |
| Inscripción | Inscripción | Inscripción | `Enrollment` | `enrollments` | Estandarizado |
| Entrega | Entrega | Entrega | `Submission` | `submissions` | Estandarizado |
| Aula | Aula | Sesión de clase | `ClassSession` | `class_sessions`, `/sessions` | Buena en UI, deuda en API |
| Plantilla | Plantilla | Plantilla de entorno | `LabTemplate` | `lab_templates`, `/templates` | Parcial: «Docker» en el nombre |
| Modelo del catálogo | Modelo del catálogo | Modelo | `TemplateModelItemDTO` | `template_models` | Parcial: «Modelo» a secas |
| Componente técnico | Tecnología | Componente técnico | `TemplateComponent` | `template_components` | Débil: tres rutas y dos palabras |
| Categoría | Categoría | Categoría | `TemplateCategory` | `template_categories` | Estandarizado |
| Solicitud de entorno | Solicitud de entorno | Solicitud de entorno | `EnvironmentRequest` | `environment_requests` | Parcial: menú y página distintos |
| Laboratorio | Laboratorio | Entorno de ejecución | `PracticeEnvironment` | `environments`, `/environments` | Parcial: «environment» e «instancia» |
| Juez Virtual | Juez Virtual | Evaluador automático | `EnvironmentTypeJuezEfimero` | `JUEZ_EFIMERO` (+ alias) | Contradictoria |
| Entorno de revisión | Vista previa del proyecto | Entorno de revisión | `ReviewEnvironment` | `review_test_environments`, `/test-environment` | Débil: sin etiqueta única |
| Modalidad | Entorno de Desarrollo / de Evaluación / dual | Modalidad de entorno | `target_environment` | `IDE_PERSISTENTE`, `JUEZ_EFIMERO`, `AMBOS` | Parcial: «dual» sin etiqueta propia |
| Estado del laboratorio | Las ocho etiquetas del badge | Estado del laboratorio | `EnvironmentStatus` | 8 códigos | Incompleta en la referencia |
| Veredicto | Las etiquetas de `OUTCOME_COPY` | Resultado de la evaluación | `Verdict` | 9 códigos (+ `PENDING`) | Contrato incompleto |
| Estado de revisión | Pendiente / En revisión / Calificada / Devuelta | Estado de revisión | `ReviewStatus` | 5 códigos | Estandarizado |
| Estado de plantilla | Pendiente de auditoría / Aprobada / Rechazada / Suspendida | Estado de gobernanza | `lab_templates.status` | 4 canónicos + alias | Parcial: «Pausada» y «Suspendida» |

---

## 4. Diferencias entre conceptos relacionados

Las decisiones de esta sección son las que la auditoría dejó abiertas. Cada una fija una frontera y dice qué palabra queda de cada lado.

### 4.1 Ejercicio, práctica, laboratorio y actividad

Son tres objetos distintos y una palabra sobrante:

- **Ejercicio**: la consigna evaluable. Tiene tipo, modalidad, plantilla, casos o rúbrica, fecha y estado de publicación. Es lo que se crea en el banco y lo que genera entregas.
- **Práctica**: la misma consigna vista desde el aula. «La práctica de la sesión» es un ejercicio asignado a un aula. No es un objeto nuevo: es una relación. Por eso puede quedarse en la prosa y no en los títulos.
- **Laboratorio**: el entorno en ejecución que abre el alumno. Tiene contenedor, volumen y estado. No tiene enunciado ni se califica.
- **Actividad**: palabra comodín que hoy aparece en la vista de evaluaciones del estudiante (`student-evaluations.component.html:7,45,52,71`). No aporta precisión: se reemplaza por «ejercicio» o «entrega» según lo que la fila represente.

**Regla:** si la frase admite «resolver», es ejercicio; si admite «abrir», es laboratorio; si admite «entregar», es entrega.

La página del docente incumple hoy las tres: el menú dice «Banco de Ejercicios», el título «Catálogo de Laboratorios», el subtítulo «Prácticas reutilizables» y el botón «Nuevo Laboratorio», y todo crea un `Exercise` (`teacher-exercises.component.html:5-6,18`).

### 4.2 Materia, curso, periodo, módulo y curso de Classroom

- **Materia** es la fila de `subjects`: se dicta en un periodo, tiene docente, código, alumnos y plantilla por defecto.
- **Periodo académico** es el rango de fechas que la contiene. No es un sinónimo: una materia pertenece a un periodo.
- **Módulo** es una unidad interna del programa de la materia, con prerrequisitos. `CourseModule` se llama «curso» en el código pero guarda `subject_id`; el mapa estudiantil expone `course_id` con el identificador de la materia (`course_module.go:25-37,66-71`).
- **Curso de Google Classroom** es un objeto externo que la materia referencia con `classroom_course_id` (`academic.go:17`). No hay integración que lo importe en el backend: solo se guarda el identificador. Es el único uso legítimo de «curso».

**Regla:** «materia» para la asignatura; «periodo» para las fechas; «módulo» para el programa; «curso de Classroom» solo cuando se nombra el objeto externo y con el calificador.

### 4.3 Plantilla, modelo, tecnología y categoría

- **Tecnología** es una pieza del catálogo (lenguaje, framework, complemento o motor de base de datos). Se combina para armar una plantilla.
- **Plantilla** es la receta ejecutable resultante, con estado de auditoría y modalidad. Es lo que usa un ejercicio y lo que instancia un laboratorio.
- **Modelo del catálogo** es una copia curada de una plantilla, con categoría y título propios, pensada para reutilizar. Conserva `source_template_id` hacia su plantilla de origen (`00001_baseline.sql:152-171`). La relación inversa existe (`lab_templates.model_id`), pero es opcional: ninguna de las dos es padre estricta de la otra.
- **Categoría** agrupa modelos. No describe capas ni tecnologías.

**Regla:** no se dice «entorno» para ninguna de las cuatro; «Docker» no forma parte de ningún nombre.

### 4.4 Entorno de práctica, instancia y entorno de revisión

- **Laboratorio** es la instancia persistente del alumno: contenedor más volumen nombrado, con URL y estado. Puede estar acotado a un ejercicio, en cuyo caso el volumen es propio de esa práctica (`environment.go:74-90`).
- **Environment** es el nombre técnico de esa instancia. No se muestra.
- **Instancia** es una palabra genérica que hoy aparece en la administración para contar laboratorios. No se usa como nombre de objeto.
- **Entorno de revisión** es otra instancia, de vida corta, que abre el docente sobre una entrega para ver la aplicación funcionando. No tiene volumen del alumno y no evalúa casos.

**Regla:** «laboratorio» para lo que el alumno abre; «vista previa del proyecto» para lo que el docente mira; «environment» solo en código.

### 4.5 Juez Virtual, Juez Efímero, entorno de evaluación y veredicto

- **Juez Virtual** es el servicio que evalúa. Es el nombre que usan los ADR y la mayor parte de la documentación.
- **Juez Efímero** es el mismo servicio nombrado por su forma de ejecución (contenedores que se descartan). Es el valor almacenado: `JUEZ_EFIMERO`.
- **Entorno de evaluación** es la modalidad declarada en una plantilla, modelo o ejercicio. No es el servicio.
- **Veredicto** es el resultado que el juez emite. No es el juez.

**Regla:** visible «Juez Virtual»; almacenado `JUEZ_EFIMERO`; el alias `JUEZ_VIRTUAL` se declara y no se propaga a nuevas capas.

### 4.6 Aula, sesión, login y vida del laboratorio

«Sesión» tiene hoy cuatro usos:

1. **Autenticación**: «Iniciá sesión con tu cuenta institucional» (`login.component.html:58,70`).
2. **Aula**: `class_sessions`, rutas `/teacher/sessions`, mensajes `ErrSessionAlreadyOpen`.
3. **Vida del laboratorio**: «entre sesiones», en comentarios de configuración del entorno.
4. **Examen o clase congelada**: «eventos de pegado en la sesión», en la copia de integridad.

**Regla:** «sesión» se reserva para autenticación. El aula se llama «aula»; la vida del laboratorio se describe como «mientras está abierto» o «al reabrir»; el examen se nombra como examen.

### 4.7 Veredicto, estado de revisión, nota y veredicto global

Son cuatro cosas distintas que suelen confundirse:

- **Veredicto**: código del juez sobre una entrega o un caso (`AC`, `WA`, …). Puede no existir en ejercicios de proyecto.
- **Estado de revisión**: si el docente debe corregir, lo está haciendo, calificó o devolvió (`review_workflow.go:15-27`). No se deduce del veredicto.
- **Nota**: puntaje de 0 a 100 asignado por el docente, con motivo obligatorio para cambiarla.
- **Veredicto global**: el del conjunto de casos, distinto del de cada caso.

**Regla:** el veredicto no implica revisión pendiente; `WA` no significa «sin corregir».

---

## 5. Sinónimos permitidos y denominaciones a evitar

| Término | Permitido en | Evitar en | Usar en su lugar |
| :--- | :--- | :--- | :--- |
| Práctica | prosa, mensajes explicativos, nombres de columnas docentes | títulos, menús, botones, filtros | Ejercicio |
| Laboratorio | instancia del alumno | catálogo de ejercicios, plantillas | Ejercicio / Plantilla |
| Laboratorio virtual | nombre del producto | una fila o un contenedor concreto | Laboratorio |
| Actividad | (no se recomienda) | toda la interfaz | Ejercicio / Entrega |
| Curso | curso de Google Classroom | materia, periodo, módulo | Materia / Periodo |
| Asignatura | prosa introductoria | etiquetas de navegación | Materia |
| Environment | código, documentación técnica | interfaz | Laboratorio |
| Instancia | documentación de infraestructura | interfaz | Laboratorio |
| Contenedor | infraestructura | interfaz de usuario | Laboratorio |
| Entorno | nunca sin calificador | toda la interfaz | Entorno de desarrollo / de evaluación / de revisión |
| Entorno de práctica | — | toda la interfaz | Laboratorio |
| Juez Efímero | documentación técnica | interfaz | Juez Virtual |
| Evaluador Automático | prosa explicativa | etiquetas y badges | Juez Virtual |
| Juez Evaluador (Sandbox) | — | manual y documentos nuevos | Juez Virtual |
| Plantilla Docker | — | menús y títulos | Plantilla |
| Modelo | pantalla del catálogo | fuera del catálogo | Modelo del catálogo |
| Componente | documentación técnica | interfaz | Tecnología |
| Herramienta | binario verificado (`tools_provided`) | ítem del catálogo | Tecnología |
| Sesión | autenticación | aula, laboratorio, examen | Aula |
| Test environment | — | contratos y documentos nuevos | Entorno de revisión |
| Activo / Iniciando / En Pausa / Examen cerrado | badge de estado del laboratorio | cualquier otro texto de estado | La etiqueta canónica del badge |

---

## 6. Contratos de estados y veredictos

### 6.1 Veredictos del juez

Los nueve códigos del dominio (`exercise.go:71-79`) tienen una tabla visible propia en el frontend, `OUTCOME_COPY` (`frontend/src/app/core/models/verdict.model.ts:31-96`), que es hoy la fuente única del lenguaje humano. Esta propuesta la adopta como capa visible y no inventa etiquetas nuevas.

| Código | Etiqueta visible | Significado | Condición que lo produce | Color | ¿Llega al alumno? |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `AC` | Cumple | Pasa todos los casos | Todos los casos ejecutados dan `AC` | Verde | Sí |
| `WA` | No cumple | Ejecutó y el resultado no coincide | Al menos un caso difiere de la salida esperada | Rojo | Sí |
| `TLE` | Tarda demasiado | Superó el tiempo permitido | El runner corta por tiempo | Ámbar | Sí |
| `MLE` | Se quedó sin memoria | Superó la memoria del cgroup | Contenedor terminado por OOM (exit 137) | Ámbar (comparte con `TLE`) | Sí |
| `RE` | Se cae al ejecutar | Error durante la ejecución | El programa termina con error | Naranja | Sí |
| `CE` | No compila | Error de compilación | El compilador falla antes de ejecutar | Naranja (comparte con `RE`) | Sí |
| `VE` | Caso de prueba inválido | La prueba tiene un problema, no el código | El checker del docente falla o expira (`comparators/custom.go:86-92`) | Naranja (comparte con `RE`) | Sí |
| `AST_VIOLATION` | Usó algo no permitido | Restricción detectada por regex | Filtro AST previo a la ejecución (`evaluation_algorithm_runner.go:129-147`) | Rojo intenso | Sí, con la etiqueta de `AST_BLOCKED` |
| `AST_BLOCKED` | Usó algo no permitido | Restricción detectada por análisis semántico | Semgrep o regla declarada por el docente (`:149-200`) | Rojo intenso | Sí |

Notas de contrato:

- `AST_VIOLATION` no tiene etiqueta propia: es un alias de `AST_BLOCKED` en la tabla visible (`OUTCOME_ALIASES` en `verdict.model.ts`). Unificar la presentación no elimina el código del dominio, que se conserva para auditoría.
- `PENDING` y `UNKNOWN` no son veredictos: son estados de presentación de la tabla visible. `PENDING` significa «sin calificar» y `UNKNOWN` «sin evaluar». No existen en el dominio.
- `SYSTEM_ERROR` aparece declarado en `frontend/src/app/core/services/student.service.ts:173,179` pero no existe en el backend ni en la tabla visible: cualquier valor así cae en `UNKNOWN`. Es una inconsistencia de tipos, no un veredicto.
- `AGENTS.md:49` documenta cinco veredictos (`AC`, `WA`, `TLE`, `RE`, `AST_BLOCKED`) y omite `MLE`, `CE`, `VE` y `AST_VIOLATION`. `docs/API/CONTRATOS.md:162` declara cinco en la interfaz `Submission`.

### 6.2 Agregación del veredicto

El veredicto global no es el peor veredicto ni un promedio: es **el primero distinto de `AC` en el orden de los casos** (`evaluation_algorithm_runner.go:285-320`). Todos los casos se ejecutan, sin detención temprana, y el detalle queda en `CaseResults`.

Consecuencias que el contrato debe declarar:

- Dos entregas con los mismos fallos en distinto orden pueden mostrar veredictos globales distintos.
- Los filtros de seguridad (`AST_*`) se aplican antes de ejecutar casos y devuelven un resultado único, sin `CaseResults`.
- El dry-run de la solución de referencia usa otra regla: `AC` si todos los casos dan `AC`; `WA` en cualquier otro caso (`evaluation_dryrun.go:122-132`).
- Los casos de prueba I/O solo aplican a `JUEZ_EFIMERO`; un ejercicio de desarrollo se corrige con rúbrica (`exercise.go:244`).

**Pendiente de decisión:** si el orden de los casos debe determinar el veredicto global o si conviene una precedencia explícita. Cambiarlo afecta datos históricos.

### 6.3 Estados del laboratorio

Ocho valores en `environment.go:10-27`, con etiqueta canónica en `status-badge.component.ts:43-64`.

| Código | Etiqueta visible | Significado | Quién lo asigna | Efecto |
| :--- | :--- | :--- | :--- | :--- |
| `pending` | Iniciando | Registro creado, contenedor en arranque | Servicio de laboratorios (`environment_service.go:566-580`) | Retiene cupo mientras arranca |
| `running` | Activo | Contenedor operativo y con latido | Servicio y reconciliador | Uso normal |
| `hibernated` | En Pausa | Contenedor retirado por inactividad o salida limpia; el volumen se conserva | Reconciliador (`environment_state_reconciler.go:114-124`) | Se reactiva al reabrir; no pierde código |
| `oom_killed` | Límite de Memoria | Terminado por falta de memoria | Reconciliador y worker de QoS | Suma un strike; al tercero hay penalización (`environment.go:35`) |
| `failed` | Error | Falló la provisión o el arranque | Servicio de laboratorios | Requiere reintento o intervención |
| `unknown` | Sin Verificar | No se pudo verificar el estado (daemon inaccesible) | Reconciliador (`:143-160`) | No destruye nada; conserva el cupo y reintenta |
| `orphaned` | Huérfano | El contenedor ya no existe en el daemon | Reconciliador (`:96-101`) | Libera cupo; el alumno puede recrear el laboratorio |
| `frozen` | Examen cerrado | Examen terminado; volumen en solo lectura | Cierre de examen (`exam_freeze_repository.go:100`) | Estado buscado, no fallo; no se reabre solo |

`AGENTS.md:48` describe cinco de estos estados. `unknown`, `orphaned` y `frozen` no tienen semántica de color asignada; hoy heredan la clase por defecto del badge. **Pendiente de decisión.**

### 6.4 Estado de revisión

Cinco valores (`review_workflow.go:15-27`), eje independiente del veredicto:

| Código | Etiqueta conceptual | Significado |
| :--- | :--- | :--- |
| `not_required` | Sin revisión | El juez calificó solo; no espera corrección manual |
| `pending` | Pendiente | Espera corrección del docente |
| `in_review` | En revisión | El docente la abrió y está trabajando |
| `graded` | Calificada | Tiene nota y feedback |
| `returned` | Devuelta | Se devolvió al alumno con feedback |

La transición no se deduce del veredicto: una entrega con `WA` puede quedar `not_required` si el ejercicio se califica solo, y una con `AC` puede requerir revisión si el docente lo decidió.

### 6.5 Estado de plantilla

Cuatro valores canónicos en la base, con alias en inglés aceptados por el código (`approvedTemplateStatusClause`, `admin_templates_repository.go:14-24`):

| Almacenado | Alias aceptado | Etiqueta visible actual | Significado |
| :--- | :--- | :--- | :--- |
| `PENDIENTE_AUDITORIA` | `pending`, `pending_audit` | Pendiente de auditoría | Espera revisión del administrador |
| `APROBADA` | `approved` | Aprobada | Disponible para ejercicios |
| `RECHAZADA` | `rejected` | Rechazada | No cumple los controles |
| `SUSPENDIDA` | `suspended`, `paused` | Pausada / Suspendida | Retirada del catálogo sin borrarse |

La interfaz usa «Pausada» para el estado que la base llama `SUSPENDIDA` (`admin-templates.component.html:81,457,685`). Es el único caso donde una etiqueta visible contradice el valor almacenado. **Pendiente de decisión:** una sola palabra para ambos.

### 6.6 Aula y participantes

| Vocabulario | Valores | Etiqueta visible |
| :--- | :--- | :--- |
| Estado del aula | `open`, `closed` | Abierta / Cerrada |
| Estado del alumno | `sin_empezar`, `trabajando`, `trabado`, `entregado`, `ausente` | Sin empezar / Trabajando / Se trabó / Entregó / Salió (`aula.component.ts:68-72`) |

Las ventanas temporales también son parte del contrato: presencia de 120 s y «trabado» de 10 minutos (`class_session.go:33-40`); duración de aula entre 15 y 480 minutos, 90 por defecto (`:42-49`).

### 6.7 Modalidades y representación de `AMBOS`

Hay tres valores, pero no viven en el mismo lugar:

| Valor | Dónde se admite | Etiqueta visible | Significado |
| :--- | :--- | :--- | :--- |
| `IDE_PERSISTENTE` | Ejercicio, plantilla, modelo, solicitud | Entorno de Desarrollo | El alumno trabaja en un entorno persistente |
| `JUEZ_EFIMERO` | Ejercicio, plantilla, modelo, solicitud | Entorno de Evaluación | Se evalúa con casos y veredicto |
| `AMBOS` | Plantilla, modelo y solicitud | Dual (Desarrollo y Evaluación) | La configuración sirve para las dos modalidades |

Lo que el código demuestra:

- `exercises.environment_type` solo acepta dos valores (`00020_dual_modality_and_template_governance.sql:42`). Un ejercicio nunca es dual: elige un modo.
- `environment_requests.target_environment` acepta los tres (`00046_environment_requests.sql:39`).
- `TemplateSupportsEnvironment` expande `AMBOS` a cualquiera de los dos modos pedidos (`admin_governance.go:58-63`).
- La pantalla del catálogo envía `JUEZ_VIRTUAL` como valor de filtro y muestra «Dual (Desarrollo y Evaluación)» para `AMBOS` (`model-library.component.ts:157-161`).
- El pipe `environmentLabel` resuelve como «Entorno de Evaluación» todo valor que no sea `IDE_PERSISTENTE` (ni sus alias `WORKSPACE`, `IDE` y `PROJECT`) (`environment-types.constants.ts:33-39`), de modo que `AMBOS` se muestra hoy como evaluación, no como dual.
- `environments.type` existe y tiene constantes, pero la creación del laboratorio no lo fija (`environment_service.go:566-580`): la columna queda en su valor por defecto y hoy no discrimina nada.

**Regla propuesta:** «modalidad» para la capacidad declarada en una configuración; «tipo de entorno» para el modo concreto de un ejercicio. `AMBOS` no se asigna a ejercicios ni a laboratorios.

### 6.8 Otros vocabularios controlados

| Vocabulario | Valores | Nota |
| :--- | :--- | :--- |
| Tipo de ejercicio | `algorithm`, `database`, `project` | Restringido por `CHECK` desde `00033` |
| Propósito | `class`, `exam` | `exam` habilita semilla por alumno y congelamiento |
| Visibilidad de caso | `example`, `public`, `hidden` | `hidden` nunca se expone al rol student |
| Estado de módulo | `locked`, `unlocked`, `in_progress`, `completed` | Bloquea ejercicios por prerrequisito |
| Estado de entorno de revisión | `starting`, `running`, `expired`, `failed` | Se apaga por inactividad a los 15 min |
| Estado de dry-run | `queued`, `running`, `done`, `failed` | Progreso por caso |
| Resultado de backup | `success`, `failed`, `verified` | Fuera del alcance de esta propuesta |

---

## 7. Decisiones pendientes de aprobación

Ninguna de estas decisiones se aplicó. Cada una indica la recomendación y qué evidencia falta para cerrarla.

### D1. Alcance de la palabra «Laboratorio»

- **Opciones:** (a) laboratorio = instancia del alumno y «laboratorio virtual» = producto; (b) laboratorio = producto y «entorno de trabajo» = instancia.
- **Recomendación:** (a). Es lo que ya muestran la lista del alumno y los botones de apertura; el cambio se limita a dejar de usar «laboratorio» para ejercicios.
- **Riesgo:** la copia del producto usa «laboratorio» en sentido amplio en manuales y presentaciones. Requiere una regla editorial, no código.
- **Para cerrarla:** aprobación de producto y revisión de `docs/MANUAL_ADMIN.md`.

### D2. Alias `JUEZ_VIRTUAL`

- **Opciones:** (a) declararlo alias aceptado y documentarlo; (b) retirarlo con migración y ventana de compatibilidad.
- **Recomendación:** (a) ahora, (b) cuando se sepa si hay filas con ese valor.
- **Qué falta:** consultar `SELECT DISTINCT target_environment FROM lab_templates` y `template_models`. Sin base disponible no se puede afirmar que no exista.
- **Impacto si no se decide:** cualquier comparación estricta nueva (`== "JUEZ_EFIMERO"`) rechaza un valor que el frontend sigue enviando.

### D3. Nombre visible de `TemplateComponent`

- **Opciones:** (a) «Tecnología»; (b) «Componente».
- **Recomendación:** (a). La pantalla ya dice «Catálogo de Tecnologías», «Registrar Nueva Tecnología» y «Guardar Tecnología»; «componente» queda como nombre conceptual y técnico.
- **Nota:** esta recomendación corrige la propuesta preliminar de la auditoría, que sugería «Componente» sin haber relevado esas cinco etiquetas.
- **Impacto si no se decide:** el catálogo mezcla «Tecnología», «Herramienta» y «Componente» en la misma vista.

### D4. «Curso» en la interfaz

- **Opciones:** (a) eliminar «curso» del vocabulario visible salvo «curso de Classroom»; (b) mantener «Cursos» como nombre de la pantalla administrativa y «Materia» dentro del formulario.
- **Recomendación:** (a). La pantalla se llama «Cursos y Periodos Académicos» y el formulario pide «Nombre de la Materia»; la API ya usa `subjects` en las consultas.
- **Impacto:** cambio de etiquetas en menú, título, KPIs y modales de administración. No toca rutas ni datos.

### D5. «Pausada» o «Suspendida»

- **Opciones:** (a) «Suspendida» (coincide con la base); (b) «Pausada» (coincide con la interfaz).
- **Recomendación:** (a) para el estado de gobernanza, y reservar «En Pausa» para el laboratorio hibernado. Dos cosas distintas no deberían compartir palabra.
- **Impacto:** tres etiquetas y la pestaña de filtro del catálogo de plantillas.

### D6. Nombre visible del modelo del catálogo

- **Opciones:** (a) «Modelo del catálogo»; (b) «Modelo oficial»; (c) «Modelo» a secas dentro de la pantalla.
- **Recomendación:** (a) en documentación y títulos; (c) dentro de la pantalla, donde el contexto ya es el catálogo.
- **Qué falta:** confirmar que «modelo» no se usa para modelos de IA en la misma pantalla (hoy ocurre en `ai_settings` y `ai_copilot`, fuera del catálogo).

### D7. Regla del veredicto global

- **Opciones:** (a) dejar el primero distinto de `AC`; (b) definir una precedencia explícita (`AST_BLOCKED` > `CE` > `RE` > `MLE` > `TLE` > `WA`).
- **Recomendación:** (a) para no reescribir historia, y documentar la regla en el contrato. Si se elige (b), aplicarla solo a evaluaciones nuevas y guardar la versión de la regla junto al resultado.
- **Impacto:** cambia la lectura de entregas antiguas si se aplica retroactivamente.

### D8. Estados sin semántica en `AGENTS.md`

- **Opciones:** (a) agregar `unknown`, `orphaned` y `frozen` con color; (b) dejarlos fuera de la guía y resolverlos por pantalla.
- **Recomendación:** (a): `unknown` = warning, `orphaned` = neutral, `frozen` = neutral con icono de candado. Son estados operativos, no fallos del alumno.
- **Impacto:** una línea en `AGENTS.md` y tres clases en el badge.

### D9. Visibilidad de `VE` para el alumno

- **Opciones:** (a) mostrarlo como «Caso de prueba inválido»; (b) tratarlo como incidencia de plataforma y no como resultado del alumno.
- **Recomendación:** (b) si el caso lo declaró el docente; (a) si el alumno puede corregir el formato de entrada. Hoy la tabla visible lo muestra siempre.
- **Qué falta:** definir qué espera el alumno cuando el problema es del enunciado.

### D10. Etiqueta del entorno de revisión

- **Opciones:** (a) «Vista previa del proyecto»; (b) «Entorno de revisión».
- **Recomendación:** (a) para la interfaz del docente, (b) en documentación. «Revisión» describe la tarea; «vista previa» describe lo que ve.
- **Qué falta:** confirmar la etiqueta actual en SpeedGrader; no hay una única hoy.

### D11. Uso de `environments.type`

- **Opciones:** (a) fijarlo explícitamente al crear el laboratorio; (b) declararlo legado y no usarlo.
- **Recomendación:** (a) si se planea abrir laboratorios de evaluación persistentes; (b) mientras tanto, no leerlo como discriminador.
- **Evidencia:** la creación no lo asigna (`environment_service.go:566-580`); la constante `EnvironmentTypeJuezEfimero` existe sin uso en creación.

### D12. «Actividad» en la vista de evaluaciones del estudiante

- **Opciones:** (a) reemplazar por «Ejercicio»; (b) dejarlo como término de esa pantalla.
- **Recomendación:** (a). La fila muestra un ejercicio con su entrega; «actividad» no distingue ninguno de los dos.

### D13. Representación de `AMBOS` en el pipe de entorno

- **Opciones:** (a) agregar la etiqueta dual al mapa; (b) dejar el fallback actual.
- **Recomendación:** (a). Hoy `AMBOS` se muestra como «Entorno de Evaluación», lo que oculta una capacidad de la plantilla.
- **Impacto:** una entrada en `ENVIRONMENT_TYPES` y revisión de los lugares que usan el pipe.

### D14. Rutas técnicas de componentes

- **Opciones:** (a) declarar `template-components` canónica y el resto alias; (b) retirar los alias.
- **Recomendación:** (a) ahora; (b) cuando no queden clientes con las rutas antiguas.
- **Evidencia:** tres prefijos para el mismo handler (`router.go:251-258`).

---

## 8. Recomendaciones para actualizar `AGENTS.md`, skills y documentación

### 8.1 Bloque propuesto para `AGENTS.md`

Se propone insertar una sección después de «Semántica de estados», sin reemplazarla:

```markdown
## Vocabulario de dominio

| Concepto | Visible | Técnico |
| :--- | :--- | :--- |
| Ejercicio | Ejercicio | exercise |
| Materia | Materia | subject |
| Aula | Aula | class_session |
| Entrega | Entrega | submission |
| Plantilla | Plantilla | lab_template |
| Modelo del catálogo | Modelo | template_model |
| Tecnología | Tecnología | template_component |
| Laboratorio | Laboratorio | environment |
| Juez Virtual | Juez Virtual | JUEZ_EFIMERO |
| Entorno de revisión | Vista previa del proyecto | review_environment |

Reglas:
- «Práctica» solo en prosa; nunca en títulos, menús ni botones.
- «Curso» solo para el curso de Google Classroom.
- «Sesión» solo para autenticación; el aula se llama aula.
- «Entorno» siempre con calificador: de desarrollo, de evaluación o de revisión.
- Los códigos del juez no se escriben a mano: salen de `OUTCOME_COPY`.
- No se crean alias nuevos; los existentes se listan en `docs/GOBERNANZA/PROPUESTA_VOCABULARIO_DOMINIO.md`.
```

No se aplicó este cambio: la fase prohíbe modificar `AGENTS.md`.

### 8.2 Skills

- **`solv-context`**: agregar la tabla de conceptos y las reglas de la sección 4. Es la skill que se carga cuando el trabajo toca dominio y hoy no tiene vocabulario.
- **`solv-design-system`**: indicar que las etiquetas de estado y veredicto salen de `status-badge` y `OUTCOME_COPY`; no se escriben literales nuevos.
- **`angular-architecture`**: no requiere cambios; la regla de nombres de selector ya está resuelta allí.
- **`gobernanza`**: agregar el checklist de la sección 8.4 al momento de escribir ADR o commits que introduzcan conceptos nuevos.

### 8.3 Documentación

| Documento | Cambio recomendado |
| :--- | :--- |
| `docs/API/CONTRATOS.md` | Reemplazar la unión de cinco veredictos por los nueve códigos y aclarar que `AST_VIOLATION` comparte etiqueta con `AST_BLOCKED` |
| `docs/API/CONTRATOS.md` | Documentar los ocho estados de laboratorio y el estado de revisión como eje separado |
| `docs/UI/MAPA_DE_VISTAS.md` | Corregir las rutas `/docente/*` y `/estudiante/*`, y la palabra «hojas de trabajo» |
| `docs/MANUAL_ADMIN.md` | Unificar «Juez Virtual» y «Entorno de Evaluación»; revisar «Juez Evaluador (Sandbox)» |
| ADR de catálogo | Anotar `docker_templates` y `courses` como nombres históricos, sin renombrar tablas |
| `docs/README.md` | Al enlazar esta propuesta, marcarla como pendiente de aprobación |

### 8.4 Checklist de revisión (sin gate automático)

Para cualquier cambio que agregue una etiqueta, un estado o un concepto:

1. ¿El concepto ya existe en la tabla de la sección 3? Si sí, se reutiliza la palabra.
2. ¿La etiqueta nueva aparece en menú, título y botón con la misma palabra?
3. ¿El texto visible evita los términos de la columna «Evitar» de la sección 5?
4. Si se agrega un estado o veredicto: ¿se actualizaron el contrato y la tabla visible en el mismo cambio?
5. Si se agrega un alias: ¿está declarado y justificado, o se puede evitar?

### 8.5 Regla para agentes

> Antes de escribir una etiqueta nueva, buscar la palabra en el menú o el título de la pantalla. Si el concepto no está en el vocabulario aprobado, preguntar antes de inventarlo. Los nombres técnicos pueden seguir en inglés; los visibles, no.

---

## 9. Evidencia utilizada

**Dominio y servicios del backend**

- `backend/internal/core/domain/exercise.go` — tipos, modalidades, propósito, visibilidad, veredictos, estructura del ejercicio.
- `backend/internal/core/domain/environment.go` — estados, tipos, instancia, volumen, plantilla.
- `backend/internal/core/domain/academic.go` — materia, inscripción, entrega.
- `backend/internal/core/domain/course_module.go` — módulo y mapa curricular.
- `backend/internal/core/domain/class_session.go` — aula, estados, ventanas.
- `backend/internal/core/domain/environment_request.go` — solicitud, usos, derivación de modalidad.
- `backend/internal/core/domain/template_component.go` — capas, modos de instalación, subcategorías.
- `backend/internal/core/domain/review_workflow.go` — estado de revisión.
- `backend/internal/core/domain/review_environment.go` — entorno de revisión y tiempos.
- `backend/internal/core/domain/admin_governance.go` — constantes de modalidad, `TemplateSupportsEnvironment`, DTO de modelo.
- `backend/internal/core/services/evaluation_algorithm_runner.go` — orden de ejecución, filtros AST, veredicto global.
- `backend/internal/core/services/evaluation_dryrun.go` — agregación del dry-run.
- `backend/internal/core/services/environment_state_reconciler.go` — transiciones a `unknown`, `orphaned`, `hibernated`, `running`.
- `backend/internal/core/services/environment_service.go` — creación de la instancia y estado inicial.
- `backend/internal/core/services/template_composer_service.go` y `admin_governance_templates.go` — tolerancia del alias `JUEZ_VIRTUAL`.
- `backend/internal/infrastructure/storage/postgres/admin_templates_repository.go` — normalización de estados de plantilla.
- `backend/internal/infrastructure/storage/postgres/exam_freeze_repository.go` — estado congelado.

**Esquema y contratos**

- `backend/migrations/00001_baseline.sql` — tablas `lab_templates`, `template_models`, `template_categories`, `exercises`, `environments`, `subjects`, `academic_periods`.
- `00020_dual_modality_and_template_governance.sql:42` — modalidad restringida del ejercicio.
- `00033_enforce_exercise_type_environment.sql` — tipos de ejercicio.
- `00044_review_test_environments.sql` — tabla del entorno de revisión.
- `00046_environment_requests.sql:39` — modalidad de la solicitud.
- `docs/API/CONTRATOS.md:162` — unión de veredictos en la interfaz `Submission`.
- `AGENTS.md:46-49` — semántica de estados y veredictos.

**Frontend**

- `frontend/src/app/core/models/verdict.model.ts` — tabla visible de resultados y alias.
- `frontend/src/app/shared/components/status-badge/status-badge.component.ts:43-64` — etiquetas de estado.
- `frontend/src/app/shared/constants/environment-types.constants.ts` — etiquetas de modalidad y fallback.
- `frontend/src/app/features/admin/templates/...` — catálogo, modelos, tecnologías, filtros y estados de plantilla.
- `frontend/src/app/features/teacher/exercises/teacher-exercises.component.html` — contradicción ejercicio/laboratorio.
- `frontend/src/app/features/admin/courses/...` — curso/materia.
- `frontend/src/app/features/student/environments/...` y `.../solver/...` — laboratorio/environment.
- `frontend/src/app/features/teacher/aula/aula.component.ts:68-72` — etiquetas de participantes.
- `frontend/src/app/features/auth/login/login.component.html:58,70` — «sesión» como autenticación.

**Método**

- Lectura directa de archivos y búsquedas de texto sobre el repositorio; sin ejecución de la aplicación ni consulta a la base (backend y PostgreSQL no disponibles durante el relevamiento).

---

## 10. Contradicciones que no se pudieron resolver

1. **¿Hay filas con `JUEZ_VIRTUAL` almacenado?** El frontend lo envía como filtro y cuatro puntos del backend lo toleran, pero sin consultar la base no se puede saber si alguna plantilla o modelo lo tiene guardado. Hasta entonces, retirarlo es riesgoso.
2. **¿Se crean laboratorios para ejercicios de evaluación?** La creación no fija `type` y el juez usa contenedores efímeros, pero no se verificó en ejecución que no existan filas con `JUEZ_EFIMERO` en `environments.type`.
3. **¿«Laboratorio» es la instancia o el producto?** El repositorio usa ambas lecturas. Es una decisión de producto, no de código.
4. **¿«Pausada» o «Suspendida»?** La base y la interfaz no coinciden y ninguna es claramente anterior. Requiere decisión editorial.
5. **¿`VE` debe llegar al alumno como resultado?** La tabla visible lo muestra; la semántica («el problema es de la prueba») sugiere otra cosa.
6. **¿El orden de los casos debe definir el veredicto global?** Hoy lo define, y cambiarlo afecta datos históricos. La decisión exige una regla de migración, no solo de código.

---

*Fin de la propuesta. Ninguna decisión de este documento está aplicada en el repositorio.*
