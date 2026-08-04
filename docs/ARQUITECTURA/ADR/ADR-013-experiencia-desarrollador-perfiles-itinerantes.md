**Estado:** Aprobado

### 1. Contexto y Origen de la Necesidad

Para los laboratorios complejos (Fase B), SOLV proporciona entornos basados en `code-server` (Visual Studio Code Web). La Experiencia del Desarrollador (DX) exige que los estudiantes puedan personalizar su entorno (temas, atajos de teclado, extensiones). Sin embargo, almacenar gigabytes de binarios de extensiones por estudiante en Volúmenes Globales de Docker dentro del servidor _On-Premise_ provocaría el agotamiento rápido del almacenamiento (Terabytes de datos irrecuperables). Además, compartir un mismo volumen de perfil entre múltiples contenedores concurrentes genera un riesgo crítico de corrupción en las bases de datos internas (SQLite) del IDE y conflictos de dependencias ("Dependency Hell") entre materias de distintos semestres.

### 2. Alternativas a Evaluar

**Alternativa Aceptada**

- **Delegación Cloud (Settings Sync) + Onboarding UI.** Aislar estrictamente los volúmenes por materia y delegar la sincronización del perfil del usuario a la infraestructura global de Microsoft/GitHub mediante la funcionalidad nativa _Settings Sync_.
    

**Alternativas Descartadas**

- **Perfil Itinerante Físico (Volumen Global `/home/coder`).**
    
    - _Motivo de descarte:_ Costo de infraestructura prohibitivo. Riesgo inminente de corrupción de estado si el alumno abre dos laboratorios simultáneamente (condición de carrera en escritura de base de datos del IDE). Arrastre de extensiones obsoletas a lo largo de los años académicos.
        
- **Entornos Estrictamente Pre-Cocinados (Sin personalización).**
    
    - _Motivo de descarte:_ Destruye la Experiencia del Desarrollador (DX). Obligar a un estudiante avanzado a programar sin sus herramientas o atajos preferidos genera rechazo hacia la plataforma institucional.
        

### 3. Decisión

Se implementará una arquitectura de almacenamiento aislado combinada con una estrategia de adopción de herramientas de la industria:

1. **Aislamiento Físico (Cero Basura Global):** Cada contenedor levantará un único Volumen Persistente (`/workspace`) dedicado exclusivamente al código fuente de **esa materia específica**. La carpeta del sistema (`/home/coder`) nacerá efímera en cada laboratorio.
    
2. **Delegación de Estado (Settings Sync):** La persistencia de la configuración visual, atajos y lista de extensiones instaladas no vivirá en la UAB, sino en la nube de Microsoft/GitHub. El estudiante utilizará el motor nativo de VS Code para sincronizar su perfil.
    
3. **Mitigación de Fricción (Onboarding UI):** Para evitar la frustración de la configuración manual reiterativa, el frontend de SOLV incluirá un flujo de _Onboarding_ (Tutorial interactivo) en el primer inicio de sesión. Este flujo guiará al estudiante para activar _Settings Sync_ con su cuenta de GitHub.
    
4. **Inyección de Credenciales de Git:** Para agilizar el trabajo diario, las credenciales básicas de control de versiones (`user.name`, `user.email`) se inyectarán como variables de entorno seguras al levantar el contenedor, permitiendo realizar _commits_ inmediatamente sin configuración manual.
    

### 4. Consecuencias y Trade-offs

**Positivas**

- **Ahorro Masivo de Infraestructura:** El servidor _On-Premise_ de la universidad ahorra cientos de Gigabytes (o Terabytes) al no almacenar binarios de extensiones de terceros.
    
- **Cero Riesgo de Corrupción:** Al ser volúmenes aislados por materia, la concurrencia de pestañas no rompe el IDE. No hay conflicto de dependencias entre un laboratorio de 1er semestre y uno de 5to.
    
- **Impacto Pedagógico (Adopción Temprana):** Se fuerza sutilmente la adopción y familiarización con GitHub desde las primeras semanas de la carrera, alineando al estudiante con los estándares de la industria del software.
    

**Negativas**

- **Dependencia Externa (Vendor Lock-In Parcial):** La comodidad del perfil itinerante depende de la disponibilidad de los servidores de GitHub/Microsoft. Si el estudiante carece de cuenta en GitHub, enfrentará fricción inicial (mitigada por el tutorial UI).