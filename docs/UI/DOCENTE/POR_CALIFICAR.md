# Vista: Por calificar

> **Especificación de interfaz y contrato de datos**
> **Rol:** Docente
> **Gobernanza:** SOLV Design System / [UX-PRINCIPLES](../UX-PRINCIPLES.md) / [ADR-053](../../ARQUITECTURA/ADR/ADR-053-revision-docente-y-libreta-de-notas.md)
> **Ruta:** `/teacher/por-calificar`

---

## 1. Qué responde

**"¿Qué trabajo manual me falta?"** Es la cola de corrección del docente, agrupada por práctica y ordenada por antigüedad. No muestra resultados del juez: muestra entregas que esperan una decisión humana.

La navegación la ubica en `DOCENCIA`, entre el panel y las evaluaciones.

---

## 2. Estructura

```
Por calificar                                  [Actualizar]
Lo que espera tu corrección, agrupado por práctica. 12 entregas en espera
────────────────────────────────────────────────────────────────────────
 Programación I                    │  Programación I
 LISTA ENLAZADA                    │  Lista enlazada            [Abrir práctica]
 7 de 12 · desde 28/10 10:00       │  ─────────────────────────────────────────
 Estructuras de Datos              │  Ana Quispe     29/10 12:00   [Calificar →]
 ORDENAMIENTO                      │  Luis Mamani    28/10 08:00   [Calificar →]
 1 de 8 · desde 29/10 09:00        │
```

- **Panel izquierdo:** una fila por práctica con materia, título, `pendientes de inscritos` y la entrega más antigua. La práctica seleccionada se marca con borde izquierdo del color del tenant.
- **Panel derecho:** las entregas pendientes de la práctica, de la más antigua a la más nueva, con el alumno y la fecha de entrega. La acción abre el SpeedGrader existente en `/teacher/revision/:submissionId`.
- El total de entregas en espera se muestra en el subtítulo y es la suma de todas las prácticas.

---

## 3. Estados

| Estado | Presentación |
| --- | --- |
| Cargando | `skeleton-loader` de lista. |
| Sin pendientes | Estado vacío: "No hay entregas por calificar" y una frase que explica cuándo aparecerán. |
| Error de red | Panel con el mensaje y botón "Reintentar". |
| Práctica seleccionada sin entregas | "Sin entregas en esta práctica", con la nota de que otro docente pudo corregirlas. |
| Sin práctica seleccionada | Marcador con icono e instrucción de elegir una práctica. |

---

## 4. Datos y refresco

| Dato | Origen |
| --- | --- |
| Prácticas pendientes | `GET /api/v1/teacher/reviews` |
| Entregas de la práctica | `GET /api/v1/teacher/courses/{id}/submissions?exercise_id=…`, filtrado en cliente por `review_status ∈ {pending, in_review}` |

No hay refresco automático: la cola cambia cuando el docente actúa. "Actualizar" recarga y conserva la práctica seleccionada si sigue teniendo pendientes.

---

## 5. Reglas visuales

- Un solo color de acento (el del tenant) y los tokens semánticos de estado; sin sombras ni gradientes.
- Fechas con el pipe `dateText` dentro de `<time>`; conteos con `[machineData]`.
- Sin modales: el detalle de corrección vive en la ruta del SpeedGrader, no encima de la lista.
- A 1024 px o menos, los paneles se apilan en una columna.

---

## 6. Fuera de alcance

- Calificar dentro de esta vista (se delega en el SpeedGrader existente).
- Métricas del curso o promedios: eso pertenece a Notas y a la materia.
- Entregas de juez que no requieren corrección: por definición no aparecen.
