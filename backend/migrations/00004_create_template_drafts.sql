-- +goose Up
CREATE TABLE IF NOT EXISTS template_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_template_drafts_tenant_user UNIQUE(tenant_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_template_drafts_user ON template_drafts(user_id, updated_at DESC);

-- +goose Down
DROP TABLE IF EXISTS template_drafts;
