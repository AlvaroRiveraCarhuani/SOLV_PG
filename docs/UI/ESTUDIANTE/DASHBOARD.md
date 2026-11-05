# Vista 1: Inicio del Estudiante

> **Especificación de interfaz y contratos**  
> **Rol:** Estudiante  
> **Ruta:** `/student`  
> **Gobernanza:** SOLV Design System · ADR-059 (entorno por práctica)

---

## 1. Modelo mental

El inicio responde tres preguntas al entrar: **qué clase está abierta**, **qué entrego pronto** y **cómo voy en mis materias**. No es un tablero de widgets ni un panel de infraestructura.

- **La materia es el cuaderno:** agrupa prácticas y conserva el contexto académico.
- **La práctica es la hoja:** tiene su enunciado, su juez y su entorno.
- **El entorno pertenece a la práctica, no a la materia:** el inicio no crea ni reanuda contenedores porque no hay una práctica elegida (ADR-059). Se abre desde la práctica o desde el aula.

Queda fuera de esta vista cualquier métrica de contenedores, CPU, RAM, IDs o nombres de nodo.

---

## 2. Estructura

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Buenos días, Álvaro.                          [ Mis entregas ]               │
│ Tienes clase abierta de Estructuras de Datos.                               │
├───────────────────────────────────────────────┬──────────────────────────────┤
│ Aula ahora                        [ Abierta ] │ Últimas entregas   Historial │
│ SIS-211                                       │ Árboles binarios    AC       │
│ Estructuras de Datos                          │            20/10 14:05:00   │
│ Práctica   Árboles binarios                   ├──────────────────────────────┤
│ Horario    08:00 – 10:00                      │ Tus materias    50% semestre │
│ En el aula 21 de 28                           │ SIS-211 Estructuras  3/4 ▓▓▓░│
│ [ Abrir práctica ]   Ver el aula              │ SIS-312 Sistemas     0/2 ░░░░│
├───────────────────────────────────────────────┤                              │
│ Próximas entregas                          1  │                              │
│ Listas enlazadas · SIS-211   06/11/2026 23:59 │                              │
└───────────────────────────────────────────────┴──────────────────────────────┘
```

### 2.1. Encabezado
- Saludo por franja horaria y nombre del alumno.
- Resumen en una línea, con este orden: fallo de entregas → clase abierta → número de entregas próximas → sin entregas. Si el pedido de entregas falla, el texto **no** afirma que el alumno está al día.
- `[ Mis entregas ]` lleva al historial.

### 2.2. Aula ahora
- Materia con su código y estado de la sesión (`open` → Abierta, `frozen` → Congelada, `closed` → Cerrada).
- Práctica de la sesión (`exercise_title`) o `Sin práctica asociada`.
- Horario como dos `<time>` con el pipe `dateText` en formato `time`.
- Presencia: `present` de `enrolled`, calculada por el latido de los últimos 120 segundos.
- Con práctica en la sesión, la acción principal es `[ Abrir práctica ]` y navega a `/student/exercises/:exerciseId`; `Ver el aula` queda como enlace secundario y, si hay más de una clase, indica cuántas hay. Sin práctica asociada, la acción es `[ Entrar al aula ]` hacia `/student/aula`.
- El aula es contexto (presencia y avances), no un paso obligatorio antes de la práctica: la presencia la emite el layout y el aviso de avance aparece también en la práctica.
- Sin clase abierta: texto explicativo y `[ Ver mis prácticas ]`. Un fallo de carga se muestra como error dentro de la tarjeta, no como ausencia de clase.

Los avances del docente **no** se listan acá: viven en el aula y en la práctica. Duplicarlos permitiría aplicar un avance desde un lugar que no muestra su contexto.

### 2.3. Próximas entregas
- Hasta cuatro entregas de `GET /api/v1/student/assignments/due`, con título, materia y fecha límite.
- Cada fila enlaza a `/student/exercises/:exerciseId`.
- Fallo de carga: mensaje de error explícito. Lista vacía: texto de estado vacío, sin botón.

### 2.4. Últimas entregas
- Las tres entregas más recientes con veredicto y fecha compacta.
- `Historial` lleva a `/student/history`.

### 2.5. Tus materias
- Una fila por materia: código, nombre, prácticas completadas sobre el total y barra de avance.
- El encabezado muestra el porcentaje global del semestre como dato técnico.
- Cada fila enlaza a `/student/courses/:subjectId` (mapa curricular).

---

## 3. Contratos

| Pedido | Uso |
| :--- | :--- |
| `GET /api/v1/student/dashboard` | Materias con totales y avance, y las cinco entregas recientes. No incluye entorno. |
| `GET /api/v1/student/assignments/due` | Entregas próximas. Su fallo se informa aparte. |
| `GET /api/v1/student/sessions/active` | Aulas abiertas del alumno, con materia, práctica y presencia. |

El resumen de materias **no** trae `active_environment`: ese campo se retiró del contrato al pasar al modelo por práctica.

---

## 4. Reglas de interfaz

1. **Un solo camino al entorno.** El inicio no abre contenedores; enlaza al aula o a la práctica.
2. **Fechas.** Solo con `dateText` dentro de `<time>`. Nunca `toLocale*` ni `| date`.
3. **Datos técnicos.** Códigos de materia, conteos y porcentajes se marcan con `[machineData]` o `<code>`; la prosa y las etiquetas quedan fuera del nodo mono.
4. **Estados.** `available`/error/empty se distinguen: un error nunca se dibuja como lista vacía.
5. **Sin personalización.** La vista es fija; se retiraron el bento con drag & drop, el selector de ancho y la persistencia de disposición en `localStorage`.

---

## 5. Tokens y movimiento

- Tipografía y tamaños con `var(--font-sans)`, `var(--text-*)`; nada de familias o `font-size` literales.
- Superficie `var(--bg-surface)`, borde `var(--border-subtle)`, radio `var(--radius-lg)`.
- Estado del aula: verde para abierta, ámbar para congelada, neutro para cerrada.
- Barra de avance con `var(--tenant-primary)` sobre `var(--bg-surface-active)`.
- Movimiento: tarjetas con primitiva global `.motion-rise` y escalonamiento `.motion-stagger` vía `[style.--motion-index]`. Sin `@keyframes` locales.
- En pantallas menores a 960px la grilla pasa a una columna.

---

## 6. Verificación

- `student-dashboard.component.spec.ts`: saludo y resumen, aula con práctica/horario/presencia, más de una clase, sin clase con fallo de entregas, entregas con fecha y enlace, avance por materia y estados vacíos.
- `npm run lint:styles`: gates de estilos, fechas, datos técnicos y movimiento.
- `ng build --configuration development`.

---

## 7. Flujo de primer ingreso

1. Sin materias: "No tienes materias matriculadas." en *Tus materias*, y el aula ofrece `[ Ver mis prácticas ]`.
2. Con materias y sin clase: el inicio muestra las entregas y el avance; el aula se anuncia cuando el docente la abre.
3. Al abrirse el aula, *Aula ahora* lleva directo a la práctica de la sesión; el ítem **Aula** del sidebar sigue disponible para ver presencia y avances.
