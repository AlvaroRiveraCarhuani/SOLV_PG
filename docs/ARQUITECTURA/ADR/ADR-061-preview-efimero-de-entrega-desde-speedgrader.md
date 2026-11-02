# ADR 061: Entorno de Prueba en Vivo de una Entrega IDE desde SpeedGrader

> **Estado:** Implementado
> **Fecha:** 2026-11-01
> **Ámbito:** Docencia, SpeedGrader, Entornos IDE
> **Relacionado:** ADR-001 (volumen persistente), ADR-011 (ciclo de vida y QoS), ADR-043 (modalidad dual), ADR-051 (PostgreSQL de aula), ADR-053 (revisión docente), ADR-059 (entornos por práctica), ADR-060 (diff y comentarios), [REVISION_POR_MODALIDAD](../../UI/DOCENTE/REVISION_POR_MODALIDAD.md)

---

## Contexto

En la revisión de un laboratorio IDE, el docente ve el snapshot del proyecto y su diff, pero no puede
comprobar si la aplicación arranca y responde. La única ejecución disponible era el runner efímero
del juez, que no aplica a un proyecto multiarchivo y que además fue bloqueado para IDE porque
re-ejecutar casos I/O no es el contrato de esa modalidad.

Abrir la aplicación desde el workspace vivo del alumno tiene tres problemas:

1. El docente necesita una ejecución reproducible de lo entregado, no de un estado que el alumno
   puede seguir cambiando.
2. El workspace puede estar hibernado, congelado o pertenecer a otro docente.
3. Un preview no puede compartir el volumen del alumno: escribir en él contamina la entrega y una
   app defectuosa podría dejarlo inutilizable.

## Decisión

### 1. Contenedor efímero sobre una copia del snapshot

Cada clic en **Probar aplicación viva** crea:

- Un volumen nombrado propio (`solv-review-env-<id>`), independiente del workspace del alumno.
- Un contenedor efímero con la imagen de la plantilla de la práctica.
- Una copia de los archivos de `submissions.code`, separados con el mismo parser del snapshot que
  usa SpeedGrader (`# === FILE: <ruta> ===`).

El contenedor no monta el volumen del alumno, no abre OpenVSCode y no comparte red de satélites. El
volumen de revisión se elimina junto con el contenedor cuando el preview vence.

### 2. La receta de arranque es explícita

El editor del laboratorio (paso **Entorno de Ejecución y Entrega**) guarda dos campos en
`exercises.config.ide`:

| Campo | Tipo | Regla |
| --- | --- | --- |
| `app_run_command` | string | Comando que arranca la app dentro del contenedor. Obligatorio para habilitar el preview. Se ejecuta con `sh -lc` después del setup de la plantilla. |
| `app_port` | int | Puerto TCP donde escucha la app. Si se omite, se usa `3000`. |

No se infiere el comando del `package.json`, del `Dockerfile` ni de la estructura del snapshot. Un
proyecto sin receta responde `409 app_run_command_missing` y explica al docente qué configurar. El
setup de la plantilla (`lab_templates.setup_script`) se ejecuta una vez por copia antes del comando;
si falla, el contenedor no arranca a medias.

### 3. La app se abre en una pestaña nueva, nunca en un iframe

La respuesta entrega una URL y la vista la abre con `window.open`/`target="_blank"` y
`rel="noopener noreferrer"`. Un iframe queda descartado: la app del alumno puede enviar
`X-Frame-Options` o una CSP que bloquee el embedding, y el docente necesita la aplicación real, no
una versión condicionada por el shell.

La pestaña se reserva en el mismo clic que dispara el `POST`, antes de esperar la respuesta, para que
el navegador no bloquee la apertura. Si la API falla, la pestaña vacía se cierra y el error se
muestra en el SpeedGrader.

### 4. Ciclo de vida automático, sin botón «Detener»

`review_test_environments` registra `status`, `container_id`, `volume_name`, `access_url`,
`last_activity_at`, `expires_at` y los últimos contadores de red. El estado activo es
`starting | running`; un índice parcial único permite un solo preview activo por entrega, así que un
fallo o un vencimiento libera el índice para reintentar sin borrar el historial.

El QoS Worker ejecuta el barrido en cada ciclo:

- Si el contador de red (`rx + tx`) creció desde el ciclo anterior, renueva el TTL 15 minutos.
- Si no hay tráfico y `expires_at` ya pasó, detiene el contenedor, elimina el volumen y marca
  `expired`.
- Un preview en `starting` que supera el tiempo de arranque más un margen se marca `failed` y se
  libera.
- Si Docker no responde, no se borra nada: se reintenta en el ciclo siguiente.

No hay una ruta de «detener» en la UI. El docente solo prueba la aplicación; el sistema es el dueño
del apagado, igual que en el resto de los laboratorios (ADR-011).

### 5. Admisión por OOM_GUARD

Antes de crear recursos, el servicio consulta `HostMonitor.CanAllocateMemory` con la memoria de la
plantilla, con piso de 512 MB. Si el nodo no tiene RAM, responde `503 oom_guard` y no crea volumen ni
contenedor. La memoria del preview no se suma dos veces a la de un laboratorio del alumno: son
contenedores distintos y el control de admisión ya descuenta la reserva de satélites compartidos.

### 6. Contrato de API

```http
POST /api/v1/teacher/submissions/{id}/test-environment
```

- **Headers:** `X-User-Id`, `X-User-Role: teacher | admin`, `X-Tenant-Id`.
- **Cuerpo:** vacío (`{}`).
- **Respuesta `200`:**
  ```json
  {
    "data": {
      "url": "https://review-3f2a9c1b7d4e.ejemplo.edu",
      "expires_at": "2026-11-01T18:45:00Z"
    },
    "error": "",
    "message": "Entorno de prueba listo"
  }
  ```
- **Idempotencia:** si la entrega ya tiene un preview `running`, se devuelve la misma URL y se
  renueva la ventana de inactividad. Si está `starting`, responde `409 review_environment_starting`.
- **Errores de configuración:** `409 review_environment_requires_ide`,
  `409 app_run_command_missing`, `409 project_snapshot_unavailable`,
  `409 template_image_missing`, `409 invalid_app_port`.
- **Errores de infraestructura:** `503 oom_guard`, `502 app_not_ready`, `502 app_start_failed`,
  `503 review_environment_unavailable`.
- **Autorización:** un estudiante recibe `403`; una entrega de otro tenant, `404`.

El esquema de la tabla vive en `backend/migrations/00044_review_test_environments.sql` y es
idempotente. No se agrega una tabla de archivos: el snapshot sigue siendo `submissions.code`.

## Consecuencias

- El docente puede comprobar que la entrega arranca sin tocar el workspace del alumno y sin pedirle
  una nueva entrega.
- Un preview sin receta no se inventa: la UI muestra el error y el editor indica los dos campos que
  faltan.
- El costo de RAM está acotado por el TTL y por OOM_GUARD; el contenedor de preview se detiene y su
  volumen se borra, así que no compite con el laboratorio del alumno.
- El snapshot ignora `node_modules`, `__pycache__`, `.venv` y binarios; el setup de la plantilla debe
  instalar dependencias. Un comando que dependa de una base satélite del nodo no funciona en este
  preview: el contenedor solo ejecuta la app y el setup local de la plantilla. Es una limitación
  explícita de esta fase, no un fallo silencioso.
- La URL no se publica en un iframe ni se firma como sesión de alumno: es una ruta de revisión
  docente con su propio ciclo de vida.
- Los previews no aparecen en el panel de laboratorios activos del alumno y el recolector de zombies
  los excluye por la etiqueta `solv.review_environment=true`.

## Alternativas descartadas

| Alternativa | Motivo |
| --- | --- |
| Reutilizar el workspace vivo del alumno | El estado cambia, puede estar congelado o hibernado y el docente no debe escribir en la entrega |
| Iframe embebido en SpeedGrader | CSP y `X-Frame-Options` de la app lo bloquean; además impide abrir herramientas externas |
| Botón «Detener» manual | Duplica el dueño del ciclo de vida y deja contenedores si el docente cierra la pestaña |
| Inferir el comando de arranque | `package.json`, `Dockerfile` y estructura del snapshot no alcanzan para saber el comando correcto; adivinar produce fallos difíciles de explicar |
| Ejecutar la app en el contenedor del juez | El juez ejecuta un archivo y casos I/O; un proyecto IDE necesita un servidor y un puerto, no un runner de casos |
| Un volumen compartido entre previews | Un preview defectuoso contamina al siguiente y rompe el aislamiento de la entrega |

## Verificación

- `go build ./...` y `go test` de `internal/core/services`, `internal/delivery/http` con los tests
  nuevos del servicio y del mapeo de errores.
- `npm run build`, `npm run lint:styles` y el spec del SpeedGrader.
- Prueba manual con Docker real: publicar una práctica IDE con `app_run_command`, entregar, abrir
  SpeedGrader, pulsar **Probar aplicación viva** y comprobar que la pestaña carga la app y que el
  preview se apaga solo tras la ventana de inactividad.
