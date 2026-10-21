# Evaluador Automático — Nivel 1: Fundamentos de Juez (`JUEZ_EFIMERO`)

> **Modo en Paso 1:** `Evaluador Automático de Ejercicios` (`JUEZ_EFIMERO`) y `Laboratorio + Evaluador` (`AMBOS`)
> **Propósito del Nivel:** Validar la creación de entornos de evaluación algorítmica sin interfaz gráfica (y entornos duales), reutilizando los componentes ya registrados en el catálogo (sin duplicarlos) y verificando el **`Comando de compilación y ejecución del Evaluador *`**, el **`Tiempo máximo por entrega (ms)`** y el piso vital reducido (`64 MB`).

---

## Resumen del Nivel 1 (`JUEZ_EFIMERO`)

| ID | Sub-nivel | Caso Académico | Lenguaje | Comando de Ejecución (`entrypoint`) | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **JUEZ-1.1** | Fácil | Algoritmos I — Evaluador Python 3.12 | `Python 3.12` | `python3 solution.py` | `PENDIENTE` |
| **JUEZ-1.2** | Medio | Estructuras de Datos — Evaluador C/C++ Compilado | `C / C++ (GCC 13)` | `g++ -O2 solution.cpp -o solution && ./solution` | `PENDIENTE` |
| **JUEZ-1.3** | Difícil | Programación de Sistemas — Evaluador Rust Compilado + Modo Dual (`Laboratorio + Evaluador`) | `Rust 1.81` | `rustc -O solution.rs -o solution && ./solution` | `PENDIENTE` |

---

## JUEZ-1.1 (Fácil): Evaluador Automático de Python 3.12

### Datos en la Vista Principal (`1. Propósito y Datos del Entorno` y `2. Arquitectura y Tecnologías del Entorno`)

| Sección / Label exacto en la Vista | Valor a Seleccionar / Copiar y Pegar |
| :--- | :--- |
| **Tarjeta de propósito (`1. Propósito y Datos del Entorno`)** | Hacer clic en **`Evaluador Automático de Ejercicios`** *(badge `Seleccionado`)* |
| **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`** | Seleccionar **`Python 3.12`** *(reutiliza el componente ya creado en `LAB-1.1`)* |
| **Nombre del Entorno \*** *(Input texto)* | Verificar que se haya autocompletado `Evaluador: Python 3.12`, o cambiarlo a: `Evaluador: Python 3.12 Estándar` |
| **Categoría Académica** *(Combobox)* | `Programación General` |
| **Descripción Pedagógica (Opcional)** *(Input texto)* | `Motor de calificación automática para ejercicios de algoritmos en Python 3.12 con aislamiento de red y límite de tiempo estricto.` |
| **Comando de compilación y ejecución del Evaluador \*** *(Input debajo de las columnas en `Modo Visual`)* | Verificar que se autocompleté **`python3 solution.py`** |
| **Tiempo máximo por entrega (ms)** *(Opcional: cambiar a pestaña `Modo Avanzado (Dockerfile / Scripts)`)* | Cambiar de `5000` a **`2000`** *(y verificar en el panel derecho `Parámetros de Evaluación` que diga `Tiempo máximo: 2000 ms`)* |

### Qué Validamos
1. Que al elegir **`Evaluador Automático de Ejercicios`**, el título de la Columna 1 pase a **`Lenguaje Principal`**, y tanto la subsección **`Interfaz Web / Frontend (Opcional)`** como la **`Columna 3 · Servicio del Nodo` (`Base de Datos`)** se oculten automáticamente.
2. Que en **`3. Capacidad del Aula y Memoria por Alumno`**, el badge indique **`Mínimo requerido: 64 MB`**, la tarjeta sugerida sea **`Entregas y Ejercicios Estándar`** (`256 MB por alumno`) y la barra segmentada muestre **`Motor de evaluación (32 MB)`**.
3. Que **`Comprobar funcionamiento`** valide el contenedor y permita hacer clic en **`Publicar Plantilla Oficial`**.

---

## JUEZ-1.2 (Medio): Evaluador Automático Compilado C/C++ (GCC 13)

### Datos en la Vista Principal

| Sección / Label exacto en la Vista | Valor a Seleccionar / Copiar y Pegar |
| :--- | :--- |
| **Tarjeta de propósito (`1. Propósito y Datos del Entorno`)** | **`Evaluador Automático de Ejercicios`** |
| **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`** | Seleccionar **`C / C++ (GCC 13)`** *(reutiliza el componente creado en `LAB-1.2`)* |
| **Nombre del Entorno \*** *(Input texto)* | `Evaluador: C / C++17 (GCC 13)` |
| **Categoría Académica** *(Combobox)* | `Programación General` |
| **Descripción Pedagógica (Opcional)** *(Input texto)* | `Evaluador competitivo con compilación optimizada (-O2) para algoritmos y estructuras de datos en C y C++.` |
| **Comando de compilación y ejecución del Evaluador \*** *(Input debajo de las columnas en `Modo Visual`)* | Verificar que se autocompleté **`g++ -O2 solution.cpp -o solution && ./solution`** |
| **Tiempo máximo por entrega (ms)** *(En pestaña `Modo Avanzado (Dockerfile / Scripts)`)* | **`3000`** |
| **3. Capacidad del Aula y Memoria por Alumno** | Verificar que por ser runtime ligero sugiera **`128 MB por alumno`** (`Entregas y Ejercicios Estándar`) con **`Mínimo requerido: 64 MB`** |

---

## JUEZ-1.3 (Difícil): Evaluador de Rust 1.81 y Prueba de Modo Dual (`Laboratorio + Evaluador`)

### Datos en la Vista Principal

| Sección / Label exacto en la Vista | Valor a Seleccionar / Copiar y Pegar |
| :--- | :--- |
| **Tarjeta de propósito (`1. Propósito y Datos del Entorno`)** | Hacer clic en la 3ra tarjeta: **`Laboratorio + Evaluador`** (`AMBOS`) |
| **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`** | Seleccionar **`Rust 1.81`** *(reutiliza el componente creado en `LAB-4.3`)* |
| **Nombre del Entorno \*** *(Input texto)* | Verificar que al no haber sido editado manualmente sugiera el prefijo **`Entorno Dual: Rust 1.81`** |
| **Categoría Académica** *(Combobox)* | `Programación General` |
| **Comando de compilación y ejecución del Evaluador \*** *(En `Modo Visual`)* | Verificar que aparezca autocompletado **`rustc -O solution.rs -o solution && ./solution`** |
| **Resumen del Entorno (Panel Derecho)** | Verificar que el pill bajo el nombre diga **`Laboratorio + Evaluador`**, que aparezca el bloque **`Parámetros de Evaluación`** (`Comando de ejecución` + `Tiempo máximo: 5000 ms`) y que la memoria del Paso 3 respete el piso del editor (`384 MB` mínimo requerido, `512 MB` sugerido) |
