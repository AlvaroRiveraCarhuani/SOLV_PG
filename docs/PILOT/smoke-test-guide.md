# Guía de Smoke Test para Piloto

## Objetivo
Validar que el sistema SOLV esté listo para el piloto con usuarios reales, cubriendo ambos modos de ejecución (JUEZ_EFIMERO e IDE_PERSISTENTE).

## Prerrequisitos
- Sistema desplegado en entorno de staging
- Usuario docente con permisos de creación
- Usuario estudiante matriculado en curso de prueba
- Plantillas aprobadas en el catálogo (al menos 1 para JUEZ, 1 para IDE)

## Caso 1: JUEZ_EFIMERO (Algoritmos)
1. **Crear ejercicio:**
   - Ingresar como docente
   - Crear ejercicio "Suma de dos números" con modalidad "Juez Virtual"
   - Agregar 3 casos de prueba (1 ejemplo, 1 público, 1 oculto)
   - Definir solución de referencia en Python
   - Publicar ejercicio

2. **Resolver como estudiante:**
   - Ingresar como estudiante
   - Abrir ejercicio, leer enunciado y casos de ejemplo
   - Escribir solución en Python
   - Enviar código
   - Verificar veredicto AC

3. **Verificar en SpeedGrader:**
   - Ingresar como docente
   - Abrir SpeedGrader para el ejercicio
   - Verificar que la submission aparece con veredicto AC
   - Verificar métricas (tiempo, memoria)

## Caso 2: IDE_PERSISTENTE (Desarrollo Web)
1. **Crear ejercicio:**
   - Ingresar como docente
   - Crear ejercicio "API REST con FastAPI" con modalidad "Workspace Interactivo"
   - Seleccionar plantilla aprobada "Python Web + PostgreSQL"
   - Definir rúbrica con 2 criterios (Endpoints REST 50%, Documentación 50%)
   - Configurar fecha límite
   - Publicar ejercicio

2. **Resolver como estudiante:**
   - Ingresar como estudiante
   - Abrir ejercicio, ver rúbrica de evaluación
   - Iniciar workspace (OpenVSCode Server)
   - Escribir código de la API
   - Entregar snapshot

3. **Calificar con rúbrica:**
   - Ingresar como docente
   - Abrir SpeedGrader para el ejercicio
   - Calificar cada criterio de la rúbrica
   - Verificar que el score se calcula correctamente (ej: Excelente + Bueno = 87.5)
   - Agregar comentarios

## Checklist Pre-Piloto
- [ ] Todos los endpoints responden correctamente
- [ ] Migraciones 00001-00021 aplicadas sin errores
- [ ] Plantillas aprobadas visibles en el catálogo
- [ ] Guards de publicación funcionan para ambos modos
- [ ] Smoke test de JUEZ_EFIMERO pasa
- [ ] Smoke test de IDE_PERSISTENTE pasa
- [ ] Métricas Prometheus están siendo colectadas
- [ ] Logs estructurados con trace_id funcionan
- [ ] Backup diario configurado (cron job)
- [ ] Runbook de incidentes documentado

## Procedimiento de Rollback
Si el piloto falla críticamente:
1. Detener el backend: `systemctl stop solv-api`
2. Restaurar desde backup: `./scripts/restore.sh /var/backups/solv/latest.dump`
3. Reiniciar backend: `systemctl start solv-api`
4. Notificar al equipo de soporte

## Contacto de Soporte
- **Equipo técnico:** devops@solv.edu
- **Respuesta a incidentes:** 24/7 on-call
