# Feature: Gestion de Administradores y Control de Acceso en Configuracion Institucional
Branch: main | Delivery strategy: ask-on-risk | Test runner: go test ./... & npm run test:ci

## Specs

- **S1**: **Modelo de Dominio, Repositorio y Servicio de Administradores (Backend Go)**:
  - En `backend/internal/domain/user.go` y `ports.go`:
    - Definir `AdministratorDTO` con campos canónicos: `ID string`, `FirstName string`, `LastName string`, `Email string`, `Status string`, `CreatedAt time.Time`, `LastLoginAt *time.Time`.
    - Definir `CreateAdministratorDTO`: `FirstName string`, `LastName string`, `Email string`.
  - En `backend/internal/infrastructure/database/user_repository.go` (o repositorio especializado):
    - `ListAdministrators(ctx, tenantID)`: Consulta los usuarios con `role = 'admin'` y `tenant_id = $1` ordenados por `created_at ASC`.
    - `CreateAdministrator(ctx, tenantID, dto)`: Pre-aprovisionamiento directo. Inserta o actualiza un usuario con `role = 'admin'`, `tenant_id = $1`, `origin = 'manual'` y `status = 'active'` (sin requerir token ni link de aceptación). Si el correo ya existe dentro del tenant, actualiza su rol a `admin` y reactiva su estado si estaba suspendido. Al iniciar sesión vía Google SSO con ese correo, el usuario obtiene acceso administrativo inmediato.
    - `RevokeAdministrator(ctx, tenantID, targetUserID, requestingUserID)`: Revoca el acceso administrativo de `targetUserID`.
  - En `backend/internal/core/services/admin_service.go` (o servicio de administración):
    - Implementar las reglas de seguridad e invariantes:
      1. **Invariante de no-auto-revocación**: Si `targetUserID == requestingUserID`, retornar error de negocio prohibiendo que el administrador se revoque a sí mismo.
      2. **Invariante de orfandad institucional**: Validar que el conteo de administradores activos en el tenant sea estrictamente mayor a 1 antes de revocar. Si queda un solo administrador, rechazar la operación.
    - Notificación asíncrona: si el servicio de correo (SMTP) está activo, despachar un correo informativo de cortesía al nuevo administrador avisando de su alta, sin bloquear la operación si el correo falla.
    - Registrar el evento en el log de auditoría (`audit_service`) para trazabilidad de creación y revocación.
  - Tests unitarios en Go para validar creación, listado e invariantes de seguridad.

- **S2**: **Rutas HTTP y Endpoints Administrativos (Delivery Go)**:
  - En `backend/internal/delivery/http/admin_administrators_handler.go`:
    - `GET /api/v1/admin/administrators`: Requiere rol `admin` vía middleware y header `X-User-Role`. Retorna lista de administradores del tenant.
    - `POST /api/v1/admin/administrators`: Valida payload JSON (`email`, `first_name`, `last_name`). Normaliza el correo electrónico a minúsculas y sin espacios. Retorna HTTP 201 con el administrador creado.
    - `POST /api/v1/admin/administrators/{id}/revoke`: Valida que el solicitante sea un administrador activo, ejecuta la revocación protegida y retorna HTTP 200 con confirmación.
  - En `backend/internal/delivery/http/router.go`:
    - Registrar las tres rutas protegidas bajo el middleware de tenant y auditoría.
  - Tests de integración y de handlers HTTP validando códigos 200, 201, 400, 403 y 422 ante violaciones de invariantes.

- **S3**: **Capa de Datos y Servicio Reactivo en Frontend (Angular 22)**:
  - En `frontend/src/app/features/admin/configuracion/services/admin-config-administradores.service.ts`:
    - Interfaces estrictas: `Administrator`, `CreateAdministratorPayload`.
    - Estado reactivo con Signals: `administrators = signal<Administrator[]>([])`, `loading = signal<boolean>(false)`.
    - Métodos: `loadAdministrators()`, `createAdministrator(payload)`, `revokeAdministrator(id)`.
    - Manejo tipado de errores conectando con notificaciones globales sin interrumpir la vista.
  - Tests unitarios del servicio cubriendo flujos de éxito y fallos de API.

- **S4**: **Interfaz de Pestaña y Modales en Configuración Institucional (UI Frontend)**:
  - En `frontend/src/app/features/admin/configuracion/admin-configuracion.component.ts` y `.html`:
    - Registrar la 5ª pestaña: `{ id: 'administradores', label: 'Administradores', description: 'Gestion de credenciales con privilegios elevados y operadores del sistema' }`.
    - Agregar el caso `@case ('administradores')` en el panel principal.
  - En `frontend/src/app/features/admin/configuracion/tabs/admin-config-administradores.component.ts`, `.html` y `.scss`:
    - Selector limpio: `admin-config-administradores` (sin prefijo `solv-`).
    - Panel `card-section` con:
      - Encabezado descriptivo con contador de administradores activos y botón primario `+ Agregar Administrador`.
      - Banner informativo neutral recordando el principio de menor privilegio y alcance de la cuenta.
      - Tabla compacta:
        - Usuario: Avatar con iniciales, Nombre completo y Correo.
        - Incorporacion: `<time>` formateado con pipe compartido `dateText`.
        - Ultimo Acceso: `<time>` con `dateText` o texto "Sin registros".
        - Estado: Badge de estado semantico.
        - Acciones: Boton "Revocar acceso" (deshabilitado si `admin.id === currentUserId` o si la lista tiene un unico elemento, con tooltip explicativo).
    - Modal de creacion `admin-create-modal` con validacion de nombre, apellido y correo institucional.
    - Modal de confirmacion con advertencia explicita para la revocacion.
    - Cumplimiento de estilos: cero colores hex, fuentes y espacios exclusivamente mediante tokens de diseno (`var(--tenant-*)`, `var(--color-*)`, `var(--text-*)`).

- **S5**: **Verificacion Integral de Calidad y Criterios de Aceptacion**:
  - Tests unitarios de backend ejecutados con `go test ./internal/...`.
  - Tests unitarios de frontend ejecutados con `npm run test:ci`.
  - Auditoria de hojas de estilo ejecutada con `npm run lint:styles`.
  - Compilacion de produccion verificada con `CI=true npm run build`.

## Tasks

- [x] **T1** (S1): Implementar en Go el repositorio, servicio, validaciones de invariantes (no-auto-revocacion y no-orfandad) y auditoria para administradores con tests unitarios.
- [x] **T2** (S2): Crear el handler HTTP `admin_administrators_handler.go`, registrar endpoints en `router.go` y escribir tests de integracion para los flujos y codigos de error.
- [x] **T3** (S3): Implementar en Angular `admin-config-administradores.service.ts` con estado basado en Signals y pruebas unitarias.
- [x] **T4** (S4): Crear el componente `AdminConfigAdministradoresComponent`, modales de alta y revocacion, e integrarlo como la 5ta pestana de `AdminConfiguracionComponent`, respetando tokens del sistema de diseno y directivas de accesibilidad.
- [x] **T5** (S5): Ejecutar el conjunto de verificaciones: `go test ./...`, `npm run test:ci`, `npm run lint:styles` y build de produccion.
