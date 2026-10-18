# Tasks: Alta Directa de Docentes, Importación Masiva Tolerante y Adaptador SMTP

## Phase 1: Servicio SMTP e Infraestructura de Correo
- [x] 1.1 Definir interface `Mailer` y estructura `EmailMessage` en `internal/core/domain/ports.go`.
- [x] 1.2 Implementar adaptador `SMTPMailer` en `internal/infrastructure/mailer/smtp_mailer.go` con soporte TLS/STARTTLS, fallback a logging local y despacho asíncrono con buffer.
- [x] 1.3 Crear plantilla HTML institucional para correos de bienvenida docente en `internal/infrastructure/mailer/templates.go`.
- [x] 1.4 Escribir pruebas unitarias para `SMTPMailer` en `internal/infrastructure/mailer/smtp_mailer_test.go` verificando modo mock y construcción MIME.

## Phase 2: Lógica de Alta Directa y Parser Tolerante de Planillas
- [x] 2.1 Implementar `RosterParser` en `internal/core/services/roster_parser.go` con lectura tolerante de CSV y escaneo heurístico de correos por regex.
- [x] 2.2 Agregar soporte para archivos `.xlsx` en `RosterParser`.
- [x] 2.3 Escribir pruebas unitarias para `RosterParser` con planillas desordenadas, cabeceras sinónimas y celdas variadas.
- [x] 2.4 Extender `TeacherService` en `internal/core/services/teacher_service.go` con `RegisterDirectTeacher`, `BatchRegisterTeachers` y `ParseRoster`.
- [x] 2.5 Actualizar repositorios de usuario en `internal/infrastructure/storage/postgres/` para admitir upsert atómico de docentes.

## Phase 3: Handlers HTTP y Rutas en Backend
- [x] 3.1 Implementar métodos en `TeacherHandler` (`internal/delivery/http/teacher_handler.go`):
  - `RegisterDirectTeacher` (`POST /api/v1/teachers/direct`)
  - `ParseRoster` (`POST /api/v1/teachers/parse-roster`)
  - `BatchRegisterTeachers` (`POST /api/v1/teachers/direct-batch`)
- [x] 3.2 Registrar las nuevas rutas en `internal/delivery/http/router.go` con middlewares de tenant y auditoría.
- [x] 3.3 Inicializar `Mailer` en `cmd/server/main.go` conectándolo a `TeacherHandler`.
- [x] 3.4 Verificar compilación backend con `go build ./...` y pruebas de integración.

## Phase 4: Modelos y Servicios en Frontend
- [x] 4.1 Actualizar interfaces en `frontend/src/app/core/models/admin.model.ts` con DTOs de previsualización y payloads de alta directa.
- [x] 4.2 Agregar métodos en `frontend/src/app/features/admin/services/admin-teachers.service.ts`:
  - `directRegisterTeacher`
  - `parseRosterFile`
  - `batchDirectRegisterTeachers`
- [x] 4.3 Escribir pruebas unitarias en `admin-teachers.service.spec.ts`.

## Phase 5: Interfaz de Usuario (Modal de Alta Directa y Carga Masiva)
- [x] 5.1 Rediseñar el modal `teacher-invite-modal` para ofrecer pestañas "Alta Directa" y "Carga Masiva".
- [x] 5.2 Implementar selector drag & drop de planillas con envío a previsualización.
- [x] 5.3 Implementar vista previa con badges de estado (correos válidos, duplicados, errores) y botón "Confirmar Alta Masiva".
- [x] 5.4 Agregar interruptor "Notificar por correo institucional" (activo por defecto).
- [x] 5.5 Conectar confirmaciones de guardado con feedback visual en `admin-teachers.component.ts`.

## Phase 6: Verificación y Control de Calidad
- [x] 6.1 Ejecutar `npm run lint:styles` para validar cumplimiento estricto del sistema de diseño (sin colores hex literales, uso de tokens).
- [x] 6.2 Ejecutar `npm run build` en frontend para garantizar compilación TypeScript sin errores.
- [x] 6.3 Ejecutar pruebas unitarias en backend y frontend.
- [x] 6.4 Validar flujo end-to-end de alta directa, previsualización de archivo e inicio de sesión por Google SSO.
