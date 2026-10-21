# Laboratorio de Práctica — Nivel 2: Monolenguaje + Complementos y Base de Datos

> **Modo en Paso 1:** `Laboratorio de Práctica (Con Editor Visual)` (`IDE_PERSISTENTE`)
> **Propósito del Nivel:** Validar el filtrado dinámico de la Columna 2 (`Complementos Compatibles`), la separación entre librerías de disco (`0 MB RAM`) y motores residentes en Columna 3 (`PostgreSQL` vs `SQLite3`).

---

## Resumen del Nivel 2 (`IDE_PERSISTENTE`)

| ID | Sub-nivel | Caso Académico | Columna 1 | Columna 2 (Complementos) | Columna 3 (BD) | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **LAB-2.1** | Fácil | Calidad de Software y Testing en Python | `Python 3.12` | `Pytest` (`pip`) | `SQLite3 (Nativo)` | `PENDIENTE` |
| **LAB-2.2** | Medio | Bases de Datos Relacionales con ORM | `Python 3.12` | `SQLAlchemy + Psycopg` (`pip`) | `PostgreSQL (Aula) v18` | `PENDIENTE` |
| **LAB-2.3** | Difícil | Frontend Rico con Múltiples Librerías UI | **Solo** `Angular 18` | `Angular Material` + `PrimeNG` (`npm`) | `Sin Base de Datos` | `PENDIENTE` |

---

## LAB-2.1 (Fácil): Calidad de Software — Python 3.12 + Pytest + SQLite3

### Paso A: Seleccionar primero `Python 3.12` en Columna 1 y abrir `+ Nuevo` en Columna 2

1. En **`Columna 1 · Principal` → `Lenguaje Principal (Elija 1)`**, marcar **`Python 3.12`** (y desmarcar cualquier interfaz web si estuviera activa).
2. En la cabecera de **`Columna 2 · Opcional (Múltiple)` (`Complementos Compatibles`)**, hacer clic en **`+ Nuevo`**. Se abrirá el modal **`Registrar Complemento o Librería`**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 2 · Complemento / Librería`** |
| **Nombre Visible \*** *(Input texto)* | `Pytest` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `Framework de pruebas unitarias y funcionales para Python` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que muestre **`Python`** (`Detectado automáticamente`) |
| **Compatible con la tecnología \*** *(Combobox)* | Verificar que ya venga preseleccionado **`Python 3.12`** |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Verificar que ya venga preseleccionado **`Automático: Paquete de Python (pip)`** |
| **Nombre del paquete o librería a instalar \*** *(Input texto)* | `pytest` |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | `pytest` |
| **Perfil de Consumo en Memoria** *(Combobox)* | Verificar que venga preseleccionado **`Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)`** |

*(Hacer clic en **`Guardar Tecnología`** — quedará marcado automáticamente **`Pytest`** en la Columna 2)*

### Paso B: Composición en la Vista Principal
- **Tarjeta de propósito (`1. Propósito y Datos del Entorno`):** `Laboratorio de Práctica (Con Editor Visual)`
- **Nombre del Entorno \*:** `Laboratorio: Testing y Calidad con Python 3.12 + Pytest`
- **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`:** `Python 3.12`
- **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`:** Verificar que esté marcado **`Pytest`** (`1 seleccionados`)
- **Columna 3 · Servicio del Nodo → `Base de Datos`:** Marcar **`SQLite3 (Nativo)`** *(con badge `+0 MB RAM`)*
- **Paso C:** Verificar que la memoria sugerida en el Paso 3 se mantenga en **`512 MB`** (ya que tanto `Pytest` como `SQLite3` suman `0 MB` residentes), hacer clic en **`Comprobar funcionamiento`** (debe detectar `python3`, `pytest` y `sqlite3`) y luego en **`Publicar Plantilla Oficial`**.

---

## LAB-2.2 (Medio): Bases de Datos Relacionales — Python 3.12 + SQLAlchemy/Psycopg + PostgreSQL v18

### Paso A: Registrar Complemento en Columna 2 (`+ Nuevo` con `Python 3.12` activo en Columna 1)

Con **`Python 3.12`** seleccionado en la Columna 1, hacer clic en **`+ Nuevo`** en la cabecera de **`Columna 2 · Opcional (Múltiple)` (`Complementos Compatibles`)**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 2 · Complemento / Librería`** |
| **Nombre Visible \*** *(Input texto)* | `SQLAlchemy + Psycopg` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `ORM relacional y driver nativo para conexión a PostgreSQL` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que muestre **`Python`** (`Detectado automáticamente`) |
| **Compatible con la tecnología \*** *(Combobox)* | Verificar que ya venga preseleccionado **`Python 3.12`** |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Verificar que ya venga preseleccionado **`Automático: Paquete de Python (pip)`** |
| **Nombre del paquete o librería a instalar \*** *(Input texto)* | `sqlalchemy psycopg[binary]` |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | *(Dejar vacío)* |
| **Perfil de Consumo en Memoria** *(Combobox)* | Verificar que venga preseleccionado **`Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)`** |

*(Hacer clic en **`Guardar Tecnología`**)*

### Paso B: Composición en la Vista Principal
- **Nombre del Entorno \*:** `Laboratorio: Bases de Datos con Python 3.12 + SQLAlchemy + PostgreSQL`
- **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`:** `Python 3.12`
- **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`:** Dejar marcado únicamente **`SQLAlchemy + Psycopg`** *(desmarcar `Pytest` haciendo clic sobre su tarjeta)*
- **Columna 3 · Servicio del Nodo → `Base de Datos`:** Marcar **`PostgreSQL (Aula) v18`** *(con badge `~50 MB RAM`)*
- **Paso C:** Verificar en **`3. Capacidad del Aula y Memoria por Alumno`** que el mínimo requerido suba a **`434 MB`** (`210 MB` editor + `50 MB` PostgreSQL + `174 MB` Python) y que la tarjeta sugerida pase a **`768 MB por alumno`**. Ejecutar **`Comprobar funcionamiento`** y **`Publicar Plantilla Oficial`**.

---

## LAB-2.3 (Difícil): Frontend Avanzado — Solo Angular 18 + Angular Material + PrimeNG

### Paso A: Seleccionar solo `Angular 18` en Columna 1 y registrar 2 Complementos en Columna 2 (`+ Nuevo`)

1. En **`Columna 1 · Principal`**:
   - Hacer clic sobre **`Python 3.12`** en `Lenguaje Principal (Elija 1)` para **DESMARCARLO**.
   - En `Interfaz Web / Frontend (Opcional)`, marcar **`Angular 18`**.
2. En **`Columna 2 · Opcional (Múltiple)` (`Complementos Compatibles`)**, verificar que el subtítulo diga **`Filtrados para Angular 18`** y hacer clic en **`+ Nuevo`**:
   - **Complemento 1 (Modal `Registrar Complemento o Librería`):**
     - **Banner superior:** `Columna 2 · Complemento / Librería`
     - **Nombre Visible \*:** `Angular Material`
     - **Descripción breve para docentes y administradores:** `Componentes UI oficiales de Material Design para Angular`
     - **Icono Oficial de la Herramienta:** Verificar que muestre **`Angular`** (`Detectado automáticamente`)
     - **Compatible con la tecnología \*:** Verificar que venga preseleccionado **`Angular 18`**
     - **¿Cómo se instala esta herramienta?:** Verificar que venga preseleccionado **`Automático: Paquete de Node.js (npm)`**
     - **Nombre del paquete o librería a instalar \*:** `@angular/material @angular/cdk`
     - **Programa a comprobar al iniciar (Opcional):** *(Dejar vacío)*
     - **Perfil de Consumo en Memoria:** `Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)`
     - Hacer clic en **`Guardar Tecnología`**.
   - **Complemento 2 (Volver a hacer clic en `+ Nuevo` en Columna 2):**
     - **Nombre Visible \*:** `PrimeNG`
     - **Descripción breve para docentes y administradores:** `Suite de componentes ricos y tablas empresariales para Angular`
     - **Icono Oficial de la Herramienta:** Verificar que muestre **`Angular`** (`Detectado automáticamente`)
     - **Compatible con la tecnología \*:** Verificar que venga preseleccionado **`Angular 18`**
     - **¿Cómo se instala esta herramienta?:** Verificar que venga preseleccionado **`Automático: Paquete de Node.js (npm)`**
     - **Nombre del paquete o librería a instalar \*:** `primeng primeicons`
     - **Programa a comprobar al iniciar (Opcional):** *(Dejar vacío)*
     - **Perfil de Consumo en Memoria:** `Librería o paquete en disco (0 MB residentes extra — solo ocupa disco)`
     - Hacer clic en **`Guardar Tecnología`**.

### Paso B: Composición en la Vista Principal
- **Nombre del Entorno \*:** `Laboratorio: Sistemas de Diseño en Angular 18 (Material + PrimeNG)`
- **Columna 1 · Principal:** Sin Lenguaje Principal, y con **`Angular 18`** marcado en `Interfaz Web / Frontend (Opcional)`.
- **Columna 2 · Opcional (Múltiple):** Verificar que estén marcados **`Angular Material`** y **`PrimeNG`** (`2 seleccionados`).
- **Columna 3 · Servicio del Nodo → `Base de Datos`:** Marcar **`Sin Base de Datos`**.
- **Paso C:** Ejecutar **`Comprobar funcionamiento`** y luego **`Publicar Plantilla Oficial`**.
