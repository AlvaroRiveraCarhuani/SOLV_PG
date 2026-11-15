-- +goose Up
-- Origen y sincronización de los datos que espejan el sistema académico de la
-- institución. `external_id` es la clave del registro en el sistema de origen,
-- `source_system` dice quién manda ese dato y `last_synced_at` cuándo se leyó.
--
-- Las columnas son opcionales: una instalación que se administra a mano sigue
-- funcionando sin llenarlas, y `enrollments.source_system` arranca en 'manual'
-- para que la reconciliación de una nómina no borre altas hechas desde el panel.

ALTER TABLE users ADD COLUMN IF NOT EXISTS external_id VARCHAR(190);
ALTER TABLE users ADD COLUMN IF NOT EXISTS source_system VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ;

ALTER TABLE subjects ADD COLUMN IF NOT EXISTS external_id VARCHAR(190);
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS source_system VARCHAR(50);
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ;

ALTER TABLE enrollments ADD COLUMN IF NOT EXISTS external_id VARCHAR(190);
ALTER TABLE enrollments ADD COLUMN IF NOT EXISTS source_system VARCHAR(50) NOT NULL DEFAULT 'manual';
ALTER TABLE enrollments ADD COLUMN IF NOT EXISTS last_synced_at TIMESTAMPTZ;

-- El identificador externo identifica a la persona dentro de una institución.
-- El índice es parcial: las filas sin identificador no compiten entre sí.
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_tenant_external_id
    ON users (tenant_id, external_id)
    WHERE external_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_tenant_external_id
    ON subjects (tenant_id, external_id)
    WHERE external_id IS NOT NULL;

-- La reconciliación borra por (tenant, materia, origen): este índice la hace
-- barata incluso con miles de matrículas.
CREATE INDEX IF NOT EXISTS idx_enrollments_subject_source
    ON enrollments (tenant_id, subject_id, source_system);

-- +goose Down
DROP INDEX IF EXISTS idx_enrollments_subject_source;
DROP INDEX IF EXISTS uq_subjects_tenant_external_id;
DROP INDEX IF EXISTS uq_users_tenant_external_id;

ALTER TABLE enrollments DROP COLUMN IF EXISTS last_synced_at;
ALTER TABLE enrollments DROP COLUMN IF EXISTS source_system;
ALTER TABLE enrollments DROP COLUMN IF EXISTS external_id;

ALTER TABLE subjects DROP COLUMN IF EXISTS last_synced_at;
ALTER TABLE subjects DROP COLUMN IF EXISTS source_system;
ALTER TABLE subjects DROP COLUMN IF EXISTS external_id;

ALTER TABLE users DROP COLUMN IF EXISTS last_synced_at;
ALTER TABLE users DROP COLUMN IF EXISTS source_system;
ALTER TABLE users DROP COLUMN IF EXISTS external_id;
