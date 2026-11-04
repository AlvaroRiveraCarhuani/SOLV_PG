-- +goose Up
-- La instancia deja de llamarse "workspace": es el entorno de una práctica.
-- La migración renombra la tabla y sus referencias; los datos no se tocan.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'workspaces')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'environments') THEN
        ALTER TABLE workspaces RENAME TO environments;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE IF EXISTS submissions RENAME COLUMN workspace_id TO environment_id;
ALTER TABLE IF EXISTS class_session_participants RENAME COLUMN workspace_id TO environment_id;

ALTER INDEX IF EXISTS idx_workspaces_tenant_id RENAME TO idx_environments_tenant_id;
ALTER INDEX IF EXISTS idx_workspaces_template_id RENAME TO idx_environments_template_id;
ALTER INDEX IF EXISTS idx_workspaces_session RENAME TO idx_environments_session;
ALTER INDEX IF EXISTS idx_workspaces_frozen_at RENAME TO idx_environments_frozen_at;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_workspaces_subject') THEN
        ALTER TABLE environments RENAME CONSTRAINT fk_workspaces_subject TO fk_environments_subject;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_workspaces_tenant') THEN
        ALTER TABLE environments RENAME CONSTRAINT fk_workspaces_tenant TO fk_environments_tenant;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'submissions_workspace_id_fkey') THEN
        ALTER TABLE submissions RENAME CONSTRAINT submissions_workspace_id_fkey TO submissions_environment_id_fkey;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_session_participants_workspace_id_fkey') THEN
        ALTER TABLE class_session_participants RENAME CONSTRAINT class_session_participants_workspace_id_fkey TO class_session_participants_environment_id_fkey;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'environments')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'workspaces') THEN
        ALTER TABLE environments RENAME TO workspaces;
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE IF EXISTS submissions RENAME COLUMN environment_id TO workspace_id;
ALTER TABLE IF EXISTS class_session_participants RENAME COLUMN environment_id TO workspace_id;

ALTER INDEX IF EXISTS idx_environments_tenant_id RENAME TO idx_workspaces_tenant_id;
ALTER INDEX IF EXISTS idx_environments_template_id RENAME TO idx_workspaces_template_id;
ALTER INDEX IF EXISTS idx_environments_session RENAME TO idx_workspaces_session;
ALTER INDEX IF EXISTS idx_environments_frozen_at RENAME TO idx_workspaces_frozen_at;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_environments_subject') THEN
        ALTER TABLE workspaces RENAME CONSTRAINT fk_environments_subject TO fk_workspaces_subject;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_environments_tenant') THEN
        ALTER TABLE workspaces RENAME CONSTRAINT fk_environments_tenant TO fk_workspaces_tenant;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'submissions_environment_id_fkey') THEN
        ALTER TABLE submissions RENAME CONSTRAINT submissions_environment_id_fkey TO submissions_workspace_id_fkey;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'class_session_participants_environment_id_fkey') THEN
        ALTER TABLE class_session_participants RENAME CONSTRAINT class_session_participants_environment_id_fkey TO class_session_participants_workspace_id_fkey;
    END IF;
END $$;
-- +goose StatementEnd
