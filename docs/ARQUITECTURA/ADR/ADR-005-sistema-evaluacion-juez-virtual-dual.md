## 1. Contexto y Origen de la Necesidad
  
SOLV debe proporcionar mecanismos para evaluar el trabajo de los estudiantes. Sin embargo, la naturaleza de las tareas de 
programación varía drásticamente: desde algoritmos matemáticos estrictos (Estructuras de Datos) hasta arquitecturas de 
software abiertas o interfaces gráficas (Programación Orientada a Objetos, Desarrollo Web).
  
Intentar automatizar la evaluación de proyectos complejos genera alta fricción para el docente al requerir la redacción de 
pruebas unitarias exhaustivas, o resulta en evaluaciones inexactas. Además, el método tradicional de revisión manual 
(descarga de archivos `.zip` o clonación de repositorios) consume un tiempo excesivo en configuración de entornos locales.
  
---
  
## 2. Decisión Arquitectónica
  
Se implementará una **Estrategia Dual de Evaluación**:
  
### Fase A (Autocalificación Algorítmica)
  
Para laboratorios lógicos, se utilizará un **Juez Virtual Híbrido** que combine:
  
- **Análisis Estático (AST)** para validar la estructura del código.  - un motor de **Ejecución Aislada (I/O)** para 
verificar resultados de salida.
  
### Fase B (Revisión Asistida y Congelamiento)
  
Para proyectos complejos, el sistema no autocalificará. En su lugar, implementará una **máquina de estados para los 
Volúmenes de Docker**.
  
Al finalizar el examen (por acción del docente o entrega voluntaria), el volumen del estudiante pasará a estado de **solo 
lectura (`:ro`)**.
  
El docente podrá levantar contenedores efímeros conectados a estos volúmenes congelados para revisar el código en ejecución 
directamente desde su navegador web en cuestión de segundos.
  
---
  
## 3. Justificación y Eficiencia (Primeros Principios)
  
### Realismo del MVP
  
Limitar el Juez Virtual automático a algoritmos reduce drásticamente la complejidad de desarrollo, garantizando un producto 
funcional para la primera versión (**MVP**).
  
### Eliminación de Fricción (Cero Setup)
  
El **"Review Mode"** elimina la necesidad de que el docente descargue archivos o instale dependencias en su máquina local. 
Todo el entorno del estudiante se reproduce instantáneamente en el servidor privado.
  
### Manejo del Factor Humano
  
El control del congelamiento de entornos se delega a **eventos manuales desde el panel del docente**, permitiendo 
flexibilidad ante retrasos en la clase o excepciones individuales, evitando la rigidez de los temporizadores estrictos.
  
---
  
## 4. Consecuencias y Trade-offs
  
- El backend (**FastAPI**) deberá gestionar rigurosamente los permisos de montaje de los volúmenes de Docker (`:rw` vs 
`:ro`) basándose en el estado de la entrega en la base de datos.
  
- Se requiere diseñar una **interfaz gráfica (UI)** clara para que el docente pueda gestionar el ciclo de vida del examen: - 
iniciar examen - congelar a toda la clase - otorgar excepciones individuales
