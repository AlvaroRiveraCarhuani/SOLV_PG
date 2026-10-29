-- +goose Up
-- Actualiza icon_key en template_components para reflejar el icono específico de cada tecnología
-- en vez del icono genérico del runtime (node) o motor compatible (mysql):
--   * node-express -> express
--   * node-nestjs -> nestjs
--   * node-prisma -> prisma
--   * node-typeorm -> typeorm
--   * mariadb-11 -> mariadb

UPDATE template_components
SET icon_key = 'express', updated_at = NOW()
WHERE slug = 'node-express';

UPDATE template_components
SET icon_key = 'nestjs', updated_at = NOW()
WHERE slug = 'node-nestjs';

UPDATE template_components
SET icon_key = 'prisma', updated_at = NOW()
WHERE slug = 'node-prisma';

UPDATE template_components
SET icon_key = 'typeorm', updated_at = NOW()
WHERE slug = 'node-typeorm';

UPDATE template_components
SET icon_key = 'mariadb', updated_at = NOW()
WHERE slug = 'mariadb-11';

-- +goose Down
UPDATE template_components
SET icon_key = 'node', updated_at = NOW()
WHERE slug IN ('node-express', 'node-nestjs', 'node-prisma', 'node-typeorm');

UPDATE template_components
SET icon_key = 'mysql', updated_at = NOW()
WHERE slug = 'mariadb-11';
