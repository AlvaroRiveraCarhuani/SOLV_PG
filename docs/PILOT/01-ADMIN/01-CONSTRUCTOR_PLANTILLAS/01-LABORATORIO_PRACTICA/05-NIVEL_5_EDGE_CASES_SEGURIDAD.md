# Laboratorio de Práctica — Nivel 5: Casos Borde, Auto-Reparación y Seguridad

> **Modo en Paso 1:** `Laboratorio de Práctica (Con Editor Visual)` (`IDE_PERSISTENTE`)

---

## Resumen del Nivel 5 (`IDE_PERSISTENTE`)

| ID | Sub-nivel | Caso de Prueba | Qué Estresa | Estado |
| :--- | :--- | :--- | :--- | :--- |
| **LAB-5.1** | Fácil | Auto-reparación reactiva al desmarcar Runtime o Plataforma Web | Limpieza inmediata de complementos incompatibles (`addonSlugs`) e invalidación de verificación | `PENDIENTE` |
| **LAB-5.2** | Medio | Edición (`PUT`) de un Componente existente y sincronización en vivo | Actualizar paquetes o versión de un componente y verificar que el Dockerfile se regenere | `PENDIENTE` |
| **LAB-5.3** | Difícil | Inyección de caracteres prohibidos en paquetes y violación del piso vital de RAM | Rechazo HTTP 400 ante `; rm -rf`, `&&` o saltos de línea, y bloqueo de `ErrRamBelowVitalFloor` | `PENDIENTE` |

---

## LAB-5.1 (Fácil): El "Administrador Indeciso" — Limpieza de Huérfanos en Vivo
- **Acción:** En **`Columna 1 · Principal`**, marcar **`Python 3.12`** (en `Lenguaje Principal (Elija 1)`) y **`Angular 18`** (en `Interfaz Web / Frontend (Opcional)`). En **`Columna 2 · Opcional (Múltiple)`**, marcar **`FastAPI + Uvicorn`** y **`Angular Material`**. Luego hacer clic sobre **`Angular 18`** en la Columna 1 para desmarcarla.
- **Resultado esperado:** **`Angular Material`** desaparece inmediatamente de la Columna 2, del bloque **`Tecnologías Seleccionadas`** en el panel derecho y del **`Dockerfile del Entorno`** en Modo Avanzado, sin quedar huérfano.

## LAB-5.2 (Medio): Edición de Componentes en el Catálogo (`PUT`)
- **Acción:** Hacer clic en el botón de editar (icono de lápiz `Editar tecnología` a la derecha de una tarjeta en la Columna 1 o Columna 2), verificar que el modal abra con título **`Editar Lenguaje Principal`**, **`Editar Plataforma Web / Frontend`** o **`Editar Complemento del Catálogo`** y el botón **`Guardar Cambios`**, modificar sus paquetes o programas verificables y guardar.
- **Resultado esperado:** El componente se actualiza sin duplicarse y el **`Dockerfile del Entorno`** / **`Programas Incluidos`** del Constructor se actualizan al instante.

## LAB-5.3 (Difícil): Guardrails de Seguridad y Piso Vital de RAM
- **Acción 1 (Paquetes maliciosos en Modal):** En el modal de tecnología, elegir `Automático: Paquete de Python (pip)` y escribir `pytest; rm -rf /` en **`Nombre del paquete o librería a instalar *`**, luego pulsar **`Guardar Tecnología`**.
  - **Resultado esperado:** El modal bloquea el guardado mostrando la alerta roja: `El nombre de paquete ";"` (o `rm`) `contiene caracteres inválidos.`
- **Acción 2 (Violación del Piso Vital de RAM en Paso 3):** En **`3. Capacidad del Aula y Memoria por Alumno`**, hacer clic en **`Configurar memoria manualmente (opcional)`** y escribir **`128`** en **`Ajuste manual por alumno (MB)`** estando en modo `Laboratorio de Práctica`.
  - **Resultado esperado:** Aparece la alerta roja `La memoria indicada (128 MB) es menor a los ... MB que necesitan las tecnologías elegidas para iniciar correctamente.` y tanto **`Comprobar funcionamiento`** como **`Publicar Plantilla Oficial`** quedan deshabilitados con el mensaje: `Ajuste la memoria RAM dentro de los límites permitidos en el Paso 3.`
