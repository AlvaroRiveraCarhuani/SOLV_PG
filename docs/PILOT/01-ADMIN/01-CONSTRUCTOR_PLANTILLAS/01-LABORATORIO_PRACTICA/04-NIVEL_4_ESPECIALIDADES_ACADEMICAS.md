# Laboratorio de Práctica — Nivel 4: Especialidades Universitarias No-Web

> **Modo en Paso 1:** `Laboratorio de Práctica (Con Editor Visual)` (`IDE_PERSISTENTE`)
> **Propósito del Nivel:** Validar cátedras de Sistemas Operativos, Redes, Ciencia de Datos y Lenguajes Formales que no responden al molde tradicional de aplicaciones web.

---

## Resumen del Nivel 4 (`IDE_PERSISTENTE`)

| ID | Sub-nivel | Caso Académico | Configuración Clave | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **LAB-4.1** | Fácil | Sistemas Operativos y Redes (Linux / Bash / Auditoría de Red) | `Debian Bookworm` + `apt` (`iproute2 netcat-openbsd tcpdump strace htop procps`) | `PENDIENTE` |
| **LAB-4.2** | Medio | Ciencia de Datos y Machine Learning (`numpy`, `pandas`, `scikit-learn`) | `Python 3.12` + Paquetes pesados + Ajuste de RAM a `2048 MB` | `PENDIENTE` |
| **LAB-4.3** | Difícil | Lenguaje Nuevo fuera de los 6 clásicos (`Rust 1.81` + Addon propio de Rust) | Crear Runtime `Rust` + Crear Complemento con `cargo` compatible con `rust` | `PENDIENTE` |

---

## LAB-4.1 (Fácil): Sistemas Operativos y Redes — Laboratorio Linux POSIX

### Paso A: Registrar en `Columna 1 · Principal` (`+ Nuevo lenguaje`)

Dentro de **`Columna 1 · Principal` → `Lenguaje Principal (Elija 1)`**, hacer clic en **`+ Nuevo lenguaje`**. Se abrirá el modal **`Registrar Lenguaje Principal`**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 1 · Lenguaje Principal`** |
| **Nombre Visible \*** *(Input texto)* | `Debian Linux (Sistemas y Redes)` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `Entorno Linux POSIX con utilidades de diagnóstico de procesos, sockets y tráfico de red` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que detecte **`Linux / Bash`** (`Detectado automáticamente`) |
| **Imagen Base Oficial (sin :latest) \*** *(Input texto)* | `debian:bookworm-slim` |
| **Comando de evaluación para Juez Virtual (Opcional)** *(Input texto)* | `bash solution.sh` |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Seleccionar **`Automático: Programa del Sistema (apt / apk)`** |
| **Nombre del paquete o librería a instalar \*** *(Input texto)* | `iproute2 netcat-openbsd tcpdump strace htop procps` |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | `bash, ip, strace, tcpdump` |
| **Perfil de Consumo en Memoria** *(Combobox)* | Seleccionar **`Ligero — Compilador nativo o consola (ej. C, C++, Bash)`** |

*(Hacer clic en **`Guardar Tecnología`**, desmarcar `Angular 18` si había quedado activa en `Interfaz Web / Frontend (Opcional)`, dejar `Sin Base de Datos` en Columna 3, comprobar funcionamiento y publicar)*

---

## LAB-4.2 (Medio): Ciencia de Datos e IA — Python Científico + Alta Memoria

### Paso A: Seleccionar `Python 3.12` en Columna 1 y abrir `+ Nuevo` en Columna 2

Con **`Python 3.12`** marcado en `Columna 1 · Principal`, hacer clic en **`+ Nuevo`** en **`Columna 2 · Opcional (Múltiple)` (`Complementos Compatibles`)**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 2 · Complemento / Librería`** |
| **Nombre Visible \*** *(Input texto)* | `Stack Ciencia de Datos (NumPy + Pandas + Scikit-Learn)` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `Librerías de análisis numérico, dataframes y modelos de aprendizaje automático` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que muestre **`Python`** (`Detectado automáticamente`) |
| **Compatible con la tecnología \*** *(Combobox)* | Verificar que venga preseleccionado **`Python 3.12`** |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Verificar que venga preseleccionado **`Automático: Paquete de Python (pip)`** |
| **Nombre del paquete o librería a instalar \*** *(Input texto)* | `numpy pandas scikit-learn matplotlib` |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | *(Dejar vacío)* |
| **Perfil de Consumo en Memoria** *(Combobox)* | Seleccionar **`Motor pesado en memoria (ej. Ciencia de Datos / Machine Learning / JVM)`** |

*(Hacer clic en **`Guardar Tecnología`**)*

### Paso B: Ajuste de Memoria en el Paso 3
- En **`3. Capacidad del Aula y Memoria por Alumno`**, verificar que el mínimo requerido suba a **`510 MB`** (`210 MB` editor + `300 MB` motor pesado) y que la tarjeta sugerida sea **`768 MB por alumno`**.
- Hacer clic en **`Configurar memoria manualmente (opcional)`** y escribir **`2048`** en el input **`Ajuste manual por alumno (MB)`** (o elegir la tarjeta **`Proyectos Integradores y Trabajos Pesados`**).
- Ejecutar **`Comprobar funcionamiento`** y **`Publicar Plantilla Oficial`**.

---

## LAB-4.3 (Difícil): Lenguaje Fuera de Semillas (`Rust 1.81`) + Su Propio Ecosistema

### Paso A: Registrar `Rust 1.81` en `Columna 1 · Principal` (`+ Nuevo lenguaje`)
- **Banner superior:** `Columna 1 · Lenguaje Principal`
- **Nombre Visible \*:** `Rust 1.81`
- **Descripción breve para docentes y administradores:** `Compilador oficial rustc y gestor de paquetes Cargo 1.81`
- **Icono Oficial de la Herramienta:** Verificar que detecte **`Rust`** (`Detectado automáticamente`)
- **Imagen Base Oficial (sin :latest) \*:** `rust:1.81-slim-bookworm`
- **Comando de evaluación para Juez Virtual (Opcional):** `rustc -O solution.rs -o solution && ./solution`
- **¿Cómo se instala esta herramienta?:** `Ninguna (La imagen base ya incluye todo)`
- **Programa a comprobar al iniciar (Opcional):** `rustc, cargo`
- **Perfil de Consumo en Memoria:** `Estándar — Lenguaje interpretado o compilado moderno (ej. Python, Go, PHP, Node)`
- Hacer clic en **`Guardar Tecnología`**.

### Paso B: Registrar Complemento de Rust en `Columna 2 · Opcional (Múltiple)` (`+ Nuevo`)
- Con **`Rust 1.81`** activo en la Columna 1, hacer clic en **`+ Nuevo`** en la Columna 2:
  1. Verificar que el combobox **`Compatible con la tecnología *`** ya venga preseleccionado con **`Rust 1.81`**.
  2. Verificar que el combobox **`¿Cómo se instala esta herramienta?`** ya venga cambiado automáticamente a **`Automático: Paquete de Rust (cargo)`** y que el placeholder del paquete sugiera `Ej: serde tokio`.
  3. Verificar que **`Icono Oficial de la Herramienta`** herede automáticamente el icono de **`Rust`**.
