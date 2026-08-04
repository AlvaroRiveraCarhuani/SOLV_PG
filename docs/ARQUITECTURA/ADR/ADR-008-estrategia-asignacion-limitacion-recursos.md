**Estado:** Aprobado

**1. Contexto y Origen de la Necesidad** Existe un riesgo crítico de agotamiento de recursos (código defectuoso, RAM/CPU) que puede colapsar el sistema SOLV On-Premise. Se requiere un mecanismo de cgroups sin delegar esta tarea al docente.

**2. Decisión Arquitectónica** Se optará por una estrategia de **Auto-Perfilado Dinámico + Auto-Escalado Heurístico Reactivo**, gestionada íntegramente por Go interactuando nativamente con el SDK de Docker.

1. **Fase de Descubrimiento (Dry Run P95):** Al registrar una nueva imagen, el backend levantará contenedores ocultos concurrentemente (3 a 5 muestras) usando un Worker Pool, ejecutará una carga sintética mínima y leerá el pico real de uso. El percentil 95 (P95) se guardará como el límite oficial (JSONB) en la base de datos.
    
2. **Escalera de Contención y Castigo Pedagógico:** Se implementará un umbral suave (`MemoryReservation` al 80%) y un techo duro (`MemoryLimitMB`). Si se agota la RAM por código roto, ocurre un `OOMKilled` para proteger al servidor.
    
3. **Auto-Escalado Transparente (Doble Filtro):** Escalará automáticamente solo si supera el 30% de la clase colapsada (con un piso mínimo de 3 estudiantes) y si el tiempo de vida del contenedor demuestra que no es un bucle infinito instantáneo.
    
4. **Worker Pool de Actualización:** El aumento de RAM se ejecutará vía `docker update`, encolado en un **Worker Pool mediante Goroutines y Canales**, verificando que el servidor físico mantenga un 10% de RAM libre de seguridad.
    

**3. Consecuencias**

- **Positivas:** Densidad máxima de contenedores basada en empirismo estadístico, resiliencia total del servidor y cero fricción administrativa para el docente.
    
- **Negativas:** Requiere programación avanzada de concurrencia en Go y cálculos continuos de cgroups.