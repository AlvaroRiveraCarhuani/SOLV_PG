# Diseño Técnico - SLICE 1: Base del Sistema y Estructura Hexagonal

## 1. Arquitectura de Capas

```
[Delivery (HTTP/Handlers)] ──> [Core (Domain & Services)] <── [Infrastructure (Postgres/Docker)]
```

### Capas del Proyecto (`backend/`):
* `internal/core/domain`: Modelos puros del sistema, constantes de veredicto y definiciones de interfaces (puertos).
* `internal/core/services`: Servicios con reglas de negocio principales.
* `internal/infrastructure/database`: Configuración de pools de conexión de `sqlx` e inicialización de esquemas.
* `internal/infrastructure/storage/postgres`: Implementación de repositorios de dominio interactuando con PostgreSQL.
* `internal/delivery/http`: Enrutador `http.ServeMux` nativo y controladores HTTP.

## 2. Esquema Relacional de Base de Datos (PostgreSQL)

```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS exercises (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'algorithm' o 'database'
    config JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY,
    student_id VARCHAR(100) NOT NULL,
    subject_id VARCHAR(100) NOT NULL,
    container_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    current_memory_mb INT NOT NULL DEFAULT 256,
    oom_strikes INT NOT NULL DEFAULT 0,
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```
