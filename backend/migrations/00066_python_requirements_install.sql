-- +goose Up
-- Perfil de Python: proyecto con requirements.txt y sin lockfile.
--
-- El perfil declara requirements.txt en `detect` pero no en `lockfiles`, así que
-- el resolver concluye que la versión no trae archivo de versiones fijas y usa
-- `install_without_lock`. Esa variante instalaba el directorio del proyecto
-- (`pip install .`), que solo existe si hay pyproject.toml o setup.py: un
-- proyecto de cátedra con solo requirements.txt moría con
-- "Directory '.' is not installable" antes de bajar un paquete.
--
-- La variante sin lockfile instala desde requirements.txt y conserva la
-- instalación del proyecto como respaldo, para que un proyecto con solo
-- pyproject.toml siga funcionando. La escritura es quirúrgica (jsonb_set) para
-- no pisar los demás campos que el administrador haya ajustado, y es
-- idempotente: reaplicarla deja el mismo valor.
UPDATE template_components
SET dependency_profile = jsonb_set(
        dependency_profile,
        '{install_without_lock}',
        to_jsonb('python3 -m venv .venv && if [ -f requirements.txt ]; then .venv/bin/pip install --no-cache-dir -r requirements.txt; else .venv/bin/pip install --no-cache-dir .; fi'::text)
    ),
    updated_at = NOW()
WHERE slug = 'python-312'
  AND dependency_profile ? 'install_without_lock';

-- +goose Down
UPDATE template_components
SET dependency_profile = jsonb_set(
        dependency_profile,
        '{install_without_lock}',
        to_jsonb('python3 -m venv .venv && .venv/bin/pip install --no-cache-dir .'::text)
    ),
    updated_at = NOW()
WHERE slug = 'python-312'
  AND dependency_profile ? 'install_without_lock';
