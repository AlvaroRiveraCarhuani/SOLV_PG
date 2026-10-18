# ADR-045: Estrategia Dual de Persistencia y Perfil Académico de Base de Datos

## Estado
Aprobado

## Fecha
2026-10-18

## Contexto
En entornos educativos de desarrollo en la nube (Cloud Development Environments - CDEs) desplegados sobre infraestructura de nodo único (servidor con 12 GB de RAM física para un padrón de 40 alumnos y picos de 10 a 15 sesiones concurrentes), el aprovisionamiento de bases de datos relacionales sin optimizar impone riesgos críticos de agotamiento de memoria física y caídas generalizadas por Out-Of-Memory (OOM Killer).

Previamente a esta decisión:
1. Las bases de datos se trataban de manera homogénea sin distinguir entre la carga pedagógica de asignaturas iniciales (SQL básico, modelado relacional) y asignaturas avanzadas (APIs fullstack, microservicios, transaccionalidad concurrente).
2. La activación de motores cliente-servidor (PostgreSQL) en el catálogo de servicios satélite no contemplaba un perfil de memoria sintonizado para aula ni límites de seguridad contra consultas no optimizadas (ej. productos cartesianos accidentales o bucles sin condición de corte).
3. Los administradores no contaban con visibilidad sobre la relación entre el motor de base de datos seleccionado, el piso de memoria requerido y la capacidad concurrente admisible en el host.

## Decisión

Adoptar una arquitectura dual estratificada por nivel curricular y un perfil de sintonización de bajo consumo para PostgreSQL:

1. **Segmentación Curricular (Dualidad Pedagógica)**:
   - **Nivel Inicial (Fundamentos de BD, Álgebra Relacional, SQL Básico)**: Motor recomendado **SQLite3**. Opera directamente sobre el sistema de archivos del espacio de trabajo persistente del estudiante (`/home/workspace`).
     - *Sobrecarga de memoria residente (RSS)*: 0 MB en reposo.
     - *Infraestructura*: Sin procesos demonio, sin puertos TCP/UNIX adicionales, gobernado por recolector de procesos `tini`.
     - *Evaluación determinista*: Permite inspección y calificación por jueces virtuales montando el archivo `.db` en modo de solo lectura (`file:test.db?mode=ro`).
   - **Nivel Avanzado (Desarrollo Fullstack, Sistemas Distribuidos, APIs)**: Motor **PostgreSQL para Aula (Optimizado)**.

2. **Perfil de Sintonización para PostgreSQL Académico (`postgresql.conf` / Flags de Arranque)**:
   - `shared_buffers = 16MB`: Acotado para tablas de ejercicios académicos (cientos o miles de registros).
   - `max_connections = 10`: Límite suficiente para cubrir CLI interactivo (`psql`), extensión visual del editor y pool de conexión de la aplicación del alumno, minimizando la creación de subprocesos `fork()` en el kernel.
   - `work_mem = 1MB`: Previene el consumo excesivo de memoria física ante planes de consulta ineficientes; las operaciones excedentes se derivan a almacenamiento temporal secundario.
   - `statement_timeout = 5000`: Límite estricto de 5 segundos para cancelar consultas no optimizadas o bucles infinitos, protegiendo los núcleos de CPU del servidor.
   - `temp_file_limit = 20MB`: Previene la saturación del disco por archivos de ordenamiento temporal descontrolados.
   - `idle_in_transaction_session_timeout = 10000`: Cierre forzado tras 10 segundos de transacciones inactivas para liberar bloqueos de tablas.
   - `synchronous_commit = off` con `fsync = on`: Protege la integridad física de los datos ante cortes de energía mientras minimiza la sobrecarga de operaciones I/O en unidades de estado sólido compartidas.
   - `huge_pages = off` y `max_parallel_workers_per_gather = 0`: Procesamiento secuencial que evita el secuestro de bloques de memoria en el kernel anfitrión.

3. **Gobernanza de Recursos en el Asistente del Administrador**:
   - En el **Paso 4 (Comandos y Servicios)**, exponer SQLite3 como opción nativa de cero consumo de RAM recomendada para materias iniciales, junto con la opción de PostgreSQL optimizado para talleres avanzados.
   - En el **Paso 5 (Memoria y Recursos)**, reflejar con precisión el piso de memoria requerido:
     - Entorno básico (SQLite / Algoritmos): Base orientativa de 384 MB a 512 MB.
     - Entorno con PostgreSQL optimizado: Piso mínimo recomendado de 550 MB a 600 MB.
   - Exponer en la métrica viva del host la capacidad estimada de alumnos simultáneos en función del hardware disponible.

4. **Hoja de Ruta hacia Servidor Institucional (32 a 64 GB)**:
   - Al disponer de mayor capacidad de cómputo, se mantendrá SQLite3 para la etapa inicial por su valor didáctico, y se migrará PostgreSQL hacia el patrón sidecar desacoplado (Modelo B) con relajación de memoria (`shared_buffers = 128MB`, `work_mem = 16MB`).

## Consecuencias

### Positivas
- Reducción del consumo de memoria residente de PostgreSQL a un rango de 40 MB a 60 MB en el servidor.
- Eliminación de la degradación por vecinos ruidosos (*noisy neighbor*) mediante límites estrictos de tiempo de consulta y archivos temporales.
- Claridad pedagógica y operativa para el docente y administrador: no se penalizan los recursos en materias donde un motor cliente-servidor es innecesario.
- Capacidad demostrada para albergar entre 15 y 20 sesiones simultáneas en el servidor piloto de 12 GB de RAM.

### Negativas
- Proyectos avanzados que requieran consultas masivas o tareas analíticas intensivas pueden alcanzar el `statement_timeout` de 5 segundos o volcar a disco por el `work_mem` de 1 MB, requiriendo templates dedicados con mayor cuota si la materia lo amerita.
