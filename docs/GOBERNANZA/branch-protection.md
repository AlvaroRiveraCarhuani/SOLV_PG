# Guía de Configuración de Protección de Ramas (`main`) — GitHub

Este documento describe el procedimiento paso a paso que el administrador del repositorio en GitHub (`@AlvaroRiveraCarhuani`) debe aplicar en la interfaz web de GitHub para activar la protección estricta sobre la rama principal `main`.

> [!IMPORTANT]
> Los agentes de desarrollo e integración continua no modifican la configuración del proveedor de Git directamente. Este procedimiento es un control humano de gobernanza.

---

## Pasos para Configurar Protección en GitHub (`main`)

### 1. Acceder a los ajustes de protección
1. Ingresar al repositorio en GitHub: [https://github.com/AlvaroRiveraCarhuani/SOLV_PG](https://github.com/AlvaroRiveraCarhuani/SOLV_PG).
2. Hacer clic en la pestaña **Settings** (Configuración).
3. En el menú lateral izquierdo, seleccionar **Branches** (en la sección *Code and automation*).
4. En la sección **Branch protection rules**, hacer clic en **Add branch protection rule** (o editar la regla existente para `main`).

### 2. Definir el patrón de rama
- **Branch name pattern:** Escribir `main`.

### 3. Activar los Controles de Gobernanza (Checklist de Seguridad)

Marcar las siguientes casillas de verificación obligatorias:

1. **Require a pull request before merging:**
   - Activar la casilla **Require a pull request before merging**.
   - **Required approvals:** Seleccionar `1`.
   - Activar **Dismiss stale pull request approvals when new commits are pushed** (revoca aprobaciones cuando se agregan nuevos commits).

2. **Require status checks to pass before merging:**
   - Activar la casilla **Require status checks to pass before merging**.
   - Activar la casilla **Require branches to be up to date before merging** (asegura que la rama esté actualizada con `main`).
   - En la barra de búsqueda de verificaciones de estado, buscar y agregar los 6 trabajos del pipeline de GitHub Actions:
     - `1. Lint & Quality Gates`
     - `2. Test Backend (Postgres Service)`
     - `3. Test Frontend`
     - `4. Build Artifacts`
     - `5. Security Audit`
     - `6. Migration Chain Verification`

3. **Bloquear Pushes Directos y Fuerza Bruta:**
   - Activar **Do not allow bypassing the above settings** (aplica las reglas también a los administradores).
   - Activar **Restrict who can push to matching branches** (impide pushes directos sin Pull Request).
   - Activar **Block force pushes** (previene `git push --force`).
   - Activar **Prevent deletions** (impide eliminar la rama `main`).

### 4. Guardar la regla
- Hacer clic en el botón verde **Create** o **Save changes** al final de la página.

---

## Verificación de Cumplimiento

Una vez aplicada la regla:
- Todo intento de `git push origin main` directo será rechazado por GitHub con HTTP 403 Forbidden.
- Todo cambio requerirá un Pull Request con el template `.github/PULL_REQUEST_TEMPLATE.md`.
- El botón **Merge pull request** estará bloqueado hasta que los 6 trabajos de CI estén en verde y se cuente con al menos 1 aprobación.
