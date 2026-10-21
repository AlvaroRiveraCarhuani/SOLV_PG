# Laboratorio de Práctica — Nivel 3: Arquitecturas Políglotas / Full-Stack

> **Modo en Paso 1:** `Laboratorio de Práctica (Con Editor Visual)` (`IDE_PERSISTENTE`)
> **Propósito del Nivel:** Validar la combinación simultánea de un Lenguaje Backend y una Plataforma Web en Columna 1, la inyección multi-stage de Node 20 en imágenes que no son Node nativas, y la convivencia de complementos `pip` + `npm` en Columna 2.

---

## Resumen del Nivel 3 (`IDE_PERSISTENTE`)

| ID | Sub-nivel | Caso Académico | Columna 1 (Combinada) | Columna 2 (Complementos) | Columna 3 (BD) | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **LAB-3.1** | Fácil | Base Híbrida Python + Angular | `Python 3.12` + `Angular 18` | Ninguno | `Sin Base de Datos` | `PENDIENTE` |
| **LAB-3.2** | Medio | API REST + SPA (`Python` + `FastAPI` + `Angular`) | `Python 3.12` + `Angular 18` | `FastAPI + Uvicorn` (`pip`) | `SQLite3 (Nativo)` | `PENDIENTE` |
| **LAB-3.3** | Difícil | Proyecto Integrador Full-Stack (`Python` + `FastAPI` + `ORM` + `Angular` + `Librería UI` + `PostgreSQL`) | `Python 3.12` + `Angular 18` | `FastAPI` (`pip`) + `SQLAlchemy + Psycopg` (`pip`) + `Angular Material` (`npm`) | `PostgreSQL (Aula) v18` | `PENDIENTE` |

---

## LAB-3.1 (Fácil): Base Híbrida Python 3.12 + Angular 18
- **Tarjeta de propósito (`1. Propósito y Datos del Entorno`):** `Laboratorio de Práctica (Con Editor Visual)`
- **Nombre del Entorno \*:** `Laboratorio: Base Full-Stack Python 3.12 + Angular 18`
- **Columna 1 · Principal → `Lenguaje Principal (Elija 1)`:** Marcar **`Python 3.12`**.
- **Columna 1 · Principal → `Interfaz Web / Frontend (Opcional)`:** Marcar **`Angular 18`**.
- **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`:** Verificar que el encabezado indique **`Filtrados para Python 3.12 + Angular 18`** y dejar todos desmarcados.
- **Columna 3 · Servicio del Nodo → `Base de Datos`:** Marcar **`Sin Base de Datos`**.
- **Qué valida:** Que al cambiar a **`Modo Avanzado (Dockerfile / Scripts)`**, el `Dockerfile del Entorno` combine `FROM python:3.12-slim-bookworm` + `COPY --from=node:20-bookworm-slim /usr/local /usr/local` + `RUN npm install -g @angular/cli@18`, y que **`Comprobar funcionamiento`** verifique `python3`, `node`, `npm` y `ng`.

---

## LAB-3.2 (Medio): API REST + SPA (Python 3.12 + FastAPI + Angular 18)

### Paso A: Registrar `FastAPI + Uvicorn` en Columna 2 (`+ Nuevo`)

Con **`Python 3.12`** y **`Angular 18`** marcados en la Columna 1, hacer clic en **`+ Nuevo`** en la cabecera de **`Columna 2 · Opcional (Múltiple)` (`Complementos Compatibles`)**. Se abrirá el modal **`Registrar Complemento o Librería`**.

| Elemento / Label exacto en el Modal | Valor a Verificar / Copiar y Pegar |
| :--- | :--- |
| **Banner superior (Contexto fijado)** | Verificar que muestre **`Columna 2 · Complemento / Librería`** |
| **Nombre Visible \*** *(Input texto)* | `FastAPI + Uvicorn` |
| **Descripción breve para docentes y administradores** *(Input texto)* | `Framework web asíncrono de alto rendimiento con servidor ASGI Uvicorn` |
| **Icono Oficial de la Herramienta** *(Barra inteligente)* | Verificar que detecte **`FastAPI`** (`Detectado automáticamente`) |
| **Compatible con la tecnología \*** *(Combobox)* | Verificar que venga preseleccionado **`Python 3.12`** |
| **¿Cómo se instala esta herramienta?** *(Combobox)* | Verificar que venga preseleccionado **`Automático: Paquete de Python (pip)`** |
| **Nombre del paquete o librería a instalar \*** *(Input texto)* | `fastapi uvicorn[standard]` |
| **Programa a comprobar al iniciar (Opcional)** *(Input texto)* | `uvicorn` |
| **Perfil de Consumo en Memoria** *(Combobox)* | Seleccionar **`Servidor de desarrollo activo en segundo plano (ej. Vite, Webpack, CLI en vivo)`** |

*(Hacer clic en **`Guardar Tecnología`**)*

### Paso B: Composición en la Vista Principal
- **Nombre del Entorno \*:** `Laboratorio: Desarrollo Web con FastAPI + Angular 18`
- **Columna 1 · Principal:** `Python 3.12` (`Lenguaje Principal (Elija 1)`) + `Angular 18` (`Interfaz Web / Frontend (Opcional)`)
- **Columna 2 · Opcional (Múltiple):** Dejar marcado **`FastAPI + Uvicorn`** (`1 seleccionados`)
- **Columna 3 · Servicio del Nodo → `Base de Datos`:** Marcar **`SQLite3 (Nativo)`** (`+0 MB RAM`)
- **Paso C:** Ejecutar **`Comprobar funcionamiento`** y luego **`Publicar Plantilla Oficial`**.

---

## LAB-3.3 (Difícil — Caso Estrella): Proyecto Integrador Full-Stack Completo
- **Nombre del Entorno \*:** `Laboratorio: Ingeniería Full-Stack (Python 3.12 + FastAPI + SQLAlchemy + Angular 18 + Material + PostgreSQL)`
- **Columna 1 · Principal:** `Python 3.12` (`Lenguaje Principal (Elija 1)`) + `Angular 18` (`Interfaz Web / Frontend (Opcional)`)
- **Columna 2 · Opcional (Múltiple) → `Complementos Compatibles`:** Marcar **`FastAPI + Uvicorn`**, **`SQLAlchemy + Psycopg`** y **`Angular Material`** (o **`PrimeNG`**) — debe mostrar el badge **`3 seleccionados`**.
- **Columna 3 · Servicio del Nodo → `Base de Datos`:** Marcar **`PostgreSQL (Aula) v18`** (`~50 MB RAM`).
- **Qué valida:** Convivencia de complementos `pip` y `npm` en simultáneo en la Columna 2, compilación multi-stage sin errores `EBADENGINE`, sugerencia automática de `1024 MB` de RAM y verificación de todos los binarios (`python3`, `node`, `npm`, `ng`, `uvicorn`, `psql`).
