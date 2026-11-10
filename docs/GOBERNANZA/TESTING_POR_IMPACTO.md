# Testing por impacto

Estado: vigente · Política v4

## Qué resuelve

Cada cambio ejecuta el conjunto de tests que aporta confianza para su impacto
real. El objetivo no es ejecutar menos por ejecutar menos: es evitar que un
cambio visual pague backend, PostgreSQL, Docker o Playwright sin una dependencia
demostrada, y mantener la cobertura cuando el cambio toca autenticación,
persistencia, aislamiento o contratos.

La selección depende de una política única, de los grafos de importación del
backend y del frontend y de las reglas críticas. No depende de una lista de
archivos escrita en el workflow ni de la interpretación de varias skills.

## Fuente única de verdad

`tools/test-impact/policy.json` es la única fuente de reglas. Allí viven:

- las suites, su entorno y si son reutilizables desde caché;
- los niveles L0, L1, L2 y L3;
- las rutas que obligan a una ejecución completa;
- las áreas funcionales, su nivel y los flujos E2E que activan;
- las reglas críticas de seguridad y sus suites obligatorias;
- las brechas de cobertura que una regla crítica declara;
- el límite de archivos y los eventos que exigen ejecución completa;
- el catálogo de flujos E2E y la configuración de Playwright de cada uno.

El workflow de GitHub Actions, el ejecutor local y las instrucciones de agentes
invocan esa política; no repiten sus globs ni sus umbrales. Para cambiarla se
edita `policy.json`, se ejecuta `cli.mjs validate` y se actualizan los tests del
selector.

## Arquitectura del sistema

El flujo está separado en etapas verificables:

1. **Análisis**: `lib/git.mjs` obtiene el diff contra la base, incluidos el
   working tree y los archivos sin seguimiento. Los artefactos solo se excluyen
   en la raíz y en directorios de salida conocidos: un fuente real en
   `backend/`, `frontend/src/` o `tools/test-impact/` no se oculta por llamarse
   `impact-report.json`. Las fuentes ignoradas relevantes se incorporan al diff.
2. **Dependencias**: `lib/backend-graph.mjs` y `lib/frontend-graph.mjs` leen
   imports, templates, estilos, rutas y contratos compartidos.
3. **Clasificación**: `lib/select.mjs` cruza cada ruta con las áreas, reglas
   críticas y triggers globales de la política.
4. **Selección**: se calcula el nivel, se eligen las suites y se suprimen las
   validaciones que no corresponden al nivel, salvo que una regla crítica o una
   spec E2E modificada las exija.
5. **Plan**: `cli.mjs` produce un plan con `schema`, `contractVersion`,
   `policyDigest`, `source` (tipo, base, commits resueltos, evento, HEAD y alcance),
   nivel, razones, decisiones, paquetes, specs, flujos y cobertura faltante.
6. **Integridad**: `lib/plan-integrity.mjs` reconstruye el diff y vuelve a
   calcular las obligaciones desde la política vigente antes de ejecutar.
7. **Ejecución**: `run.mjs` recorre solo las suites del plan. `--all` significa
   «todas las suites del plan», no «todas las suites de la política».
8. **Resultados**: cada suite queda en `executed`, `cached`, `simulated`,
   `skipped`, `pending` o `failed`. El informe se guarda con `--report`, lleva
   un recibo de ejecución y un sidecar firmado de evidencia; se valida con
   `cli.mjs report --plan`.
9. **Agregación**: `cli.mjs aggregate` comprueba que los informes parciales de
   CI cubren todas las suites obligatorias del plan y que cada job usa su propia
   clave.
10. **Reutilización**: solo las suites deterministas marcadas en la política
    pueden usar caché, y solo con una clave de política, runner, comando,
    toolchain, entorno, contenido y una firma de la entrada.

## Integridad del plan

El runner no confía en el JSON del plan. Antes de ejecutar cualquier suite,
`validatePlan` comprueba de forma independiente:

1. `schema` y `contractVersion` compatibles con el motor instalado.
2. `policyVersion` compatible con `policy.json`.
3. `policyDigest` igual al SHA-256 de la política vigente. Un plan generado
   antes de un cambio de política se rechaza.
4. `source` con tipo `git` o `files`, base y commits resolubles, evento y
   alcance completos; para `git`, `headCommit` debe ser el HEAD real y
   `source.event` debe coincidir con `GITHUB_EVENT_NAME` cuando el job lo
   define. Un plan no puede declararse local para evitar las reglas de un
   evento programado o de release.
5. Reconstrucción del diff desde `source`, incluido el working tree cuando
   corresponde. Si el árbol tiene cambios no representados, el plan se rechaza.
6. Correspondencia exacta entre los archivos modificados del plan y el diff
   reconstruido, incluidos estados y rutas de renombrado.
7. Inventario completo de suites: faltantes y desconocidas se rechazan.
8. Alcance exacto de cada suite: `selected`, paquetes, specs, recursos, build de
   backend y frontend, y modo completo.
9. Nivel, título, reglas críticas, rutas desconocidas y cobertura faltante
   comparados con el cálculo vigente.
10. Decisiones de nivel y de cada suite coherentes con las obligaciones.

El runner no acepta un plan de un commit anterior ni un plan generado sin
`--working-tree` cuando hay cambios locales. Un plan de CI solo es válido en el
checkout que coincide con su `headCommit`. Esta comprobación impide el falso
verde de un plan de documentación ejecutado contra un árbol con cambios
críticos.

Un plan que oculte una suite, vacíe las reglas críticas, cambie el nivel,
elimine cobertura faltante o reduzca el build se rechaza con código de salida
distinto de cero. No se ejecuta una versión debilitada ni se escribe un
resultado verde: el informe queda en `validationStatus: rejected`.

El `policyDigest` no es una firma de ejecución: cualquiera que lea la política
puede recalcularlo. La autoridad es la reconstrucción: aunque el plan tenga un
digest correcto, sus suites y archivos deben coincidir con lo que produce el
selector sobre el diff y la política actuales. Para probar ejecución se usan el
recibo HMAC y el sidecar de evidencia, no el digest del plan.

### Plan sintético y plan de CI

Un plan `source.kind: files` es explícito y útil para diagnosticar un archivo o
un icono sin esperar a un diff. No prueba el diff real del árbol de trabajo, así
que el runner lo marca `synthetic: true` y nunca lo declara validación completa,
aunque todas sus suites terminen bien. Un `--all` sintético termina con código
2; `cli report` devuelve estado parcial y `cli aggregate` no lo acepta como
cobertura de CI. Un plan sin suites obligatorias tampoco puede declarar
`validated: true`.

CI genera el plan desde git en un job y lo comparte como artefacto. Los jobs
consumidores lo validan otra vez con `cli.mjs outputs` o con `run.mjs`; no
pueden sustituirlo por un plan más pequeño. El agregado final exige que cada
suite obligatoria tenga un informe con estado `executed` o `cached`, código 0,
un recibo HMAC y un sidecar `*.evidence.json` verificable.

## Contrato de ejecución

Hay tres conceptos distintos:

| Concepto | Comando | Qué significa |
| :--- | :--- | :--- |
| Plan completo | `cli.mjs plan` | Todas las suites necesarias según el impacto |
| Ejecución parcial | `run.mjs --partial --suite <id>` | Una parte deliberada del plan; no valida el cambio completo |
| Ejecución completa | `run.mjs --all` | Todas las suites obligatorias del plan terminaron correctamente |

Reglas del contrato:

- `--all` ejecuta todas las suites seleccionadas y obligatorias del plan.
- `--suite` exige `--partial`. Sin `--partial` el runner falla y explica que una
  suite aislada no valida el cambio completo.
- `--if-selected` permite que un job condicional no falle cuando su suite no fue
  seleccionada. El informe la deja en `skipped` y conserva las demás en
  `pending`; no declara validación.
- Una suite obligatoria no seleccionada o desconocida es un error.
- Una ejecución parcial deja `partial: true` y enumera las suites pendientes.
- Un fallo detiene la ejecución y deja `failed` en el informe; el éxito global
  es imposible.
- Un informe parcial nunca puede tener `validationStatus: complete` ni
  `validated: true`.
- Una suite cacheada cuenta como cobertura solo si su clave y su resultado son
  válidos; se muestra como `cached`, no como `executed`.
- Una suite omitida por irrelevancia es `skipped`; una suite obligatoria que
  todavía no se ejecutó es `pending`. No se mezclan.

`run.mjs --report <ruta>` escribe el contrato del informe, le añade un recibo
HMAC con `runId`, `jobId`, plan, suite y resultado, y escribe un sidecar
`<ruta>.evidence.json` firmado con el detalle de las suites. El recibo del
informe no se acepta solo: `cli.mjs report --plan` exige el sidecar, comprueba
su firma, su digest, su `runId`, su `jobId` y que sus suites coincidan. Devuelve
0 solo para validación completa real, 1 si hay fallos y 2 si es parcial,
simulada, sintética o sin suites obligatorias.

`cli.mjs aggregate` hace lo mismo entre jobs: si una suite obligatoria no tiene
informe firmado, evidencia verificable y estado real, el agregado falla aunque
todos los jobs hayan terminado en verde. En CI cada job usa su propia clave
`SOLV_TEST_RECEIPT_KEY_<JOB>`; `ci-summary` recibe todas y exige que el `jobId`
del informe corresponda a la clave que lo firma. Un job no puede firmar el
informe de otro ni reutilizar una clave genérica. La firma y la evidencia no
sustituyen la ejecución: identifican el run, el job y las suites ejecutadas,
pero un proceso con acceso a la clave puede fabricar ambos.

En local el runner crea `tools/test-impact/.cache/receipt.key` con permisos
restrictivos; esa clave evita ediciones accidentales y de terceros sin acceso a
la cuenta, pero no protege frente al mismo usuario que puede leerla. La garantía
local no se presenta como autenticidad criptográfica frente a ese agente. Los
tests del motor usan `SOLV_TEST_TRUST_MODE=local` y no pueden producir informes
firmados como jobs de CI.

## Niveles de validación

La política define cuatro niveles. El nivel se calcula por riesgo e impacto, no
por cantidad de archivos. Una línea de autenticación puede ser L3; veinte iconos
pueden ser L0.

| Nivel | Propósito | Qué ejecuta | Qué evita |
| :--- | :--- | :--- | :--- |
| L0 | Validación inmediata | Gates locales cuando hay estilos o scripts, validación de recursos, spec del componente si existe | Backend, PostgreSQL, Docker, E2E y build sin necesidad |
| L1 | Validación del módulo | Specs unitarias del módulo y sus dependencias, tipos y build del lado afectado | Integración, Docker y E2E genéricos |
| L2 | Validación de integración | Unitarias del lado, integración PostgreSQL, E2E del área y build del lado | Docker y migraciones si no corresponden |
| L3 | Validación integral | Suites completas del lado, integración, seguridad, Docker o migraciones según la regla, E2E del catálogo | Nada que la regla crítica exija |

Las suites completas de un lado solo se activan por una regla crítica, un
cambio de contrato compartido, una configuración global, un cambio estructural
o una ruta que el selector no puede clasificar. Un cambio visual localizado no
hereda el nivel de la aplicación entera.

## Reglas críticas y escalamiento

Autenticación, sesiones, autorización, aislamiento de tenants, ejecución de
contenedores, WebSockets, barrera AST, contratos compartidos y migraciones
tienen reglas propias. Son un suelo de seguridad, no una excusa para ejecutar
todo:

- ejecutan todos los paquetes unitarios del lado afectado;
- ejecutan integración con PostgreSQL;
- ejecutan Docker cuando se toca la ejecución de contenedores;
- añaden los flujos E2E de seguridad;
- activan auditoría de seguridad;
- en migraciones, ejecutan la cadena completa y todos los flujos E2E.

Las reglas son aditivas. Un cambio de autenticación junto a un icono conserva
la validación del icono y suma seguridad, integración y el nivel L3. Nunca se
elige una regla y se descarta otra que aporte cobertura.

### WebSockets

La regla `websocket-backend` cubre el handler, el hub y los handlers efímeros
de docentes, el registro de rutas y los emisores de mensajes. Exige backend
unitario completo, integración, build de backend, seguridad y los flujos de
sesión persistente y concurrencia. El handshake exige una sesión válida por
cookie `solv_session` o `Authorization: Bearer`, ignora `X-User-Id`/
`X-Tenant-Id` y valida `Origin` contra la allowlist de CORS. El hub indexa por
`(tenant_id, user_id)`, el logout persiste la revocación en PostgreSQL, el
middleware HTTP rechaza los JWT revocados y las conexiones abiertas se cierran;
las otras instancias descubren la revocación por sondeo de la misma tabla. El
terminal docente exige rol, tenant, permiso sobre el contenedor y autorización
en cada `exec`.
Las pruebas de estas rutas pasan; lo que sigue
sin verificarse en este entorno es:

- la ejecución completa de las suites de integración, Docker y E2E; las de
  revocación sí se ejecutaron contra PostgreSQL de test;
- el cierre por expiración o revocación en un navegador real;
- la reconexión del terminal frontend y la liberación de listeners en E2E.

La regla `websocket-frontend` cubre el terminal del docente. Exige specs del
grafo, build, seguridad, los flujos de sesión y concurrencia, y declara cuatro
brechas: handshake sin token explícito, cierre sin reconexión ni distinción de
token expirado, mensajes de otro tenant y liberación de listeners al cerrar. El
servicio de avisos de avance del estudiante hace polling HTTP, no WebSocket, y
por eso no activa esta regla crítica.

Las pruebas agregadas cubren el índice del hub por tenant, el rechazo de un
token expirado, el rechazo de una sesión después de logout, el cierre de una
conexión abierta y la autorización del terminal docente. Las pruebas unitarias
de WebSocket y seguridad pasan; la validación integral sigue pendiente hasta
que se ejecuten las suites de integración y E2E en CI.

Mostrar una suite no demuestra que la brecha esté cubierta. Por eso el plan
publica `coverageGaps` y los tests de regresión comprueban que no se vacíe al
manipular el plan. La suite `security` ejecuta las pruebas
`TestWebSocketSecurityRejectsMissingToken`,
`TestWebSocketSecurityRejectsIdentityHeaders`,
`TestWebSocketSecurityRejectsUntrustedOrigin` y
`TestWebSocketSecurityAcceptsValidSession`. Las cuatro pasan: el handshake sin
token devuelve 401, las cabeceras de identidad no sustituyen la sesión, un
origen no permitido devuelve 403 y una sesión válida abre la conexión. La
cobertura de integración y E2E real de las brechas restantes requiere
PostgreSQL, Docker y Playwright, y no se ejecutó en esta fase.

El selector escala a ejecución completa cuando:

- no hay diff verificable o el diff está vacío;
- una ruta no tiene área, regla crítica ni trigger global;
- cambia una configuración global, un contrato compartido o una interfaz;
- se elimina o renombra código;
- el plan supera el límite de archivos;
- no puede justificar una selección reducida con la evidencia disponible.

El escalamiento se explica en `levelReasons`; no se presenta como una decisión
silenciosa.

## Plantillas Angular, shell y build

El modelo distingue seis casos:

1. **HTML estático o decorativo**: no tiene directivas, bindings, eventos ni
   interpolación. Si tiene spec de componente, se ejecuta esa spec; si no, no
   exige una compilación artificial.
2. **Plantilla Angular con lógica**: contiene `@if`, `@for`, `@switch`, `{{ }}`,
   bindings, eventos, referencias o `*ngIf`. Se detecta por sintaxis, no por una
   lista de nombres de archivo.
3. **Componente con spec**: el grafo de imports resuelve la spec propia y las
   specs que consumen el componente.
4. **Componente sin spec**: si la plantilla tiene lógica, el plan exige build
   de frontend y registra el archivo en `uncovered.frontend`. No selecciona las
   128 specs: no cubren esa plantilla y encarecen sin aportar señal.
5. **Shell global**: `frontend/src/index.html` tiene un área propia, nivel L2,
   build de frontend y un flujo de humo `01-core-lifecycle`. No arrastra todas
   las specs ni todos los E2E.
6. **HTML que depende de servicios, rutas o contratos**: lo resuelve el área o
   la regla crítica correspondiente; si la plantilla no tiene spec, se suma el
   build como validación mínima.

El bootstrap real (`main.ts`, `app.config.ts`, `app.ts`, `app.html` y
`app.scss`) tiene una regla propia L3: ejecuta todas las specs, los 24 flujos
E2E y el build de frontend. No usa los nombres antiguos `app.component.*`, que
no existen en Angular 22. `app.html` conserva el build aunque sea visual porque
es el componente raíz y no una plantilla decorativa aislada.

Un cambio de autenticación frontend conserva el build aunque el archivo sea
visual. La optimización `allVisualOnly` solo puede retirar el build cuando no
hay reglas críticas, plantilla sin spec ni bootstrap; nunca anula una obligación
crítica ni el componente raíz.

## Cómo selecciona

1. Lee el diff contra la base del PR o del push. Si no hay base, el diff está
   vacío o la base no existe, no adivina: ejecuta todo.
2. Clasifica cada ruta con las áreas de la política. Una ruta sin área, sin regla
   crítica y sin regla global es desconocida y activa la ejecución completa.
3. Para Go, construye el grafo de imports y recorre dependencias inversas:
   cambiar un paquete ejecuta los paquetes que lo importan.
4. Para Angular, resuelve imports relativos y alias del `tsconfig`, enlaces de
   `templateUrl`/`styleUrl` y dependencias SCSS. Cambiar un componente, pipe o
   servicio ejecuta las specs que lo importan de forma directa o transitiva.
5. Calcula el nivel máximo de las áreas, reglas críticas, triggers y cambios
   estructurales. Un cambio exclusivamente visual de un componente baja a L0;
   un estilo global, una ruta o un contrato no baja.
6. Añade los flujos E2E del área solo desde L2. Las specs E2E modificadas
   directamente y las reglas críticas conservan sus flujos en cualquier nivel.
7. Resuelve la plantilla: lógica sin spec exige build y cobertura faltante; HTML
   decorativo no dispara todas las specs; SCSS huérfano conserva la estrategia
   conservadora.
8. Reporta los archivos de código que quedaron sin tests o specs. El reporte no
   los oculta ni inventa cobertura.

Una eliminación o un renombrado de código activa el alcance completo del lado
afectado. Un icono o un asset público valida el recurso y no ejecuta Go,
PostgreSQL, Docker ni Playwright. Un cambio visual de una pantalla conserva su
spec de componente, pero no paga E2E ni build sin necesidad.

## Suites

| Suite | Qué comprueba | Entorno | Caché |
| :--- | :--- | :--- | :--- |
| `gates` | Estilo, tokens, tipografía, fechas, terminología y sus tests | Node | Sí |
| `asset-validation` | Estructura y firma de recursos estáticos | Node | Sí |
| `backend-unit` | Paquetes Go afectados y dependientes, sin Docker ni PostgreSQL | Go | Sí |
| `backend-integration` | Integración Go contra PostgreSQL efímero | Go + PostgreSQL | No |
| `backend-docker` | Integración que arranca contenedores reales | Go + Docker | No |
| `frontend-unit` | Specs Angular afectadas por el grafo de imports | Node + Vitest | Sí |
| `frontend-e2e` | Flujos Playwright por área, contrato o seguridad | Stack completo | No |
| `migrations` | Migraciones up, down y re-aplicación | Go + PostgreSQL | No |
| `build` | Build de backend y/o frontend según el plan | Go + Node | No |
| `security` | `go vet`, pruebas negativas de WebSocket, dependencias frontend y secretos | Go + Node | No |

La caché solo se usa para suites deterministas y con una clave que incluye el
plan estable, el HEAD, el diff versionado, el diff preparado, el contenido de
los archivos sin seguimiento y de las fuentes ignoradas relevantes, el hash de
la política, el digest del runner, el comando efectivo, la versión de Node, Go y
npm, los lockfiles de dependencias instaladas y las variables de entorno que
afectan a las suites (`CI`, `NODE_ENV`, `TZ`, `DATABASE_URL`, `TEST_DB_*`,
`JWT_SECRET` y `SOLV_*`, salvo la identidad efímera de la ejecución
`SOLV_TEST_EXECUTION_ID` y `SOLV_TEST_RUN_ID`). Los valores se incluyen como
hash, no en claro. Cada
entrada lleva una firma HMAC y una entrada alterada se rechaza. Un cambio en
cualquiera de esos inputs invalida la entrada. Fallos y dry-runs nunca se
escriben en caché. Integración, Docker, migraciones, E2E, build y seguridad no
se cachean. Una suite cacheada sigue apareciendo en el informe con su estado y
su tiempo; no se presenta como ejecutada. Si Git no puede leer HEAD, el diff o
el estado del working tree, el motor no calcula una clave vacía: la ejecución
falla o escala a cobertura completa, nunca reutiliza una caché dudosa.

## Aislamiento de la base de pruebas

La integración y las migraciones aceptan únicamente `TEST_DB_DSN` o
`TEST_DATABASE_URL` con host loopback, puertos de test y una base cuyo nombre
empiece por `solv_test`, `solv_migration_test` o `solv_e2e`. `DATABASE_URL` del
despliegue se ignora para elegir el destino. `test-db.sh` exige host, puerto,
usuario y contraseña de test explícitos, valida el destino antes de crear nada y
no cae a `solv_dev_db`.

El workflow E2E (`e2e.yml`) usa un proyecto Compose propio y volúmenes propios;
nunca ejecuta `down -v` sobre los volúmenes del despliegue. Los recursos Docker de
las pruebas se etiquetan con `solv.test.run` y la limpieza solo toca los del run
actual o residuos antiguos etiquetados.

## Interfaz para agentes

El agente no necesita leer varios documentos para decidir qué ejecutar. Debe
usar el plan como contrato:

```bash
node tools/test-impact/cli.mjs plan --base origin/main --working-tree --format text
```

La salida incluye:

- el nivel y su título;
- las suites seleccionadas y las omitidas;
- el motivo de cada decisión;
- las razones del nivel;
- paquetes, specs y flujos E2E afectados;
- archivos sin cobertura y brechas críticas declaradas;
- el modo `targeted`, `full` o `neutral`.

Flujo obligatorio al terminar una implementación:

1. identificar los archivos realmente cambiados;
2. ejecutar el plan con la base correcta;
3. revisar el nivel y las decisiones;
4. ejecutar las suites seleccionadas, no todas las suites por costumbre;
5. no ejecutar una suite que el plan omitió sin una razón registrada;
6. recoger el informe y los tiempos con `--report`;
7. leer el informe con `cli.mjs report` y comprobar `validated`;
8. informar qué se ejecutó, qué se simuló, qué quedó pendiente y por qué;
9. declarar la cobertura faltante y las limitaciones;
10. si el plan escala a completo, no reducirlo por conveniencia.
Si el código cambia después del plan, se vuelve al paso 1: no se reutiliza el
plan ni el informe anterior. Una suite en caché se informa como `cached`, no
como una ejecución nueva, y un dry-run nunca cuenta como ejecución.

Un agente no puede afirmar que un cambio está validado si el informe es parcial,
simulado o sintético, si alguna suite obligatoria está pendiente o si el plan
declaró cobertura faltante que el cambio necesitaba. Si ejecuta una suite fuera
del plan, debe justificarlo y registrarlo; no debe presentarla como si el plan la
hubiera pedido.

## Comandos

```bash
node tools/test-impact/cli.mjs validate
node tools/test-impact/cli.mjs plan --base origin/main --working-tree --format text
node tools/test-impact/cli.mjs explain backend/internal/core/services/auth_service.go
node tools/test-impact/cli.mjs report --file impact-report.json --plan impact-plan.json

# Ejecuta todas las suites obligatorias del plan y escribe el informe.
node tools/test-impact/cli.mjs plan --base origin/main --working-tree \
  --output impact-plan.json
node tools/test-impact/run.mjs --all --plan impact-plan.json \
  --report impact-report.json
node tools/test-impact/cli.mjs report --file impact-report.json --plan impact-plan.json

# Ejecuta una suite de un plan guardado. Una ejecución parcial debe declararse.
node tools/test-impact/cli.mjs plan --base origin/main --working-tree \
  --output impact-plan.json
node tools/test-impact/run.mjs --partial --suite frontend-unit \
  --plan impact-plan.json --report impact-report-frontend-unit.json

# Job condicional: si la suite no fue seleccionada, no falla y no valida.
node tools/test-impact/run.mjs --partial --suite build --if-selected \
  --plan impact-plan.json --report impact-report-build.json

# Agrega los informes parciales de CI contra las suites obligatorias del plan.
node tools/test-impact/cli.mjs aggregate --plan impact-plan.json --dir impact-reports

# Benchmark reproducible de quince escenarios aislados.
node tools/test-impact/bench.mjs
node tools/test-impact/bench.mjs --format md
```

`--all` recorre las suites de `policy.json`, pero ejecuta únicamente las que el
plan marcó como seleccionadas. Una suite pedida con `--suite` sin `--partial`
termina en error. `--dry-run` imprime los comandos y marca las suites como
`simulated`; no incrementa `executed`, no escribe caché y no valida el cambio.
Un `--all --dry-run` termina con código 2, igual que un plan sintético, para que
un consumidor que solo mire el código de salida no lo confunda con un verde.

## Informe y métricas

`run.mjs --report <ruta>` escribe un JSON con:

- identidad del plan: contrato, política, digest de política, digest del plan,
  modo, nivel y `source`;
- `runId`, `jobId` y recibo HMAC de la ejecución;
- `partial`, `dryRun`, `synthetic` y `validated`;
- suites seleccionadas, omitidas, ejecutadas, cacheadas, simuladas, pendientes y
  fallidas;
- estado, código y tiempo real de cada suite;
- comandos y resultados de los runners;
- totales calculados y estado de validación.

`cli.mjs report --file <ruta> --plan <plan>` valida el contrato global, la
correspondencia con el plan y el recibo, y devuelve 0 solo si `validated` es
verdadero. Si faltan totales, una suite obligatoria no aparece, un dry-run
declara suites ejecutadas, un plan sintético se declara completo o la firma no
coincide, el informe se rechaza. El tiempo de análisis y planificación se mide en
`plan.metrics`; no se inventan estimaciones de suites que no se ejecutaron.

## Integración con GitHub Actions

`.github/workflows/ci.yml` calcula el plan en el job `impact` y publica el
artefacto `impact-plan`. Los jobs de assets, backend, integración, Docker,
frontend, E2E, migraciones, build y seguridad descargan ese plan, ejecutan su
suite con `--partial --suite <id> --if-selected` y publican un artefacto
`impact-report-<id>`.

- `push` y `pull_request`: plan por impacto.
- `schedule` y ejecuciones manuales: plan completo.
- Cambios en la política, el selector, el CI, la infraestructura o las
  dependencias: plan completo.
- Un cambio visual L0 no instala Go ni ejecuta `go vet`; el job de lint conserva
  el chequeo de formato y vet solo para cambios de backend.
- `test-assets` existe para validar iconos e imágenes sin levantar el stack.
- El job de build usa los scopes del plan: un endpoint no instala dependencias
  frontend, y un icono no compila ninguno de los dos lados.
- Si el selector falla o no puede comparar el diff, el job falla; no se degrada
  a «sin pruebas».
- Cada job recibe una clave propia `SOLV_TEST_RECEIPT_KEY_<JOB>` y
  `SOLV_TEST_EXECUTION_ID` desde `github.run_id`/`github.run_attempt`.
- Antes de habilitar el pipeline hay que crear los secretos por job. Si falta la
  clave de un job, ese job falla de forma explícita; no se genera una clave
  compartida ni se acepta la clave de otro job.
- Cada informe se firma con el plan, la ejecución y `GITHUB_JOB`; su sidecar
  `*.evidence.json` viaja en el mismo artefacto. `ci-summary` recibe todas las
  claves, rechaza informes sin evidencia, de otra ejecución, de otro job o
  duplicados, y no acepta una clave genérica en CI.

Todos los jobs consumidores hacen checkout con `fetch-depth: 0`: el runner
reconstruye el diff desde `source.baseCommit` y `source.headCommit`, y un clon
superficial no contiene esos objetos. Sin esa historia, la validación rechaza el
plan y el job falla en lugar de ejecutar una versión debilitada.

`ci-summary` descarga el plan y todos los informes parciales, ejecuta
`cli.mjs aggregate` y después verifica el resultado de cada job. Una suite
obligatoria sin informe, simulada, pendiente o fallida bloquea el agregado. Un
job no seleccionado debe quedar `skipped`; un job seleccionado debe terminar en
`success`. La firma y la identidad de job se comprueban además del estado
declarado por el JSON.

`golangci-lint` y ESLint siguen siendo informativos mientras exista el backlog
declarado en `CONTRIBUTING.md`; sus pasos se identifican como advisory y no se
presentan como validación de seguridad. El job `lint` publica `advisory` y
`ci-summary` deja una advertencia visible, sin convertirla en gate obligatorio.
La suite `security` ejecuta `go vet` cuando el plan toca backend, las pruebas
negativas de WebSocket contra el handler real, auditoría de dependencias del
frontend y secretos.

El flujo E2E vive en `.github/workflows/e2e.yml` y se ejecuta en modo programado
o manual. Usa el mismo plan, un stack aislado con base y volumen propios, y
publica su propio informe. La ejecución real de GitHub Actions no se puede
validar solo con leer el YAML.

## Escenarios de referencia

Estos quince planes se generaron con `bench.mjs` sobre el selector y los grafos
reales, con un cambio aislado por escenario. No usan el working tree completo:
el árbol de trabajo actual contiene modificaciones previas y no representa un
plan único.

| # | Escenario | Nivel | Suites seleccionadas | Build | Omitidas principales | Cobertura faltante | Coste estimado |
| ---: | :--- | :--- | :--- | :--- | :--- | :--- | ---: |
| 1 | Icono SVG | L0 | `asset-validation` | No | Backend, PostgreSQL, Docker, E2E, build, seguridad | — | 1 s |
| 2 | HTML decorativo | L0 | `frontend-unit` | No | Backend, PostgreSQL, Docker, E2E, build, seguridad | — | 45 s |
| 3 | HTML Angular con bindings y sin spec | L1 | `build` | Frontend | Backend, PostgreSQL, Docker, E2E, specs | `admin-environment-requests.component.html` | 45 s |
| 4 | Formulario | L1 | `frontend-unit`, `build` | Frontend | Backend, PostgreSQL, Docker, E2E | — | 90 s |
| 5 | Servicio frontend | L1 | `frontend-unit`, `build` | Frontend | Backend, PostgreSQL, Docker, E2E | — | 90 s |
| 6 | Endpoint Go | L2 | `backend-unit`, `backend-integration`, `frontend-e2e`, `build` | Backend | Docker, migraciones, seguridad | — | 200 s |
| 7 | Autenticación backend | L3 | `backend-unit`, `backend-integration`, `frontend-e2e`, `build`, `security` | Backend | Docker, migraciones | — | 275 s |
| 8 | Autenticación frontend | L3 | `frontend-unit`, `frontend-e2e`, `build`, `security` | Frontend | Backend, PostgreSQL, Docker, migraciones | — | 230 s |
| 9 | WebSocket backend | L3 | `backend-unit`, `backend-integration`, `frontend-e2e`, `build`, `security` | Backend | Docker, migraciones | 7 brechas declaradas | 300 s |
| 10 | WebSocket frontend | L3 | `frontend-unit`, `frontend-e2e`, `build`, `security` | Frontend | Backend, PostgreSQL, Docker, migraciones | 4 brechas declaradas | 195 s |
| 11 | Motor del juez | L3 | `backend-unit`, `backend-integration`, `backend-docker`, `frontend-e2e`, `build`, `security` | Backend | Migraciones | — | 615 s |
| 12 | Migración SQL | L3 | `backend-unit`, `backend-integration`, `frontend-e2e`, `migrations`, `build`, `security` | Backend | Docker | — | 870 s |
| 13 | Infraestructura Docker | L3 | `backend-unit`, `backend-integration`, `backend-docker`, `frontend-e2e`, `build`, `security` | Backend | Migraciones | — | 615 s |
| 14 | Contrato compartido | L3 | `backend-unit`, `backend-integration`, `frontend-unit`, `frontend-e2e`, `build`, `security` | Backend + frontend | Docker, migraciones | — | 380 s |
| 15 | `index.html` | L2 | `frontend-e2e`, `build` | Frontend | Backend, PostgreSQL, Docker, specs, seguridad | — | 80 s |

Razones de selección y omisión verificadas por el benchmark:

- **Icono**: el recurso estático aislado solo activa su validación. El resto se
  omite porque no hay paquete, spec, flujo ni compilación afectados.
- **HTML decorativo**: el grafo resuelve una spec de componente y no hay lógica
  de plantilla; no se compila ni se ejecuta E2E.
- **HTML con bindings y sin spec**: no hay spec que ejecutar y una compilación
  de frontend cubre los bindings y las directivas. Se registra la falta de spec
  en lugar de fingir cobertura con 128 specs.
- **Formulario y servicio frontend**: specs del grafo más build de frontend. No
  se paga backend, PostgreSQL, Docker ni E2E de área en L1.
- **Endpoint Go**: paquete afectado, integración PostgreSQL, flujo de ciclo de
  vida y build de backend. Docker y migraciones se omiten.
- **Autenticación backend**: 14 paquetes unitarios, integración, dos flujos de
  seguridad, build y auditoría. No se ejecuta Docker ni migraciones.
- **Autenticación frontend**: 128 specs, dos flujos, build y seguridad. El build
  se conserva aunque el cambio sea visual porque la regla `auth-frontend` es
  crítica.
- **WebSocket backend**: 14 paquetes, integración, tres flujos, build, seguridad
  y siete brechas declaradas. No se ejecuta Docker.
- **WebSocket frontend**: dos specs del grafo, tres flujos, build, seguridad y
  cuatro brechas declaradas. El servicio de avisos por polling no se clasifica
  como WebSocket.
- **Juez e infraestructura Docker**: backend completo, integración, Docker, seis
  flujos de juez/seguridad, build y auditoría.
- **Migración**: backend completo, integración, los 24 flujos E2E, cadena de
  migraciones, build y auditoría.
- **Contrato compartido**: backend y frontend completos, integración, dos
  flujos, build de ambos lados y auditoría.
- **`index.html`**: build de frontend y un flujo de humo. No selecciona todas
  las specs ni los 24 E2E.

El benchmark mide el tiempo de selección de cada escenario y lo muestra como
`selectionMs`; ese dato no es el tiempo de ejecución de las suites. Los pesos del
coste son orientativos y se expresan en segundos: cada suite tiene
un peso base, el frontend completo suma 60 s, el backend completo 30 s y cada
flujo E2E adicional 25 s. No son mediciones de suites reales. El agregado de los
quince escenarios es una estimación de 4031 s; ejecutar el conjunto completo de
la política para cada uno habría costado aproximadamente 18 465 s con los mismos
pesos, un 78 % menos estimado. No es una medición real: sirve para comparar
órdenes de magnitud, no para prometer una duración.

### Comparación antes y después

| Escenario | Antes de E7 (estimado) | Después de E7 (estimado) | Diferencia |
| :--- | ---: | ---: | :--- |
| Icono SVG | 1231 s (todas las suites) | 1 s | 99,9 % menos |
| Formulario | 1231 s (todas las suites) | 90 s | 92,7 % menos |
| HTML con lógica y sin spec | 0 s y falso verde | 45 s (build real) | +45 s para eliminar el falso verde |
| `index.html` | 750 s (128 specs, 24 E2E y build) | 80 s | 89,3 % menos |
| Login HTML L3 | 705 s (128 specs y 24 E2E, sin build) | 230 s | 67,4 % menos y build conservado |
| WebSocket backend | 200 s (sin regla propia) | 300 s | +100 s para cubrir seguridad y brechas |

El aumento en WebSocket y en la plantilla sin spec no es una regresión de coste:
es el precio de dejar de producir un verde falso. El resto de los escenarios baja
porque el plan deja de pagar infraestructura que no aporta señal.

## Regresión del selector

`node --test tools/test-impact/test/*.test.mjs` cubre:

- niveles L0 a L3, cambios visuales y estilos globales;
- contratos compartidos y reglas críticas aditivas;
- dependencias transitivas de Go y Angular;
- plantillas con lógica, HTML decorativo, `index.html` y build de autenticación;
- WebSockets backend y frontend, con brechas declaradas;
- eliminaciones, renombrados y configuraciones globales;
- rutas desconocidas, diff vacío y límite de archivos;
- módulos sin cobertura;
- planes completos y dirigidos;
- integridad: suites en falso, suite faltante, suite desconocida, política
  antigua, digest alterado, regla crítica eliminada, archivos incoherentes,
  nivel degradado, build oculto, HEAD distinto y working tree no representado;
- ejecución parcial explícita, suite pendiente, suite no seleccionada, dry-run,
  informe completo, agregación de CI, job pendiente, job fallido, informe sin
  recibo, informe sin evidencia, recibo de otra clave, job firmado con la clave
  de otro, duplicado y de otra ejecución;
- caché invalidada por archivo, fuente ignorada con basename de artefacto,
  lockfiles de dependencias, política, runner, comando, toolchain y entorno;
  firma de caché, fallos y dry-runs no cacheados;
- validación semántica de recursos estáticos;
- correspondencia entre política, documentación y catálogo E2E.

## Limitaciones conocidas

- La ejecución real de GitHub Actions no está validada en este entorno; solo se
  validaron el YAML, el agregado local y las reglas que lo alimentan.
- No se ejecutaron Docker ni suites E2E reales. PostgreSQL y las migraciones se
  ejecutaron solo para las pruebas de revocación contra una base de test; el
  resto de la integración y E2E queda preparado para CI con infraestructura
  efímera. La suite `security` sí ejecutó `go vet` y las pruebas de WebSocket;
  las de handshake, aislamiento, revocación y autorización del terminal pasan.
  La validación integral del plan sigue pendiente.
- Un plan `source.kind: files` es sintético y no valida el árbol de trabajo. Es
  útil para diagnosticar, no para cerrar un cambio.
- La granularidad de Go es por paquete. Un cambio ejecuta todos los tests de
  integración del paquete `tests/integration`, no un subconjunto por archivo.
- Un archivo nuevo sin spec no se puede cubrir con una suite que no existe. El
  plan lo reporta como no cubierto para que se agregue la prueba.
- Las pruebas unitarias cubren aislamiento del hub, expiración/revocación,
  permisos de `exec` y cierre de conexiones abiertas. Las pruebas de integración
  verifican persistencia, reinicio, dos instancias, HTTP, WebSocket y limpieza
  contra PostgreSQL de test. Siguen sin validarse en este entorno la reconexión
  del terminal, la liberación de listeners en E2E y la ejecución completa de
  Docker/E2E.
- La revocación persistente vive en `session_revocations`; el cierre entre
  instancias usa un sondeo de 2 s, no pub/sub. Si PostgreSQL no responde, la
  autenticación y el logout fallan cerrados; las conexiones locales del hub se
  cierran cuando el sondeo no puede consultar la tabla.
- La garantía local del recibo y de la evidencia no es criptográfica frente al
  mismo usuario que puede leer la clave de `.cache`. En CI cada job usa un
  secreto propio; `ci-summary` recibe todas las claves y rechaza la de otro job.
- La firma y el sidecar prueban que el runner produjo el informe con la clave de
  la ejecución; no convierten en correcta una prueba que el propio código del job
  haya falseado. La ejecución real de GitHub Actions y sus secretos no se pudo
  verificar en este host.
- `asset-validation` comprueba estructura y firma, no renderiza SVG ni valida
  CSS/SCSS con un compilador. Los formatos sin validador se aceptan y declaran el
  límite.
- Los tests E2E dependen de que el backend y el frontend estén levantados. El
  runner no los levanta por sí solo.
- La caché no se usa en integración, Docker, migraciones, E2E, build ni
  seguridad. No hay caché de resultados de infraestructura.
- `test-db.sh` necesita el cliente `psql`; si no está instalado falla de forma
  explícita. En este host no está disponible.
- Un `SIGKILL` del proceso de pruebas no ejecuta la limpieza ordenada; el paso
  `if: always()` del workflow cubre cancelaciones, no un `SIGKILL` del runner.
- El coste del benchmark es una estimación con pesos, no una medición de las
  suites. Las suites pesadas no se ejecutaron en esta fase.
