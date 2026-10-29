# node-service-bindings Specification

## Purpose

Definir el contrato de servicios satélite declarativos por rol para una plantilla de laboratorio: cardinalidad, aprovisionamiento de varios motores, pin del entorno, capacidad del nodo, aislamiento por credenciales y contrato de inicialización. El juez efímero queda fuera de alcance.

> **Nota de implementación:** esta especificación describe el estado final. Mientras la fase 4 (pipeline de N bindings) no esté desplegada, la validación conserva el tope temporal de un binding por plantilla para no aceptar configuraciones que el arranque todavía ignora.

## Requirements

### Requirement: Bindings declarativos y cardinalidad por rol

El sistema DEBE representar los servicios de una plantilla como una lista de bindings con `role`, `engine`, `component_id`, `version` y `env_alias`. Los roles iniciales son `primary_database` y `cache`. El dominio DEBE rechazar más de una base principal, más de un binding por motor, un rol incoherente con el motor y un alias repetido. `category` DEBE aceptarse como alias de lectura de `role` para payloads anteriores.

#### Scenario: PostgreSQL y Redis conviven

- DADO un payload con `primary_database` PostgreSQL y `cache` Redis
- CUANDO se crea o actualiza la plantilla
- ENTONCES el sistema DEBE aceptarla con dos bindings y persistir ambos

#### Scenario: MySQL y PostgreSQL no conviven

- DADO un payload con dos bindings `primary_database` (MySQL y PostgreSQL)
- CUANDO se crea o actualiza la plantilla
- ENTONCES el sistema DEBE responder `422` con el código `multiple_primary_database`

#### Scenario: Motor duplicado o desconocido

- DADO un payload con Redis repetido, o con un motor fuera del dominio soportado
- CUANDO se valida la plantilla
- ENTONCES el sistema DEBE rechazarlo con `duplicate_service_engine` o `unsupported_service_engine`

#### Scenario: Payload heredado con category

- DADO un payload anterior con `category: "cache"` y `engine: "redis"` sin `role`
- CUANDO se normaliza
- ENTONCES el sistema DEBE interpretarlo como `role: "cache"`

### Requirement: Resolución y aprovisionamiento de todos los bindings

El sistema DEBE resolver todos los bindings de la plantilla y aprovisionarlos en orden determinista (primario primero, auxiliares después). El arranque DEBE esperar readiness de todos, inyectar el entorno de todos y abortarse si uno falla, sin destruir contenedores ni volúmenes compartidos. El scale-to-zero DEBE contar el conjunto de motores de cada laboratorio.

#### Scenario: Dos motores listos

- DADO un laboratorio con PostgreSQL de aula y Redis gestionado
- CUANDO arranca
- ENTONCES el laboratorio DEBE recibir `DATABASE_URL` y `REDIS_URL` y estar conectado a la red de satélites

#### Scenario: Un motor no responde

- DADO un laboratorio con dos bindings y uno que no alcanza readiness en 30 s
- CUANDO arranca
- ENTONCES el arranque DEBE fallar completo y el workspace no DEBE quedar en `running`

#### Scenario: Scale-to-zero con dos motores

- DADOS dos laboratorios activos: uno usa Redis y PostgreSQL, el otro solo PostgreSQL
- CUANDO se hiberna el primero
- ENTONCES Redis DEBE detenerse solo si ningún otro laboratorio lo usa
- Y PostgreSQL DEBE seguir disponible para el segundo

### Requirement: Pin del entorno al publicar

El sistema DEBE persistir un snapshot del entorno al publicar un ejercicio con imagen, entrypoint, timeout, RAM base, bindings resueltos, contrato de init y huella de verificación. El arranque DEBE leer el snapshot. Para ejercicios sin snapshot, DEBE resolver la plantilla viva y registrar el fallback. Editar una plantilla NO DEBE cambiar el entorno de ejercicios ya publicados.

#### Scenario: Edición posterior a la publicación

- DADO un ejercicio publicado que fija PostgreSQL + Redis
- CUANDO un administrador cambia la plantilla a MySQL sin Redis
- ENTONCES el ejercicio DEBE seguir arrancando con PostgreSQL + Redis

#### Scenario: Ejercicio heredado

- DADO un ejercicio publicado antes del corte, sin `environment_snapshot`
- CUANDO arranca
- ENTONCES el sistema DEBE resolver la plantilla viva y registrar que usó el fallback

### Requirement: Capacidad del nodo separada de la cuota del alumno

Cada adaptador DEBE declarar `expected_mb` y `reserve_mb`. La cuota del laboratorio NO DEBE incluir la memoria de servicios compartidos. La admisión DEBE descontar `reserve_mb` una vez por identidad de satélite y calcular la capacidad como `(allocatable_mb − Σ reserve_mb) / per_lab_quota_mb`. La interfaz DEBE rotular el costo del satélite como del nodo.

#### Scenario: Treinta alumnos con un satélite

- DADO un nodo con 8 GB y un satélite de 256 MB reservado
- CUANDO se calcula la capacidad para una cuota de 640 MB por alumno
- ENTONCES la cuenta DEBE restar 256 MB una vez y no 256 MB por alumno

#### Scenario: Dos servicios auxiliares

- DADO un nodo con dos satélites reservados
- CUANDO se calcula la capacidad
- ENTONCES DEBE restar la suma de ambos techos una sola vez

### Requirement: Aislamiento por motor y por scope

Toda credencial y nombre de recurso DEBE derivarse de `tenant + alumno + materia + alias`. PostgreSQL DEBE usar un rol y una base por scope sin superusuario; MongoDB DEBE exigir autenticación y un usuario con `readWrite` sobre su base; Redis DEBE usar ACL por usuario con patrón de claves del scope y comandos peligrosos denegados. Un motor sin aislamiento verificable NO DEBE habilitarse.

#### Scenario: Alumno no alcanza la base de otro

- DADO un alumno con credenciales de su scope
- CUANDO intenta conectarse a la base de otro alumno o listar bases ajenas
- ENTONCES el motor DEBE denegar la operación

#### Scenario: Redis acotado

- DADO un alumno con ACL de su prefijo
- CUANDO ejecuta `FLUSHALL` o lee claves fuera de su prefijo
- ENTONCES Redis DEBE denegar la operación

### Requirement: Contrato de inicialización por motor

Cada adaptador DEBE declarar `init_kind` (`sql`, `mongo-js`, `redis-cmds` o `none`). Un `db_init_script` en un motor sin contrato de init DEBE rechazarse en la validación. La aplicación DEBE sellarse por `(exercise_id, service_alias)` en cada motor.

#### Scenario: Init aplicado una vez

- DADO un ejercicio con `db_init_script` y un primario SQL
- CUANDO se reactiva el mismo laboratorio
- ENTONCES el script NO DEBE ejecutarse de nuevo

#### Scenario: Init en un motor sin contrato

- DADO un ejercicio con `db_init_script` y un primario Redis
- CUANDO se valida la plantilla
- ENTONCES el sistema DEBE rechazarlo con `service_init_not_supported`

### Requirement: Studio con base principal y auxiliares

La Columna 3 del Studio DEBE separar «base principal» (selección única, incluida «Sin base de datos») de «servicios auxiliares» (uno por rol). El payload DEBE enviar todos los bindings. El modal de solicitud docente y el copiloto DEBEN usar el mismo contrato.

#### Scenario: Selección de PostgreSQL y Redis

- DADO el Studio en modo IDE persistente
- CUANDO el administrador elige PostgreSQL como principal y Redis como auxiliar
- ENTONCES el resumen DEBE mostrar ambos servicios, su rol y el costo del nodo

#### Scenario: Dos bases principales

- DADO el Studio con PostgreSQL seleccionado
- CUANDO el administrador intenta elegir MySQL como principal
- ENTONCES la interfaz DEBE reemplazar la selección, no acumularla

### Requirement: Catálogo y adaptador con responsabilidades separadas

El catálogo DEBE decidir qué motores se ofrecen y su disponibilidad. El adaptador DEBE decidir imagen, versión, límites, readiness, aislamiento e init. El aprovisionador NO DEBE usar una imagen o un techo que no provenga del adaptador. Desactivar un motor en el catálogo DEBE ocultarlo del Studio.

#### Scenario: Motor desactivado

- DADO Redis desactivado en el catálogo
- CUANDO se abre el Studio
- ENTONCES Redis NO DEBE ofrecerse como auxiliar

#### Scenario: Imagen gobernada por el adaptador

- DADO un adaptador que fija `redis:7.2-alpine`
- CUANDO el catálogo muestra una versión distinta
- ENTONCES el contenedor DEBE levantarse con la imagen del adaptador y la interfaz DEBE mostrar la del adaptador
