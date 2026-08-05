# ADR-025: Invitaciones a Docentes mediante Transacciones Atómicas SQL

* **Estado:** Aceptado
* **Fecha:** 2026-08-04
* **Slice:** 9

## Contexto y Problema
El escalamiento de usuarios al rol `teacher` requiere un mecanismo seguro que impida la elevación no autorizada de privilegios, garantice la validez del correo institucional y prevenga ataques de reuso de tokens de invitación.

## Decisión Tomada
1. Generar tokens criptográficos aleatorios de un solo uso asociados al correo del docente (`teacher_invitations`).
2. Implementar la operación de aceptación dentro de una transacción atómica SQL (`AcceptInvitationTx`):
   - Validar que el token existe, no ha sido usado (`used = FALSE`) y no está expirado.
   - Validar que el email del usuario logueado en sesión coincide exactamente con el email de la invitación.
   - Actualizar el rol del usuario a `teacher`.
   - Marcar el token como usado (`used = TRUE`).

## Consecuencias
* **Positivas:**
  * Prevención total de race conditions y elevaciones de privilegio mediante bloqueos de fila (`FOR UPDATE`) en la transacción.
  * Trazabilidad completa de asignación de roles.
