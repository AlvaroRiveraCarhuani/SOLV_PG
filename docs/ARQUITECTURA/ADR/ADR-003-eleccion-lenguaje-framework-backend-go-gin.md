# ADR 003: Elección del Lenguaje, Framework Backend y Gestión de Recursos (Go y `net/http`)

**Estado:** Aceptado

> [!NOTE] Evolución del Enrutamiento Backend
> Durante la planificación inicial del Slice 1 se contempló el uso del framework **Gin**. En los Slices 4 al 7 se adoptó formalmente la biblioteca estándar de **Go (Golang)** mediante `net/http.ServeMux` (nativo en Go 1.26), logrando cero dependencias externas de enrutamiento y una huella de memoria en reposo inferior a 15 MB.

---

## 1. Contexto y Origen de la Necesidad
El sistema SOLV actúa como un orquestador central que debe recibir peticiones web simultáneas de múltiples estudiantes y traducirlas en comandos de infraestructura de bajo nivel hacia el Demonio de Docker. Al operar bajo una arquitectura On-Premise con recursos de hardware estrictamente finitos, se requiere un núcleo de backend que garantice concurrencia masiva (I/O Bound) y validación estricta de datos, pero con un consumo mínimo de memoria RAM.

---

## 2. Decisión Arquitectónica
Se utilizará **Go (Golang) 1.26** como lenguaje principal del orquestador, implementando el enrutador HTTP nativo (`http.ServeMux`) y la librería estándar `encoding/json` para la validación de datos, eliminando la dependencia de frameworks externos (como Gin). Se utilizará el SDK oficial `docker/client` para Go para la comunicación directa a través del socket de Docker.

---

## 3. Justificación y Eficiencia (Primeros Principios)

- **Concurrencia Nativa (Goroutines):** Go maneja la concurrencia a través de Goroutines ligeras que consumen aproximadamente 2 KB de memoria al arrancar, permitiendo procesar cientos de peticiones de manera paralela sin saturar el servidor.
- **Enrutamiento Nativo Eficiente:** Las capacidades integradas en Go (`ServeMux` con comodines y verificación de métodos HTTP) reducen la necesidad de librerías externas, manteniendo el núcleo ligero.
- **Eficiencia Extrema de Memoria:** El orquestador compilado en Go consume entre 10 MB y 20 MB en reposo, reservando recursos para los laboratorios de los estudiantes.
- **Validación Estricta:** El tipado estricto y el model binding nativo mediante `encoding/json` protegen al motor de Docker de caídas causadas por inyecciones malformadas.

---

## 4. Consecuencias y Trade-offs

- **Costo:** La sintaxis centrada en el manejo explícito de errores y la gestión manual de concurrencia incrementan la curva de desarrollo inicial.
- **Ventaja:** Se genera un único archivo binario estático, logrando un despliegue sin fricción y sin infiernos de dependencias en el servidor de la universidad.