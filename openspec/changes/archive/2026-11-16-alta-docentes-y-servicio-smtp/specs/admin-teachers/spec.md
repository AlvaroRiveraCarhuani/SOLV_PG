# admin-teachers Specification

## Purpose
Allow administrative staff to provision teachers directly (individually or in bulk via tolerant spreadsheet upload) and trigger optional onboarding notifications, eliminating the friction of manual token delivery.

## Requirements

### Requirement: Direct Teacher Provisioning (Alta Directa Individual)
The system MUST allow an administrator to register a teacher directly by providing their institutional email, creating or promoting their user record in the `users` table with `role = 'teacher'`. First name and last name fields MUST be optional, as Google Workspace SSO will refresh identity details upon first login.

#### Scenario: Successful direct registration of a new teacher
- GIVEN an authenticated administrator on `/admin/docentes`
- WHEN submitting an institutional email `nuevo.profesor@uab.edu.bo` without previous user account
- THEN a user record is created in `users` with `role = 'teacher'` and active status
- AND a success toast is shown to the administrator
- AND the teacher list reflects the new active instructor.

#### Scenario: Direct promotion of an existing student to teacher
- GIVEN an existing user with `role = 'student'` and email `docente.asistente@uab.edu.bo`
- WHEN the administrator registers `docente.asistente@uab.edu.bo` via direct provisioning
- THEN the existing user role is updated to `teacher`
- AND the operation succeeds without duplicate key errors.

#### Scenario: Re-registering an already active teacher is idempotent
- GIVEN a user already having `role = 'teacher'`
- WHEN the administrator attempts direct provisioning with that same email
- THEN the system confirms the active teacher status without raising a fatal error.

---

### Requirement: Tolerant Bulk Teacher Import (Carga Masiva)
The system MUST parse `.xlsx` and `.csv` files flexibly, identifying institutional email columns using fuzzy header matching (`correo`, `email`, `mail`, `dirección`) or scanning cell values with RFC 5322 regex heuristics when headers are irregular or absent.

#### Scenario: Spreadsheet with non-standard header names
- GIVEN an uploaded Excel file where the email column is titled "Correo Institucional del Docente"
- WHEN the file parser processes the sheets
- THEN the system identifies the column as the primary email source
- AND extracts all unique email addresses found in data rows.

#### Scenario: Spreadsheet without recognized headers (regex scan fallback)
- GIVEN an uploaded file with arbitrary column headers
- WHEN header keyword matching finds no match
- THEN the parser scans row cells using email format regex heuristics
- AND aggregates all valid institutional email candidates.

---

### Requirement: Bulk Import Validation and Preview Modal
The system MUST present an interactive Preview step in the UI before committing bulk changes, displaying metrics for total detected emails, valid candidates, duplicate rows, and invalid format items, with an explicit "Confirmar Alta Masiva" action.

#### Scenario: Preview presentation before persistence
- GIVEN a parsed file containing 20 total emails (18 valid, 1 malformed, 1 duplicate within file)
- WHEN the parser completes client-side staging
- THEN the preview modal displays "18 válidos", "1 duplicado", "1 inválido"
- AND the administrator can inspect the sanitized list before committing.

#### Scenario: Execution of confirmed bulk provisioning
- GIVEN a validated list of 18 emails in the preview modal
- WHEN the administrator clicks "Confirmar Alta Masiva"
- THEN the backend inserts or updates the 18 users with `role = 'teacher'` in a single batch
- AND the UI notifies the administrator of the number of provisioned teachers.

---

### Requirement: Notification Dispatch Option
The teacher provisioning modal MUST provide an optional toggle "Notificar a los docentes por correo institucional" (enabled by default). When checked and SMTP is configured, an onboarding notification MUST be dispatched asynchronously.

#### Scenario: Provisioning with notification enabled
- GIVEN an administrator provisioning one or more teachers with the notification toggle active
- WHEN the provisioning completes
- THEN the backend schedules welcome notification jobs to the SMTP dispatch queue
- AND the HTTP response returns immediately without blocking on mail delivery.

#### Scenario: Provisioning with notification disabled
- GIVEN an administrator provisioning teachers with the notification toggle unchecked
- WHEN the provisioning completes
- THEN the users are registered as teachers
- AND no mail jobs are enqueued to the SMTP service.
