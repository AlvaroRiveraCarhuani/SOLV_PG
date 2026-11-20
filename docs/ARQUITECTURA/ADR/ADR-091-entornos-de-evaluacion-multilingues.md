# ADR-091: Entornos de Evaluación Multilingües

- **Estado:** Aceptado
- **Fecha:** 2026-11-20
- **Ámbito:** Core (juez, plantillas, catálogo de tecnologías)
- **Decisores:** Arquitectura SOLV

## Contexto

Hasta ahora un entorno de evaluación se declaraba con un solo lenguaje implícito: el runtime elegido en el constructor definía la imagen, el `entrypoint` y, a la vez, el lenguaje que el docente podía asignar al ejercicio. La modalidad `JUEZ_EFIMERO` (Entorno de Evaluación) quedaba así atada a una imagen por lenguaje y a una plantilla por lenguaje.

Ese modelo obliga a crear y aprobar una plantilla por cada lenguaje, aunque compartan memoria, cuota y política de auditoría. También deja el catálogo de lenguajes repartido en tres sitios distintos: las plantillas `SYSTEM_SEED_RUNNER`, la tabla `language_profiles` y los componentes de tecnología. Un ejercicio de C++ creado sobre una plantilla de Python no se puede corregir, y un entorno nuevo de C++ duplica la imagen del lenguaje sin agregar capacidad.

Los jueces que ya resuelven este problema (DMOJ, Judge0, DOMjudge, VJudge) separan dos conceptos: el **entorno** (cuota, imagen base, política) y el **ejecutor de lenguaje** (comando de compilación y ejecución, imagen específica cuando hace falta). La entrega declara su lenguaje y el juez elige el ejecutor correspondiente.

## Decisión

Un Entorno de Evaluación declara **uno o más ejecutores de lenguaje**. Cada ejecutor guarda:

| Campo | Contrato |
| :--- | :--- |
| `language` | Clave canónica del juez (`python`, `javascript`, `cpp`, `c`, `csharp`, `java`, `go`, `rust`, `php`) o lenguaje propio con imagen. |
| `entrypoint` | Comando de compilación y ejecución. Obligatorio. Se ejecuta dentro de `/tmp` con la solución copiada a `solution.<ext>` o `Solution.java`/`main.go` según el lenguaje. |
| `docker_image` | Imagen del ejecutor. Vacía significa «usa la imagen del catálogo del juez para este lenguaje». |
| `timeout_ms` | Tiempo límite por caso. `0` hereda el del ejercicio y el perfil del lenguaje. |
| `memory_limit_mb` | Memoria por ejecución. `0` hereda la cuota del entorno. |

Los ejecutores viven en `template_language_runners` (migración `00070`), con clave `(template_id, language)` y borrado en cascada. La migración rellena los entornos existentes desde el runtime de su imagen para que ninguna plantilla aprobada quede sin ejecutor.

Reglas del contrato, aplicadas en el dominio (`NormalizeLanguageRunners`) y no en la entrega HTTP:

1. Los alias se canonizan antes de guardar (`c++` → `cpp`, `c#`/`cs` → `csharp`, `js`/`node` → `javascript`, `py`/`python3` → `python`, `golang` → `go`, `rs` → `rust`).
2. Un lenguaje se declara una sola vez por entorno y el orden es determinista: el primero es el lenguaje primario.
3. Un lenguaje fuera del catálogo del juez exige `docker_image`; sin ella el motor no tendría cómo ejecutarlo.
4. `:latest` sigue prohibido, igual que en la imagen del entorno.
5. El tiempo se acota a 100–60000 ms y la memoria a 64–4096 MB, además del techo del host.
6. Un entorno admite hasta 12 lenguajes.

El **lenguaje primario** conserva el contrato anterior: su ejecutor alimenta el `entrypoint` de la plantilla y la comprobación de arranque del worker de auditoría. Los demás lenguajes no agregan requisitos de imagen al entorno.

### Selección en la creación de ejercicios

- `exercise.language` pasa a ser el **lenguaje primario o por defecto** del ejercicio, no un candado único.
- Al crear o actualizar un ejercicio cuyo entorno declara ejecutores, el servicio exige que `exercise.language` esté entre ellos. El desajuste se informa con `language_not_supported` (422) y no se descubre en la primera entrega.
- Al evaluar, el juez resuelve el ejecutor a partir del lenguaje de la entrega (`ResolveLanguageRunner`). Si el entorno declara lenguajes y la entrega trae uno que no está declarado, responde `language_not_supported` (422).
- Un entorno sin ejecutores conserva el comportamiento anterior: imagen y `entrypoint` únicos, válidos solo para el lenguaje del ejercicio.

### Imagen del ejecutor

Cuando un ejecutor no declara imagen, el motor usa la imagen de catálogo de su lenguaje: la estrategia del lenguaje en el juez (`lang_python.go`, `lang_cpp.go`, …) y, si existe, el pin de `language_profiles`. Así, agregar un lenguaje no obliga a construir una imagen que contenga todos los runtimes, y un lenguaje que necesita paquetes propios (NumPy, una versión específica de JDK) sigue pudiendo declarar su imagen.

### Pinning de la publicación

El `environment_snapshot` del ejercicio publica los ejecutores congelados: editar la plantilla después de publicar no cambia la forma de corregir lo ya publicado. La huella de verificación (`verification_fingerprint`) incluye la lista, de modo que un cambio de lenguajes invalida la verificación anterior.

### Lenguajes de la práctica y elección del alumno

El catálogo del entorno no obliga a ofrecer todos sus lenguajes en cada práctica. El ejercicio persiste en `config` dos datos:

- `allowed_languages`: subconjunto de lenguajes del entorno que el alumno puede elegir. Vacío conserva el contrato anterior (solo `language`).
- `boilerplates`: código de partida por lenguaje, con claves canónicas y limitado a los lenguajes admitidos.

El editor del docente toma los lenguajes del entorno al elegir la plantilla y permite escribir un esqueleto por lenguaje. La API pública del ejercicio (`GET /api/v1/exercises/{id}`) devuelve ambas listas al alumno, que elige con un selector y recibe el código de partida correspondiente. Al entregar, el servidor comprueba que el lenguaje esté en `allowed_languages` antes de resolver el ejecutor; si no lo está responde `422 language_not_supported` y nombra los lenguajes admitidos. El navegador nunca decide qué imagen ni qué comando se ejecuta.

## Consecuencias

**Positivas**

- Un solo Entorno de Evaluación cubre varios lenguajes con la misma cuota, modalidad y auditoría.
- El docente elige el lenguaje del ejercicio dentro del catálogo real del entorno.
- El catálogo de lenguajes queda en un solo lugar consultable; `GetTemplateForLanguage` puede preferir plantillas que declaran el lenguaje.
- Duplicar una plantilla conserva sus lenguajes.
- Un lenguaje propio (Julia, Lua, Ruby) se agrega con su imagen y su comando, sin tocar el motor.

**Costos y límites**

- La migración deduce el lenguaje de los entornos existentes a partir de la imagen y del comando; una imagen atípica puede quedar sin ejecutor y debe completarse desde el constructor.
- El techo de memoria por ejecutor se valida contra el host, pero el límite real sigue siendo la suma de ejecuciones concurrentes que admite la admisión del nodo.
- El lenguaje de la entrega ya no está limitado al `language` del ejercicio cuando el entorno declara varios: pasa a mandar el catálogo del entorno. Las reglas AST y los casos de prueba se aplican igual, en el lenguaje entregado. La práctica puede restringir ese catálogo con `allowed_languages`.
- Un esqueleto por lenguaje se guarda solo mientras ese lenguaje siga admitido: quitar un lenguaje de la práctica descarta su código de partida para no ofrecer un inicio que ya no se corrige.

## Alternativas descartadas

1. **Una columna JSON `allowed_languages` en `lab_templates`.** Simple de escribir, pero cada lenguaje necesita su propio comando y, a veces, su imagen; un arreglo de cadenas obliga a derivar el comando por convención y deja el contrato sin validar.
2. **Una imagen por entorno con todos los runtimes preinstalados.** Multiplica el tamaño y la superficie de vulnerabilidades de cada entorno y no escala a lenguajes nuevos sin reconstruir y reauditar.
3. **Mantener una plantilla por lenguaje.** Es el estado actual: duplica la cuota, la auditoría y el mantenimiento, y no permite que un ejercicio cambie de lenguaje dentro del mismo entorno.
4. **Resolver el lenguaje en el navegador.** El ejecutor debe elegirse en el servidor a partir de la entrega; el cliente no dicta con qué imagen ni con qué comando se corrige.

## Referencias

- `backend/internal/core/domain/language_runner.go`
- `backend/migrations/00070_template_language_runners.sql`
- `backend/internal/core/services/evaluation_algorithm_runner.go`
- `backend/internal/core/services/evaluation_exercise_crud.go`
- `docs/ARQUITECTURA/ADR/ADR-030-catalogo-plantillas-docker-aprobacion.md` (gobernanza de imágenes)
- `backend/migrations/00059_environment_snapshot.sql` (pin del entorno al publicar)
