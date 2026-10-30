# Mapa Integral de Vistas y Arquitectura de Navegación de SOLV

> **Documento Maestro de Arquitectura de Información (IA)**  
> **Sistema:** SOLV — Plataforma de Orquestación de Laboratorios Virtuales  
> **Alcance:** Multi-tenant Dinámico · Todos los Roles (Admin, Docente, Estudiante) · Contratos de Interfaz  
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System / Decisión D1–D6  

---

## 1. Arquitectura Global y Resolución Dinámica Multi-Tenant

El sistema opera bajo un esquema **100% Multi-Tenant Dinámico**. No existen dominios, nombres de facultades ni logotipos fijos en el código.

### 1.1. Flujo de Entrada y Derivación por Rol

```mermaid
graph TD
    Entry["Acceso Web: /login"] --> GoogleSSO["Autenticación Google Workspace / SSO"]
    GoogleSSO --> BackendAuth["Backend: /api/v1/auth/callback"]
    BackendAuth --> TenantResolution["Resolución de Tenant (Dominio email -> tenant_id)"]
    TenantResolution --> IssueJWT["Emisión de JWT (solv_session HttpOnly con tenant_id, user_id, role)"]
    IssueJWT --> AppShell["Carga Frontend Angular (Zoneless + Signals)"]
    AppShell --> TenantBootstrap["TenantService: Carga dinámica de Configuración (Nombre, Logo, Paleta OKLCH)"]
    
    TenantBootstrap --> RoleGuard{"Guard de Enrutamiento"}
    RoleGuard -->|"role == 'admin'"| AdminRoute["/admin/dashboard (Shell de Administración)"]
    RoleGuard -->|"role == 'teacher'"| DocenteRoute["/docente/dashboard (Shell de Docente)"]
    RoleGuard -->|"role == 'student'"| EstudianteRoute["/estudiante/dashboard (Shell de Estudiante)"]
```

### 1.2. Principio de Inyección Visual de Tenant
* **Topbar e Identidad:** El componente `solv-topbar` lee de `TenantService.config()` de forma reactiva:
  * Si hay `logo_url`, renderiza la imagen institucional.
  * Si no hay logo o falla la carga, renderiza `brand-shield` con las iniciales calculadas (`tenantInitials()`).
  * Inyecta la variable CSS `--tenant-primary` en formato OKLCH ajustada para contraste WCAG AA.

---

## 2. Árbol General de Navegación

```text
/ (Raíz)
├── /login .............................. Portal institucional de inicio de sesión
│
├── /admin (Rol: Administrador de Institución)
│   ├── /admin/dashboard ................ Salud del servidor, consumo de recursos y contenedores
│   ├── /admin/docentes ................. Gestión del plantel docente e invitaciones
│   ├── /admin/plantillas ............... Catálogo y aprobación de plantillas Docker
│   ├── /admin/configuracion ............ Períodos académicos, marca institucional y servidor
│   └── /admin/auditoria ................ Registro inmutable de eventos y trazabilidad
│
├── /teacher (Rol: Docente)
│   ├── /teacher/dashboard .............. Hoy: qué espera al docente, con la tabla de materias
│   ├── /teacher/aula ................... Qué pasa ahora en clase: entornos abiertos y telemetría
│   ├── /teacher/por-calificar .......... Cola de corrección agrupada por práctica
│   ├── /teacher/cursos/:id ............. Materia: prácticas, entregas, alumnos y libreta
│   ├── /teacher/revision/:submissionId . SpeedGrader: revisión de una entrega
│   ├── /teacher/exercises .............. Banco de ejercicios del docente
│   ├── /teacher/templates .............. Catálogo de plantillas de entorno
│   ├── /teacher/anti-plagio ............ Similitud estructural entre entregas
│   └── /teacher/configuracion .......... Preferencias de asistencia con IA
│
└── /estudiante (Rol: Estudiante)
    ├── /estudiante/dashboard ........... Cuaderno académico (cursos, hojas de trabajo y entregas)
    ├── /estudiante/laboratorios/:id .... Modo Inmersivo: OpenVSCode Server (95%+ viewport)
    └── /estudiante/juez/:ejercicioId ... Juez Virtual: Monaco Editor, AST Semgrep y veredictos
```

---

## 3. Rol Administrador: Fichas Técnicas de Pantallas

### 3.1. Dashboard de Infraestructura y Salud (`/admin/dashboard`)
* **Propósito:** Supervisar la estabilidad del servidor host (CPU, RAM, Disco) y la carga de contenedores en tiempo real.
* **Información que muestra:**
  * 4 Tarjetas KPI de Hardware: Memoria RAM utilizada/total con barra de umbral semántico, Carga media de vCPU, Almacenamiento NVMe (`/var/lib/docker`) y Concurrencia de contenedores (Activos / Hibernados / Límite).
  * Gráfico de Carga Temporal: Consumo de los últimos 60 minutos con línea proyectada de capacidad máxima.
  * Tabla de Contenedores Activos: Alumno, Curso, Imagen Docker base, RAM consumida, TTL restante y estado de red.
* **Acciones Principales:**
  * `[Hibernar Todo]` — Pasa todos los contenedores inactivos a estado hibernado para liberar RAM.
  * `[Detener Contenedor]` — Finaliza una instancia específica que exceda límites de consumo.
  * `[Ver Logs de Docker]` — Abre un panel lateral con salida directa del motor Docker.

### 3.2. Gestión de Docentes e Invitaciones (`/admin/docentes`)
* **Propósito:** Administrar el ciclo de vida del personal docente asignado a la institución.
* **Información que muestra:**
  * Tabla de Docentes: Nombre completo, correo institucional, cursos asignados, último acceso y estado (Activo / Invitación Pendiente / Inactivo).
  * Contador de cupos docentes utilizados vs contratados.
* **Acciones y Disparadores:**
  * `[+ Invitar Profesor]` — Abre Modal de Invitación: solicita correo electrónico institucional; genera token de un solo uso con TTL de 72 horas y envía enlace por correo.
  * `[Reenviar Enlace]` — Renueva la vigencia de una invitación pendiente.
  * `[Revocar Acceso]` — Abre diálogo de confirmación para deshabilitar credenciales del profesor.

### 3.3. Catálogo y Gobernanza de Plantillas Docker (`/admin/plantillas`)
* **Propósito:** Mantener la lista blanca de imágenes Docker seguras que los docentes pueden seleccionar al crear laboratorios.
* **Información que muestra:**
  * Grilla de Plantillas Aprobadas: Nombre del perfil (ej. *Python Data Science*, *C++ Algoritmos*, *PostgreSQL 16*), tag fijado (prohibido `:latest`), límite de RAM por defecto y fecha de validación.
  * Bandeja de Solicitudes Pendientes: Peticiones enviadas por docentes solicitando imágenes nuevas.
* **Acciones y Disparadores:**
  * `[+ Nueva Plantilla]` — Modal para registrar imagen Docker aprobada, asignando cuota base de RAM/CPU y comprobando existencia en registro.
  * `[Revisar Solicitud]` — Abre diálogo de revisión técnica: inspección de capas, vulnerabilidades y aprobación/rechazo con justificación.

### 3.4. Configuración Institucional (`/admin/configuracion`)
* **Propósito:** Control centralizado de parámetros académicos, marca de la institución y políticas del servidor en 3 pestañas:
  * **Pestaña 1 (Períodos Académicos):** Listado de semestres (Activo, Planificado, Archivado). Botón `[+ Nuevo Período]` y acción `[Archivar Período]` (requiere confirmación manual escribiendo el código del semestre; pasa los cursos y laboratorios a modo solo lectura `:ro`).
  * **Pestaña 2 (Marca e Identidad White-Label):** Subida de logotipo institucional, nombre formal de la universidad y selector de color primario en OKLCH con validación automática de contraste WCAG AA (4.5:1). Incluye vista previa en vivo sin recargar la página.
  * **Pestaña 3 (Servidor y Respaldos):** Cuotas de RAM máxima por estudiante, tiempo de inactividad antes de hibernar (por defecto 15 minutos) y panel de respaldos automáticos con checksum SHA-256 y botón `[Generar Respaldo Ahora]`.

### 3.5. Registro de Auditoría (`/admin/auditoria`)
* **Propósito:** Consulta de registros inmutables de seguridad y eventos del sistema.
* **Información que muestra:**
  * Tabla paginada de eventos: Marca de tiempo ISO-8601, Actor (correo o sistema), Acción ejecutada (ej. `AUTH_LOGIN`, `CONTAINER_START`, `PERIOD_ARCHIVE`), Dirección IP y Veredicto/Estado HTTP.
  * Filtros avanzados por rango de fechas, actor y tipo de operación.
* **Acciones:**
  * `[Exportar CSV/JSON]` — Descarga del segmento filtrado para auditorías externas.

---

## 4. Rol Docente: Fichas Técnicas de Pantallas

### 4.1. Hoy (`/teacher/dashboard`)
* **Propósito:** Responder una sola pregunta: qué espera al docente hoy. Especificación completa en [DOCENTE/REDISENO_PANEL_DOCENTE.md](DOCENTE/REDISENO_PANEL_DOCENTE.md).
* **Información que muestra:**
  * Una línea de cabecera con la fecha y el período académico.
  * Filas de trabajo que se ocultan sin contenido: la práctica por vencer (o el aula abierta), la cola de corrección y los entornos abiertos.
  * Tabla de materias con inscritos, entornos abiertos ahora, entregas por calificar y alumnos sin entregar.
* **Contratos:** `GET /api/v1/teacher/overview?period_id=`, refrescado cada 60 segundos. Todas las cifras las calcula el servidor.
* **Acciones:**
  * Clic en el nombre de la materia — Navega a `/teacher/cursos/:id`.
  * `[Calificar]` — Navega a `/teacher/por-calificar`.
  * `[Ver entornos]` — Navega a `/teacher/aula`.
  * `[Abrir aula]` en la cabecera, solo cuando no hay aula abierta — Navega a `/teacher/aula` para abrir la sesión.
  * `[Ir al aula]` en la fila de sesión — Navega a `/teacher/aula` con el tablero de la clase en curso.

### 4.2. Detalle de Curso (`/docente/cursos/:id`)
* **Propósito:** Gestión interna de una materia específica mediante 3 pestañas:
  * **Laboratorios:** Lista cronológica de actividades prácticas con sus fechas límite y estado (Borrador / Publicado / Cerrado).
  * **Estudiantes:** Nómina de alumnos, estado de avance individual y enlaces directos a sus entornos.
  * **Métricas:** Estadísticas agregadas de tiempos de resolución, tasa de éxito y problemas comunes.

### 4.3. Asistente de Creación/Edición de Laboratorio (`/docente/laboratorios/nuevo`)
* **Propósito:** Formulario paso a paso para desplegar una nueva actividad práctica.
* **Flujo en 3 Pasos:**
  * **Paso 1 (Datos Generales):** Título, curso asignado, fechas de apertura y cierre, descripción pedagógica.
  * **Paso 2 (Modalidad Técnica):**
    * *Opción A - Entorno Completo (IDE):* Selección de plantilla Docker del catálogo de la institución, límite de RAM y tiempo de inactividad permitido.
    * *Opción B - Juez Virtual (Algoritmos):* Selección de lenguaje base, tiempo máximo de ejecución por caso de prueba y memoria límite.
  * **Paso 3 (Restricciones y Evaluación):** Carga de casos de prueba (públicos para el estudiante y privados para calificación) y reglas de análisis estático de código (Semgrep).
* **Acciones:**
  * `[Guardar Borrador]` — Guarda los cambios sin hacer visible el laboratorio a los alumnos.
  * `[Publicar a la Clase]` — Habilita el laboratorio en el calendario de los estudiantes.

### 4.4. Aula (`/teacher/aula`)
* **Propósito:** Responder una sola pregunta mientras ocurre la clase: quién está, quién falta y quién se trabó. Con aula abierta muestra el tablero de la sesión; sin aula muestra el formulario de apertura y, si hay entornos encendidos, su telemetría, sin lista de faltantes y sin botón de cerrar.
* **Contratos:** `GET /api/v1/teacher/sessions/open`, `GET /api/v1/teacher/sessions/{id}/board` (refresco cada 5 segundos), `POST /api/v1/teacher/sessions` y `PATCH /api/v1/teacher/sessions/{id}/close`.
* **Información que muestra (con aula):**
  * Cabecera con materia, hora de apertura y de fin, práctica de la sesión, presentes sobre inscritos y `[Cerrar aula]`.
  * Marcador: "Todo en orden: N de M entregaron, 0 trabados" o los chips de trabados, sin entregar, sin empezar y salieron.
  * Tabla de alumnos con estado (trabado, sin empezar, salió, entregó, trabajando), hora y veredicto de la entrega, intentos y entorno, con filtros y búsqueda.
  * Telemetría acotada a los entornos de la materia de la sesión.
* **Información que muestra (sin aula):** formulario de apertura —materia y práctica publicada opcional— y los entornos abiertos ahora de todas las materias.
* **Acciones:**
  * `[Abrir aula]` — Abre la sesión de una materia con una duración (por defecto 1 h 30); una segunda aula abierta en la misma materia responde conflicto.
  * `[Cerrar aula]` — Cierra la sesión; es idempotente y corta la presencia de esa clase.
  * Cierre por vencimiento — al llegar a `ends_at` el aula se cierra sola en la siguiente lectura o apertura, sin proceso periódico. La pantalla deja el modo en vivo y avisa a qué hora se cerró; la materia queda libre para abrir otra aula.
  * `[Consola]` — Abre la terminal de sombra del entorno del alumno, con la sesión auditada.
* **Presencia:** la emite el cliente del alumno cada minuto sobre el aula abierta de sus materias (`GET /api/v1/student/sessions/active`, `POST /api/v1/student/sessions/{id}/heartbeat`). Una señal de más de 120 segundos no cuenta.
* **Fuera de alcance:** reiniciar contenedores y ajustar recursos son acciones del administrador; el docente no opera infraestructura. El modo proyección, la agrupación de excepciones por causa y la actualización por SSE están especificados y pendientes.

### 4.5. Juez Virtual — Auditoría y Calificación (`/docente/juez`)
* **Propósito:** Banco de ejercicios algorítmicos y revisión manual de código entregado.
* **Información que muestra:**
  * Banco de Problemas del profesor clasificados por dificultad y etiquetas.
  * Cola de Envíos Pendientes de Calificación.
* **Pantalla de Auditoría de Código:**
  * Editor Monaco en modo solo lectura (`:ro`) con la solución enviada por el alumno.
  * Detalle de ejecución de casos de prueba (tiempos en milisegundos y memoria utilizada).
  * Resumen de reglas Semgrep aplicadas (ej. verificación de no uso de librerías prohibidas).
  * Campo para retroalimentación cualitativa y botón `[Confirmar Calificación]`.

---

## 5. Rol Estudiante: Fichas Técnicas de Pantallas

### 5.1. Dashboard del Estudiante (`/estudiante/dashboard`)
* **Propósito:** Centro de mando organizado bajo el modelo mental del **Cuaderno Académico**.
* **Información que muestra:**
  * Banner de Próximas Entregas: Alertas claras de laboratorios o ejercicios con fecha límite cercana.
  * Sección de Cursos: Cada materia funciona como una libreta donde se agrupan cronológicamente sus hojas de trabajo (laboratorios).
  * Estado de cada Laboratorio: Etiqueta semántica (*No iniciado*, *En progreso*, *Entregado*, *Calificado*).
* **Acciones:**
  * `[Iniciar Laboratorio]` / `[Continuar Trabajo]` — Abre la vista inmersiva del laboratorio activo.
  * `[Ver Retroalimentación]` — Muestra los comentarios y la nota otorgada por el docente.
* **Presencia en el aula:** el layout del estudiante consulta cada minuto si sus materias tienen un aula abierta (`GET /api/v1/student/sessions/active`) y emite su señal (`POST /api/v1/student/sessions/{id}/heartbeat`) mientras la pestaña está visible. Esa señal es la que el docente ve como "dentro" en `/teacher/aula`; se corta al cerrarse el aula o al dejar de estar matriculado.

### 5.2. Laboratorio Activo en Modo Inmersivo (`/estudiante/laboratorios/:id`)
* **Propósito:** Espacio de desarrollo libre de distracciones donde el 95%+ del área visual corresponde a **OpenVSCode Server**.
* **Componentes de la Pantalla:**
  * **Topbar Mínimo Superior:**
    * Nombre de la materia y número de laboratorio.
    * Indicador de tiempo restante de la sesión.
    * Botón `[Extender Tiempo]` (añade tiempo si las políticas del laboratorio lo permiten).
    * Botón `[Guardar y Salir]` (hiberna el contenedor de forma segura y regresa al dashboard).
  * **Área Central Completa:**
    * Iframe seguro comunicándose mediante `window.postMessage` para sincronización de estado, sin exponer identificadores de contenedor, puertos ni consumo interno.

### 5.3. Juez Virtual — Resolución y Evaluación (`/estudiante/juez/:ejercicioId`)
* **Propósito:** Resolución de problemas de lógica y algoritmos con evaluación formativa instantánea.
* **Componentes de la Pantalla:**
  * **Panel Izquierdo:** Consigna del problema, ejemplos de entrada/salida y restricciones de diseño pedagógico (lo que está permitido o prohibido usar).
  * **Panel Superior Derecho:** Monaco Editor interactivo con resaltado de sintaxis y autocompletado para el lenguaje configurado.
  * **Panel Inferior Derecho (Consola y Veredictos):**
    * Botón `[Ejecutar Pruebas]` — Corre el código frente a los casos de prueba visibles.
    * Botón `[Enviar Solución]` — Ejecuta la validación completa (casos privados + análisis estático Semgrep).
    * Indicador de Veredicto Semántico:
      * `AC` (Aceptado): Verde.
      * `WA` (Respuesta Incorrecta): Rojo.
      * `TLE` (Tiempo Límite Excedido): Ámbar.
      * `RE` (Error de Ejecución): Naranja.
      * `AST_BLOCKED` (Restricción de Código Violada): Rojo intenso con explicación constructiva de la regla infringida.

---

## 6. Matriz de Estados Globales del Sistema

| Estado Técnico | Color Semántico | Significado para el Usuario |
| :--- | :--- | :--- |
| `running` | Verde éxito | Contenedor o servicio funcionando con normalidad |
| `pending` | Amarillo advertencia | Contenedor levantando, comprobación en cola o invitación enviada |
| `hibernated` | Gris neutro | Contenedor pausado en disco para preservar recursos |
| `failed` / `oom_killed` | Rojo error | Proceso finalizado por error o falta de memoria |
| `AC` | Verde éxito | Solución aprobada en todos los casos |
| `WA` | Rojo error | La salida generada no coincide con la esperada |
| `TLE` | Ámbar advertencia | El algoritmo superó el tiempo máximo asignado |
| `RE` | Naranja error | Error fatal durante la ejecución del programa |
| `AST_BLOCKED` | Rojo intenso | El código contiene estructuras o funciones no permitidas por la cátedra |
