-- +goose Up
-- Correcciones del perfil de dependencias.
--
-- La preparación corre en /home/workspace, la misma ruta en la que el alumno ve
-- su proyecto. Por eso los almacenes de las familias que antes apuntaban a
-- /opt/solv/cache viven ahora en .solv-cache dentro del árbol: el enlace al
-- volumen copia rutas del proyecto, y una ruta absoluta fuera de él nunca
-- llegaba al aula. runtime_env es el entorno del aula: sin él, la herramienta
-- busca el almacén en su ruta por defecto y no lo encuentra. La caché compartida
-- por nodo queda como optimización de disco, no como condición para entregar.

-- Java: Maven. La pieza java-gradle es un complemento y la preparación corre
-- sobre la imagen base del runtime, así que un proyecto Gradle se detiene con un
-- mensaje que dice qué falta en lugar de fallar dentro de Maven.
UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["pom.xml"],"lockfiles":[],"install":"test -f pom.xml || { echo \"java-21 solo resuelve Maven; un proyecto Gradle necesita una imagen con Gradle\"; exit 1; }; mvn -B -q dependency:go-offline","install_without_lock":"test -f pom.xml || { echo \"java-21 solo resuelve Maven; un proyecto Gradle necesita una imagen con Gradle\"; exit 1; }; mvn -B -q dependency:go-offline","materialize":"tree","dependency_paths":[".solv-cache/m2"],"cache_env":{"MAVEN_OPTS":"-Dmaven.repo.local=/home/workspace/.solv-cache/m2"},"runtime_env":{"MAVEN_OPTS":"-Dmaven.repo.local=/home/workspace/.solv-cache/m2"},"verify":"mvn -q -DskipTests package","network":"build","offline_flags":"-o"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'java-21';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["go.mod"],"lockfiles":["go.sum"],"install":"go mod download","install_without_lock":"go mod download","materialize":"tree","dependency_paths":[".solv-cache/go-mod",".solv-cache/go"],"cache_env":{"GOMODCACHE":"/home/workspace/.solv-cache/go-mod","GOPATH":"/home/workspace/.solv-cache/go"},"runtime_env":{"GOMODCACHE":"/home/workspace/.solv-cache/go-mod","GOPATH":"/home/workspace/.solv-cache/go"},"verify":"go build ./...","network":"build","offline_flags":"GOFLAGS=-mod=mod GOPROXY=off"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'go-122';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["Cargo.toml"],"lockfiles":["Cargo.lock"],"install":"cargo fetch","install_without_lock":"cargo fetch","materialize":"tree","dependency_paths":[".solv-cache/cargo"],"cache_env":{"CARGO_HOME":"/home/workspace/.solv-cache/cargo"},"runtime_env":{"CARGO_HOME":"/home/workspace/.solv-cache/cargo"},"verify":"cargo check","network":"build","offline_flags":"--offline"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'rust-stable';

-- .NET restaura en obj además de la carpeta de paquetes: sin el archivo de
-- activos, dotnet build --no-restore no arranca en el aula.
UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["*.csproj","*.sln","Directory.Packages.props"],"lockfiles":["packages.lock.json"],"install":"dotnet restore","install_without_lock":"dotnet restore","materialize":"tree","dependency_paths":[".solv-cache/nuget","obj"],"cache_env":{"NUGET_PACKAGES":"/home/workspace/.solv-cache/nuget","DOTNET_CLI_TELEMETRY_OPTOUT":"1"},"runtime_env":{"NUGET_PACKAGES":"/home/workspace/.solv-cache/nuget"},"verify":"dotnet build --no-restore","network":"build"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'dotnet-8';

-- C y C++ no declaran dependencias de proyecto con CMake: enlazar build era
-- entregar salida de compilación como si fueran paquetes. Conan y vcpkg
-- necesitan perfiles propios y una imagen que traiga esas herramientas.
UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["CMakeLists.txt","conanfile.txt","vcpkg.json"],"materialize":"none"}'::jsonb,
    updated_at = NOW()
WHERE slug IN ('c-gcc13', 'cpp-gcc13');

-- +goose Down
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
SET dependency_profile = '{"version":1,"detect":["*.csproj","*.sln","Directory.Packages.props"],"lockfiles":["packages.lock.json"],"install":"dotnet restore","install_without_lock":"dotnet restore","materialize":"cache","dependency_paths":["/opt/solv/cache/nuget"],"cache_env":{"NUGET_PACKAGES":"/opt/solv/cache/nuget","DOTNET_CLI_TELEMETRY_OPTOUT":"1"},"verify":"dotnet build --no-restore","network":"build"}'::jsonb,
    updated_at = NOW()
WHERE slug = 'dotnet-8';

UPDATE template_components
SET dependency_profile = '{"version":1,"detect":["CMakeLists.txt","conanfile.txt","vcpkg.json"],"lockfiles":[],"install":"cmake -S . -B build -DCMAKE_BUILD_TYPE=Release","install_without_lock":"cmake -S . -B build -DCMAKE_BUILD_TYPE=Release","materialize":"tree","dependency_paths":["build"],"cache_env":{},"verify":"cmake --build build","network":"build"}'::jsonb,
    updated_at = NOW()
WHERE slug IN ('c-gcc13', 'cpp-gcc13');
