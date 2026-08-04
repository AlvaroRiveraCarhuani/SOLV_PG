**Estado:** Aprobado

### 1. Contexto y Origen de la Necesidad

El frontend del sistema SOLV no es una página web tradicional, sino una **Single Page Application (SPA)** de alta interactividad. Debe cumplir dos roles drásticamente distintos (según el ADR 005):

1. **Panel de Orquestación y Dashboard:** Gestionar el estado global de la sesión del usuario (Autenticación OAuth2 / JWT), la creación de materias, y la redirección a entornos de desarrollo pesados (contenedores con VS Code Server para la Fase B).
    
2. **Interfaz del Juez Virtual (Fase A):** Proveer un entorno de codificación incrustado, estilo programación competitiva (_Codeforces/LeetCode_), capaz de comunicarse en tiempo real mediante WebSockets con el backend (FastAPI) para enviar código y recibir resultados de evaluación (I/O) y diagnósticos del AST sin recargar la página.
    

Se requiere un _stack_ tecnológico en el cliente que evite fugas de memoria en conexiones persistentes y brinde una Experiencia de Desarrollador (DX) nativa sin sobrecargar el navegador.

### 2. Alternativas a Evaluar

**Batalla 1: El Framework del Cliente (SPA)**

- **Opción 1 (Aceptada): Angular (con RxJS).** Framework estructurado y fuertemente tipado.
    
- **Opción 2 (Descartada): React / Vue.js.** Frameworks populares y flexibles. _Motivo de descarte:_ Alta dependencia de librerías de terceros para enrutamiento y manejo de estado. La gestión manual de WebSockets puede derivar en fugas de memoria si los ciclos de vida de los componentes no se limpian exhaustivamente.
    
- **Opción 3 (Descartada): HTMX / Vanilla JS.** _Motivo de descarte:_ Incapacidad para manejar eficientemente el estado complejo y las conexiones bidireccionales persistentes requeridas por un IDE web.
    

**Batalla 2: El Motor del Editor de Código (Fase A)**

- **Opción 1 (Aceptada): Monaco Editor.** El núcleo de código abierto de Visual Studio Code.
    
- **Opción 2 (Descartada): CodeMirror 6.** Editor web ligero. _Motivo de descarte:_ Requiere configuración manual extensiva y múltiples _plugins_ de terceros para alcanzar un nivel de autocompletado e _IntelliSense_ aceptable para estudiantes universitarios.
    

### 3. Decisión

Se implementará el frontend utilizando **Angular** (mediante _Standalone Components_ para modularidad) como framework principal, integrando **Monaco Editor** para la interfaz del Juez Virtual.

La arquitectura visual seguirá el principio de **Carga Cognitiva Reducida (Enfoque Dual)**:

1. **Vista de Evaluación Estricta:** Un componente _Split-Pane_ (Panel Dividido) nativo en Angular. A la izquierda, la descripción en Markdown; a la derecha, la instancia de Monaco Editor. Todo el flujo de evaluación se maneja vía API REST a FastAPI.
    
2. **Vista de Proyecto:** El frontend delega la interfaz. Al solicitar un laboratorio complejo, Angular renderizará un `<iframe>` a pantalla completa (o redirección directa) hacia el subdominio generado por Traefik, exponiendo el contenedor Docker que ejecuta `code-server` (VS Code Web).
    

### 4. Justificación y Eficiencia (Primeros Principios)

- **Gestión de WebSockets (RxJS):** La integración nativa de RxJS en Angular permite tratar las conexiones asíncronas con los contenedores (estado de latencia, consumo de RAM, logs en vivo) como _Observables_. Esto garantiza que, al destruir un componente (ej. el alumno cierra la pestaña o cambia de vista), las suscripciones a los WebSockets se cancelen automáticamente, protegiendo el navegador del cliente contra fugas de memoria (Memory Leaks).
    
- **Consistencia de Experiencia (DX):** Al utilizar Monaco Editor en el Juez Virtual y VS Code Server en la Fase B, el estudiante interactúa con el mismo motor de renderizado de texto, atajos de teclado y resaltado de sintaxis en toda la plataforma SOLV. El cambio de contexto entre "hacer un examen" y "desarrollar un proyecto" es visualmente imperceptible.
    

### 5. Consecuencias y Trade-offs (Análisis de Costo)

**Costo**

- **Curva de Aprendizaje:** Angular y RxJS imponen una curva de aprendizaje inicial más pronunciada en comparación con React o Vue debido a su rigidez estructural y el uso intensivo de TypeScript y programación reactiva.
    
- **Peso del _Bundle_:** Monaco Editor es una librería pesada (~3MB a 5MB), lo que incrementará marginalmente el tiempo del _First Contentful Paint_ (FCP) en la primera visita del usuario.
    

**Mitigación**

- Se utilizará el sistema de _Lazy Loading_ (Carga Perezosa) del enrutador de Angular para diferir la descarga de los _scripts_ de Monaco Editor. El motor de código solo se descargará en la red del usuario cuando este ingrese explícitamente a una vista de "Examen / Juez Virtual", manteniendo el _Dashboard_ y el inicio de sesión ultraligeros y rápidos.