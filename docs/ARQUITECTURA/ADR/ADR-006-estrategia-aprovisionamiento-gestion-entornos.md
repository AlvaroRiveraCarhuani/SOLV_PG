## 1. Contexto y Origen de la Necesidad  
  
SOLV requiere un mecanismo para que los docentes creen laboratorios de programación. El desafío radica en diseñar una interfaz que sea extremadamente fácil de usar para lenguajes comunes (cero fricción), pero que a la vez no limite la enseñanza de tecnologías emergentes, frameworks modernos, o requiera mantenimiento constante por parte del departamento de TI de la universidad.  
  
Además, depender exclusivamente de APIs externas (como la búsqueda en vivo de Docker Hub) representa un riesgo crítico de inactividad por bloqueos de IP (Rate Limits).  
  
---  
  
## 2. Decisión Arquitectónica  
  
Se implementará un modelo de aprovisionamiento híbrido de tres capas (Infraestructura Visual vs. Software Dinámico):  
  
### Capas de Infraestructura (Catálogo Local y Generador Dinámico)  
  
Para lenguajes y bases de datos estándar, se utilizará una interfaz visual basada en clics. Los metadatos de las imágenes oficiales se almacenarán en una base de datos local (caché), eliminando la dependencia de búsquedas en APIs externas.  
  
FastAPI orquestará estas piezas generando archivos `docker-compose` en memoria al vuelo.  
  
### Capa de Software (Scripts de Arranque / DevContainers)  
  
Los frameworks (ej. React, Laravel) no se pre-instalarán en las imágenes. El docente definirá la receta de instalación mediante un **"Script de Arranque"** en texto plano, el cual SOLV inyectará y ejecutará en el contenedor en tiempo de ejecución.  
  
### Escotilla de Escape (Tirón a Ciegas / Blind Pull)  
  
Para lenguajes inexistentes en el catálogo o imágenes privadas de la universidad, el docente ingresará el nombre exacto de la imagen en un input manual.  
  
El sistema ejecutará una descarga directa (`docker pull`), esquivando buscadores externos y eliminando la necesidad de aprobación de un Administrador.  
  
---  
  
## 3. Justificación y Eficiencia (Primeros Principios)  
  
### Mantenimiento Cero (A prueba de futuro)  
  
Al separar la infraestructura base de los scripts de instalación de frameworks, SOLV nunca quedará obsoleto. Las actualizaciones de frameworks recaen en el comando que escribe el docente, no en el código fuente de la plataforma.  
  
### Alta Resiliencia  
  
El uso del catálogo local y el **Blind Pull** protegen a la universidad de caídas de terceros o bloqueos por exceso de peticiones a Docker Hub.  
  
### Ahorro de Almacenamiento  
  
Fomentar imágenes base limpias + scripts de arranque evita saturar el servidor con imágenes pesadas que contienen dependencias innecesarias preinstaladas.  
  
---  
  
## 4. Consecuencias y Trade-offs  
  
### Tiempos de carga prolongados en el primer inicio  
  
Si un docente define un script de arranque que descarga múltiples dependencias pesadas de internet (ej. un `npm install` masivo), el estudiante experimentará un tiempo de espera mayor (1–2 minutos) al iniciar el laboratorio.  
  
Se asume como un costo aceptable frente a la flexibilidad obtenida.  
  
### Sensibilidad a errores tipográficos  
  
En el modo de **"Escotilla de Escape"**, si el docente escribe incorrectamente el nombre de la imagen, la orquestación fallará.  
  
Se asume que este flujo avanzado requiere precisión técnica por parte del usuario.