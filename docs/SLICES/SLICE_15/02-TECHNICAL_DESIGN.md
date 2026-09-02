# Diseño Técnico — SLICE 15: Notificaciones Proactivas (In-App & Email)

## 1. Arquitectura de Despacho de Notificaciones

```mermaid
graph TD
    Trigger[Evento del Sistema / Workspace OOM / Entrega Calificada / Mantenimiento] --> NotifService[NotificationService]
    NotifService --> InAppWorker[In-App Storage Worker]
    NotifService --> EmailQueue[Email Dispatcher Queue]
    
    InAppWorker --> DB[(PostgreSQL 18 - notifications table)]
    EmailQueue --> SMTP[Email Provider / Resend o AWS SES]
    
    Client[Cliente Web / Angular 22] -->|GET /api/v1/notifications| NotifHandler[NotificationHandler]
    NotifHandler --> InAppWorker
```

## 2. Componentes del Sistema

### 2.1 Modelo de Notificaciones In-App
- Tabla `notifications` (`id`, `user_id`, `tenant_id`, `type`, `title`, `message`, `severity`, `read_at`, `created_at`).
- Severidades: `info`, `warning`, `error`, `critical`.
- Marcar como leída de forma individual o masiva (`PATCH /api/v1/notifications/{id}/read`).

### 2.2 Despachador de Correo Electrónico
- Envío asíncrono reservado para eventos de severidad `critical` o `warning` institucional (por ejemplo: anuncio de mantenimiento programado, fallo en backup, credencial de invitación).
- Reglas anti-fatiga: Limitación de tasa máxima (rate-limit) de correos por usuario para evitar saturación de bandejas de entrada.

## 3. Estado del Incremento
- **Backend:** Planificado.
- **Frontend:** Planificado.
