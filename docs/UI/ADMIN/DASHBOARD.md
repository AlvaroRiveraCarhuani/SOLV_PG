# Vista 1: Dashboard del Administrador de Institución (Centro de Salud y Recursos)

> **Especificación Oficial de Interfaz, Componentes y Wireframes**  
> **Rol:** Administrador de Institución (Tenant Admin)  
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System / ADR-014 / ADR-024 / ADR-027 / Slice 16  

---

## 1. Diagrama de Arquitectura de Pantalla (Mermaid HD)

```mermaid
graph TD
    subgraph ShellAdmin ["Shell del Administrador (Layout de Alta Densidad)"]
        Topbar["Topbar Superior: Logo Institucional | Host Tag | [Personalizar Vista] | Perfil Admin"]
        Sidebar["Sidebar Izquierdo (5 Módulos): Inicio y Salud | Docentes | Plantillas | Configuración | Auditoría"]
        
        subgraph MainContent ["Área Principal de Contenido"]
            subgraph KpisHardware ["1. Tarjetas KPI de Hardware Real (Above the Fold)"]
                KpiRAM["1. Memoria RAM (28.2 / 32.0 GB - Umbral Semántico 88%)"]
                KpiCPU["2. Procesamiento CPU (6.8 / 16 vCPUs - Carga media 42%)"]
                KpiDisk["3. Almacenamiento NVMe (184 / 512 GB - Espacio en /var/lib/docker)"]
                KpiContainers["4. Concurrencia (28 Activos / 40 Máx - 4 Hibernados, 0 Caídos)"]
            end
            
            subgraph GridMonitoreo ["2. Grilla de Monitoreo Macro (2 Columnas)"]
                ColMaterias["Columna Izquierda (2/3): Distribución de Carga por Materia"]
                ColIncidencias["Columna Derecha (1/3): Panel de Incidencias Técnicas (OOM / Fallos)"]
            end

            subgraph ModalDetalle ["3. Modal de Contenedores por Materia"]
                ModalBarra["Barra de Filtro: Input de búsqueda + Selector de estado"]
                ModalTabla["Tabla de Workspaces de la Materia (RAM, Estado, Acción Reiniciar/Pausar)"]
            end
        end
    end

    Topbar --> MainContent
    Sidebar --> MainContent
    KpisHardware --> GridMonitoreo
    ColMaterias -->|Clic en [Ver Contenedores]| ModalDetalle
```

---

## 2. Anatomía Visual y Wireframes ASCII Técnicos

### 2.1 Dashboard Principal de Monitoreo

La pantalla organiza la información de infraestructura para evitar saturar al administrador con listas interminables de estudiantes:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ SOLV | Universidad Adventista de Bolivia                 Host: asus-lab-srv · Docker 27.2  Admin │
├──────────────┬───────────────────────────────────────────────────────────────────────────────────┤
│              │ SALUD Y RECURSOS DEL SERVIDOR                      [lucide:sliders] [Personalizar]│
│ [x] Inicio   │ Monitoreo macro de hardware, carga por materia e incidencias de infraestructura.   │
│ [ ] Docentes ├──────────────────────┬──────────────────────┬──────────────────┬──────────────────┤
│ [ ]Plantillas│ 1. MEMORIA RAM       │ 2. PROCESAMIENTO CPU │ 3. DISCO / NVME  │ 4. CONTENEDORES  │
│ [ ] Config.  │ [||||||||||||||=] 88%│ [||||||||    ] 42%   │ [||||||      ]36%│ [|||||||||||| ]70%│
│ [ ] Auditoría│ 28.2 / 32.0 GB       │ 6.8 / 16 vCPUs       │ 184 / 512 GB     │ 28 / 40 Máx      │
│              │ [!] 3.8 GB libres    │ Normal               │ 328 GB libres    │ 4 Hibernados     │
│              ├──────────────────────┴──────────────────────┴──────────────────┴──────────────────┤
│              │ DISTRIBUCIÓN DE CARGA POR MATERIA (2/3)   │ INCIDENCIAS TÉCNICAS (1/3)            │
│              │ ┌───────────────────────────────────────┐ │ ┌───────────────────────────────────┐ │
│              │ │ Materia / Curso   │ Activos │ RAM Cons│ │ │ [!] OOM Killed (Exit code 137)    │ │
│              │ ├───────────────────┼─────────┼─────────┤ │ │ WS-089 (Carlos Ruiz en Prog. Avan)│ │
│              │ │ Programación Avan │ 18 Alum │ 4.3 GB  │ │ │ Excedió 512MB por bucle de memoria│ │
│              │ │ Prof. C. García   │         │[Ver Dtl]│ │ │ [Ver Logs]    [Reiniciar]         │ │
│              │ ├───────────────────┼─────────┼─────────┤ │ ├───────────────────────────────────┤ │
│              │ │ Algoritmos Complej│ 10 Alum │ 2.5 GB  │ │ │ [i] Umbral de RAM del Host        │ │
│              │ │ Prof. A. Torres   │         │[Ver Dtl]│ │ │ Servidor al 88%. El servicio QoS  │ │
│              │ ├───────────────────┼─────────┼─────────┤ │ │ pausará entornos tras 15m inact.  │ │
│              │ │ Bases de Datos I  │ 0 Alum  │ 0 MB    │ │ └───────────────────────────────────┘ │
│              │ │ Prof. M. López    │ (4 Hib) │[En rep] │ │                                       │ │
│              │ └───────────────────────────────────────┘ │                                       │ │
└──────────────┴───────────────────────────────────────────┴───────────────────────────────────────┘
```

---

### 2.2 Modo Personalización de la Vista (Drag & Drop con Persistencia Local)

Al pulsar el botón `[Personalizar]`, la interfaz entra en modo de edición interactivo mediante `@angular/cdk/drag-drop`:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ [i] Modo Personalización: Arrastrá los bloques para ordenar la vista. [Cancelar] [Guardar Orden] │
├──────────────┬───────────────────────────────────────────────────────────────────────────────────┤
│              │ ┌ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - ┐ │
│ [x] Inicio   │ : [::] 1. MEMORIA RAM       [::] 2. CPU            [::] 3. DISCO  [::] 4. CONTEN: │
│              │ └ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - ┘ │
│              │ ┌ - - - - - - - - - - - - - - - - - - - ┐ ┌ - - - - - - - - - - - - - - - - - - ┐ │
│              │ : [::] DISTRIBUCIÓN DE CARGA POR MATERIA: : [::] INCIDENCIAS TÉCNICAS           : │
│              │ :                                       : :                                     : │
│              │ └ - - - - - - - - - - - - - - - - - - - ┘ └ - - - - - - - - - - - - - - - - - - ┘ │
└──────────────┴───────────────────────────────────────────────────────────────────────────────────┘
```

- **Persistencia en Navegador:** La disposición final de las tarjetas y paneles se almacena en el `localStorage` del cliente (`solv_admin_dashboard_layout`), garantizando que cada administrador mantenga su distribución personalizada sin generar tráfico ni escrituras en la base de datos.

---

### 2.3 Modal: Detalle de Contenedores por Materia (Con Búsqueda y Filtro)

Al hacer clic en `[Ver Dtl]` en una materia, se abre una ventana modal para inspeccionar a los estudiantes de esa clase sin saturar la pantalla principal:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Contenedores Activos: Programación Avanzada (18 alumnos)                             [lucide:x]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [lucide:search] Buscar alumno o ID... (ej: Carlos, WS-089)   │ Filtro: [ Todos los estados     v ]│
│                                                              │   ├── Todos los estados           │
│                                                              │   ├── En ejecución (Running)      │
│                                                              │   └── Con fallos (OOM Killed)     │
├──────────────────────────────────────────────────────────────┴───────────────────────────────────┤
│ ┌──────────────┬──────────────────┬──────────────────────┬─────────────┬───────────────────────┐ │
│ │ ID Workspace │ Estudiante       │ Memoria Consumida    │ Estado      │ Acción                │ │
│ ├──────────────┼──────────────────┼──────────────────────┼─────────────┼───────────────────────┤ │
│ │ WS-089       │ Carlos Ruiz      │ 512 MB (Límite alcanz│ [OOM Killed]│ [Reiniciar Contenedor]│ │
│ │ WS-090       │ Alvaro Rivera    │ 210 MB / 512 MB      │ [Running]   │ [Pausar]              │ │
│ │ WS-091       │ Elena Morales    │ 195 MB / 512 MB      │ [Running]   │ [Pausar]              │ │
│ └──────────────┴──────────────────┴──────────────────────┴─────────────┴───────────────────────┘ │
│                                                                                      [ Cerrar ]  │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Especificación Visual y Semántica de Estados

1. **Semáforos de Umbrales de Hardware (Host Linux):**
   - **Memoria RAM:**
     - `< 79%`: Saludable (`#16A34A` - Verde).
     - `80% - 89%`: Alerta de ocupación (`#D97706` - Ámbar).
     - `>= 90%`: Crítico (`#DC2626` - Rojo).
   - **Disco NVMe (`/var/lib/docker`):**
     - `< 70%`: Saludable (`#16A34A`).
     - `70% - 85%`: Advertencia preventiva para depuración de imágenes intermedias.
     - `> 85%`: Crítico con riesgo de corrupción de volumen.
2. **Traducción Semántica de Incidencias de Contenedores:**
   - La plataforma intercepta el evento de terminación `Exit code 137` de Linux (`oom_score_adj`) y lo traduce al administrador como `"OOM Killed - Límite de Memoria Excedido"`, indicando qué estudiante y en qué materia ocurrió para proceder al reinicio con un solo clic.

---

## 4. Reglas de Negocio y Separación de Responsabilidades

1. **Escalabilidad por Agregación:**
   - Para evitar degradación en facultades con más de 300 alumnos concurrentes, la vista macro siempre agrega datos a nivel de curso/materia. La consulta de contenedores individuales se delega al modal contextual paginado o filtrable.
2. **Límites de los Roles:**
   - El **docente** supervisa el avance pedagógico de sus estudiantes en su curso.
   - El **administrador** atiende contingencias técnicas de infraestructura (reiniciar contenedores con errores de memoria, supervisar cuotas de disco y memoria del host).
3. **Persistencia Desacoplada del Layout:**
   - La personalización visual mediante Drag & Drop no muta registros en el backend; se resuelve localmente en el cliente, respetando la velocidad del navegador.

---

## 5. Contrato de Integración y Endpoints (v0.16.0)

| Método | Endpoint | Parámetros / Query | Propósito |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/admin/dashboard/metrics` | — | Métricas en tiempo real del host: RAM usada/total, vCPUs, espacio en disco y total de contenedores. |
| `GET` | `/api/v1/admin/dashboard/courses-load` | `?period_id={id}` | Distribución agregada de laboratorios activos y memoria consumida por materia. |
| `GET` | `/api/v1/admin/courses/{id}/workspaces` | `?search={q}&status={status}` | Lista detallada de contenedores de una materia específica con soporte de búsqueda y filtrado. |
| `GET` | `/api/v1/admin/dashboard/incidents` | `?limit=10` | Lista de anomalías técnicas recientes (OOM Killer, errores de inicio de contenedor). |
| `POST` | `/api/v1/workspaces/{id}/restart` | — | Reinicia un contenedor caído por OOM restableciendo sus recursos de forma inmediata. **Ruta real registrada en el router**; el antiguo prefijo `/admin/workspaces/...` no existía y devolvía 404 silencioso. |
| `POST` | `/api/v1/workspaces/{id}/pause` | — | Pausa forzada de un contenedor con comportamiento anómalo (el rol `admin` pasa la guarda del handler). Mismo caso: el prefijo `/admin/...` devolvía 404. Tras cada pausa/reinicio el dashboard recarga el modal (`reloadToken`) y muestra toast de éxito/error. |

---

## 6. Integridad de los Datos Mostrados (Salud y Recursos)

1. **Identidades reales, nunca UUIDs**: `GET /admin/dashboard/containers`, `GET /admin/courses/{id}/workspaces` y `GET /admin/dashboard/incidents` resuelven nombre y correo del estudiante desde el directorio (`users` vía `ListStudentsDirectory`). Si el estudiante no está en el directorio se muestra el rótulo neutro «Estudiante» **sin correo y sin UUID**; jamás se concatena el UUID para fabricar un correo (`uuid@uab.edu.bo` está prohibido). Las materias también se resuelven por nombre (fallback «Materia»), no por UUID.
2. **Memoria real o «Sin métricas»**: el consumo por contenedor se lee de Docker (`GetContainerMetrics`) y se expone con el flag `metrics_available`. Sin telemetría (`false`) la fila del modal muestra el rótulo *Sin métricas* en lugar de una barra inventada; está prohibido el antiguo cálculo `limit / 3`. El campo `image_tag` de estos DTO queda vacío hasta resolverse desde `lab_templates.docker_image` (nada de tags hardcodeados).
3. **Estados**: `running`/`oom_killed` provienen de la tabla `workspaces`; el semáforo de la tabla de carga usa `--tenant-primary` para RAM y alumnos (el morado `--color-purple-600` quedó fuera de la paleta).
4. **Cobertura de tests**: `admin_courses_test.go` verifica que ni el nombre ni el correo contengan el UUID (fallback y directorio), y que las métricas sin Docker devuelvan `(0, false)`.
