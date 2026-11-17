# Tasks: Dimensionamiento medido de RAM y CPU del Entorno de Desarrollo

## Phase 1: Medición del nodo piloto

- [x] 1.1 Registrar el contexto del nodo en `medicion.md` §0: núcleos, RAM total, swap, versión de Docker y `MemAvailable` con la plataforma levantada y sin entornos.
- [x] 1.2 Ejecutar M1 (curva de arranque del IDE) y M2 (régimen de 30 min) con tres corridas cada uno.
- [ ] 1.3 Ejecutar M3 (`npm install` y `pip install`) y M4 (proyecto de un archivo vs proyecto completo).
- [x] 1.4 Ejecutar M5 (10, 20 y 30 entornos, en oleadas y simultáneos) registrando `free -m`, `MemAvailable` y los 503 con su error.
- [ ] 1.5 Recoger M6 (OOM reales del historial) y M7 (swap durante M5).
- [ ] 1.6 Ejecutar C1 a C5: CPU por entorno compilando, curva de contención y latencia de la API durante la oleada.
- [ ] 1.7 Ejecutar R1 y R2: los tres números de cuota de un mismo entorno y frecuencia de cuota ajustada.
- [ ] 1.8 Cerrar la planilla con p50/p95 por escenario y marcar lo que no se pudo medir.

## Phase 2: Decisión de números

- [ ] 2.1 Calcular la base de memoria con la regla de `medicion.md` y anotar el perfil elegido.
- [ ] 2.2 Elegir el escalón de CPU que mantiene la compilación dentro de 1.5× del tiempo sin límite.
- [ ] 2.3 Revisar el techo de seguridad y los perfiles del préstamo con la curva de M5.
- [x] 2.4 Escribir el ADR "Límite de CPU por entorno y dimensionamiento medido" con la planilla adjunta y la entrada en `INDEX.md`.

## Phase 3: Política y dominio

- [x] 3.1 Agregar `PracticeCPUMillis` a `DefaultServerPolicies()` y a la validación de políticas, con `0` como sin límite.
- [x] 3.2 No hace falta columna: la política se guarda en el config del tenant (`server_policies`), así que `practice_cpu_millis` viaja en ese JSON y se completa con el default al leer una política vieja.
- [ ] 3.3 Ajustar `loanQuotaTiersMB` y la base con los valores decididos en la fase 2, en un solo lugar.
- [x] 3.4 Pruebas unitarias: valor por defecto, valor en 0, valor ausente y validación de rango.

## Phase 4: Aplicación en el contenedor

- [x] 4.1 Resolver `cpu_millis` en el servicio y pasarlo en el config del contenedor.
- [x] 4.2 Aplicar `NanoCPUs` y `PidsLimit` en `StartPracticeEnvironmentContainer`, sin constantes en infraestructura.
- [x] 4.3 Persistir `cpu_millis` junto a `memory_limit_mb` en el entorno (migración `00062`, `RecordResourceQuota` al crear, reactivar y congelar).
- [x] 4.4 Pruebas de integración con Docker local: `HostConfig.Memory`, `NanoCPUs` y `PidsLimit` del contenedor creado.

## Phase 5: Reconciliación en la interfaz

- [x] 5.1 Exponer `memory_profile_mb` y `cpu_millis` en el DTO del entorno.
- [x] 5.2 Mostrar los tres valores con nombre en la ficha del entorno y en la vista del docente. La ficha del alumno los muestra al abrir el entorno; la vista del docente los recibe al abrir el IDE de la práctica (`memory_limit_mb`, `memory_profile_mb`, `cpu_millis`).
- [x] 5.3 Mostrar el aviso de cuota ajustada solo cuando la concedida difiere de la declarada.
- [ ] 5.4 Pruebas de componente para los dos casos: cuota igual y cuota menor.

## Phase 6: Verificación y documentación

- [ ] 6.1 Repetir M5 y C3 con el límite aplicado y comparar contra la medición sin límite.
- [ ] 6.2 Actualizar `docs/OPERATIONS/capacidad.md` §4 y §5 con la tabla medida y lo que sigue sin medir.
- [ ] 6.3 Ejecutar el plan de impacto de `tools/test-impact` sobre las suites afectadas y adjuntar el informe.
