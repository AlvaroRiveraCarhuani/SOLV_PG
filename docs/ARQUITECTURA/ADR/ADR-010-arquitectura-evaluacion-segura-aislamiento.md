**Estado:** Aprobado

**1. Contexto y Origen de la Necesidad** Para la Autocalificación Algorítmica, SOLV debe ejecutar código no confiable sin arriesgar la seguridad del servidor universitario (bombas lógicas, APIs externas) y sin causar fricción técnica al docente al crear los laboratorios.

**2. Decisión Arquitectónica** Se implementará el patrón de **Contenedores Efímeros (Disposable Runners)** combinado con un Juez Híbrido (AST + I/O), orquestado por el backend en **Go**.

- **UI Adaptativa:** Interfaz sencilla de Entradas/Salidas esperadas y Switches de restricción pedagógica.
    
- **Filtro Anti-Trampas (AST):** El sistema en Go evaluará el código a través de un Árbol de Sintaxis Abstracta para bloquear librerías de red/sistema y forzar el cumplimiento de los Switches pedagógicos antes de asignar recursos de Docker.
    
- **Aislamiento Extremo:** Go instanciará un contenedor desechable con la directiva `--network none` (Red Cero) y volúmenes de código en modo solo lectura (`:ro`).
    
- **Evaluación Agnóstica:** Go inyectará la entrada por `stdin`, capturará el `stdout` y comparará resultados en texto plano, sin importar el lenguaje evaluado.
    
- **Protección de Infraestructura:** Las solicitudes de evaluación masiva se encolarán utilizando un patrón de **Worker Pool con Goroutines y Canales**, limitando la concurrencia activa de Runners para no saturar la CPU. Se aplicará Rate Limiting tras intentos fallidos.
    

**3. Consecuencias**

- **Positivas:** Seguridad Zero-Trust, escalabilidad multi-lenguaje inminente y cero necesidad de redactar tests unitarios complejos por parte del docente.
    
- **Negativas:** Exige un manejo milimétrico del ciclo de vida y limpieza de contenedores efímeros desde Go, incluso ante Timeouts por bucles infinitos.