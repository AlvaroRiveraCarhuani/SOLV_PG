# Evaluador Automático — Nivel 2: Librerías Científicas y Estructuras en Juez

> **Modo en Paso 1:** `Evaluador Automático de Ejercicios` (`JUEZ_EFIMERO`)
> **Propósito del Nivel:** Validar que el Juez Virtual también puede evaluar código que requiere librerías de terceros en disco (como `numpy` para métodos numéricos o estructuras de grafos) sin requerir interfaz gráfica ni servidor web.

---

## Resumen del Nivel 2 (`JUEZ_EFIMERO`)

| ID | Sub-nivel | Caso Académico | Lenguaje + Complementos | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **JUEZ-2.1** | Fácil | Métodos Numéricos / Álgebra Lineal | `Python 3.12` + `Stack Ciencia de Datos` (`numpy`) | `PENDIENTE` |
| **JUEZ-2.2** | Medio | Verificación Formal / Pruebas con Pytest | `Python 3.12` + `Pytest` (`entrypoint: pytest -q solution.py`) | `PENDIENTE` |
| **JUEZ-2.3** | Difícil | Algoritmos con Límite Estricto de Memoria (`128 MB`) y Tiempo (`1000 ms`) | `C / C++ (GCC 13)` + `timeout_ms: 1000` + `128 MB` | `PENDIENTE` |

---

## JUEZ-2.1 (Fácil): Evaluador de Métodos Numéricos (Python + NumPy)
- **Tarjeta de propósito (`1. Propósito y Datos del Entorno`):** `Evaluador Automático de Ejercicios`
- **Nombre del Entorno \*:** `Evaluador: Métodos Numéricos con Python + NumPy`
- **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`:** `Python 3.12`
- **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`:** Seleccionar **`Stack Ciencia de Datos (NumPy + Pandas + Scikit-Learn)`**
- **Comando de compilación y ejecución del Evaluador \*:** `python3 solution.py`
- **Qué valida:** Que un Evaluador Automático pueda construir una imagen local `solv-local/stack:...` con paquetes `pip` adicionales, sugiera `512 MB por alumno` en Juez por tratarse de un complemento pesado y exija **`Comprobar funcionamiento`** antes de habilitar **`Publicar Plantilla Oficial`**.

## JUEZ-2.2 (Medio): Evaluador Basado en Pruebas Unitarias (`pytest`)
- **Tarjeta de propósito (`1. Propósito y Datos del Entorno`):** `Evaluador Automático de Ejercicios`
- **Nombre del Entorno \*:** `Evaluador: Pruebas Unitarias con Pytest`
- **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`:** `Python 3.12`
- **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`:** Marcar únicamente **`Pytest`**
- **Comando de compilación y ejecución del Evaluador \*:** Reemplazar `python3 solution.py` por **`pytest -q solution.py`**
- **Qué valida:** Personalización del comando de evaluación en el Modo Visual y reflejo inmediato en el bloque **`Parámetros de Evaluación`** del panel derecho (`Comando de ejecución: pytest -q solution.py`).

## JUEZ-2.3 (Difícil): Evaluador Competitivo Estricto (C++ + Límite de 1000 ms y 128 MB)
- **Tarjeta de propósito (`1. Propósito y Datos del Entorno`):** `Evaluador Automático de Ejercicios`
- **Nombre del Entorno \*:** `Evaluador: Programación Competitiva C++ (Estricto)`
- **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`:** `C / C++ (GCC 13)`
- **Modo Avanzado (`Modo Avanzado (Dockerfile / Scripts)`):** En el campo **`Tiempo máximo por entrega (ms)`**, escribir **`1000`**.
- **3. Capacidad del Aula y Memoria por Alumno:** Verificar que esté seleccionada la tarjeta **`Entregas y Ejercicios Estándar`** (`128 MB por alumno`).
