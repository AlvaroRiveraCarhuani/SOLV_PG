# Docente — Nivel 4: Contenido y reutilización

> **Rutas:** `/teacher/exercises`, `/teacher/templates`, `/teacher/cursos/:id`
> **Propósito:** comprobar que el docente reutiliza prácticas, solicita un entorno al administrador y usa la libreta para cerrar el ciclo de notas.
> **Criterio del nivel:** DOC-4.1, DOC-4.2 y DOC-4.3 en `APROBADO`.

---

## Resumen del Nivel 4

| ID | Subnivel | Caso | Ruta principal | Resultado esperado | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **DOC-4.1** | Fácil | Reutilizar una práctica del banco | `/teacher/exercises` | Práctica duplicada en otra materia sin perder casos ni configuración | `PENDIENTE` |
| **DOC-4.2** | Medio | Solicitar un entorno al administrador | `/teacher/templates` | Solicitud enviada, seguimiento visible y entorno aprobado disponible | `PENDIENTE` |
| **DOC-4.3** | Difícil | Revisar la libreta y exportar notas | `/teacher/cursos/:id` | Notas coherentes, celdas sin corregir marcadas y exportación válida | `PENDIENTE` |

---

## DOC-4.1 (Fácil): Reutilizar una práctica del banco

**Objetivo:** localizar una práctica existente y duplicarla a otra materia conservando casos, lenguaje y configuración.

**Precondiciones:** docente con al menos dos materias; práctica de evaluación de DOC-1.2 publicada.

**Datos de prueba:** práctica origen `DOC-PILOTO Suma de enteros`; materia destino `DOC-PILOTO-02`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/exercises`. | Se muestra el listado de prácticas reutilizables con buscador y filtros. |
| 2 | Buscar `DOC-PILOTO Suma de enteros`. | La práctica aparece una sola vez y muestra materia y tecnología. |
| 3 | Hacer clic en `Duplicar`. | Se abre el modal `Duplicar laboratorio` con la materia destino. |
| 4 | Seleccionar `DOC-PILOTO-02` y confirmar. | La copia se crea en la materia destino y el modal se cierra. |
| 5 | Abrir la copia y revisar su configuración. | Conserva lenguaje, formato, casos y límite de tiempo; el título permite distinguirla del original. |
| 6 | Editar el título de la copia y guardar. | Solo cambia la copia; la práctica original permanece intacta. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura del listado, del modal de duplicación y de ambas prácticas con títulos distintos |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** la copia pierde casos o límites, se modifica el original, o la práctica se duplica en una materia sin permiso.

---

## DOC-4.2 (Medio): Solicitar un entorno al administrador

**Objetivo:** enviar una solicitud de entorno, seguir su estado y comprobar que, una vez aprobada, aparece disponible en el editor de prácticas.

**Precondiciones:** cuenta docente; administrador disponible para aprobar; catálogo sin duplicados del entorno solicitado.

**Datos de prueba:** nombre `DOC-PILOTO Entorno de análisis`; imagen base oficial; propósito `Análisis de datos`.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/templates`. | Se muestra el catálogo disponible y la acción `Solicitar entorno`. |
| 2 | Hacer clic en `Solicitar entorno`. | Se abre el formulario de solicitud con perfil pedagógico e imagen base. |
| 3 | Completar nombre, propósito, imagen oficial y justificación; enviar. | La solicitud se registra con estado pendiente o `En revisión`. |
| 4 | Verificar el seguimiento en `/teacher/templates`. | La solicitud aparece con su estado y no se puede publicar mientras no esté aprobada. |
| 5 | Como administrador, aprobar la solicitud. | El estado cambia a aprobado y el entorno queda en el catálogo. |
| 6 | Volver al editor de una práctica de desarrollo. | El entorno aprobado aparece en `Entorno homologado` y puede seleccionarse. |
| 7 | Intentar seleccionar una solicitud rechazada o pendiente. | No aparece como opción publicable o muestra el motivo del bloqueo. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la solicitud enviada, del estado aprobado y del entorno en el editor |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** se puede publicar con una solicitud pendiente, el estado no se actualiza, o el entorno aprobado no aparece en el editor.

---

## DOC-4.3 (Difícil): Revisar la libreta y exportar notas

**Objetivo:** verificar que la libreta refleja las calificaciones, distingue lo no corregido y exporta un archivo con las mismas cifras.

**Precondiciones:** materia con al menos dos estudiantes y dos prácticas; una entrega calificada y otra sin corregir.

**Datos de prueba:** `EST-PILOTO-01` con nota; `EST-PILOTO-02` sin corregir; exportación CSV y Excel.

### Pasos

| # | Acción | Resultado esperado |
| :--- | :--- | :--- |
| 1 | Abrir `/teacher/cursos/:id` y seleccionar la pestaña `Libreta`. | Se muestra la tabla de estudiantes por práctica y la columna `Nota de Laboratorio`. |
| 2 | Revisar la celda de `EST-PILOTO-01`. | Muestra la nota registrada en DOC-2.2 o DOC-2.3. |
| 3 | Revisar la celda de `EST-PILOTO-02`. | Muestra `—` con la leyenda `Sin corregir`; no se interpreta como cero. |
| 4 | Verificar la nota de laboratorio. | Coincide con las notas corregidas y no incluye las celdas sin corregir como ceros. |
| 5 | Abrir la exportación y elegir `Exportar CSV Plano`. | El archivo se descarga y contiene las mismas notas y celdas vacías. |
| 6 | Repetir con `Descargar Excel Estilizado (.xlsx)`. | El archivo abre en una hoja de cálculo y conserva encabezados y notas. |
| 7 | Comparar la libreta en pantalla con ambos archivos. | No hay diferencias de cifras, estudiantes ni prácticas. |

### Evidencia y cierre

| Campo | Registro |
| :--- | :--- |
| **Estado** | `PENDIENTE` |
| **Evidencia** | Captura de la libreta y ambos archivos exportados |
| **Ejecutado por** | — |
| **Fecha y hora** | — |
| **Observaciones** | — |

**Falla si:** una celda sin corregir se exporta como cero, la nota de laboratorio no coincide, o la exportación omite estudiantes o prácticas.
