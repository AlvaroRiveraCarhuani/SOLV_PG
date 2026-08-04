**Estado:** Aprobado

**1. Contexto y Origen de la Necesidad** El sistema SOLV debe resolver el problema de que los estudiantes se queden rezagados copiando código, equilibrando esta flexibilidad con el aislamiento de los contenedores.

**2. Alternativas a Evaluar**

- **Alternativa Aceptada (Sincronización de Volúmenes):** Que el backend en Go copie y sobrescriba los archivos a nivel de sistema de archivos (Host) desde el volumen del docente al alumno mediante instantáneas.
    
- **Descartadas:** Git Inyectado (conflictos de fusión) y Eventos por WebSockets (complejidad arquitectónica extrema para el MVP).
    

**3. Decisión** Se optará por la **Sincronización de Volúmenes mediante Instantáneas**, orquestada íntegramente por el backend en Go.

- **Publicación:** Se crea un Snapshot temporal del volumen del docente.
    
- **Suscripción:** El contenedor del estudiante se pausa y el backend sobrescribe su volumen con el Snapshot.
    
- **Optimización de I/O:** Para evitar cuellos de botella, el copiado se encolará utilizando un patrón de **Worker Pool con Goroutines y Canales** acotado por un semáforo. Se incluirá una función opcional de backup local (`.solv_backups/`).
    

**4. Consecuencias**

- **Positivas:** Alto impacto pedagógico, determinismo técnico y eficiencia de recursos en el almacenamiento On-Premise.
    
- **Negativas:** Incrementa la responsabilidad del backend en Go, exigiendo control riguroso de concurrencia al gestionar los estados de Docker.