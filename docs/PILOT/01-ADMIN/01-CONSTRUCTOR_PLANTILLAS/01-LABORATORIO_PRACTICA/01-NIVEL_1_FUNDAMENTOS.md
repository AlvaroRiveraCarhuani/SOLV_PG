# Laboratorio de Práctica — Nivel 1: Fundamentos (Columna 1 Pura)

> **Modo en Paso 1:** `Laboratorio de Práctica (Con Editor Visual)` (`IDE_PERSISTENTE`)
> **Propósito del Nivel:** Validar el registro y compilación de motores primarios en la Columna 1 (`Lenguaje y Web`) sin complementos de Columna 2 ni bases de datos de Columna 3.

---

## Resumen del Nivel 1 (`IDE_PERSISTENTE`)

| ID | Sub-nivel | Caso Académico | Estrategia en Runtime | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **LAB-1.1** | Fácil | Introducción a la Programación (Python 3.12 Puro) | `Ninguna` (`none`) | `EN CURSO` |
| **LAB-1.2** | Medio | Estructuras de Datos en C/C++ (`gcc` + `gdb` + `valgrind` + `cmake`) | `apt` (`gdb valgrind cmake`) | `PENDIENTE` |
| **LAB-1.3** | Difícil | Diseño de Interfaces Web (Solo Angular 18 sin Lenguaje Backend) | `npm` (`@angular/cli@18`) | `PENDIENTE` |

---

## LAB-1.1 (Fácil): Introducción a la Programación — Python 3.12 Puro

### Paso A: Registrar el Lenguaje Principal (`+ Nuevo lenguaje` en `Columna 1 · Principal`)

En la sección **`2. Arquitectura y Tecnologías del Entorno`**, dentro de **`Columna 1 · Principal` (`Lenguaje y Web` → `Lenguaje Principal (Elija 1)`)**, hacer clic en el botón **`+ Nuevo lenguaje`**. Se abrirá el modal **`Registrar Lenguaje Principal`**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 1 · Lenguaje Principal`** *(Clasificación fijada por el contexto del Constructor — no tocar `Cambiar tipo`)* |
| **Nombre Visible \*** *(Input texto)* | `Python 3.12` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `Intérprete oficial de Python 3.12 sobre Debian Bookworm para fundamentos y algoritmos` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que al escribir el nombre cambie solo a **`Python`** con el texto **`Detectado automáticamente`** *(sin necesidad de abrir `Cambiar icono`)* |
| **Imagen Base Oficial (sin :latest) \*** *(Input texto)* | `python:3.12-slim-bookworm` |
| **Comando de evaluación para Juez Virtual (Opcional)** *(Input texto)* | `python3 solution.py` |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Verificar que ya venga seleccionado **`Ninguna (La imagen base ya incluye todo)`** |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | `python3` |
| **Perfil de Consumo en Memoria** *(Combobox)* | Verificar que ya venga seleccionado **`Estándar — Lenguaje interpretado o compilado moderno (ej. Python, Go, PHP, Node)`** |

*(Hacer clic en **`Guardar Tecnología`** — el modal se cerrará y seleccionará automáticamente **`Python 3.12`** en la Columna 1)*

### Paso B: Datos del Entorno en la Vista Principal (`1. Propósito y Datos del Entorno` y `2. Arquitectura y Tecnologías del Entorno`)

| Sección / Label exacto en la Vista | Valor a Seleccionar / Copiar y Pegar |
| :--- | :--- |
| **Tarjeta de propósito (`1. Propósito y Datos del Entorno`)** | **`Laboratorio de Práctica (Con Editor Visual)`** *(con badge `Seleccionado`)* |
| **Nombre del Entorno \*** *(Input texto)* | Verificar que se haya autocompletado `Laboratorio: Python 3.12`, o cambiarlo a: `Laboratorio: Fundamentos de Python 3.12` |
| **Categoría Académica** *(Combobox)* | `Programación General` *(o la primera categoría disponible)* |
| **Descripción Pedagógica (Opcional)** *(Input texto)* | `Entorno limpio con Python 3.12 estándar para prácticas de lógica, estructuras de control y funciones en primer semestre.` |
| **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`** | Verificar que **`Python 3.12`** esté marcado |
| **Columna 1 · Principal → `Interfaz Web / Frontend (Opcional)`** | *(Dejar sin seleccionar ninguna)* |
| **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`** | *(Dejar sin seleccionar ninguno)* |
| **Columna 3 · Servicio del Nodo → `Base de Datos`** | Verificar que esté marcado **`Sin Base de Datos`** |
| **3. Capacidad del Aula y Memoria por Alumno** | Verificar que esté marcada la tarjeta **`Prácticas y Ejercicios de Clase`** (`512 MB por alumno`, badge `Sugerido`) y que muestre `Mínimo requerido: 384 MB` |

### Paso C: Verificación en el Panel Derecho (`Resumen del Entorno`)
1. En el panel derecho (**`Resumen del Entorno`** → pestaña **`Resumen`**), verificar que figuren:
   - **`Imagen Contenedora`:** `python:3.12-slim-bookworm` con badge **`Imagen Oficial`**
   - **`Tecnologías Seleccionadas (1)`:** `Python 3.12` (`Base`)
   - **`Memoria por alumno`:** `512 MB`
   - **`Programas Incluidos`:** `python3`
2. Hacer clic en **`Comprobar funcionamiento`** y verificar que aparezca el recuadro verde **`¡Entorno Verificado!`** reportando la versión de `python3`.
3. Hacer clic en **`Publicar Plantilla Oficial`**.

### Bitácora de Ejecución (LAB-1.1)
- **Estado:** `EN CURSO`
- **Observaciones detectadas:** *(Pendiente de resultados de prueba)*

---

## LAB-1.2 (Medio): Estructuras de Datos en C/C++ (`gcc` + `gdb` + `valgrind` + `cmake`)

### Paso A: Selección Visual Directa desde el Catálogo (Sin Comandos)

El catálogo institucional ya provee separados **C (GCC 13)** (`solution.c`) y **C++ (G++ 13)** (`solution.cpp`) en la **Columna 1**, además de los complementos **Depuración y Memoria (GDB + Valgrind)** y **Construcción CMake + Ninja** en la **Columna 2**.

*(Opcional — Si desea probar el alta guiada en Columna 2 con `+ Nuevo complemento`):*
| Elemento / Label exacto en el Modal | Valor a Verificar / Seleccionar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 2 · Complemento / Librería`** |
| **Compatible con la tecnología** | Verificar que venga preseleccionado **`C++ (G++ 13)`** y que el instalador quede fijado automáticamente en **`Programa del Sistema (apt / apk)`** (bloqueando `npm`, `pip`, etc.) |
| **Nombre Visible \*** | `Toolchain Depuración C++ (GDB + Valgrind + CMake)` |
| **Paquetes o herramientas a incluir \*** | `gdb valgrind cmake` *(en un solo campo; el sistema instala y verifica cada paquete automáticamente sin preguntar memoria ni comandos)* |

### Paso B: Datos del Entorno en la Vista Principal (`1. Propósito y Datos del Entorno` y `2. Arquitectura y Tecnologías del Entorno`)

| Sección / Label exacto en la Vista | Valor a Seleccionar / Copiar y Pegar |
| :--- | :--- |
| **Tarjeta de propósito (`1. Propósito y Datos del Entorno`)** | **`Laboratorio de Práctica (Con Editor Visual)`** |
| **Nombre del Entorno \*** *(Input texto)* | `Laboratorio: Estructuras de Datos en C/C++ (GCC 13 + Valgrind)` |
| **Categoría Académica** *(Combobox)* | `Programación General` |
| **Descripción Pedagógica (Opcional)** *(Input texto)* | `Entorno de programación de bajo nivel con compilador C/C++17/20, depurador GDB y auditoría de memoria dinámica con Valgrind.` |
| **Columna 1 · Principal -> `Lenguaje Principal (Elija 1)`** | Seleccionar **`C++ (G++ 13)`** (o **`C (GCC 13)`** para C estructurado) |
| **Columna 1 · Principal -> `Interfaz Web / Frontend (Opcional)`** | *(Dejar sin seleccionar ninguna)* |
| **Columna 2 · Opcional (Múltiple) -> `Complementos Compatibles`** | Marcar **`Depuración y Memoria (GDB + Valgrind)`** y **`Construcción CMake + Ninja`** |
| **Columna 3 · Servicio del Nodo -> `Base de Datos`** | **`Sin Base de Datos`** |

### Paso C: Verificación Esperada en el Panel Derecho (`Resumen del Entorno`)
1. Verificar que **`Imagen Contenedora`** muestre `solv-local/stack:...` con el badge **`Construcción Local`**, memoria recomendada de **`384 MB`** y programas incluidos `g++`, `gcc`, `make`, `gdb`, `valgrind`, `cmake`.
2. Hacer clic en **`Comprobar funcionamiento`** y verificar que detecte las versiones de `g++`, `gdb`, `valgrind` y `cmake`.
3. Hacer clic en **`Publicar Plantilla Oficial`**.

### Bitácora de Ejecución (LAB-1.2)
- **Estado:** `PENDIENTE`
- **Observaciones detectadas:** —

---

## LAB-1.3 (Difícil dentro de Nivel 1): Solo Plataforma Web Autónoma — Angular 18 Puro (Sin Lenguaje Backend)

### Paso A: Registrar la Interfaz Web (`+ Nueva web` en `Columna 1 · Principal`)

Dentro de **`Columna 1 · Principal`**, bajar a la segunda subsección **`Interfaz Web / Frontend (Opcional)`** y hacer clic en el botón **`+ Nueva web`**. Se abrirá el modal **`Registrar Plataforma Web / Frontend`**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 1 · Interfaz Web / Frontend`** *(y que NO aparezca el campo `Comando de evaluación para Juez Virtual`)* |
| **Nombre Visible \*** *(Input texto)* | `Angular 18` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `Framework web SPA con Angular CLI 18 sobre Node.js 20 LTS` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que detecte solo **`Angular`** (`Detectado automáticamente`) |
| **Imagen Base Oficial (sin :latest) \*** *(Input texto)* | Verificar que ya venga pre-llenado con **`node:20-bookworm-slim`** |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Verificar que ya venga preseleccionado **`Automático: Paquete de Node.js (npm)`** |
| **Nombre del paquete o librería a instalar \*** *(Input texto)* | `@angular/cli@18` |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | `node, npm, ng` |
| **Perfil de Consumo en Memoria** *(Combobox)* | Verificar que venga preseleccionado **`Estándar — Lenguaje interpretado o compilado moderno (ej. Python, Go, PHP, Node)`** |

*(Hacer clic en **`Guardar Tecnología`** — marcará automáticamente **`Angular 18`** dentro de `Interfaz Web / Frontend (Opcional)`)*

### Paso B: Datos del Entorno en la Vista Principal (`1. Propósito y Datos del Entorno` y `2. Arquitectura y Tecnologías del Entorno`)

| Sección / Label exacto en la Vista | Valor a Seleccionar / Copiar y Pegar |
| :--- | :--- |
| **Tarjeta de propósito (`1. Propósito y Datos del Entorno`)** | **`Laboratorio de Práctica (Con Editor Visual)`** |
| **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`** | **Ninguno** *(Si quedó marcado un lenguaje anterior en `Lenguaje Principal (Elija 1)`, hacer clic sobre esa tarjeta para DESMARCARLA)* |
| **Columna 1 · Principal → `Interfaz Web / Frontend (Opcional)`** | Verificar que **`Angular 18`** quede como la única tecnología seleccionada en Columna 1 |
| **Nombre del Entorno \*** *(Input texto)* | `Laboratorio: Desarrollo Frontend con Angular 18` |
| **Categoría Académica** *(Combobox)* | `Desarrollo Web` *(o la categoría disponible)* |
| **Descripción Pedagógica (Opcional)** *(Input texto)* | `Entorno dedicado exclusivamente a arquitectura Frontend con Angular 18 CLI y Node.js 20 LTS, sin motor backend acoplado.` |
| **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`** | *(Dejar sin seleccionar ninguno)* |
| **Columna 3 · Servicio del Nodo → `Base de Datos`** | **`Sin Base de Datos`** |

### Paso C: Verificación Esperada en el Panel Derecho (`Resumen del Entorno`)
1. Verificar que **`Tecnologías Seleccionadas (1)`** muestre `Angular 18` con el rol **`Web`** y que **`Programas Incluidos`** muestre `node`, `npm`, `ng`.
2. Hacer clic en **`Comprobar funcionamiento`** y luego en **`Publicar Plantilla Oficial`**.

### Bitácora de Ejecución (LAB-1.3)
- **Estado:** `PENDIENTE`
- **Observaciones detectadas:** —
