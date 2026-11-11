# ADR-068: Proyectos Iniciales y Versiones Inmutables en `IDE_PERSISTENTE`

- **Estado:** Aceptado
- **Fecha:** 2026-11-10
- **Actualización:** 2026-11-11 — regla de compatibilidad entre proyecto inicial y entorno.
- **Ámbito:** Docencia, entornos de práctica, integridad de datos

## Contexto

`exercises.boilerplate` era a la vez el proyecto inicial editable por el docente y la
base que SpeedGrader, el antiplagio y el timeline usaban para comparar entregas
históricas. Editar el ejercicio reemplazaba ese texto en el lugar. Una entrega de
un periodo anterior quedaba comparándose contra un proyecto que el alumno nunca
recibió.

La modalidad `JUEZ_EFIMERO` no tiene este problema: su base es el enunciado del
ejercicio y no se versiona por entorno. La corrección de Fase 1 se limita a
`IDE_PERSISTENTE`.

## Decisión

1. **Proyecto estable por ejercicio.** `workspace_projects` tiene una fila por
   `exercise_id`. Su `current_published_version_id` es la fuente de verdad de la
   versión publicada.
2. **Borrador mutable separado.** `workspace_project_drafts` guarda el texto en
   edición con revisión optimista. Un proyecto tiene un solo borrador.
3. **Versiones inmutables.** Cada publicación crea una fila en
   `workspace_project_versions` con `version_number`, texto, manifiesto y hash.
   Un trigger rechaza `UPDATE` y `DELETE`; el historial de entregas no se puede
   reescribir.
4. **Espejo de compatibilidad.** `exercises.boilerplate` sigue existiendo para
   los consumidores que aún lo leen. La publicación actualiza el espejo, el hash
   y `boilerplate_source_version_id` dentro de la misma transacción. Un trigger
   impide cambiar el texto sin cambiar la versión de origen.
5. **Referencias legacy.** El `boilerplate` histórico de un IDE sin proyecto se
   copia a `workspace_legacy_project_references`. No recibe número de versión,
   no se publica y no se enlaza desde entregas ni entornos.
6. **Clasificación de la entrega.** `submissions.base_status` toma uno de:
   - `version`: la entrega referencia `workspace_project_version_id`.
   - `no_base`: el entorno nació sin proyecto.
   - `legacy_unknown`: la entrega es anterior a Fase 1.
   El juez conserva `base_status = NULL` y sigue usando `exercises.boilerplate`.
7. **Integridad multi-tenant y de alcance.** Las FKs compuestas impiden que una
   versión, un entorno o una entrega mezclen tenant, proyecto o ejercicio. La FK
   de entorno en `submissions` se agregó `NOT VALID` para no romper filas
   históricas; las filas nuevas y las actualizaciones sí se validan.
8. **Publicación cerrada por defecto.** `WORKSPACE_PROJECTS_PUBLISH_ENABLED`
   solo habilita la publicación cuando vale exactamente `true`. Cualquier otro
   valor, incluido el ausente, devuelve `403 workspace_publishing_disabled`.
9. **Entornos.** `environments.workspace_seed_mode` distingue `project`, `none`
   y `legacy`; `workspace_init_status` conserva los estados transitorios. La
   correspondencia `legacy` ↔ `legacy` es bidireccional y una versión solo puede
   existir con modo `project`.
10. **Importación como borrador (Fase 2).** La carpeta del navegador y el ZIP se
    convierten en el mismo snapshot normalizado antes de tocar la base. El
    servicio común valida rutas, texto UTF-8, colisiones, límites y el formato
    de cabeceras; después llama al mismo guardado con revisión optimista. La
    importación nunca publica ni escribe `exercises.boilerplate`.
11. **Inicialización del volumen individual.** Al crear un entorno
    `IDE_PERSISTENTE` se resuelve una sola vez la versión publicada del
    ejercicio y se persiste en `environments.workspace_project_version_id`
    antes de crear el volumen. El sembrado ocurre después de crear el volumen y
    antes de arrancar OpenVSCode. Cada alumno recibe su propia copia física de
    los archivos; la versión publicada y el borrador del docente no se tocan.
    Un entorno que ya tiene versión asignada no consulta la publicación actual.
12. **Sembrado conservador.** El volumen se considera apto solo si está vacío o
    si contiene un diario propio de la misma versión. Un archivo ajeno, una
    modificación, una versión distinta, un estado ambiguo o un volumen `legacy`
    se conservan y se rechazan con error; no existe una operación de re-siembra
    destructiva. El diario y el marcador viven en `.solv_workspace/` dentro del
    volumen, nunca en el host.
13. **Recuperación y verificación.** Un diario `pending` de la misma versión
    permite completar únicamente archivos faltantes. La promoción a
    `setup_done` exige comparar el árbol completo contra el snapshot y escribir
    el marcador `verified`. Al reabrir un entorno ya inicializado se verifica
    ese marcador; si falta, el entorno pasa a cuarentena en vez de sembrarse de
    nuevo. Los estados operativos del contenedor se mantienen separados de
    `workspace_init_status`.

## Superficie HTTP

- `GET /api/v1/teacher/exercises/{id}/project`
- `PUT /api/v1/teacher/exercises/{id}/project/draft`
- `POST /api/v1/teacher/exercises/{id}/project/import/folder`
- `POST /api/v1/teacher/exercises/{id}/project/import/zip`
- `POST /api/v1/teacher/exercises/{id}/project/generate`
- `POST /api/v1/teacher/exercises/{id}/project/publish`

Las rutas exigen `teacher` o `admin`, resuelven tenant y usuario desde el
contexto autenticado y comprueban que el ejercicio sea `IDE_PERSISTENTE` y que
la materia pertenezca al docente (un administrador puede operar dentro de su
tenant). La publicación se mantiene apagada por defecto.

### Contrato de importación

- **Carpeta:** `multipart/form-data` con un campo `manifest` y un archivo por
  entrada. El manifiesto declara `{ path, field, size }`; el nombre del campo
  nunca se interpreta como ruta. Cuerpo HTTP máximo 4 MiB.
- **ZIP:** `multipart/form-data` con un único campo `archive` y
  `expected_revision`. ZIP comprimido máximo 10 MiB, 1000 entradas, 20 MiB
  expandidos y 2 MiB por entrada antes de aplicar el límite del proyecto.
- **Resultado final:** máximo 1 MiB de contenido, 300 archivos y 512 KiB por
  archivo. Las rutas se preservan relativas; se rechazan rutas absolutas,
  `..`, separadores ambiguos, duplicados, colisiones de normalización Unicode,
  binarios, entradas cifradas, symlinks y tamaños inconsistentes.
- **Formato:** el mismo snapshot `# === FILE: <ruta> ===` que consumen
  SpeedGrader, antiplagio y timeline. El contenido se normaliza a UTF-8/LF y un
  salto final; una cabecera dentro del texto se rechaza para no partir el
  archivo en silencio.

### Contrato de generación asistida

La generación con IA es una tercera fuente de archivos, no un sistema de
proyectos paralelo. El docente describe lo que necesita y recibe una propuesta
que revisa antes de incorporarla al editor.

- **Entrada:** JSON `{ description, purpose?, technology?, template_id?,
  stack_config?, requirements? }` con cuerpo HTTP máximo de 16 KiB.
  `description` es obligatoria; los campos de texto juntos no pasan de 4 KiB y
  cada uno tiene su propio tope. `template_id` y las piezas de `stack_config`
  se acotan a 100 caracteres y `addon_ids` a 20 elementos. La etiqueta
  `technology` queda como respaldo para clientes anteriores.
- **Proveedor:** el institucional ya configurado (`AISettingsService`). La ruta
  usa `CompleteJSON` con la acción `generate-workspace-project`, de modo que
  hereda cifrado AES-256-GCM de la API Key, cascada BYOK, cuota horaria,
  `response_format: json_object`, escudo anti-SSRF y presupuesto de tiempo por
  perfil de motor. Ninguna credencial, URL ni modelo llegan desde el navegador.
- **Salida del motor:** `{ "files": [{ "path", "content" }], "summary" }`.
  El decodificador rechaza campos desconocidos y contenido fuera del objeto.
  La respuesta no puede declarar imágenes, runtimes, capacidades, variables de
  infraestructura ni comandos: el contrato no tiene esos campos.
- **Validación:** la propuesta pasa por `normalizeWorkspaceProjectFiles`, el
  mismo normalizador que una carpeta o un ZIP: rutas relativas, sin `..`, sin
  `\\`, sin duplicados ni colisiones NFC, UTF-8/LF, sin binarios, máximo 1 MiB,
  300 archivos y 512 KiB por archivo. Una sola entrada inválida descarta la
  propuesta completa.
- **Sin efectos:** la ruta no crea proyecto, no guarda borrador, no publica, no
  escribe volúmenes ni ejecuta el código generado. Aplicar la propuesta solo
  cambia los archivos en pantalla; conservarla exige el guardado del borrador y
  publicarla sigue siendo una acción aparte.
- **Errores:** `504 ai_timeout`, `409 ai_disabled`, `429 ai_quota_exceeded`,
  `400` para entrada inválida o proveedor no configurado
  (`ai_provider_unavailable`) y `422 ai_response_invalid` cuando la respuesta no
  cumple el contrato o no pasa la normalización. No se registran claves ni el
  prompt completo.

## Compatibilidad entre proyecto inicial y entorno (2026-11-11)

El proyecto inicial y el entorno homologado se eligen por separado. Esa
separación permite reutilizar un proyecto entre materias, pero también permite
publicar un proyecto cuyo stack no coincide con las herramientas de la imagen.
La regla es explícita: el sistema detecta la contradicción, la muestra y pide
confirmación; no bloquea la publicación ni intenta validar el código. La
detección de familias vive en el catálogo compartido
(`TECH_FAMILY_BY_ID`), de modo que el proyecto y el entorno no mantienen listas
paralelas. Al importar una carpeta, la comprobación ocurre antes de subir los
archivos y la importación queda a la espera de confirmación si hay
incompatibilidad; un ZIP solo puede comprobarse después de leer su contenido.

### Regla

1. **Detección del proyecto.** `workspace-project-stack.ts` lee manifiestos y
extensiones del snapshot: dependencias de `package.json` (Angular, React,
   Next.js, Vue, Svelte), archivos `.py`, `requirements.txt`, `pyproject.toml`,
   `pom.xml`, `go.mod`, `Cargo.toml`, `composer.json`, `*.csproj` y fuentes
   C/C++. Devuelve familias tecnológicas, no una nota de calidad.
2. **Detección del entorno.** El stack del entorno sale de `stack_config` y,
   como respaldo, del nombre, la imagen y la descripción de la plantilla, con la
   misma inferencia que usa el catálogo (`inferTemplateTechnologies`).
3. **Comparación.** Una familia del proyecto que el entorno no declara produce
   una advertencia. No se compara versión, no se ejecuta el proyecto y no se
   inspecciona la calidad del código.
4. **Aviso y salida.** La tarjeta del proyecto lista las tecnologías sobrantes y
   sugiere hasta tres entornos compatibles del catálogo. Aplicar uno actualiza
   la plantilla y el límite de memoria del ejercicio; `[Cambiar entorno]` lleva
   al selector. La sugerencia solo aparece cuando ambos stacks se reconocen.
5. **Generación asistida.** El campo de tecnología del panel de IA es de solo
   lectura y sale del entorno seleccionado. Sin entorno no se genera. La
   petición envía `template_id` y `stack_config` (runtime, plataforma web,
   complementos y base de datos) además de la etiqueta legible. El backend
   valida esos identificadores y arma el prompt con el stack estructurado como
   «Entorno homologado (stack obligatorio)»; un cliente anterior que solo mande
   `technology` sigue funcionando.
6. **Publicación.** Si la incompatibilidad sigue al publicar el proyecto, el
   modal muestra el aviso y exige una casilla de confirmación; el botón de
   publicar queda deshabilitado hasta marcarla. Es una confirmación de interfaz:
   la API no la exige y no se agregó una bandera al contrato de publicación.
7. **Alcance.** Aplica solo a proyectos iniciales de `IDE_PERSISTENTE`.
   `JUEZ_EFIMERO` no pasa por este flujo. No valida sintaxis, dependencias ni
   ejecución; la preparación de dependencias (ADR-069) y la prueba del docente
   siguen siendo las señales de que el entorno puede trabajar.

### Límites y excepciones

- Si no se reconoce el stack del proyecto o del entorno, no hay aviso. Es
  preferible callar antes que inventar una incompatibilidad.
- La detección es heurística. Un `.py` auxiliar dentro de un proyecto JavaScript
  puede disparar la advertencia; por eso es un aviso y no un bloqueo.
- Un proyecto mixto puede ser intencional. La confirmación permite publicarlo y
  la decisión final queda en el docente.
- El aviso no impide publicar por API ni cambia el contrato HTTP. Tampoco
  corrige archivos, instala herramientas ni migra el proyecto a otro stack.
- La sugerencia solo considera los entornos aprobados que ya están en el paso
  del editor; no crea entornos ni solicita plantillas nuevas.
- La compatibilidad de stack no garantiza que el proyecto compile o se ejecute.
  Solo evita que la contradicción entre proyecto y entorno pase inadvertida.

## Experiencia docente (Fase 2)

El componente `ide-project-initial` se integra dentro de `step-ide-environment`
del editor de ejercicios:

- Estado del borrador, revisión, versión publicada, cantidad de archivos y flag
  de publicación.
- Importación por File System Access API, alternativa `webkitdirectory`, ZIP o
  arrastre de carpeta/ZIP.
- Árbol de archivos, editor de contenido, alta, renombrado/movimiento y borrado
  con las mismas reglas de ruta que el backend.
- Guardado con revisión optimista; un `409` muestra conflicto y no sobrescribe.
- Publicación con confirmación, resumen de cambios y versión a crear. Si el flag
  está apagado no se muestra un botón de publicación: solo el estado
  informativo.
- Historial de versiones con fecha y snapshot de solo lectura.

El mismo componente ofrece **Generar con IA** en un panel inline: formulario
corto (qué proyecto, qué debe hacer, requisitos) y el stack del entorno
seleccionado como tecnología de solo lectura, estado de espera
cancelable, propuesta con resumen, lista de archivos y vista de solo lectura, y
acciones de descartar o usar la propuesta. El panel no bloquea el árbol ni el
editor: mientras el motor responde, el docente puede seguir trabajando.
Si hay cambios sin guardar, usar la propuesta pide confirmación explícita; si el
borrador quedó en conflicto, la aplicación se bloquea hasta recargar para no
pisar una revisión más nueva.

## Consumidores adaptados

- **SpeedGrader:** el proyecto base sale de
  `workspace_project_versions.text_snapshot` de la versión de la entrega. Solo
  `base_status = version` produce `base_files`. `no_base` y `legacy_unknown` no
  inventan un diff.
- **Antiplagio:** la resta de base usa la misma versión exacta. Una entrega
  legacy o sin base no resta el espejo actual.
- **Timeline:** el keyframe inicial usa la versión exacta; el juez mantiene su
  base histórica.
- **`JUEZ_EFIMERO`:** `BaseStatus` nulo y `exercises.boilerplate` sin cambios de
  contrato.

## Inicialización del volumen individual

El punto de entrada es el flujo normal de apertura del alumno
(`POST /api/v1/environments/start` con `subject_id` y `exercise_id`), no un
endpoint separado.

1. Si el entorno es nuevo, `resolveWorkspaceSeed` lee
   `workspace_projects.current_published_version_id` y valida tenant, ejercicio
   y proyecto. La versión se guarda en el entorno antes de crear el volumen.
   Sin proyecto publicado el entorno nace en modo `none` y estado
   `not_required`.
2. Tras `EnsureVolumeExists`, el servicio pasa el snapshot normalizado al
   orquestador Docker. Un contenedor Alpine efímero, sin red y con el volumen
   montado únicamente en `/environment`, escribe archivos con uid/gid 1000. No
   se escribe ninguna ruta arbitraria del host ni se reutiliza el volumen de
   otro alumno.
3. Antes de escribir, el volumen debe estar vacío o contener un diario
   `.solv_workspace/pending.json` de la misma versión. El diario se escribe
   antes que el contenido. Un archivo extraño, un symlink, una ruta fuera del
   árbol o un contenido distinto impiden continuar y dejan el volumen intacto.
4. Al terminar se lee y compara el árbol completo contra el snapshot; solo
   entonces se escribe `.solv_workspace/init.json` con estado `verified` y se
   promueve el entorno a `setup_done`. Si falla, queda `failed` o
   `quarantined` según el caso y no se arranca el IDE.
5. Una reapertura verifica el marcador. Si el volumen fue reemplazado o perdió
   el marcador, el entorno pasa a cuarentena: no se asume que esté vacío ni se
   vuelve a sembrar. Si el alumno modificó sus archivos, el marcador sigue
   válido y su trabajo se conserva.

### Máquina de estados

| Estado | Significado | Acción al reabrir |
| :--- | :--- | :--- |
| `pending` | Versión asignada, volumen aún no preparado | Reintentar siembra |
| `volume_created` | Volumen creado; puede haber diario parcial | Reanudar solo la misma versión |
| `setup_done` | Árbol verificado y marcador escrito | Verificar marcador, no sembrar |
| `failed` | Error transitorio (Docker, escritura, actualización) | Reintentar |
| `quarantined` | Contenido ajeno, versión distinta, marcador perdido o estado ilegible | Detenerse y pedir revisión |
| `missing` | Volumen no disponible/ambiguo | Cuarentena; no recrear ni sembrar |
| `legacy` | Entorno histórico | No reinterpretar ni sembrar |
| `not_required` | Sin proyecto o juez efímero | No sembrar |

Los estados operativos (`running`, `hibernated`, `failed`) siguen siendo del
contenedor. `workspace_init_status` no los reemplaza ni se usa para decidir si
el IDE está disponible.

### Concurrencia y recuperación

Las aperturas del mismo `(tenant, alumno, materia, práctica)` se serializan con
un lock en proceso. El lock es deliberadamente local al nodo On-Premise; la
unicidad de `(tenant_id, student_id, subject_id, exercise_id)` sigue siendo la
última barrera de la base. Una caída después de crear el volumen deja el diario
`pending`; el siguiente intento completa solo archivos faltantes y vuelve a
verificar. Una caída después de escribir el marcador se resuelve como entorno
ya inicializado. No hay reparación automática de contenido modificado ni
borrado para "empezar de nuevo".

La restricción es explícita: SOLV se despliega como una única instancia de
backend On-Premise (un proceso `go run cmd/api/main.go` o su binario), y la
configuración versionada (`compose.yml`, `compose.dev-asus.yml`) no declara
réplicas ni balanceo del backend. Por eso el lock en memoria alcanza. Si en el
futuro se ejecutan dos instancias contra la misma base, la serialización deberá
moverse a PostgreSQL (por ejemplo `pg_advisory_xact_lock` por workspace) antes
de habilitar ese despliegue; la unicidad de la tabla ya impide duplicados, pero
no evita que dos procesos creen y siembren el volumen a la vez.

### Entregas y aislamiento

`submissions.workspace_project_version_id` se toma del entorno, no de la
publicación actual. Por eso un alumno con V1 conserva V1 aunque el docente
publique V2; un entorno nuevo recibe V2. SpeedGrader, antiplagio y timeline
consumen esa misma referencia. `JUEZ_EFIMERO` no pasa por este camino y mantiene
su contrato.

La API traduce los rechazos a `409 workspace_seed_volume_not_empty`,
`409 workspace_seed_conflict`, `409 workspace_seed_quarantined` o
`409 workspace_seed_version_invalid`; un fallo operativo del volumen responde
`503 workspace_seed_unavailable`. El solver muestra el mensaje del backend y no
presenta un IDE vacío como éxito.

## Migración y rollback

La migración es `00053_workspace_projects.sql` (goose, solo `Up` en el runner).
Incluye tablas, columnas, FKs, checks, backfill de `legacy_unknown`, backfill de
referencias legacy, hashes SHA-256 calculados con la función nativa de
PostgreSQL y triggers de inmutabilidad.

El `Down` del archivo es destructivo y no se ejecuta en producción. Un rollback
real exige: detener la publicación, exportar proyectos/versiones, retirar primero
entregas y entornos que referencian versiones, y recién después eliminar las
tablas. No se debe borrar una versión para "liberar" una entrega.

## Consecuencias

- Una entrega queda atada a la versión que el entorno recibió, no al estado
  actual del ejercicio.
- El docente puede seguir editando el borrador sin romper entregas anteriores.
- El espejo `boilerplate` se conserva para compatibilidad, pero ya no es la
  fuente de verdad del proyecto publicado.
- La inicialización ocurre en el flujo normal de apertura; no hay un endpoint
  aislado que el alumno o el frontend deban recordar invocar.
- El coste es un modelo adicional, triggers de integridad y operaciones Docker
  efímeras para sembrar y verificar; la ventaja es que el historial y el trabajo
  del alumno dejan de depender de una edición posterior del ejercicio.
- La detección de compatibilidad es una advertencia de herramientas, no una
  garantía de que el proyecto compile o se ejecute; validar el código sigue
  siendo responsabilidad del docente.

## Verificación

- `go test ./internal/core/services -run TestWorkspaceImport` cubre carpeta,
  normalización, límites, binarios, traversal, duplicados y archivos ZIP
  inseguros.
- `TestWorkspaceProjectPhase2ImportAuthorizationAndPublishFlow` y
  `TestWorkspaceProjectPhase2HTTPImportContracts` usan PostgreSQL 18
  descartable y verifican autorización por docente/tenant, `JUEZ_EFIMERO`
  rechazado, importación, conflicto de revisión, publicación fail-closed,
  publicación autorizada e inmutabilidad del historial.
- `workspace-project.utils.spec.ts`, `workspace-project-import.spec.ts` e
  `ide-project-initial.component.spec.ts` cubren selección, árbol, edición,
  guardado, conflicto, errores de importación y publicación deshabilitada.
- `practice_environment_workspace_seed_test.go` e
  `practice_environment_workspace_seed_internal_test.go` cubren asignación,
  entorno sin proyecto, legacy, reapertura verificada, marcador perdido,
  cuarentena, concurrencia, V1/V2 y hash inválido.
- `workspace_projects_phase3_test.go` usa PostgreSQL 18 descartable para
  comprobar asignación persistida, reapertura sin re-siembra, entrega atada a
  V1, aislamiento entre alumnos y tenants, `JUEZ_EFIMERO` y cuarentena.
- `workspace_projects_phase3_docker_test.go` usa volúmenes Docker descartables
  para comprobar árbol anidado, contenidos, marcador, modificación del alumno,
  rechazo de V2, archivos ajenos y reanudación del diario.
- `workspace_project_generate_test.go` cubre la generación con proveedor
  simulado: normalización y orden, traversal, rutas absolutas, separadores
  ambiguos, duplicados, binarios, campos desconocidos, contenido extra, topes de
  archivo y cantidad, `JUEZ_EFIMERO`, otro tenant, IA deshabilitada, proveedor
  ausente, timeout y límites del formulario.
- `workspace_project_generate_handler_test.go` ejercita el endpoint completo:
  200 con propuesta, 403 para estudiante, 404 para ejercicio de otro tenant, 422
  para juez y para rutas inseguras, 400 para JSON inválido y 409 con la IA
  deshabilitada.
- `ide-project-initial.component.spec.ts` agrega apertura del generador,
  propuesta visible sin guardar, confirmación por cambios locales, bloqueo en
  conflicto, descarte, error del proveedor y guardado posterior con el flujo
  existente.
- `workspace-project-stack.spec.ts` cubre la detección de familias y la
  comparación con el entorno. `ide-project-initial.component.spec.ts` suma los
  casos de aviso, generación atada al entorno, sugerencia de entornos
  compatibles y confirmación antes de publicar.
