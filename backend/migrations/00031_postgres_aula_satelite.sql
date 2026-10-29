-- +goose Up
-- PostgreSQL de aula deja de ser un servicio compartido de la plataforma y pasa a ser
-- un satélite dedicado más (ADR-051):
--   * base_image fija postgres:18-alpine, igual que el resto de los motores gestionados,
--   * ram_delta_mb con la huella del contenedor del motor (384 MB), que se reserva una
--     sola vez por nodo y nunca por laboratorio,
--   * descripción con el contrato de aislamiento por alumno y materia.
--
-- El slug se conserva para no romper plantillas ni referencias existentes; cambia el
-- nombre visible, que hasta ahora anunciaba la versión 16 aunque la imagen ya era 18.

UPDATE template_components
SET name = 'PostgreSQL 18 (Aula)',
    description = 'Motor relacional auto-orquestado del aula: rol, base y credenciales exclusivos por alumno y materia, separado de la base de la plataforma',
    icon_key = 'postgres',
    base_image = 'postgres:18-alpine',
    service_engine = 'postgres',
    ram_delta_mb = 384,
    sub_category = 'relational',
    is_active = true,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'postgres-16';

-- +goose Down
-- Restaura la ficha anterior: servicio compartido del aula con huella de 50 MB.
UPDATE template_components
SET name = 'PostgreSQL 16',
    description = 'Base de datos relacional compartida del aula, con base y usuario aislados por alumno y materia',
    icon_key = 'postgres',
    base_image = 'postgres:18-alpine',
    service_engine = 'postgres',
    ram_delta_mb = 50,
    sub_category = 'relational',
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'postgres-16';
