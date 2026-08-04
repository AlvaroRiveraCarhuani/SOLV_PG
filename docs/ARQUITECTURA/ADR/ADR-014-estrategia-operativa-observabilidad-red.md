**Estado:** Aprobado

---
## 1. Contexto y Origen de la Necesidad

El sistema SOLV ha definido una arquitectura sólida para la orquestación de contenedores, la evaluación algorítmica y la persistencia de datos en un entorno _On-Premise_. Sin embargo, al transicionar hacia una etapa de producción en la Universidad Adventista de Bolivia (UAB), surgen tres vulnerabilidades operativas críticas que deben ser mitigadas:

1. **Cajas Negras y Falta de Telemetría:** El orquestador escrito en Go carece de visibilidad interna en tiempo real. En escenarios de alta concurrencia (ej. 40 estudiantes activos), es imposible diagnosticar cuellos de botella en el disco (I/O Wait), fugas de Goroutines o latencia de creación de contenedores sin una capa de observabilidad.
    
2. **Movimiento Lateral y Riesgo de Red:** Al instanciar múltiples entornos de desarrollo en la Fase B (VS Code Server), los contenedores comparten subredes por defecto. Esto abre una brecha donde un estudiante podría intentar comunicarse con el contenedor de otro compañero o atacar los servicios de infraestructura interna.
    
3. **Exposición de Credenciales:** La gestión tradicional mediante archivos `.env` físicos en el servidor representa un riesgo inaceptable para los tokens de OAuth (Google SSO), contraseñas de PostgreSQL y claves JWT.
    

---

## 2. Decisiones Arquitectónicas

Se implementará una estrategia operativa integral basada en tres pilares de Ingeniería de Confiabilidad del Sitio (SRE) y seguridad _Zero Trust_:

### Pilar A: Observabilidad y Telemetría en Tiempo Real

Se adopta el ecosistema **Prometheus + Grafana**.

- **Prometheus (Backend):** Se encargará del _scraping_ periódico de métricas. El backend en Go (Gin) expondrá un endpoint protegido `/metrics` utilizando la librería `prometheus/client_golang` para emitir métricas de infraestructura (Goroutines, uso de Heap) y métricas de negocio (laboratorios activos, tiempos de creación).
    
- **Grafana (Frontend):** Servirá como el panel de control exclusivo para el Administrador del Sistema, visualizando la salud del servidor físico y las alertas de inactividad de los contenedores (identificación de "Zombies").
    

### Pilar B: Topología de Red Aislada (Zero Trust)

Se implementa una política estricta de aislamiento de red a nivel del demonio de Docker.

- **Desactivación de ICC (Inter-Container Communication):** El orquestador instruirá al SDK de Docker para aprovisionar las redes puente de los estudiantes con la directiva `com.docker.network.bridge.enable_icc=false`. Los entornos podrán acceder a internet para descargar dependencias, pero estarán completamente ciegos ante las direcciones IP de sus compañeros.
    
- **Air-Gap Lógico de Infraestructura:** La base de datos (PostgreSQL) y el binario de Go residirán en una red interna dedicada y aislada de los laboratorios. El proxy inverso Traefik actuará como el único puente autorizado para el tráfico web entrante.
    

### Pilar C: Inyección de Secretos Dinámica (Secret Management)

Se abandona el uso de archivos `.env` estáticos o variables _hardcodeadas_ en el código fuente.

- La configuración de la aplicación en Go dependerá de la inyección de secretos en tiempo de ejecución (Run-Time Injection), utilizando gestores de secretos modernos (como Doppler o herramientas nativas equivalentes compatibles con el despliegue en Fedora Linux).
    
- Esto garantiza que los tokens institucionales solo existan en la memoria volátil del proceso de Go.

**Pilar D: Disaster Recovery (Recuperación ante Desastres)**

Para mitigar el riesgo de pérdida total de datos por un fallo físico crítico en el hardware local (Single Point of Failure en el servidor On-Premise), se implementa una política de copias de seguridad automatizadas y externalizadas bajo un enfoque de "Cero Inversión de Capital" (Zero CAPEX):

- **Ejecución Desacoplada:** El proceso es gestionado directamente por el sistema operativo anfitrión mediante un `Cron Job` nocturno, manteniendo el orquestador principal en Go completamente libre de esta carga de procesamiento.
    
- **Extracción y Compresión:** Diariamente, un script ejecuta un `pg_dump` del contenedor de PostgreSQL y comprime los directorios físicos de los volúmenes de Docker, capturando tanto el estado de las evaluaciones como el código fuente intacto.
    
- **Externalización Eficiente:** Los paquetes comprimidos se sincronizan automáticamente con la infraestructura en la nube de Google Drive utilizando la herramienta `rclone`. Esto garantiza la supervivencia de la información crítica fuera de la red local, aprovechando los recursos institucionales existentes sin incurrir en costos de facturación de proveedores Cloud (SaaS/IaaS) y facilitando una eventual restauración mediante interfaces accesibles.
---

## 3. Justificación y Eficiencia (Primeros Principios)

- **Impacto Mínimo en Recursos (Zero CAPEX):** Prometheus y Grafana son herramientas de código abierto altamente optimizadas. El _scraping_ de métricas en texto plano desde Go toma microsegundos y la base de datos de series de tiempo (TSDB) comprime los datos de manera extrema, protegiendo la memoria RAM y el almacenamiento del servidor _On-Premise_.
    
- **Seguridad Inherente:** Desactivar la comunicación inter-contenedor no consume recursos adicionales de hardware; de hecho, reduce la carga sobre la tabla de ruteo del host y anula por completo la superficie de ataque lateral.
    
- **Auditoría y Resiliencia:** Visualizar las métricas permite demostrar empíricamente el comportamiento del algoritmo heurístico de asignación de recursos, justificando matemáticamente que el servidor no colapsará.
    

---

## 4. Consecuencias y Trade-offs (Análisis de Riesgo)

### Costos

- **Complejidad de Despliegue Secuencial:** El flujo de arranque del sistema SOLV se vuelve dependiente de pasos previos. El gestor de secretos y el sistema de telemetría deben estar operativos y saludables antes de que el orquestador principal en Go pueda inicializarse de forma segura.
    
- **Mantenimiento Operativo:** Requiere gestionar y actualizar contenedores adicionales en la infraestructura base del servidor.
    

### Beneficios

- El sistema evoluciona de ser una simple aplicación web a una plataforma SRE auditable, proactiva e inatacable desde su red interna.
    
- La mitigación de riesgos operativos garantiza la estabilidad de la plataforma durante los exámenes críticos, logrando el aislamiento pedagógico requerido por el Juez Virtual y los entornos interactivos.