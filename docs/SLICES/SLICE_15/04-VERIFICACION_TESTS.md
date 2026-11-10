# Verificación de Pruebas — SLICE 15: Notificaciones Proactivas (In-App & Email)

## 1. Estado del Slice
- **Estado General:** In-app implementado y verificado; correo pendiente.

## 2. Pruebas ejecutadas
1. **Persistencia In-App:** emisión, guardado y consulta por el destinatario (`TestSlice15_Notifications_CompleteSuite`).
2. **Aislamiento Multi-Tenant:** un usuario no puede marcar como leída una notificación ajena (404).
3. **Conteo de no leídas:** decremento al marcar una o todas como leídas.
4. **Filtros y consolidación:** severidad, tipo de evento, no leídos, agrupación de fallas de respaldo y saneo de enlaces históricos (`TestSlice15_Notifications_FiltersAndConsolidation`).
5. **Contrato del servicio:** validaciones, consolidación opt-in y saneo de enlaces (`notification_service_test.go`).
6. **Handler HTTP:** filtros inválidos (400) e id inválido (400) (`notification_handler_test.go`).
7. **Recordatorio de vencimiento:** emisión in-app junto al correo (`deadline_reminder_worker_test.go`).
8. **Frontend:** servicio, campana y página de avisos (11 specs con Vitest).

## 3. Pendiente de verificar
- Envío de correo desde el despachador y preferencias por usuario.
