# Manual Operativo — SLICE 15: Notificaciones Proactivas (In-App & Email)

## 1. Endpoints del Sistema de Notificaciones

| Método | Ruta | Descripción | Rol Requerido |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/notifications` | Listar notificaciones del usuario con paginación | Autenticado |
| `GET` | `/api/v1/notifications/unread-count` | Conteo de notificaciones no leídas | Autenticado |
| `PATCH` | `/api/v1/notifications/{id}/read` | Marcar notificación específica como leída | Autenticado |
| `POST` | `/api/v1/notifications/mark-all-read` | Marcar todas las notificaciones como leídas | Autenticado |

## 2. Variables de Entorno del Despachador de Correo
```env
SMTP_HOST=smtp.resend.com
SMTP_PORT=587
SMTP_USER=resend
SMTP_PASSWORD=re_123456789
EMAIL_FROM="SOLV Notifications <notificaciones@solv.uab.edu.bo>"
```
