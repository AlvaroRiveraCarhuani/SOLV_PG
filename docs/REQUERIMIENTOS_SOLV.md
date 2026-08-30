# Especificación de Requerimientos del Sistema SOLV

> **Documento Oficial de Requerimientos Funcionales y No Funcionales**
> **Proyecto:** SOLV (Sistema de Orquestación de Laboratorios Virtuales)
> **Autor:** Alvaro Rivera Carhuani
> **Estado:** Corregido y Alineado con Arquitectura Hexagonal, Multi-Tenancy y Decisiones D1–D6

---

## 1. Requerimientos Funcionales (RF)

### Módulo 1: Gestión de Identidad, Usuarios y Multi-Tenancy

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-1.1** |
| **Nombre** | Autenticación Institucional Multi-Tenant |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Restringe el acceso a miembros autorizados de la comunidad universitaria.* |
| **Descripción** | El sistema debe autenticar a los usuarios mediante Google OAuth 2.0 / SSO, filtrando de forma dinámica los dominios de correo permitidos según la configuración de cada institución o facultad (Tenant) en la base de datos (`allowed_domains`). |
| **Criterios de Aceptación** | 1. Rechazar intentos de inicio de sesión de dominios de correo no autorizados en la configuración del Tenant activo.<br>2. Emitir la cookie segura HttpOnly `solv_session` integrada con el proxy Traefik v3 (ForwardAuth) tras la confirmación de Google. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-1.2** |
| **Nombre** | Importación Manual Unidireccional desde Google Classroom (D6) |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: Agiliza la inscripción masiva de estudiantes sin ingreso manual.* |
| **Descripción** | El sistema debe permitir al docente importar manualmente la lista de estudiantes inscritos en sus cursos de Google Classroom mediante peticiones `GET` de solo lectura a la API de Classroom, poblando automáticamente la tabla de inscripciones (`enrollments`). |
| **Criterios de Aceptación** | 1. Sincronizar de forma unidireccional la lista de correos al seleccionar un curso desde el panel del docente.<br>2. Preservar la soberanía de los datos en PostgreSQL sin requerir permisos de escritura sobre Google Classroom. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-1.3** |
| **Nombre** | Asignación Restrictiva de Roles Docentes |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Seguridad administrativa.* |
| **Descripción** | El rol de "Docente" debe asignarse únicamente mediante una lista blanca (whitelist) preconfigurada en la base de datos o por invitaciones de un solo uso mediante tokens transaccionales. |
| **Criterios de Aceptación** | 1. Un usuario no listado en la whitelist o sin invitación válida asume el rol de "Alumno" por defecto al iniciar sesión. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-1.4** |
| **Nombre** | Bloqueo de Escalamiento de Privilegios |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Integridad de la seguridad.* |
| **Descripción** | El sistema debe bloquear cualquier intento a nivel de API REST de modificar el rol o los permisos de un usuario con privilegios de alumno. |
| **Criterios de Aceptación** | 1. Retornar error HTTP 403 (Forbidden) si un token o cookie de alumno intenta acceder o mutar endpoints administrativos. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-1.5** |
| **Nombre** | Aislamiento Lógico Multi-Tenant por Discriminador (D2) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Garantiza la privacidad entre instituciones.* |
| **Descripción** | El sistema debe aislar lógicamente los datos de múltiples instituciones en una misma instancia de base de datos relacional mediante la columna discriminadora `tenant_id` y la validación obligatoria en el middleware `TenantMiddleware`. |
| **Criterios de Aceptación** | 1. Filtrar el 100% de las consultas SQL y peticiones HTTP por el `tenant_id` derivado del contexto autenticado. |

---

### Módulo 2: Orquestación y Entornos de Desarrollo

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-2.1** |
| **Nombre** | Interfaz Visual de Creación de Laboratorios |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Facilidad de uso para docentes sin conocimientos avanzados de Docker Engine.* |
| **Descripción** | El frontend debe permitir configurar plantillas de contenedores mediante selectores gráficos (lenguajes: Python, Java, C++, C#, Node.js; BD: PostgreSQL 18). |
| **Criterios de Aceptación** | 1. El backend debe generar y ejecutar la estructura del contenedor mediante la SDK de Docker sin requerir archivos YAML manuales por parte del docente. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-2.2** |
| **Nombre** | Personalización Dinámica de Contenedores |
| **Prioridad (MoSCoW)** | **Could (Podría tener)** - *Justificación: Flexibilidad para materias avanzadas.* |
| **Descripción** | La interfaz debe proveer un campo de texto para inyectar comandos de inicialización (ej. `npm install`) durante el arranque del contenedor. |
| **Criterios de Aceptación** | 1. El contenedor debe ejecutar los comandos inyectados antes de marcarse como "Listo" (`running`) en el orquestador. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-2.3** |
| **Nombre** | Entornos de Edición de Código Dual (Monaco Native & OpenVSCode Server) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Abstracción del hardware local del estudiante.* |
| **Descripción** | El sistema debe proveer dos modalidades de edición: a) Componente Monaco Editor nativo en Angular para la resolución rápida de ejercicios (Fase A); y b) Instancia de OpenVSCode Server incrustada en Iframe seguro para laboratorios complejos (Fase B). |
| **Criterios de Aceptación** | 1. Los cambios realizados en el editor web deben reflejarse de forma inmediata en el volumen persistente del servidor. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-2.4** |
| **Nombre** | Persistencia de Preferencias del Editor |
| **Prioridad (MoSCoW)** | **Could (Podría tener)** - *Justificación: Mejora de la experiencia de usuario (UX).* |
| **Descripción** | Las configuraciones del editor web (tema, tamaño de fuente, atajos) deben guardarse en el perfil del usuario en la base de datos relacional. |
| **Criterios de Aceptación** | 1. Al iniciar sesión desde un nuevo equipo, el editor debe cargar las configuraciones visuales guardadas. |

---

### Módulo 3: Evaluación Dual y Juez Virtual

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-3.1** |
| **Nombre** | Juez Virtual para Autocalificación Sandbox (I/O) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Automatización de la evaluación algorítmica.* |
| **Descripción** | El sistema debe ejecutar el código del alumno en contenedores efímeros aislados sin red (`network_mode: none`), inyectando casos de prueba (Standard Input) y comparando el resultado (Standard Output) con las respuestas esperadas. |
| **Criterios de Aceptación** | 1. Devolver un veredicto preciso (`AC`, `WA`, `TLE`, `RE`) por cada caso de prueba. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-3.2** |
| **Nombre** | Pre-chequeo Sintáctico Estático AST con Semgrep (D4) |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: Prevención de trampas o uso de funciones prohibidas antes de gastar recursos de Docker.* |
| **Descripción** | El sistema debe analizar el Árbol de Sintaxis Abstracta (AST) del código fuente usando el motor **Semgrep** en `< 100ms` para buscar reglas bloqueadas por lenguaje antes de permitir su ejecución. |
| **Criterios de Aceptación** | 1. Abortar la creación del Sandbox y retornar el veredicto `AST_BLOCKED` si se detecta una función o librería prohibida en el AST. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-3.3** |
| **Nombre** | Congelamiento de Entornos y Modo Solo Lectura |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: Integridad de las entregas tras finalizar exámenes.* |
| **Descripción** | El sistema debe pasar el entorno de evaluación del estudiante a modo solo lectura (Read-Only) al expirar el tiempo límite configurado. |
| **Criterios de Aceptación** | 1. Deshabilitar la edición de archivos en la interfaz y mostrar un banner informativo cuando expire el examen. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-3.4** |
| **Nombre** | Revisión Asistida para Docentes |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: Agiliza la calificación de proyectos visuales.* |
| **Descripción** | El docente debe disponer de un botón en su panel para acceder al entorno congelado de un estudiante en modo de solo lectura. |
| **Criterios de Aceptación** | 1. El docente puede navegar por el código e interfaz del alumno sin alterar el estado original de los archivos. |

---

### Módulo 4: Consolidación de Resultados

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-4.1** |
| **Nombre** | Dashboard de Métricas Docente |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Centralización de la información académica.* |
| **Descripción** | El sistema debe proveer una vista consolidada mostrando qué estudiantes aprobaron/reprobaron las pruebas automáticas en tiempo real. |
| **Criterios de Aceptación** | 1. La tabla debe permitir filtrarse por laboratorio, materia y estado de aprobación. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-4.2** |
| **Nombre** | Visualización Segura de Notas (D6) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Privacidad de los datos estudiantiles.* |
| **Descripción** | El sistema debe mostrar las calificaciones exclusivamente en la interfaz gráfica del docente (Solo Lectura), bloqueando la edición o alteración manual de los resultados calculados por el Juez. |
| **Criterios de Aceptación** | 1. Las calificaciones se muestran como datos de solo lectura respaldados por la BD relacional. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-4.3** |
| **Nombre** | Gestión Integrada de Copias de Seguridad (Backups) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Prevención de pérdida de datos ante fallos de hardware.* |
| **Descripción** | El panel de administración debe incluir una función para generar y restaurar volcados completos de la base de datos PostgreSQL y los volúmenes persistentes de los estudiantes. |
| **Criterios de Aceptación** | 1. Permitir descargar un archivo comprimido con el backup y restaurarlo desde la interfaz web o mediante script administrativo en el servidor. |

---

### Módulo 5: Lógica de Operación de Recursos (Orquestador)

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-5.1** |
| **Nombre** | Hibernación Inteligente de Contenedores |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: Optimización de memoria RAM en el servidor On-Premise.* |
| **Descripción** | El sistema debe ejecutar `docker pause` en contenedores que no registren interacción web durante un periodo de inactividad configurable (ej. 15 minutos). |
| **Criterios de Aceptación** | 1. Pausar el contenedor tras 15 minutos sin peticiones HTTP/WebSocket desde el cliente. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-5.2** |
| **Nombre** | Reanudación Instantánea de Laboratorios |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: UX fluida sin pérdida de tiempo de clase.* |
| **Descripción** | El sistema debe ejecutar `docker unpause` automáticamente en el momento en que el usuario inactivo recarga la página o interactúa con el editor. |
| **Criterios de Aceptación** | 1. El entorno debe volver a ser interactivo en menos de 2 segundos tras la reconexión. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-5.3** |
| **Nombre** | Motor de Asignación de Límites (Cgroups & OOM Guard) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Protección del servidor principal contra saturación.* |
| **Descripción** | El orquestador debe imponer límites duros de memoria RAM (256MB) y cuotas de CPU al instanciar cualquier contenedor de laboratorio. |
| **Criterios de Aceptación** | 1. Si un proceso estudiantil excede la RAM asignada, el sistema operativo liquidará ese contenedor específico (OOM Killed) sin afectar los demás servicios. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RF-5.4** |
| **Nombre** | Observabilidad en Vivo (Prometheus / Grafana) |
| **Prioridad (MoSCoW)** | **Could (Podría tener)** - *Justificación: Monitoreo técnico para administradores.* |
| **Descripción** | Integración de un panel de métricas (Prometheus/Grafana) para visualizar el consumo de RAM/CPU del servidor físico y el estado de los contenedores. |
| **Criterios de Aceptación** | 1. Mostrar gráficos en tiempo real del uso de recursos del servidor anfitrión. |

---

## 2. Requerimientos No Funcionales (RNF)

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-01** |
| **Nombre** | Aislamiento de Red e iptables (D4 / Zero Trust) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Seguridad crítica contra vulnerabilidades en red local.* |
| **Descripción** | El sistema debe bloquear la comunicación inter-contenedor mediante la cadena `DOCKER-USER` en `iptables` y binding explícito a `127.0.0.1`. |
| **Criterios de Aceptación** | 1. Todo intento de conexión de red entre el contenedor de un alumno y otro contenedor o el host debe ser rechazado (`DROP`). |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-02** |
| **Nombre** | Arquitectura On-Premise Autónoma |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Soberanía tecnológica y funcionamiento ante caídas de internet externo.* |
| **Descripción** | El sistema de orquestación, compilación y base de datos debe operar al 100% en la red LAN de la institución utilizando imágenes Docker cacheadas localmente. |
| **Criterios de Aceptación** | 1. Aprovisionar laboratorios nuevos incluso ante caídas del enlace WAN (asumiendo sesión SSO validada previamente). |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-03** |
| **Nombre** | Distribución Multi-Hipervisor / Docker Compose |
| **Prioridad (MoSCoW)** | **Should (Debería tener)** - *Justificación: Empaquetamiento profesional del software.* |
| **Descripción** | Todo el entorno de producción (Go, Docker, Traefik v3, PostgreSQL 18) debe estar empaquetado en un archivo orquestado `compose.yml` para despliegue automatizado en servidores Linux. |
| **Criterios de Aceptación** | 1. Iniciar el 100% de los servicios de la infraestructura mediante un solo comando (`docker compose up -d`). |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-04** |
| **Nombre** | Eficiencia de Huella del Backend (Go 1.26 Hexagonal) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Maximizar la memoria disponible para los laboratorios estudiantiles.* |
| **Descripción** | El binario compilado del orquestador backend desarrollado en Go 1.26 con `net/http.ServeMux` debe mantener una huella de memoria extremadamente baja. |
| **Criterios de Aceptación** | 1. El consumo de RAM del proceso backend en estado inactivo (idle) no debe superar los 50 MB. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-05** |
| **Nombre** | Gestión Dinámica de Tráfico (Traefik v3 Ingress) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Escalabilidad y acceso inmediato sin reinicio de servicios.* |
| **Descripción** | El proxy inverso Traefik v3 debe actualizar dinámicamente sus reglas de enrutamiento escuchando el socket de Docker para asignar subdominios únicos en tiempo real. |
| **Criterios de Aceptación** | 1. El tiempo de latencia entre la creación de un nuevo contenedor y la disponibilidad pública de su subdominio debe ser menor a 1 segundo. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-06** |
| **Nombre** | Concurrencia y Escalabilidad Vertical |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Viabilidad operativa en aulas reales.* |
| **Descripción** | El orquestador y el proxy inverso deben ser capaces de gestionar la ejecución simultánea de múltiples laboratorios sin degradación severa del servicio. |
| **Criterios de Aceptación** | 1. El sistema debe soportar al menos 40 contenedores de laboratorio activos simultáneamente en un nodo anfitrión de 16 GB RAM y 8 núcleos. |

| Campo | Detalle de la Especificación |
|---|---|
| **Identificador** | **RNF-07** |
| **Nombre** | Cifrado en Tránsito TLS/SSL Automático (Let's Encrypt / desec.io) |
| **Prioridad (MoSCoW)** | **Must (Debe tener)** - *Justificación: Protección de tokens de sesión HttpOnly y código fuente.* |
| **Descripción** | Todas las comunicaciones entre el cliente y el servidor (Traefik Ingress) deben estar cifradas obligatoriamente bajo el protocolo HTTPS. |
| **Criterios de Aceptación** | 1. Forzar redirección automática de HTTP (80) a HTTPS (443) y emisión automática de certificados SSL Wildcard mediante Let's Encrypt y DNS `desec.io`. |
