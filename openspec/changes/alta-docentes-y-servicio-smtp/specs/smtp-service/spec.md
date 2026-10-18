# smtp-service Specification

## Purpose
Provide a decoupled, non-blocking email delivery engine in the Go backend using standard SMTP (compatible with Google Workspace Relay, Brevo, Resend, or on-premise relays) with resilient local fallbacks and institutional HTML templates.

## Requirements

### Requirement: Standard SMTP Adapter Configuration
The backend MUST load SMTP parameters from standard environment variables (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`). When valid host and credentials are provided, the adapter MUST establish TLS/STARTTLS connections to dispatch transactional messages.

#### Scenario: Production Google Workspace Relay dispatch
- GIVEN environment variables configured with `SMTP_HOST=smtp-relay.gmail.com`, `SMTP_PORT=587`, `SMTP_USER=notificaciones@uab.edu.bo`
- WHEN a mail dispatch request is executed
- THEN the service connects via STARTTLS, authenticates, and transmits the MIME payload to the destination server.

#### Scenario: TLS port 465 SSL connection
- GIVEN `SMTP_PORT=465` with `SMTP_TLS=true`
- WHEN mail is dispatched
- THEN the service uses direct TLS dialer instead of plain handshake followed by STARTTLS.

---

### Requirement: Local Development Fallback
When `SMTP_HOST` is unset, empty, or set to `mock`/`stdout`, the SMTP service MUST NOT throw fatal runtime errors or panic. Instead, it MUST log the outgoing message (recipient, subject, text snippet) to standard output/logger for inspection during development and automated tests.

#### Scenario: Dispatch without configured SMTP credentials
- GIVEN a local development environment with `SMTP_HOST=""`
- WHEN the application attempts to send an onboarding email to `docente.test@uab.edu.bo`
- THEN the service logs `[SMTP Mock] Outgoing mail to docente.test@uab.edu.bo with subject: Bienvenido a SOLV`
- AND returns a successful job completion to the caller without network error.

---

### Requirement: Asynchronous Non-Blocking Mail Queue
Mail dispatch operations MUST execute asynchronously via worker goroutines and bounded Go channels. API endpoint handlers MUST NOT block HTTP response times waiting for SMTP handshake, network latency, or external rate limits.

#### Scenario: High concurrency teacher bulk provisioning
- GIVEN a batch of 50 teachers provisioned in a single request
- WHEN the notification trigger enqueues 50 emails
- THEN the API returns HTTP 201 immediately with the count of registered teachers
- AND the background worker pool consumes and dispatches messages at a controlled throughput.

---

### Requirement: Institutional HTML Email Template
Sent onboarding emails MUST render a semantic, branded HTML template including the institutional tenant name, SOLV system title, instructions to authenticate via Google Workspace SSO, and an explicit action button pointing to the frontend URL.

#### Scenario: Welcome email contains direct SSO link
- GIVEN a teacher `alvaro@uab.edu.bo` provisioned in tenant UAB
- WHEN the welcome email is compiled
- THEN the HTML body contains tenant branding ("Universidad Adventista de Bolivia"), instructions to click "Iniciar sesión con Google", and a button linking to `FRONTEND_URL/login`
- AND includes a plain-text alternative part in the MIME multipart structure.
