# Propuesta: Servicios Satélite por Rol, Pin del Entorno y Contratos de Capacidad y Aislamiento

## Intención

Habilitar combinaciones legítimas como «PostgreSQL + Redis» sin convertir la Columna 3 en una multi-selección libre que permita MySQL + PostgreSQL. El cambio no es de interfaz: reemplaza el contrato de un solo motor por bindings con rol, hace explícitos la capacidad del nodo, la reproducibilidad del entorno de un ejercicio y el aislamiento entre alumnos.

La consulta original («¿por qué no puedo elegir Redis y después PostgreSQL?») destapó cinco brechas que este cambio cierra:

1. Un solo motor en Studio, payload, compositor, aprovisionamiento y scale-to-zero.
2. RAM de satélites compartidos contada en la cuota de cada alumno, con tres números distintos para el mismo motor.
3. Ejercicios publicados que cambian de imagen, RAM y base cuando se edita la plantilla.
4. Aislamiento por convención: superusuario de PostgreSQL, MongoDB sin auth y Redis con prefijo en variable de entorno.
5. Catálogo que publica motores cuya imagen y límites reales están hardcodeados en el aprovisionador.

## Alcance

### Dentro del Alcance

- Bindings con rol (`primary_database`, `cache`), cardinalidad y validación server-side.
- Catálogo con `service_role` y adaptadores de motor como única fuente de imagen, límites, readiness, aislamiento e init.
- Aprovisionamiento de N bindings, entorno combinado, readiness de todos, init por motor y scale-to-zero por conjunto.
- Pin del entorno al publicar (`environment_snapshot`) y lectura en el arranque, con fallback para ejercicios anteriores.
- Capacidad: `expected_mb` / `reserve_mb`, reserva del nodo y fórmula de admisión sin satélites en la cuota del alumno.
- Aislamiento por scope con tenant en PostgreSQL, MySQL/MariaDB, MongoDB y Redis.
- Studio con «base principal» y «auxiliares»; copiloto y modal docente con el mismo contrato.

### Fuera del Alcance

- Juez efímero con servicios (sigue aislado y sin red).
- Publicación multi-nodo y colocación de satélites por nodo.
- Versionado completo de plantillas.
- Manifiesto `.solv/services.json` dentro del workspace.
- Identidad versionada del satélite (fase 7, diferida).

## Capacidades

### Nuevas Capacidades

- `node-service-bindings`: contrato de bindings por rol, resolución, aprovisionamiento de N motores, pin del entorno, capacidad del nodo y aislamiento por scope.

### Capacidades Modificadas

- `template-environment-studio`: la Columna 3 pasa a «Datos y servicios» con base principal y auxiliares.
- `admin-templates`: validación de bindings en alta y edición; capacidades del nodo con rol, reserva e init.
- `template-request-modal` y copiloto: mismo contrato de servicios.

## Enfoque

Primero el contrato y la validación (fase 1), después la capacidad y el pin (fases 2 y 3), luego los adaptadores y el pipeline de N bindings (fase 4), el aislamiento (fase 5) y por último la interfaz (fase 6). La fase 2 no puede quedar después de la 4 porque aprovisionar dos motores con la cuenta de memoria actual empeora la admisión; la fase 5 no puede quedar después de la 6 porque no se habilita Redis auxiliar sin ACL.

## Áreas Afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `backend/internal/core/domain` | Modificado | Roles, bindings, invariantes, init, capacidad |
| `backend/internal/core/services/workspace_*` | Modificado | Resolución y aprovisionamiento de N bindings |
| `backend/internal/core/services/template_composer_service.go` | Modificado | Composición con varios servicios |
| `backend/internal/core/services/admin_governance_*` | Modificado | Validación de bindings, piso vital y capacidades |
| `backend/internal/infrastructure/docker` | Modificado | Adaptadores de motor, aislamiento y readiness |
| `backend/migrations` | Nuevo | `template_components.service_role`, `exercises.environment_snapshot` |
| `frontend/src/app/features/admin/templates/studio` | Modificado | Base principal + auxiliares, payload y memoria |
| `frontend/src/app/features/teacher/templates` | Modificado | Mismo contrato de servicios |
| `frontend/src/app/features/admin/services` | Modificado | Copiloto con `services` |
| `docs/ARQUITECTURA/ADR` | Nuevo / Modificado | ADR-050 y nota de reemplazo en ADR-046 §7 |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Cambio de credenciales invalida laboratorios existentes | Alta | Reaprovisionamiento en el siguiente arranque, prueba con un laboratorio real y rollback por fase |
| Migración de identidad de satélite rompe volúmenes | Media | Fase 7 separada, con nombres nuevos y copia/renombrado verificado antes de borrar |
| Dos motores elevan el consumo del nodo | Media | Reserva explícita y rechazo temprano en la admisión |
| Aislamiento de Redis/Mongo deja datos alcanzables | Alta | ACL y auth antes de habilitar el auxiliar; prueba de dos alumnos en integración |
| El pin no cubre ejercicios viejos | Media | Fallback a plantilla viva con log y plan de migración de publicaciones |
| La UI habilita combinaciones que el backend rechaza | Baja | La validación de dominio es la misma que consume el Studio |

## Rollback

- Migraciones aditivas y nullable; no se borran datos.
- Cada fase se puede revertir sin tocar las anteriores.
- La fase 5 debe desplegarse con un laboratorio de prueba y un plan de reescritura de credenciales; revertirla exige volver a la derivación anterior y reiniciar los laboratorios afectados.
- La fase 6 se puede ocultar con una bandera de interfaz sin desactivar el contrato.

## Criterios de Aceptación

- [ ] Una plantilla con PostgreSQL + Redis se crea, se publica y arranca con `DATABASE_URL` y `REDIS_URL`.
- [ ] Una plantilla con MySQL + PostgreSQL se rechaza con `422 multiple_primary_database`.
- [ ] Un ejercicio publicado conserva su entorno después de editar la plantilla.
- [ ] La capacidad del nodo descuenta la reserva del satélite una vez, no por alumno.
- [ ] Un alumno no puede conectarse a la base de otro ni ejecutar `FLUSHALL` en Redis.
- [ ] El scale-to-zero no apaga un motor que otro laboratorio sigue usando.
- [ ] El Studio, el copiloto y el modal docente hablan el mismo contrato.
- [ ] `go test ./...`, `npm run lint:styles` y `CI=true npm run build` pasan en cada fase.
