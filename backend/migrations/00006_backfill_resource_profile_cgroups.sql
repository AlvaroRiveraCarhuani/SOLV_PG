-- +goose Up
-- Backfill resource_profile for lab_templates derived from base_ram_mb:
-- min_mb = base_ram_mb / 2, high_mb = base_ram_mb * 1.5, max_mb = base_ram_mb * 2
UPDATE lab_templates
SET resource_profile = jsonb_build_object(
    'min_mb', (base_ram_mb / 2),
    'high_mb', ((base_ram_mb * 3) / 2),
    'max_mb', (base_ram_mb * 2)
)
WHERE base_ram_mb > 0;

-- +goose Down
-- Revert is a no-op since this is a data backfill aligning resource_profile with base_ram_mb
