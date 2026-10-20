-- +goose Up
UPDATE template_components
SET base_image = 'eclipse-temurin:21-jdk-jammy', updated_at = NOW()
WHERE slug = 'java-21' AND base_image = 'eclipse-temurin:21-alpine';

-- +goose Down
UPDATE template_components
SET base_image = 'eclipse-temurin:21-alpine', updated_at = NOW()
WHERE slug = 'java-21' AND base_image = 'eclipse-temurin:21-jdk-jammy';
