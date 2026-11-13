# ADR 076: Despublicar una Práctica Publicada

> **Estado:** Implementado
> **Fecha:** 2026-11-13
> **Ámbito:** Docencia, Ejercicios
> **Relacionado:** ADR-075 (reinicio de la base por alumno)

---

## Contexto

Publicar era una acción de una sola dirección. Una práctica publicada no podía ocultarse sin
borrarla, y el docente no tenía forma de retirar una configuración equivocada sin dejar la práctica
visible para los alumnos.

## Decisión

Se agrega `POST /api/v1/exercises/{id}/unpublish`, disponible para docentes y administradores del
tenant. Devuelve el estado a `draft` y la práctica deja de estar disponible para los estudiantes.

- No borra contenido, rúbrica, entregas ni las bases que los alumnos ya abrieron.
- La página de la práctica muestra "Volver a borrador" solo cuando está publicada.
- La acción pide confirmación explícita y avisa qué se conserva.

## Consecuencias

- El docente puede retirar una práctica de circulación sin perder trabajo ni historial.
- Los alumnos dejan de verla de inmediato; sus entregas y entornos siguen existiendo.
- Volver a publicar usa el mismo camino de siempre y no requiere migración.

## Alternativas descartadas

- **Pausar en lugar de despublicar.** Requería un tercer estado y reglas propias de visibilidad; el
  borrador ya representa "no visible para alumnos".
- **Borrar y recrear la práctica.** Pierde entregas, estadísticas y bases abiertas.

## Verificación

- Tests de servicio: vuelve a `draft`, conserva el contenido y devuelve no encontrado para un
  ejercicio inexistente.
- `go build ./...`, `go vet ./...` y build de frontend.
