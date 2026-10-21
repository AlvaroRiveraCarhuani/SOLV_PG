# Evaluador Automático — Nivel 3: Edge Cases y Aislamiento de Juez

> **Modo en Paso 1:** `Evaluador Automático de Ejercicios` (`JUEZ_EFIMERO`)
> **Propósito del Nivel:** Validar que el Constructor limpie y bloquee configuraciones incompatibles con el aislamiento de un contenedor efímero (Plataformas Web, Bases de Datos de Aula y `entrypoint` vacío).

---

## Resumen del Nivel 3 (`JUEZ_EFIMERO`)

| ID | Sub-nivel | Caso de Prueba | Qué Estresa | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **JUEZ-3.1** | Fácil | Conmutación en vivo de `Laboratorio` a `Evaluador` con Angular y PostgreSQL marcados | Limpieza automática de `webPlatformSlug`, `databaseSlug` y complementos web | `PENDIENTE` |
| **JUEZ-3.2** | Medio | Intento de componer Evaluador sin Lenguaje Principal (Solo Web) | Guard en Backend (`Rejects Empty RuntimeID even if WebPlatformID is provided in JUEZ_EFIMERO`) | `PENDIENTE` |
| **JUEZ-3.3** | Difícil | Validación de `entrypoint` vacío, `timeout_ms` fuera de rango y piso vital de 64 MB | Rechazo de publicación sin comando de evaluación válido | `PENDIENTE` |

---

## JUEZ-3.1 (Fácil): Conmutación Reactiva de Modo (`IDE_PERSISTENTE` → `JUEZ_EFIMERO`)
- **Acción:** En la tarjeta **`Laboratorio de Práctica (Con Editor Visual)`**, marcar **`Python 3.12`** (en `Lenguaje Principal (Elija 1)`), **`Angular 18`** (en `Interfaz Web / Frontend (Opcional)`), **`Angular Material`** (en `Complementos Compatibles`) y **`PostgreSQL (Aula) v18`** (en `Base de Datos`). Luego hacer clic arriba en la tarjeta **`Evaluador Automático de Ejercicios`** (o en **`Laboratorio + Evaluador`**).
- **Resultado esperado:** El Constructor oculta y desmarca automáticamente `Angular 18`, `Angular Material` y `PostgreSQL (Aula) v18`, dejando únicamente **`Python 3.12`** y recalculando la memoria sobre el piso de **`64 MB`** (en `Evaluador Automático de Ejercicios`).

## JUEZ-3.2 (Medio): Bloqueo de Plataforma Web en Modo Juez
- **Acción:** Estando en **`Evaluador Automático de Ejercicios`**, verificar que en **`Columna 1 · Principal`** solo aparezca la sección **`Lenguaje Principal (Elija 1)`** con el botón **`+ Nuevo lenguaje`** (sin mostrar `Interfaz Web / Frontend (Opcional)` ni el botón `+ Nueva web`), y que al hacer clic en **`+ Nuevo lenguaje`** el modal abra bloqueado en **`Columna 1 · Lenguaje Principal`**.

## JUEZ-3.3 (Difícil): Guardrails de `entrypoint` y Límites de Ejecución
- **Acción 1 (Entrypoint vacío):** Con **`Python 3.12`** seleccionado en modo **`Evaluador Automático de Ejercicios`**, borrar todo el texto del input **`Comando de compilación y ejecución del Evaluador *`** debajo de las columnas.
  - **Resultado esperado:** Tanto **`Comprobar funcionamiento`** como **`Publicar Plantilla Oficial`** se bloquean mostrando el aviso en ámbar: `Indique el comando de compilación y ejecución del Evaluador Automático.`
- **Acción 2 (Violación del piso de 64 MB en Juez):** Volver a escribir `python3 solution.py`, abrir **`Configurar memoria manualmente (opcional)`** en el Paso 3 y escribir **`32`** en **`Ajuste manual por alumno (MB)`**.
  - **Resultado esperado:** Aparece la alerta roja `La memoria indicada (32 MB) es menor a los 64 MB que necesitan las tecnologías elegidas para iniciar correctamente.` y se bloquean los botones de comprobación y publicación.
