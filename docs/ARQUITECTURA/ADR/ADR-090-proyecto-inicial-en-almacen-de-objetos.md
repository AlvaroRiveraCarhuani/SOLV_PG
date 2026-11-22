# ADR-090: Proyecto Inicial de Árbol Grande como Artefacto Versionado

- **Estado:** Propuesto
- **Fecha:** 2026-11-18
- **Ámbito:** Docencia, proyectos iniciales, uso de disco
- **Relacionado:** ADR-068 (proyectos iniciales y versiones inmutables), ADR-069 (preparación de dependencias), ADR-058 (avance del docente en aula), ADR-073 (almacenamiento propio), ADR-084 (cuotas de disco), ADR-035 (respaldos y retención)

## Contexto

El proyecto inicial de un ejercicio de Entorno de Desarrollo se guarda hoy como
texto en PostgreSQL: el docente importa una carpeta o un ZIP, el sistema lo
normaliza a un snapshot `# === FILE: <ruta> ===` y la versión publicada queda en
`workspace_project_versions.text_snapshot`. Esa decisión (ADR-068) hizo que el
historial sea inmutable, que el diff de SpeedGrader y la resta del antiplagio
comparen contra una versión exacta, y que el respaldo no crezca con el código de
cada ejercicio.

El costo de esa decisión es un techo explícito: **1 MiB de contenido, 300
archivos y 512 KiB por archivo**, con un ZIP de hasta 10 MiB y 1000 entradas
antes de aplicar el límite del proyecto. El mismo tope aparece otra vez al
preparar dependencias, que escribe la versión en el árbol con 300 archivos y
1 MiB por archivo.

Ese techo es correcto para un laboratorio y queda corto para un caso docente
real: materias de mantenimiento, evolución o arquitectura donde la consigna es
tomar un sistema heredado grande y trabajarlo durante el semestre. Un monólito
de 2.000 archivos no entra, y subir el techo del snapshot no es una opción: cada
versión metería decenas de MB en la base, en el respaldo y en el árbol que
dibuja el navegador.

El depósito de objetos ya existe y ya se respalda: el avance del docente
(ADR-058) guarda árboles completos como `tar.gz` con hash SHA-256 en SeaweedFS,
fuera de PostgreSQL, con exclusiones configurables. `infra/scripts/backup.sh`
copia ese volumen y lo sube afuera cuando hay destino configurado. La pieza que
falta no es almacenamiento: es usar ese camino para el proyecto inicial.

## Medición

Antes de decidir se midió la pieza que más condiciona el resultado: la
preparación de dependencias por versión, con los perfiles reales del catálogo y
el runner real del worker, sobre proyectos reales de este repositorio. Nodo
On-Premise único, contenedor de 1 CPU y 2 GB por preparación, 18 de noviembre de
2026. Disco del nodo al medir: **13 GB libres de 98 GB (87 % usado)**.

| Caso | Proyecto | Dependencias | Caché de trabajo | Árbol | Baja al alumno | 30 alumnos | Instalar | Comprobar |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Python · API típica | 58,6 MB | `.venv` 55,7 MB | — | 114,4 MB | 114,3 MB (11,2 %) | ~3,4 GB | 34,0 s | 4,6 s |
| Python · locust | 58,6 MB | `.venv` 60,8 MB | — | 119,5 MB | 119,4 MB (11,7 %) | ~3,4 GB | 31,6 s | 4,5 s |
| Go · `backend/` | 92,8 MB | `.solv-cache` 216,1 MB | `.solv-home` 305,4 MB | 614,3 MB | 308,9 MB (30,2 %) | ~9,1 GB | 33,1 s | 182,8 s |
| Node · `frontend/` Angular | 42,8 MB | 650,7 MB | — | 693,4 MB | 563,3 MB (55,0 %) | ~16,5 GB | 64,2 s | 3,8 s |

Dos columnas importan más que el total. **Árbol** es lo que ocupa el servidor
por versión publicada. **Baja al alumno** es lo que el enlace copia al volumen:
solo las rutas declaradas en `dependency_paths`, no la caché de trabajo. La
cuota por entorno es de 1 GB y se mide al entregar.

Estos números son **capacidad del nodo**, no parte de la decisión. El camino del
proyecto grande funciona igual con cualquier familia del catálogo: lo que cambia
es cuántos cursos entran en este servidor y cuánto tarda cada preparación. La
decisión de este ADR es una sola —que exista o no la segunda puerta— y no se
elige por lenguaje.

Lectura de los números:

- **Python entra con margen.** 11 % de la cuota y 3,4 GB para un aula de 30.
- **Go entra, ajustado.** 30 % de la cuota, 9,1 GB para 30 y cuatro minutos de
  preparación por versión, porque la comprobación compila todo el módulo.
- **Node no entra en este hardware.** 55 % de la cuota antes de escribir una
  línea y 16,5 GB para 30 alumnos sobre 13 GB libres. No es configuración: es
  disco.
- **El proyecto de origen ya no entra por el camino liviano.** El `frontend/`
  medido tiene 1.564 archivos y 42,8 MB: cinco veces el tope de archivos y
  cuarenta veces el de contenido. El caso que motiva este ADR es real y está en
  este repositorio.

### Defectos que la medición dejó a la vista

Los tres perfiles estaban publicados y ninguno funcionaba en su caso principal.
La validación del perfil es de forma, no de ejecución.

| Perfil | Defecto | Estado |
| :--- | :--- | :--- |
| `python-312` | Un proyecto con `requirements.txt` y sin lockfile usaba `pip install .` y moría con `Directory '.' is not installable` | Corregido en la migración `00066` |
| `java-21` | La imagen del runtime no trae Maven y el perfil invoca `mvn` | Abierto |
| `go-122` | Fija `golang:1.22-bookworm` y el `go.mod` del proyecto pide 1.25: con `GOPROXY=off` no hay toolchain | Abierto |

Además: el runner de preparación no descarga imágenes (falla con `No such image`
si la imagen no está en el nodo), el perfil de Node depende de correr como root
—`corepack enable` escribe en `/usr/local/bin` y falla con `EACCES` bajo
`1000:1000`— y la comprobación de Go deja 305 MB de caché de compilación dentro
del árbol preparado, que no viaja al alumno pero ocupa disco del servidor y
cuenta contra el techo de 2 GB por preparación.

### Alcance de la medición

Se midió la preparación de dependencias, no el camino del artefacto que este ADR
propone. Los números sirven para dos cosas: fijar el presupuesto de disco del
camino nuevo y decidir a qué familia se ofrece primero. El costo de subir y
extraer un `tar.gz` de 100 MiB no se midió.

## Decisión

1. **Dos caminos por tamaño, no por calidad.** El proyecto liviano conserva el
   contrato de ADR-068 sin cambios: carpeta, ZIP o generación asistida, árbol
   editable en el navegador, 1 MiB y 300 archivos. El proyecto completo es un
   camino nuevo para árboles que superan ese techo. El sistema no opina sobre la
   calidad del código en ninguno de los dos. **La decisión es una sola: que
   exista o no ese segundo camino.** La familia tecnológica del primer curso es
   capacidad y no condiciona el diseño.

2. **El árbol grande es un artefacto, no texto en la base.** Se guarda como
   `tar.gz` en el depósito de objetos (ADR-058 §6, ADR-073), con su SHA-256. La
   versión publicada guarda la referencia y el hash; `text_snapshot` deja de ser
   la fuente para ese camino y no se escribe.

3. **La versión sigue siendo inmutable.** El hash del artefacto entra en la
   versión y en la clave de preparación de dependencias. Un `force-push` en el
   repositorio del docente no cambia lo que el alumno recibió.

4. **La entrega sigue atada a la versión.** `submissions.workspace_project_version_id`
   se toma del entorno, como ya ocurre. Se conserva el diff base del SpeedGrader
   y la resta del antiplagio. Esto es lo que un clonado desde Git no da.

5. **Sin edición en el navegador para el artefacto.** El docente ve un resumen
   —archivos, tamaño, hash y tecnologías detectadas—, no el árbol editable.
   Corrige en su máquina y publica una versión nueva. El árbol completo se sigue
   viendo en el SpeedGrader para corregir.

6. **Límites del artefacto, configurables y separados del camino liviano.**
   Valores propuestos: 100 MiB comprimido, 20.000 archivos, 512 MiB expandido y
   8 MiB por archivo. Se derivan de lo medido (1.564 archivos / 42,8 MB en un
   proyecto grande real) y son configuración, no constantes de código. El camino
   liviano no hereda ninguno.

7. **El sembrado reutiliza lo que ya existe.** El mismo contenedor efímero sin
   red monta el volumen, pero extrae el archivo en lugar de escribir texto. El
   diario `.solv_workspace/pending.json` y el marcador `verified` siguen
   gobernando: un volumen con trabajo del alumno no se toca, no hay re-siembra
   destructiva y una versión distinta se rechaza.

8. **Autorización igual a la de publicar.** Docente de la materia o
   administrador de la institución, tenant y usuario del contexto autenticado. No
   se agregan permisos nuevos.

9. **Dos puertas cerradas por defecto.** La publicación sigue detrás de
   `WORKSPACE_PROJECTS_PUBLISH_ENABLED`. La carga del archivo tiene además su
   propia bandera, `WORKSPACE_PROJECT_ARCHIVE_ENABLED`, apagada hasta que la
   siembra por extracción esté probada de punta a punta; encender la carga sin
   la publicación solo deja borradores que no se pueden convertir en versión.

10. **El alta mide el disco antes de aceptar.** Publicar un artefacto exige que
    el nodo tenga espacio para el árbol preparado y para las dependencias que
    declara el perfil. Un proyecto que no cabe se rechaza al publicar, no en la
    primera clase. Este ADR no fija el número: lo mide la admisión con las
    cifras de la sección anterior.

## Superficie HTTP

- `POST /api/v1/teacher/exercises/{id}/project/import/archive`
  `multipart/form-data` con `archive` y `expected_revision`. Responde el resumen
  del artefacto: `files`, `bytes`, `sha256`, `technologies`.
- `GET /api/v1/teacher/exercises/{id}/project` incluye `archive` con
  `sha256`, `bytes`, `files` y `prepared_at` cuando la versión publicada es un
  artefacto.
- `POST .../project/publish` no cambia su contrato: publica la versión vigente
  del camino que corresponda.

Un artefacto no se descarga desde el navegador del alumno ni desde el del
docente: solo se usa para sembrar y para el diff base.

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| :--- | :--- |
| Subir el techo de 1 MiB y 300 archivos | Cada versión metería decenas de MB en la base, en `pg_dump` y en el árbol que dibuja el navegador; el límite existe para proteger esas tres cosas a la vez |
| Clonar desde una URL de Git | Exige credenciales de terceros en la plataforma, abre salida arbitraria desde el backend y, sobre todo, deja la entrega sin versión de origen: se pierde el diff base |
| Precargar el repositorio en la imagen del entorno | El volumen montado en `/home/workspace` tapa el contenido de la imagen en esa ruta, y multiplica imágenes por ejercicio y versión (ya descartado en ADR-069) |
| Copiar el proyecto grande con `setup_script` al abrir | Bloquea el arranque del editor una vez por alumno, que es el problema que ADR-069 quita |
| Una tabla nueva por artefacto | Duplica `workspace_project_versions`; alcanza con una columna de referencia y el hash en la fila existente |

## Consecuencias

**A favor**

- El proyecto semestral deja de estar fuera de alcance sin tocar el camino que
  hoy funciona.
- La base y su respaldo no crecen con el código de cada ejercicio.
- El historial, el diff del SpeedGrader y la resta del antiplagio conservan la
  versión exacta que el alumno recibió.
- El depósito ya está en el respaldo: no aparece un sistema de almacenamiento
  nuevo que operar.

**Costos y cuidados**

- El docente pierde la edición archivo por archivo en el navegador para estos
  proyectos. Es el precio y conviene decirlo en la interfaz, no en el manual.
- El respaldo del depósito pasa a ser crítico: ahora guarda el original de una
  cursada y no solo avances. La retención de siete días de `backup.sh` es corta
  para ese contenido; la copia fuera del nodo deja de ser opcional.
- El disco del nodo se vuelve el límite real del producto. Medido: Node no entra
  para 30 alumnos en este servidor.
- La preparación de dependencias sigue teniendo sus propios límites de 300
  archivos y 1 MiB por archivo al escribir la versión en el árbol. El camino del
  artefacto los necesita levantados también, o el proyecto grande se publica y
  no se prepara.

## Migración

`00067_workspace_project_artifacts.sql` (goose, solo `Up` en el runner):

- `workspace_project_versions` y `workspace_project_drafts` reciben
  `artifact_key TEXT`, `artifact_sha256 CHAR(64)`, `artifact_bytes BIGINT` y
  `artifact_files INTEGER`, todas nulas. Las filas existentes quedan sin
  artefacto y siguen por el camino liviano sin cambios.
- Un `CHECK` por tabla exige los cuatro campos juntos o ninguno, para que una
  fila no quede a medio camino entre los dos modos.
- El trigger de inmutabilidad de ADR-068 se mantiene: el artefacto se escribe al
  crear la versión, nunca después.
- El borrador conserva los mismos campos como área de preparación: la carga del
  archivo revisa el árbol sin publicar, igual que el borrador de texto.
- Las guardas de 00053 sobre las versiones (`text_snapshot` no vacío, hasta 300
  archivos y 1 MiB) se reemplazan por variantes que distinguen el modo: sin
  artefacto se conservan tal cual; con artefacto el snapshot queda vacío y los
  topes pasan a 20.000 archivos y 128 MiB expandidos. Sin este cambio una
  versión con artefacto no podía existir.
- El `Down` borra las versiones con artefacto (el esquema anterior no las
  admite) desactivando el trigger de inmutabilidad dentro de su propia
  transacción, vacía la preparación de los borradores y restaura las guardas
  originales. Los objetos del depósito no se tocan: quedan para la retención.

No se migran versiones existentes ni se borra `text_snapshot`: el espejo de
compatibilidad sigue vivo.

### Estado del corte mínimo

Implementado en este corte:

- `POST /api/v1/teacher/exercises/{id}/project/import/archive` recibe un único
  `tar.gz` en multipart, con `expected_revision` y la misma autorización que el
  ZIP liviano.
- El archivo se valida completo antes de guardar: rutas relativas y
  normalizadas, sin symlinks, hardlinks ni entradas especiales, sin el
  directorio `.solv_workspace`, sin duplicados ni colisiones, y con topes de
  32 MiB comprimidos, 128 MiB expandidos, 20.000 archivos y 4 MiB por archivo.
- Se guarda en el depósito de objetos con la clave
  `workspace-projects/<tenant>/<ejercicio>/<sha256>.tar.gz` y queda como
  borrador con su huella, tamaño y cantidad de archivos.
- La siembra del volumen extrae el artefacto de la versión publicada con las
  mismas guardas, verifica el SHA-256 descargado contra el emitido y reutiliza
  el diario `pending.json` y el marcador `verified` existentes.

- La publicación acepta un borrador con artefacto: la versión queda con
  `text_snapshot` vacío, `content_sha256` igual al hash del archivo, sus números
  en `file_count`/`total_bytes` y el manifiesto por archivo extraído del
  tar.gz, así que el historial conserva la lista sin volver a abrir el depósito.
- La revisión del docente resuelve el proyecto base desde el depósito cuando la
  entrega apunta a una versión con artefacto, y arma el diff por archivo con ese
  árbol. El espejo de texto no se usa en ese caso.
- La entrega del alumno aplica los topes del proyecto grande cuando el entorno
  fue sembrado con artefacto (128 MiB, 20.000 archivos y 4 MiB por archivo) y
  conserva los del camino liviano en cualquier otro caso. Sin esto el alumno no
  podía entregar el mismo árbol que el docente sembró.
- La migración 00069 vuelve a fijar las guardas de versión de 00067. Una base
  que ya había aplicado 00067 antes del ajuste conserva los CHECK de 00053 y
  rechaza publicar un artefacto; goose no repite una versión aplicada.
- Esa entrega se empaqueta en `tar.gz` y se guarda en el depósito con la clave
  `submissions/<tenant>/<ejercicio>/<entrega>.tar.gz` (migración 00068). La fila
  conserva `artifact_key`, `artifact_sha256`, `artifact_bytes` y
  `artifact_files`, y `submissions.code` queda vacío: los 128 MiB no entran en
  PostgreSQL. La revisión del docente descarga el objeto, verificá su huella y
  expande el árbol para el visor por archivo.
- El espejo `exercises.boilerplate` guarda texto, así que para un artefacto
  queda vacío y atado a la versión nueva. El trigger de 00053 exige que el
  espejo coincida con el snapshot de su origen, y ese snapshot es vacío: el
  espejo no miente sobre lo que hay en la base.
- Publicar un artefacto no encola la preparación de dependencias: el worker lee
  el snapshot de texto y todavía no extrae el archivo. La versión se publica y
  el entorno instala al empezar, como antes de ADR-069.
- La carga está detrás de `WORKSPACE_PROJECT_ARCHIVE_ENABLED`, apagada por
  defecto, para no dejar la puerta abierta antes de que la cadena esté
  verificada de punta a punta.

Todavía no incluido:

1. Pantalla de carga del `.tar.gz` y vista del árbol del borrador en el
   frontend: hoy el endpoint existe sin interfaz.
2. El motor antiplagio y la línea de tiempo todavía leen `boilerplate`, que queda
   vacío en las versiones con artefacto.
3. Subir los topes a los valores del ADR exige extraer en streaming dentro del
   volumen, sin expandir el árbol en memoria.
4. Preparación de dependencias sobre artefactos: el worker tiene que extraer el
   archivo y respetar sus topes en lugar de los 300 archivos y 1 MiB del texto.
5. El historial del alumno y las vistas que leen `submissions.code` no muestran
   todavía el árbol de una entrega con artefacto; la revisión docente sí.

## Verificación

- Prueba de normalización y límites del artefacto: rutas seguras, sin binarios,
  sin symlinks, sin entradas cifradas, tope de archivos y de tamaño, hash
  reproducible.
- Prueba de sembrado con volumen descartable: extracción completa, diario y
  marcador, rechazo de una versión distinta y conservación del trabajo del
  alumno.
- Prueba de autorización: estudiante rechazado, ejercicio de otro tenant
  rechazado, ejercicio de Entorno de Evaluación rechazado.
- Prueba de entrega: la sumisión queda con la versión del artefacto y el
  SpeedGrader obtiene el árbol base desde el depósito.
- Prueba de admisión: un artefacto cuyo árbol preparado no cabe en el nodo se
  rechaza con un mensaje que dice cuánto falta.

## Pendientes antes de aceptar

1. Medir el costo de subir, guardar y extraer un artefacto de 100 MiB en este
   nodo. La sección de medición no lo cubre.
2. Decidir la política de retención del depósito y de la copia fuera del nodo.
3. Capacidad, no decisión: con la cuota de 1 GB y 13 GB libres, un aula de 30
   con Node no entra y una de Python sí. El camino no cambia por eso; cambia
   cuántos cursos caben en este nodo.
4. Levantar el tope de archivos de la preparación de dependencias, que hoy
   rechaza el mismo árbol que este ADR quiere aceptar.
