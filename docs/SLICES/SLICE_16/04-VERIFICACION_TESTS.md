# Verificación de Pruebas — SLICE 16: Backups y Retención Institucional

## 1. Estado del Slice
- **Estado General:** Planificado (Pendiente de implementación en backend e infraestructura).

## 2. Plan de Pruebas a Ejecutar tras Implementación
1. **Volcado y Consistencia:** Ejecución de snapshot `pg_dump`, comprobación de integridad y generación de checksum SHA-256.
2. **Rotación de Retención:** Simulación de 10 ejecuciones y verificación de que se eliminen los archivos que superen los `BACKUP_RETENTION_DAYS`.
3. **Restauración Controlada en Sandbox:** Prueba de restauración en base de datos efímera para verificar que el backup no esté corrupto.
