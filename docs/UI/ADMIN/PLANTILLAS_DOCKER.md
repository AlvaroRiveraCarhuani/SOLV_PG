# Vista 5: Gobernanza y Aprobación de Plantillas Docker

> **Especificación Oficial de Interfaz, Componentes y Wireframes**  
> **Rol:** Administrador de Institución (Tenant Admin)  
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System / ADR-030 / Decisión D4  

---

## 1. Diagrama de Arquitectura de Gobernanza (Mermaid HD)

```mermaid
sequenceDiagram
    autonumber
    participant D as Docente / Wizard
    participant B as Backend API (Go)
    participant A as Administrador / UI
    participant R as Registro Docker Hub

    D->>B: POST /api/v1/docker-templates/requests (Perfil, imagen, justificación)
    B->>B: Registra solicitud en estado [pending] y emite evento
    B-->>A: Notificación en bandeja de administración (Badge numérico)

    alt Administrador Aprueba Solicitud
        A->>A: Abre modal [Revisar y Aprobar]
        A->>A: Asigna cuota real de RAM (256/512/1024 MB) y CPU
        A->>A: Configura visibilidad (Global / Materia solicitante / Específica)
        A->>B: POST /api/v1/admin/docker-templates/requests/{id}/approve
        B->>R: Verifica existencia de imagen y tag en Docker Hub
        B->>B: Inserta en tabla docker_templates (status = active)
        B-->>A: Publicada en Catálogo Oficial
        B-->>D: Notifica al docente que su plantilla ya está disponible
    else Administrador Rechaza Solicitud
        A->>A: Abre modal [Rechazar Solicitud]
        A->>A: Ingresa motivo obligatorio (mínimo 15 caracteres)
        A->>B: POST /api/v1/admin/docker-templates/requests/{id}/reject (reason)
        B->>B: Actualiza estado a [rejected] con feedback
        B-->>D: Muestra motivo del rechazo en el panel del docente
    end
```

---

## 2. Anatomía Visual y Wireframes ASCII Técnicos

La interfaz presenta dos pestañas principales: **Solicitudes Pendientes** (con contador reactivo) y **Catálogo Activo**. Además, dispone del alternador de vista **[ Tarjetas | Lista ]** para adaptarse a la densidad de trabajo del administrador.

### 2.1 Vista en Tarjetas: Grilla Lado a Lado (Solicitudes Pendientes)

En formato tarjetas, las solicitudes se disponen en una grilla responsiva de 2 columnas donde cada tarjeta permite leer la justificación pedagógica completa sin truncamiento:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ SOLV | Universidad                                                           [lucide:bell] Admin │
├──────────────┬───────────────────────────────────────────────────────────────────────────────────┤
│              │ PLANTILLAS DE ENTORNOS                              Vista: [ Tarjetas | Lista ]   │
│ [ ] Inicio   │ Pestañas: [x] Solicitudes Pendientes (2)   |   [ ] Catálogo Activo (6)            │
│ [ ] Docentes ├───────────────────────────────────────────────────────────────────────────────────┤
│ [*]Plantillas│ ┌───────────────────────────────────────┐ ┌───────────────────────────────────────┐ │
│ [ ] Config.  │ │ Rust 1.80 con Herramientas            │ │ C++20 con Compilador Clang 18         │ │
│ [ ] Auditoría│ │ [ Pendiente ]      Perfil: Estándar   │ │ [ Pendiente ]      Perfil: Estándar   │ │
│              │ │ Prof. Carlos García · Prog. Avanzada  │ │ Prof. Ana Torres · Algoritmos Complej.│ │
│              │ │ Imagen: rust:1.80-slim                │ │ Imagen: silkeh/clang:18               │ │
│              │ │                                       │ │                                       │ │
│              │ │ Justificación pedagógica:             │ │ Justificación pedagógica:             │ │
│              │ │ "Práctica de concurrencia segura sin  │ │ "Necesitamos soporte de Concepts y    │ │
│              │ │ recolector de basura (GC) y Clippy."  │ │ Ranges para optimización en memoria." │ │
│              │ │                                       │ │                                       │ │
│              │ │ [ Rechazar ]    [ Revisar y Aprobar ] │ │ [ Rechazar ]    [ Revisar y Aprobar ] │ │
│              │ └───────────────────────────────────────┘ └───────────────────────────────────────┘ │
└──────────────┴───────────────────────────────────────────────────────────────────────────────────┘
```

---

### 2.2 Vista en Lista Compacta (Tabla de Alta Densidad)

Para cuando existen múltiples solicitudes acumuladas, la vista de lista proporciona una tabla rápida con acciones directas:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ SOLV | Universidad                                                           [lucide:bell] Admin │
├──────────────┬───────────────────────────────────────────────────────────────────────────────────┤
│              │ PLANTILLAS DE ENTORNOS                              Vista: [ Tarjetas | Lista ]   │
│ [ ] Inicio   │ Pestañas: [x] Solicitudes Pendientes (2)   |   [ ] Catálogo Activo (6)            │
│ [ ] Docentes ├───────────────────────────────────────────────────────────────────────────────────┤
│ [*]Plantillas│ ┌───────────────────────────────────────────────────────────────────────────────┐ │
│ [ ] Config.  │ │ Plantilla / Imagen     │ Solicitante   │ Materia       │ Perfil   │ Fecha     │ Acciones│ │
│ [ ] Auditoría│ ├────────────────────────┼───────────────┼───────────────┼──────────┼───────────┼─────────┤ │
│              │ │ Rust 1.80              │ C. García     │ Prog. Avanzada│ Estándar │ Hoy 10:30 │[Revisar]│ │
│              │ │ rust:1.80-slim         │               │               │          │           │         │ │
│              │ │ C++20 Clang 18         │ A. Torres     │ Algor.Complej.│ Estándar │ Ayer 16:20│[Revisar]│ │
│              │ │ silkeh/clang:18        │               │               │          │           │         │ │
│              │ └───────────────────────────────────────────────────────────────────────────────┘ │
└──────────────┴───────────────────────────────────────────────────────────────────────────────────┘
```

---

### 2.3 Modal: [ Revisar y Aprobar ] con Selector Compacto y Revelación Progresiva

Este diálogo resuelve la asignación de hardware y el alcance sin sobrecargar la pantalla:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Aprobar Plantilla y Fijar Cuotas                                                     [lucide:x]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Imagen Docker Validada: [ rust:1.80-slim                                                       ] │
│                                                                                                  │
│ Asignación de Memoria RAM por Contenedor Estudiante:                                             │
│ [ 256 MB ]   [ 512 MB (Sugerido) ]   [ 1024 MB ]                                                 │
│                                                                                                  │
│ Límite de CPU:                                                                                   │
│ [ 0.5 vCPU ]   [ 1.0 vCPU (Sugerido) ]   [ 2.0 vCPU ]                                            │
│                                                                                                  │
│ Alcance y Visibilidad:                                                                           │
│ [ Catálogo Global (Toda la universidad)                                                        v ]│
│   ├── Catálogo Global (Toda la universidad)                                                      │
│   ├── Solo materia solicitante ('Programación Avanzada')                                         │
│   └── Otra materia específica...  ────────────────────────┐                                      │
│                                                           ▼                                      │
│ (Se revela únicamente si se elige 'Otra materia'):                                               │
│ [ Seleccionar materia de destino: Sistemas Operativos I                                        v ]│
│                                                                                                  │
│                                              [ Cancelar ]   [ Confirmar y Publicar en Catálogo ] │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

### 2.4 Modal: [ Rechazar Solicitud ]

Requiere justificación obligatoria para evitar retroalimentación opaca hacia el profesor:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Rechazar Solicitud de Plantilla                                                      [lucide:x]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Docente solicitante: Prof. Carlos García · Programación Avanzada                                 │
│ Plantilla: Rust 1.80 con Herramientas                                                            │
│                                                                                                  │
│ Motivo de Rechazo (Obligatorio, visible para el docente):                                        │
│ ┌──────────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │ La imagen solicitada excede el límite de peso en disco del cluster local (máximo 1.5 GB).    │ │
│ │ Por favor solicitar una variante alpine o slim con menos paquetes preinstalados.            │ │
│ └──────────────────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                                  │
│                                                      [ Cancelar ]   [ Confirmar Rechazo ]        │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

### 2.5 Pestaña 2: Catálogo Activo y Control de Ciclo de Vida

Permite supervisar los entornos en producción y desactivar versiones obsoletas:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Pestañas: [ ] Solicitudes Pendientes (2)   |   [x] Catálogo Activo (6 Oficiales)                 │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ┌──────────────────────────────────────────────────────────────────────────────────────────────┐ │
│ │ Entorno Oficial           │ Imagen Base           │ RAM / CPU   │ Labs Usándola │ Estado     │ Acción│ │
│ ├───────────────────────────┼───────────────────────┼─────────────┼───────────────┼────────────┼───────┤ │
│ │ Python 3.11 Data Science  │ solv/python:3.11-ds   │ 512MB / 1.0 │ 12 Prácticas  │ [Activa] │ [Pause│ │
│ │ Go 1.23 Backend Services  │ golang:1.23-bookworm  │ 512MB / 1.0 │ 8 Prácticas   │ [Activa] │ [Pause│ │
│ │ Java 21 LTS OpenJDK       │ eclipse-temurin:21    │ 1024MB/ 1.0 │ 15 Prácticas  │ [Activa] │ [Pause│ │
│ │ Node.js 20 LTS Fullstack  │ node:20-alpine        │ 512MB / 1.0 │ 6 Prácticas   │ [Activa] │ [Pause│ │
│ │ Python 3.9 Legacy         │ python:3.9-slim       │ 256MB / 0.5 │ 0 Prácticas   │ [Pausada]│ [Rean]│ │
│ └──────────────────────────────────────────────────────────────────────────────────────────────┘ │
│ Nota: Al pausar una plantilla, ya no aparece en el selector para nuevos laboratorios, pero no    │
│ altera los entornos ya creados ni las entregas pasadas.                                          │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Reglas de Negocio, Gobernanza y Seguridad

1. **Gobernanza Institucional de Recursos (Decisión D4):**
   - El docente solo sugiere la intensidad pedagógica (Ligera / Estándar / Intensiva). La potestad de asignar megabytes de RAM y vCPUs recae 100% en el Administrador para salvaguardar la capacidad del hardware.
2. **Validación de Imagen en Registro:**
   - Antes de completar la aprobación, el backend efectúa un sondeo al registro oficial para certificar que el repositorio y tag especificados existen y son públicamente descargables.
3. **Inmutabilidad y Deprecación Suave:**
   - Las plantillas en uso nunca se eliminan físicamente (`hard delete`) para preservar la reproducibilidad histórica de entregas y auditorías. Se marcan como `paused` para ocultarlas del asistente de creación de nuevos laboratorios.
4. **Visibilidad Granular:**
   - Por defecto, las plantillas aprobadas se publican en el Catálogo Global. No obstante, si se restringe a una materia, solo los docentes asignados a esa materia la verán en su lista de opciones.

---

## 4. Contrato de Integración y Endpoints (v0.16.0)

| Método | Endpoint | Parámetros / Payload | Propósito |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/admin/docker-templates/requests` | `?status=pending` | Lista solicitudes docentes pendientes de revisión. |
| `POST` | `/api/v1/admin/docker-templates/requests/{id}/approve` | `{ "ram_limit_mb": 512, "cpu_limit": 1.0, "scope": "global", "course_id": null }` | Aprueba la solicitud, fija límites y publica la plantilla. |
| `POST` | `/api/v1/admin/docker-templates/requests/{id}/reject` | `{ "reason": "La imagen excede el límite de peso..." }` | Rechaza la solicitud con justificación obligatoria. |
| `GET` | `/api/v1/admin/docker-templates` | `?include_paused=true` | Lista el catálogo institucional completo de plantillas. |
| `PATCH` | `/api/v1/admin/docker-templates/{id}/status` | `{ "is_active": false }` | Pausa o reactiva una plantilla en el catálogo oficial. |
