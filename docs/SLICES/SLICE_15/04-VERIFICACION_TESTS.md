# Verificación de Pruebas — SLICE 15: Notificaciones Proactivas (In-App & Email)

## 1. Estado del Slice
- **Estado General:** Planificado (Pendiente de implementación en backend y frontend).

## 2. Plan de Pruebas a Ejecutar tras Implementación
1. **Persistencia In-App:** Emisión de evento, guardado en tabla `notifications` y consulta por el usuario destinatario.
2. **Aislamiento Multi-Tenant:** Validación de que los usuarios no puedan leer notificaciones de otros tenants.
3. **Cálculo de Conteo No Leídas:** Verificación de decremento atómico al marcar elementos como leídos.
4. **Anti-Fatiga de Correo:** Comprobación de descarte o consolidación tras superar el límite de envíos por hora.
