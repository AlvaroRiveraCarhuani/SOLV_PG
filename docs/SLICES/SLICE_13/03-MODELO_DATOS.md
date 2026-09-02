# SLICE 13 — Modelo de Datos y Migraciones

## Esquema Relacional de Base de Datos (PostgreSQL 18)

### 1. Tabla `subjects` (Extensión Docente)
```sql
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS teacher_id UUID REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_subjects_teacher ON subjects(teacher_id);
```

### 2. Tabla `exercises` (Campos de Configuración y Estado)
```sql
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS boilerplate TEXT DEFAULT '';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'draft';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS language VARCHAR(50) DEFAULT 'python';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS time_limit_ms INT DEFAULT 1000;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS memory_limit_mb INT DEFAULT 128;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS db_config JSONB DEFAULT '{}'::jsonb;
```

### 3. Tabla `submissions` (Auditoría y Override)
```sql
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS manual_override BOOLEAN DEFAULT FALSE;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS override_reason TEXT DEFAULT '';
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS score INT;
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS graded_by UUID REFERENCES users(id) ON DELETE SET NULL;
```

### 4. Tabla `submission_comments` (Nueva Tabla de Feedback Anclado)
```sql
CREATE TABLE IF NOT EXISTS submission_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_submission_comments_sub ON submission_comments(submission_id);
CREATE INDEX IF NOT EXISTS idx_submission_comments_tenant ON submission_comments(tenant_id);
```
