# Manual del Docente — Plataforma Académica SOLV

Bienvenido a la guía de operación docente del Juez Virtual y Plataforma Académica SOLV. Este manual detalla las herramientas avanzadas para la creación de laboratorios, estructuración curricular, exámenes blindados y auditoría forense de entregas.

---

## 1. El Nuevo Editor de Ejercicios de 3 Pasos

El editor de ejercicios opera mediante una arquitectura reactiva dividida en 3 pasos clave:

### Paso 1: Identidad y Pedagogía
- **Modalidad y Propósito:** Selección entre ejercicio de clase (`class`) o examen oficial (`exam`).
- **Parámetros Técnicos:** Título, dificultad (Fácil, Medio, Difícil), límites de tiempo por caso (ms) y memoria RAM (MB).
- **Complejidad Esperada (Opcional):** Define la clase asintótica teórica ($O(1)$, $O(N)$, $O(N \log N)$, $O(N^2)$) para su posterior comparación en SpeedGrader.
- **Etiquetas Temáticas (Tags):** Registro de contenidos para el panel analítico y módulo de recomendaciones.

### Paso 2: Contrato de Entrada y Casos de Prueba
- **Formato Declarativo:** Especificación estructural mediante `FormatBuilder` visual o JSON directo.
- **Visibilidad de 3 Estados por Caso:**
  - `Ejemplo (example)`: Visible en el enunciado con entrada y salida esperada para guiar al estudiante.
  - `Público (public)`: Evaluado normalmente en el envío.
  - `Oculto (hidden)`: Caso privado de evaluación sin revelación de datos.
- **Pesos Relativos:** Ponderación asignada a cada caso de prueba para el cálculo final del puntaje (0–100).

### Paso 3: Publicación y Ajustes Finales
- **Solución de Referencia y Reglas AST Custom:** Código canónico para validación previa y restricciones estáticas (bloqueos de importaciones o funciones prohibidas).
- **Vista Previa del Estudiante:** Botón con ícono de ojo (👁️) que abre el modal de simulación a pantalla completa para verificar exactamente qué verá el alumno.

---

## 2. Formato Declarativo y Fuzzing Estructural v2

El motor de formato estructurado permite generar casos masivos automáticamente sin tipeo manual:

1. **Presets Visuales:** Inserción directa de patrones comunes (Entero Simple, Vector de N elementos, Matriz N×M, Cadenas y Grafos).
2. **Generación con Fuzzing:** El botón **"Generar Casos con Fuzzing"** interpreta la estructura y genera automáticamente casos extremos (límites de rango, valores cero/negativos y vectores masivos).
3. **Validación en Vivo:** La tabla de casos valida instantáneamente que las entradas tipeadas cumplan el contrato del ejercicio.

---

## 3. Checklist de Publicación

Antes de cambiar el estado de un ejercicio a `Publicado`, el sistema ejecuta un pre-chequeo automático:

- **Bloqueantes (Blockers):** Impiden la publicación (ej. falta de título, ausencia de casos de prueba o solución de referencia no probada).
- **Advertencias (Warnings):** Recomendaciones pedagógicas (ej. menos de 3 casos privados o falta de etiquetas temáticas).
- **Información (Info):** Confirmaciones de estado.

---

## 4. Mapa Curricular y Prerrequisitos de Módulos

En la ruta `/teacher/courses/:courseId/curriculum` o en el panel del curso:

1. **Creación de Módulos:** Estructuración de la materia en unidades temáticas (ej. "Módulo 1: Fundamentos", "Módulo 2: Árboles y Grafos").
2. **Prerrequisitos de Desbloqueo:** Selección de módulos previos necesarios. El sistema efectúa un chequeo topológico acíclico impidiendo dependencias circulares.
3. **Exención de Exámenes:** Los ejercicios marcados con `purpose = 'exam'` permanecen siempre accesibles para los estudiantes aunque pertenezcan a un módulo bloqueado.

---

## 5. Panel de Métricas por Tag y Dificultad

Disponible en `/teacher/courses/:courseId/analytics`:

- **Distribución por Dificultad:** Conteo y porcentaje de éxito ($AC / Total$) en ejercicios fáciles, medios y difíciles.
- **Top Tags:** Tasa de efectividad del curso por temas (ej. 45% de éxito en "recursión").
- **Casos más Fallados:** Identificación de los casos de prueba donde el curso presenta mayor índice de respuestas incorrectas.

---

## 6. SpeedGrader y Herramientas Forenses

Al ingresar a la revisión de una entrega (`/teacher/revision/:submissionId`):

### Auditoría Forense: Time-Travel Replay
- Al pulsar **"Replay de Escritura"**, se abre el reproductor interactivo.
- **Slider y Controles:** Reproducción paso a paso o continua (0.5x, 1x, 2x, 4x) de la sesión de tipeo del estudiante.
- **Resaltado de Pegados:** Fragmentos insertados de forma masiva se destacan en rojo con la marca del segundo exacto del evento de portapapeles.

### Benchmark O(N) (Espejo Forense)
- Muestra la complejidad temporal ($O(1)$, $O(N)$, $O(N \log N)$, $O(N^2)$) y espacial empírica detectada.
- **Alerta de Discrepancia:** Si la complejidad detectada difiere de la esperada en el ejercicio, el sistema muestra una advertencia en rojo (`Esperado: O(N log N) | Detectado: O(N²)`).
- **Invariante:** El análisis es 100% forense y no altera la nota del estudiante.

---

## 7. Exámenes Blindados con Semilla Determinista

Para evitar la copia entre estudiantes durante parciales o exámenes finales:

1. En el Paso 1 del Editor, activar la opción **"Generar semilla determinista por estudiante"** (disponible solo si `purpose = 'exam'`).
2. Cada alumno recibirá un conjunto de datos de prueba único derivado determinísticamente de su ID de estudiante.
3. El snapshot exacto de casos se almacena en la entrega (`generated_cases`), permitiendo re-ejecutar o auditar la evaluación ante cualquier reclamo.
