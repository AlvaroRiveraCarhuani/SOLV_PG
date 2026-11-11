-- +goose Up
-- Perfil de preparación de dependencias de proyecto por familia tecnológica.
-- El backend no ramifica por lenguaje: lee este perfil y ejecuta el mismo procedimiento.
ALTER TABLE template_components
    ADD COLUMN IF NOT EXISTS dependency_profile JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["requirements.txt","pyproject.toml","Pipfile","setup.py"],"lockfiles":["requirements.lock","uv.lock","poetry.lock","Pipfile.lock"],"install":"python3 -m venv .venv && .venv/bin/pip install --no-cache-dir -r requirements.txt","install_without_lock":"python3 -m venv .venv && .venv/bin/pip install --no-cache-dir .","materialize":"tree","dependency_paths":[".venv"],"cache_env":{"PIP_CACHE_DIR":"/opt/solv/cache/pip","PIP_DISABLE_PIP_VERSION_CHECK":"1"},"verify":".venv/bin/python -m pip check","network":"build","offline_flags":"PIP_NO_INDEX=1"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'python-312';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["package.json"],"lockfiles":["pnpm-lock.yaml","package-lock.json","yarn.lock","bun.lockb"],"install":"corepack enable && (pnpm install --frozen-lockfile || npm ci --no-audit --no-fund)","install_without_lock":"npm install --no-audit --no-fund","materialize":"tree","dependency_paths":["node_modules"],"cache_env":{"npm_config_cache":"/opt/solv/cache/npm","PNPM_HOME":"/opt/solv/cache/pnpm"},"verify":"corepack enable && (pnpm list --depth=0 --offline || npm ls --depth=0 --offline)","network":"build","offline_flags":"npm_config_offline=true"}'::jsonb,
    updated_at = NOW()
WHERE slug IN ('node-20', 'angular-web', 'react-vite', 'vue-vite', 'nextjs-web', 'vite-web');

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["pom.xml","build.gradle","build.gradle.kts","settings.gradle","settings.gradle.kts"],"lockfiles":[],"install":"mvn -B -q dependency:go-offline","install_without_lock":"mvn -B -q dependency:go-offline","materialize":"cache","dependency_paths":["/opt/solv/cache/m2"],"cache_env":{"MAVEN_OPTS":"-Dmaven.repo.local=/opt/solv/cache/m2","GRADLE_USER_HOME":"/opt/solv/cache/gradle"},"verify":"mvn -o -q -DskipTests package","network":"build","offline_flags":"-o"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'java-21';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["go.mod"],"lockfiles":["go.sum"],"install":"go mod download","install_without_lock":"go mod download","materialize":"cache","dependency_paths":["/opt/solv/cache/go-mod"],"cache_env":{"GOMODCACHE":"/opt/solv/cache/go-mod","GOPATH":"/opt/solv/cache/go"},"verify":"go build ./...","network":"build","offline_flags":"GOFLAGS=-mod=mod GOPROXY=off"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'go-122';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["Cargo.toml"],"lockfiles":["Cargo.lock"],"install":"cargo fetch","install_without_lock":"cargo fetch","materialize":"cache","dependency_paths":["/opt/solv/cache/cargo"],"cache_env":{"CARGO_HOME":"/opt/solv/cache/cargo"},"verify":"cargo check --offline","network":"build","offline_flags":"--offline"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'rust-stable';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["composer.json"],"lockfiles":["composer.lock"],"install":"composer install --no-interaction --no-progress --prefer-dist","install_without_lock":"composer install --no-interaction --no-progress --prefer-dist","materialize":"tree","dependency_paths":["vendor"],"cache_env":{"COMPOSER_HOME":"/opt/solv/cache/composer","COMPOSER_CACHE_DIR":"/opt/solv/cache/composer/cache"},"verify":"composer validate --no-check-publish && php -l vendor/autoload.php","network":"build","offline_flags":"COMPOSER_DISABLE_NETWORK=1"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'php-83';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["*.csproj","*.sln","Directory.Packages.props"],"lockfiles":["packages.lock.json"],"install":"dotnet restore","install_without_lock":"dotnet restore","materialize":"cache","dependency_paths":["/opt/solv/cache/nuget"],"cache_env":{"NUGET_PACKAGES":"/opt/solv/cache/nuget","DOTNET_CLI_TELEMETRY_OPTOUT":"1"},"verify":"dotnet build --no-restore","network":"build"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'dotnet-8';

-- C y C++ comparten familia de dependencias. Un Makefile suelto no declara paquetes
-- externos: ese proyecto no tiene nada que preparar y queda fuera de detect.
UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["CMakeLists.txt","conanfile.txt","vcpkg.json"],"lockfiles":[],"install":"cmake -S . -B build -DCMAKE_BUILD_TYPE=Release","install_without_lock":"cmake -S . -B build -DCMAKE_BUILD_TYPE=Release","materialize":"tree","dependency_paths":["build"],"cache_env":{},"verify":"cmake --build build","network":"build"}'::jsonb,
    updated_at = NOW()
WHERE slug IN ('c-gcc13', 'cpp-gcc13');

-- +goose Down
ALTER TABLE template_components
    DROP COLUMN IF EXISTS dependency_profile;
