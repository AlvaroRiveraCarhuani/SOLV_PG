# Feature: Notificaciones Transaccionales Academicas Multi-Tenant por Correo (Evaluaciones, Recordatorios a 12h y Bienvenida Docente)
Branch: main | Delivery strategy: ask-on-risk | Test runner: go test ./...

## Specs

- **S1**: **Plantillas de Correo Multi-Tenant y Marca Blanca (Backend Go)**:
  - En `backend/internal/infrastructure/mailer/templates.go`:
    - Definición de estructura `EmailBranding` (`InstitutionName`, `LogoURL`, `BaseDomain`, `SupportEmail`) en dominio.
    - Renderizador base reutilizable `renderEmailLayout`:
      - Encabezado con logo institucional dinámico (`https://{base_domain}{logo_url}`) y fallback en texto formal.
      - Eliminación estricta de toda mención a la marca interna "SOLV" en asuntos, títulos y pies de página.
      - Redacción formal institucional en español neutro (tratamiento de "usted").
      - Pie institucional limpio con referencia a los servicios académicos del tenant.
    - `BuildTeacherWelcomeEmail(branding, teacherName, loginURL)`:
      - Reformulación del correo existente a tono formal, eliminando tuteo y referencias a SOLV.
    - `BuildGradeFeedbackEmail(branding, studentName, assignmentTitle, subjectName, reviewURL)`:
      - Aviso de publicación de corrección y retroalimentación pedagógica (sin exponer nota cruda).
    - `BuildDeadlineReminderEmail(branding, studentName, assignmentTitle, subjectName, formattedDueDate, hasStartedWorkspace, workspaceURL)`:
      - Aviso urgente a 12 horas con bifurcación si el estudiante tiene borrador en workspace o no ha iniciado.
  - Tests unitarios en `backend/internal/infrastructure/mailer/templates_test.go` verificando ausencia de marca interna, escape HTML y renderizado de texto plano.

- **S2**: **Disparador de Notificación de Evaluación en Revisión Docente (Core/Services Go)**:
  - En `TeacherService.OverrideSubmission` y `RubricService.EvaluateSubmission`:
    - Al asentar la calificación o retroalimentación:
      - Obtener correo, nombre del estudiante y datos del ejercicio/materia.
      - Resolver metadatos de branding del tenant.
      - Si el mailer está disponible (`s.mailer != nil`), encolar asíncronamente con `s.mailer.SendAsync(...)`.
      - Operación desacoplada: no interrumpe el guardado de la nota ni genera fallas ante problemas SMTP.
  - Tests unitarios verificando el despacho al canal asíncrono.

- **S3**: **Worker Programado de Recordatorios de Vencimiento a 12 Horas con Deduplicación (Backend Go)**:
  - Migración `00029_deadline_reminders.sql` con tabla `deadline_reminders_sent` y restricción única `(tenant_id, exercise_id, student_id)`.
  - En `PostgresDeadlineReminderRepository`:
    - Búsqueda de ejercicios publicados con fecha de vencimiento dentro de la ventana de urgencia (~12 horas).
    - Detección de estudiantes inscritos sin entrega registrada.
    - Comprobación de si existe workspace iniciado para personalizar el aviso.
  - En `DeadlineReminderWorker`:
    - Tarea periódica de escaneo en segundo plano con ticker de 15 minutos.
    - Control de idempotencia y deduplicación.
    - Despacho asíncrono vía `mailer.SendAsync`.
  - Tests unitarios del worker con reloj y repositorios simulados.

- **S4**: **Verificación de Calidad y Tests**:
  - Ejecución de `go test ./internal/...` en `backend/`.
  - Verificación de no-bloqueo y funcionamiento en modo mock.

## Tasks

- [x] **T1** (S1): Implementar `EmailBranding`, layout base multi-tenant y plantillas (`BuildTeacherWelcomeEmail`, `BuildGradeFeedbackEmail`, `BuildDeadlineReminderEmail`) con sus pruebas unitarias en `templates_test.go`.
- [x] **T2** (S2): Integrar el despacho asíncrono de notificación de evaluación en `TeacherService.OverrideSubmission` y `RubricService` con sus pruebas unitarias.
- [x] **T3** (S3): Implementar el repositorio y worker de recordatorios de vencimiento a 12 horas con control de idempotencia y pruebas unitarias.
- [x] **T4** (S4): Ejecutar la verificación completa de pruebas en Go y confirmar que todos los tests pasen limpiamente.
