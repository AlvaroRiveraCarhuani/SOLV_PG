### 1. Contexto y Origen de la Necesidad

El sistema SOLV opera en una arquitectura _On-Premise_ con recursos de hardware estrictamente finitos. Según lo establecido en el ADR 001, los proyectos de los estudiantes deben persistir a lo largo del semestre. Sin embargo, permitir un acceso ininterrumpido o automatizado provocaría el colapso del servidor (agotamiento de RAM/CPU por picos de usuarios fuera de horario) o el llenado crítico del disco duro a largo plazo. Se requiere una estrategia que concilie la necesidad humana del estudiante (repasar código, hacer tareas en horarios atípicos) con la supervivencia de la infraestructura física, eliminando variables estáticas (como horarios _hardcodeados_ o límites mágicos de contenedores) que no reflejan la realidad universitaria.

### 2. Alternativas a Evaluar

**Alternativa Aceptada**

- **Opción Dinámica Integral:** Un modelo basado en Hibernación de volúmenes, asignación de recursos guiada por Interfaz de Usuario (Carga Perezosa), priorización QoS mediante Máquina de Estados manual (Docente) y cálculo matemático de RAM en tiempo real para control de concurrencia.
    

**Alternativas Descartadas**

- **Opción 1: Horarios de Acceso Estrictos.** Bloquear el sistema fuera del horario de clases.
    
    - _Motivo de descarte:_ Falla en considerar el factor humano (estudiantes que trabajan, ventanas libres o repasos nocturnos), generando una pésima experiencia de usuario y rechazo al sistema.
        
- **Opción 2: Límite Global Estático de Contenedores.** Definir en código un máximo de, por ejemplo, "100 contenedores simultáneos".
    
    - _Motivo de descarte:_ Inexactitud técnica. 100 contenedores de un lenguaje compilado (C++) no consumen lo mismo que 100 contenedores de un framework pesado (Spring Boot). Un límite estático puede provocar un colapso del servidor mucho antes de alcanzar el tope.
        
- **Opción 3: Limpieza mediante _Cron Jobs_ del Sistema Operativo.**
    
    - _Motivo de descarte:_ Riesgo crítico de pérdida de datos en caliente si el _Cron Job_ se ejecuta mientras un estudiante trabaja en un horario atípico.
        

### 3. Decisión

Se implementará un orquestador inteligente en FastAPI que gobernará el ciclo de vida de los entornos a través de 5 pilares arquitectónicos:

1. **Aterrizaje Forzado en Solo Lectura (_Lazy Loading_ / UX):**
    
    - Al acceder al sistema fuera de una clase activa, el estudiante visualizará su proyecto en modo "Solo Lectura", extraído directamente de la base de datos (consumo de RAM del entorno: 0%).
        
    - El contenedor de desarrollo solo se instanciará si el usuario realiza una acción explícita e intencional (`[▶️ Despertar Entorno]`), asumiendo un tiempo de carga (_Cold Start_). Esto filtra los "repasos rápidos" y protege la CPU.
        
2. **Calidad de Servicio (QoS) y Máquina de Estados:**
    
    - El docente dispondrá de un control manual (`Iniciar Clase en Vivo`) que alterará el estado de la materia en PostgreSQL (`is_live = True`).
        
    - Los estudiantes inscritos en una materia con estado `is_live` recibirán **Prioridad VIP** en el enrutador (Traefik/FastAPI), garantizando la reserva de RAM durante el periodo de enseñanza.
        
    - Para mitigar errores humanos, este estado caducará automáticamente tras un _Timeout_ de seguridad (ej. 3 horas).
        
3. **Control de Admisión por Hardware (Cálculo Dinámico):**
    
    - Las peticiones sin prioridad VIP pasarán por un cálculo de disponibilidad de recursos en tiempo real (mediante `psutil`), evaluando la memoria física del servidor antes de ejecutar la API de Docker:
    
   $$
\text{Capacidad}_{\text{actual}} =
\left\lfloor
\frac{
\text{RAM}_{\text{disponible}} - \text{RAM}_{\text{reserva\_seguridad}}
}{
\text{RAM}_{\text{estimada\_del\_perfil}}
}
\right\rfloor
$$
    
    - Si la capacidad es ≤0, la petición se encola en una "Sala de Espera" visual, evitando el error catastrófico de falta de memoria en el _Host_ (`OOM`).
        
4. **Hibernación por Inactividad (_Idle Timeout_):**
    
    - El backend monitoreará el tráfico de WebSockets de la terminal y el editor. Tras 15 minutos de inactividad absoluta, el contenedor será destruido (`docker rm -f`) para liberar recursos, manteniendo el Volumen de Docker en estado latente (Hibernación).
        
5. **Limpieza Semestral (_Garbage Collection_ Profundo):**
    
    - Al finalizar el ciclo académico, cuando el administrador cambie la materia al estado `ARCHIVADA`, el sistema extraerá y comprimirá los archivos de los volúmenes, los almacenará como historial en la base de datos (o S3), y ejecutará la purga masiva de los Volúmenes físicos de Docker, recuperando el almacenamiento masivo.
        

### 4. Consecuencias

**Positivas**

- **Resiliencia Basada en la Realidad:** El servidor jamás colapsará por falta de memoria, ya que las decisiones de instanciación se basan en matemáticas de hardware en tiempo real, no en estimaciones.
    
- **Empatía con el Usuario:** Los estudiantes pueden acceder al sistema 24/7 para estudiar o programar sin fricción, adaptándose a cualquier realidad socioeconómica o laboral.
    
- **Eficiencia de Costos:** Se maximiza el uso de la infraestructura _On-Premise_ existente, simulando elasticidad _Cloud_ a través de la destrucción y recreación eficiente de contenedores.
    

**Negativas**

- **Alta Complejidad de Implementación:** Exige un desarrollo avanzado en FastAPI para coordinar métricas del sistema operativo, WebSockets para detección de inactividad, transacciones de base de datos y la API nativa de Docker.