# Índice Maestro de Pruebas: Constructor de Entornos y Catálogo de Componentes

> **Ruta del Módulo:** `/admin/plantillas/estudio` y Biblioteca de Componentes (`/admin/plantillas`)
> **Organización:** Separación estricta en dos directorios según el modo de ejecución (`01-LABORATORIO_PRACTICA` vs `02-EVALUADOR_AUTOMATICO`), divididos internamente por niveles y con **3 sub-casos progresivos (`Fácil` → `Medio` → `Difícil`)** por cada nivel.

---

## 1. Reglas y Comportamiento Consolidado del Constructor y Catálogo

1. **Columna 1 con subsecciones siempre visibles y botones dedicados:**
   - La Columna 1 (`Lenguaje y Web`) muestra permanentemente ambos sub-grupos en modo Laboratorio: **`Lenguaje Principal (Elija 1)`** con el botón **`+ Nuevo lenguaje`** y **`Interfaz Web / Frontend (Opcional)`** con el botón **`+ Nueva web`**.
2. **Panel Derecho con guardrails unificados:**
   - Tanto **`Comprobar funcionamiento`** como **`Publicar Plantilla Oficial`** permanecen deshabilitados hasta seleccionar tecnologías válidas, y muestran el motivo exacto del bloqueo encima de los botones.
3. **Capacidad Dual `AMBOS` en Paso 1:**
   - El Paso 1 (`1. Propósito y Datos del Entorno`) ofrece las 3 tarjetas: **`Laboratorio de Práctica (Con Editor Visual)`**, **`Evaluador Automático de Ejercicios`** y **`Laboratorio + Evaluador`** (`AMBOS`), reutilizando la misma plantilla oficial para laboratorio e instancias de juez.
4. **Modal contextualizado con bloqueo de clasificación y herencia de ecosistema:**
   - Al abrir desde **`+ Nuevo lenguaje`**, **`+ Nueva web`** o **`+ Nuevo`** (Columna 2), el modal fija el contexto en un banner superior (`Columna 1 · Lenguaje Principal`, `Columna 1 · Interfaz Web / Frontend` o `Columna 2 · Complemento / Librería`) con opción `Cambiar tipo`, preselecciona la tecnología compatible y ajusta automáticamente el método de instalación (`pip`, `npm`, `apt`, `cargo`, etc.).
5. **Smart Icon Bar con autodetección y galería bajo demanda:**
   - El campo **`Icono Oficial de la Herramienta`** muestra una barra compacta con el icono detectado (`Detectado automáticamente`) y oculta la grilla/URL detrás de los botones **`Cambiar icono`** y **`Usar URL / .svg`**.
6. **Gobernanza en Catálogo de Tecnologías y Plantillas Oficiales:**
   - Taxonomía unificada, detector de complementos huérfanos, botón rápido `+ Complemento` por lenguaje y validación de piso vital dinámico de RAM (ADR-045).

---

## 2. Pista A: `01-LABORATORIO_PRACTICA/` (`IDE_PERSISTENTE` — Con Editor Visual)

| Nivel | Archivo | `.1` Fácil | `.2` Medio | `.3` Difícil |
| :--- | :--- | :--- | :--- | :--- |
| **Nivel 1: Fundamentos (Columna 1 Pura)** | [`01-LABORATORIO_PRACTICA/01-NIVEL_1_FUNDAMENTOS.md`](01-LABORATORIO_PRACTICA/01-NIVEL_1_FUNDAMENTOS.md) | **LAB-1.1:** Python 3.12 Puro (`none`) | **LAB-1.2:** C/C++ con toolchain (`apt`: `gdb valgrind cmake`) | **LAB-1.3:** Solo Plataforma Web Autónoma — Angular 18 (`npm`) sin Backend |
| **Nivel 2: Monolenguaje + Complementos y BD** | [`01-LABORATORIO_PRACTICA/02-NIVEL_2_MONOLENGUAJE_COMPLEMENTOS.md`](01-LABORATORIO_PRACTICA/02-NIVEL_2_MONOLENGUAJE_COMPLEMENTOS.md) | **LAB-2.1:** Python + `Pytest` + SQLite3 | **LAB-2.2:** Python + `SQLAlchemy/Psycopg` + PostgreSQL v18 | **LAB-2.3:** Solo Angular 18 + `Angular Material` + `PrimeNG` |
| **Nivel 3: Políglota / Full-Stack** | [`01-LABORATORIO_PRACTICA/03-NIVEL_3_FULLSTACK_POLIGLOTA.md`](01-LABORATORIO_PRACTICA/03-NIVEL_3_FULLSTACK_POLIGLOTA.md) | **LAB-3.1:** Python + Angular (Base multi-stage) | **LAB-3.2:** Python + FastAPI + Angular + SQLite3 | **LAB-3.3:** Python + FastAPI + ORM + Angular + Librería UI + PostgreSQL v18 |
| **Nivel 4: Especialidades Académicas** | [`01-LABORATORIO_PRACTICA/04-NIVEL_4_ESPECIALIDADES_ACADEMICAS.md`](01-LABORATORIO_PRACTICA/04-NIVEL_4_ESPECIALIDADES_ACADEMICAS.md) | **LAB-4.1:** Sistemas Operativos y Redes (Linux + `tcpdump strace iproute2`) | **LAB-4.2:** Ciencia de Datos e IA (`numpy pandas scikit-learn` + 2048 MB RAM) | **LAB-4.3:** Lenguaje No-Clásico (`Rust 1.81` + Addon propio de Rust) |
| **Nivel 5: Edge Cases y Guardrails** | [`01-LABORATORIO_PRACTICA/05-NIVEL_5_EDGE_CASES_SEGURIDAD.md`](01-LABORATORIO_PRACTICA/05-NIVEL_5_EDGE_CASES_SEGURIDAD.md) | **LAB-5.1:** Auto-reparación en vivo al desmarcar Runtime con Addons | **LAB-5.2:** Edición (`PUT`) de componente en catálogo y propagación | **LAB-5.3:** Caracteres prohibidos en paquetes y piso vital de RAM |

---

## 3. Pista B: `02-EVALUADOR_AUTOMATICO/` (`JUEZ_EFIMERO` — Sin Interfaz Gráfica)

| Nivel | Archivo | `.1` Fácil | `.2` Medio | `.3` Difícil |
| :--- | :--- | :--- | :--- | :--- |
| **Nivel 1: Fundamentos de Juez (Estándar)** | [`02-EVALUADOR_AUTOMATICO/01-NIVEL_1_FUNDAMENTOS_JUEZ.md`](02-EVALUADOR_AUTOMATICO/01-NIVEL_1_FUNDAMENTOS_JUEZ.md) | **JUEZ-1.1:** Evaluador Python 3.12 (`python3 solution.py`, 64/256 MB) | **JUEZ-1.2:** Evaluador Compilado C/C++ (`g++ -O2 solution.cpp -o solution && ./solution`) | **JUEZ-1.3:** Evaluador Rust 1.81 + Prueba de Modo Dual (`Laboratorio + Evaluador`) |
| **Nivel 2: Librerías y Estructuras en Juez** | [`02-EVALUADOR_AUTOMATICO/02-NIVEL_2_LIBRERIAS_Y_ESTRUCTURAS.md`](02-EVALUADOR_AUTOMATICO/02-NIVEL_2_LIBRERIAS_Y_ESTRUCTURAS.md) | **JUEZ-2.1:** Evaluador Python + `Stack Ciencia de Datos` (`numpy`) | **JUEZ-2.2:** Evaluador Python con `Pytest` (`pytest -q solution.py`) | **JUEZ-2.3:** Evaluador Competitivo C++ Estricto (`1000 ms` y `128 MB`) |
| **Nivel 3: Edge Cases y Aislamiento de Juez** | [`02-EVALUADOR_AUTOMATICO/03-NIVEL_3_EDGE_CASES_AISLAMIENTO.md`](02-EVALUADOR_AUTOMATICO/03-NIVEL_3_EDGE_CASES_AISLAMIENTO.md) | **JUEZ-3.1:** Cambio en vivo de `Laboratorio` a `Evaluador` con Web/BD seleccionadas (Auto-limpieza) | **JUEZ-3.2:** Intento de crear Evaluador usando solo Plataforma Web sin Lenguaje Backend (Rechazo) | **JUEZ-3.3:** Validación de `entrypoint` vacío o comando malformado y piso mínimo de 64 MB |
