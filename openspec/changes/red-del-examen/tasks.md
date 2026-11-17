# Tasks: Política de red del examen

## Phase 1: Dominio y validación

- [x] 1.1 Agregar `exam_network` (`none`|`egress`) y `exam_network_reason` al ejercicio, con `none` por defecto.
- [x] 1.2 Validar que `egress` exija motivo y que solo se aplique a ejercicios de examen.
- [x] 1.3 Pruebas unitarias de la validación y del valor por defecto.

## Phase 2: Red interna del examen

- [x] 2.1 Definir la red interna `solv-exam-net` sin puerta de enlace y conectar Traefik a ella. Traefik no entra a la red interna: el editor se publica con un proxy (`solv-exam-proxy`) que vive en las dos redes, porque Docker no permite publicar puertos de un contenedor que solo está en una red interna.
- [x] 2.2 Crear el contenedor de examen en esa red cuando la política sea `none`, conservando el puerto del editor.
- [x] 2.3 Adjuntar el satélite por su red interna cuando el examen lo declare (`solv-exam-satellites-net`).
- [x] 2.4 Agregar la regla de defensa en `DOCKER-USER` derivada de la etiqueta del contenedor (`solv.exam.no-egress=true`).
- [x] 2.5 Pruebas de integración: editor alcanzable, dominio externo no resuelto, satélite alcanzable.

## Phase 3: Verificación al publicar

- [x] 3.1 Implementar la verificación de compilación y ejecución del proyecto base en un contenedor sin salida (`VerifyExamProjectOffline`: volumen temporal, red interna, setup y arranque declarado, corte a los 5 minutos).
- [x] 3.2 Rechazar la publicación con el paso que necesitó red, indicando comando o dependencia (`422 EXAM_OFFLINE_VERIFICATION_FAILED` con el paso y las últimas líneas del log).
- [x] 3.3 Pruebas con un proyecto que funciona sin red y otro que necesita una instalación remota.

## Phase 4: Interfaz y auditoría

- [x] 4.1 Mostrar al alumno que el examen no tiene acceso a internet, o que lo tiene por excepción.
- [x] 4.2 Registrar la excepción en `audit_logs` con docente, ejercicio y motivo (`EXAM_NETWORK_EGRESS_GRANTED`).
- [x] 4.3 Mostrar el estado de la verificación en la pantalla de publicación del docente (aviso en el checklist con `passed`, `failed`, `skipped` o sin verificar).
- [x] 4.4 Pruebas de componente para los dos avisos.

## Phase 5: Verificación y documentación

- [ ] 5.1 Corrida manual en el nodo: abrir, compilar y entregar un examen sin salida.
- [ ] 5.2 Corrida manual con la excepción activa y su registro de auditoría.
- [x] 5.3 Escribir el ADR, su entrada en `INDEX.md` y cerrar la decisión abierta en `separar-practica-y-examen` (ADR-085).
- [ ] 5.4 Ejecutar el plan de impacto de `tools/test-impact` sobre las suites afectadas.
