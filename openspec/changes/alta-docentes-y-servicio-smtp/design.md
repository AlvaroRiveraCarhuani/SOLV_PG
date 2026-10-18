# Design: Alta Directa de Docentes, Importación Masiva Tolerante y Adaptador SMTP

## Technical Approach

El diseño implementa el pre-aprovisionamiento de docentes en SOLV mediante dos capacidades complementarias:
1. **Pre-aprovisionamiento de Identidades (Alta Directa Individual y Masiva)**: Registro inmediato en la tabla `users` con `role = 'teacher'`. Al apoyarse en Google Workspace SSO, cuando el docente inicia sesión por primera vez, `auth_service.go` detecta su fila existente y le asigna su rol sin requerir canje de tokens transaccionales.
2. **Motor de Carga Masiva Tolerante (`parse-roster`)**: Endpoint de pre-procesamiento que admite archivos `.xlsx` y `.csv`. Aplica coincidencia difusa de cabeceras (`correo`, `email`, `mail`) y escaneo heurístico de celdas por expresiones regulares (RFC 5322), entregando un resumen estructurado (válidos, duplicados, malformados) para previsualización antes de la persistencia.
3. **Servicio SMTP Asíncrono Desacoplado**: Paquete de infraestructura en Go (`internal/infrastructure/mailer`) que implementa la interfaz `domain.Mailer`. Admite conexiones TLS/STARTTLS estándar (Google Workspace Relay, Brevo, Resend, servidores On-Premise) con fallback automático a logger en consola para desarrollo local.

---

## Architecture Decisions

### Decision 1: Pre-registro directo en tabla `users` vs Tokens transaccionales
**Choice**: Permitir creación y elevación directa en `users` con `role = 'teacher'` usando `ON CONFLICT (email) DO UPDATE SET role = 'teacher'`.
**Alternatives considered**: Obligar al administrador a generar tokens de 72 horas para cada docente o crear contraseñas locales.
**Rationale**: Al usar Google SSO institucional, las credenciales no se almacenan localmente. Si el usuario ya existe en `users` al autenticarse en Google, el sistema lee directamente sus privilegios sin enlaces temporales ni riesgo de expiración.

### Decision 2: Endpoint de Previsualización separado (`parse-roster` + `direct-batch`)
**Choice**: Dividir la carga masiva en dos pasos HTTP:
  - `POST /api/v1/teachers/parse-roster`: Analiza el archivo y devuelve candidatos validados, duplicados y errores sin persistir.
  - `POST /api/v1/teachers/direct-batch`: Persiste la nómina confirmada por el usuario.
**Alternatives considered**: Guardado directo al subir el archivo sin paso previo de validación.
**Rationale**: Evita altas accidentales o inserción de datos basura cuando el administrador sube una planilla con columnas incorrectas o correos repetidos.

### Decision 3: Adaptador SMTP nativo en Go con canal de trabajo en background
**Choice**: Implementar `SMTPMailer` usando la librería estándar `net/smtp` y `crypto/tls` con un canal en memoria con buffer (`chan *EmailMessage`) y worker goroutine.
**Alternatives considered**: Enviar correos de forma sincrónica en el handler HTTP o depender de SDKs cerrados de terceros (ej. AWS SDK o Resend SDK).
**Rationale**: La librería estándar de Go no agrega dependencias pesadas y es universalmente compatible (Google Workspace Relay, relays locales y proveedores cloud). El despacho asíncrono asegura que la respuesta HTTP sea instantánea (<50ms).

---

## Data Flow

### 1. Flujo de Alta Directa Individual
```
[Admin UI] 
   │ 
   ▼ POST /api/v1/teachers/direct { email, first_name, last_name, notify }
[TeacherHandler] ──→ [TeacherService.RegisterDirectTeacher]
                           │
                           ├──→ [UserRepository.UpsertTeacher] ──→ PostgreSQL (users)
                           │
                           └──→ (if notify) Mailer.SendAsync() ──→ Worker Channel ──→ SMTP Server
```

### 2. Flujo de Carga Masiva (Excel / CSV)
```
[Admin UI] 
   │ 
   ├── 1. Selecciona .xlsx o .csv
   ▼ POST /api/v1/teachers/parse-roster (Multipart File)
[TeacherHandler] ──→ [RosterParser.ExtractEmails]
   │                      ├── Fuzzy header matching ("correo", "email")
   │                      └── Fallback regex scanner
   ▼ HTTP 200 { valid: [...], duplicates: [...], invalid: [...] }
[Modal de Previsualización] (Muestra métricas y resumen de correos)
   │ 
   ├── 2. Admin presiona "Confirmar Alta Masiva"
   ▼ POST /api/v1/teachers/direct-batch { emails: [...], notify: true }
[TeacherHandler] ──→ [TeacherService.BatchRegisterTeachers]
                           │
                           ├──→ SQL Batch Upsert ──→ PostgreSQL (users)
                           │
                           └──→ (if notify) Mailer.SendAsyncBatch() ──→ SMTP Server
```

---

## File Changes

| File | Action | Description |
| :--- | :--- | :--- |
| `backend/internal/core/domain/ports.go` | Modify | Define `Mailer` interface y modelo `EmailMessage`. |
| `backend/internal/infrastructure/mailer/smtp_mailer.go` | Create | Implementa `Mailer` con `net/smtp`, STARTTLS/TLS, fallback mock y cola en background. |
| `backend/internal/infrastructure/mailer/templates.go` | Create | Plantilla HTML institucional para correos de bienvenida (branding UAB + SOLV). |
| `backend/internal/core/services/roster_parser.go` | Create | Parser tolerante para `.xlsx` y `.csv` con detección de cabeceras y regex scanning. |
| `backend/internal/core/services/teacher_service.go` | Modify | Agrega métodos `RegisterDirectTeacher`, `BatchRegisterTeachers` y `ParseRoster`. |
| `backend/internal/delivery/http/teacher_handler.go` | Modify | Expone endpoints `POST /api/v1/teachers/direct`, `POST /api/v1/teachers/parse-roster`, `POST /api/v1/teachers/direct-batch`. |
| `backend/internal/delivery/http/router.go` | Modify | Conecta los nuevos endpoints en el enrutador HTTP de Go. |
| `frontend/src/app/core/models/admin.model.ts` | Modify | Agrega DTOs de previsualización masiva y payloads de alta directa. |
| `frontend/src/app/features/admin/services/admin-teachers.service.ts` | Modify | Agrega métodos `directRegisterTeacher`, `parseRoster` y `batchDirectRegister`. |
| `frontend/src/app/features/admin/teachers/components/teacher-invite-modal/teacher-invite-modal.component.*` | Modify | Rediseño para permitir Alta Directa individual, pestaña de Carga Masiva y previsualización. |

---

## Interfaces / Contracts

### API Contracts (Backend HTTP)

#### `POST /api/v1/teachers/direct`
Payload:
```json
{
  "email": "profesor@uab.edu.bo",
  "first_name": "Carlos",
  "last_name": "Mendoza",
  "send_notification": true
}
```
Response `201 Created`:
```json
{
  "id": "uuid",
  "email": "profesor@uab.edu.bo",
  "role": "teacher",
  "status": "active",
  "notified": true,
  "message": "Docente registrado exitosamente"
}
```

#### `POST /api/v1/teachers/parse-roster`
Payload: `multipart/form-data` con campo `file` (`.xlsx` o `.csv`).
Response `200 OK`:
```json
{
  "total_detected": 15,
  "valid_emails": ["docente1@uab.edu.bo", "docente2@uab.edu.bo"],
  "duplicate_emails": ["docente1@uab.edu.bo"],
  "invalid_emails": ["correo-invalido"]
}
```

#### `POST /api/v1/teachers/direct-batch`
Payload:
```json
{
  "emails": ["docente1@uab.edu.bo", "docente2@uab.edu.bo"],
  "send_notification": true
}
```
Response `201 Created`:
```json
{
  "total_processed": 2,
  "registered_count": 2,
  "notified": true,
  "message": "2 docentes registrados exitosamente"
}
```
