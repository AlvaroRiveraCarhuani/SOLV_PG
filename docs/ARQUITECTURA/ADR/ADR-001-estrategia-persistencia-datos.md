
## Estado
Aceptado

## 1. Contexto y Origen de la Necesidad

Las asignaturas de programación y laboratorios en la universidad tienen un hilo de continuidad; los proyectos se desarrollan a lo largo de semanas o semestres.

Utilizar contenedores 100% efímeros (que destruyan la información al cerrarse) rompería el flujo académico y degradaría severamente la experiencia del usuario final (estudiante y docente).

Por lo tanto, el sistema necesita un mecanismo que permita mantener la persistencia del código fuente de los estudiantes incluso cuando los contenedores sean detenidos, eliminados o reiniciados.

---

## 2. Decisión Arquitectónica

Se implementará el uso de **Volúmenes Persistentes (Docker Volumes)** montados dinámicamente en los contenedores efímeros de cada estudiante.

Cada contenedor tendrá asociado un volumen independiente que almacenará el código fuente del usuario.

---

## 3. Justificación y Eficiencia (Primeros Principios)

A nivel de sistema operativo, esta solución delega la escritura del código directamente al sistema de archivos del servidor host (Ubuntu Server), evitando intermediarios innecesarios o procesos de sincronización pesados.

Al utilizar **Docker Volumes** en lugar de **Bind Mounts**, se delega la gestión de permisos al demonio de Docker, lo cual proporciona:

- aislamiento entre estudiantes
- control de acceso más seguro
- menor riesgo de manipulación directa del sistema de archivos del host

Esto mejora la seguridad y simplifica la administración del entorno.

---

## 4. Consecuencias y Trade-offs (Análisis de Costo)

### Costo

Se sacrifica espacio de almacenamiento a largo plazo en el servidor central, ya que los volúmenes persistirán incluso cuando los contenedores sean eliminados.

### Mitigación

Se deberá implementar:

- un script de **recolección de basura (Garbage Collection)**
- o políticas automáticas de **retención de datos**

Esto permitirá eliminar volúmenes asociados a estudiantes inactivos o cursos finalizados al terminar cada ciclo académico.

---

## 5. Escenarios de Falla (Resiliencia)

Si ocurre alguno de los siguientes eventos:

- el contenedor entra en un bucle infinito (uso de RAM al 100%)
- el contenedor se detiene inesperadamente
- el servidor físico sufre un reinicio abrupto

El contenedor puede ser destruido o recreado, pero **la integridad del código fuente del estudiante permanecerá intacta**, ya que los datos están almacenados en el volumen persistente del servidor.