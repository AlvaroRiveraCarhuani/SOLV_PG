# Proposal: Alta Directa de Docentes, Importación Masiva Tolerante y Adaptador SMTP

## Contexto y Motivación
Actualmente, el alta de docentes en SOLV se apoya exclusivamente en un flujo de emisión de tokens transaccionales de 72 horas con entrega manual de enlaces. Aunque seguro para autogestión, este enfoque presenta fricción operativa en administraciones académicas:
1. **Falta de Despacho de Correo Real**: El modal muestra una opción de envío de enlace por correo, pero el backend carece de un cliente SMTP o proveedor transaccional, obligando a copiar y pegar enlaces manualmente.
2. **Ausencia de Pre-aprovisionamiento (Alta Directa)**: En un entorno con Google Workspace SSO institucional, el administrador debería poder pre-registrar a los docentes en la base de datos (por correo) para que al pulsar "Iniciar sesión con Google" ingresen con su rol de profesor sin intermediación de tokens ni enlaces efímeros.
3. **Necesidad de Carga Masiva Tolerante**: Al inicio de semestre, un administrador necesita importar planillas de nómina docente (Excel/CSV) que no siempre respetan esquemas de columnas rígidos, requiriendo detección inteligente de cabeceras, escaneo por expresiones regulares y previsualización de validación antes de confirmar.

## Alcance
- **Dominio Admin / Docentes**:
  - Modal rediseñado con soporte para Alta Directa (correo obligatorio, nombres opcionales que luego actualiza Google SSO).
  - Componente de Carga Masiva (.xlsx/.csv) con parser tolerante a columnas desordenadas y previsualización con resumen (correos detectados, válidos, duplicados).
  - Endpoints en backend para creación directa de docentes (`POST /api/v1/teachers/direct`) e importación masiva (`POST /api/v1/teachers/import-roster`).
- **Dominio Infraestructura / Correo (SMTP)**:
  - Adaptador SMTP desacoplado en Go (`net/smtp` / `crypto/tls`) configurable por variables de entorno (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`).
  - Fallback a logger en consola si no hay credenciales configuradas (entorno de desarrollo local).
  - Despacho asíncrono con plantilla de bienvenida institucional HTML (marca SOLV + tenant UAB) e interruptor opcional en la UI para notificar.

## Capabilities

### New Capabilities
- `admin-teachers`: Alta directa individual de docentes, importación masiva por planilla Excel/CSV tolerante con previsualización previa.
- `smtp-service`: Servicio de correo transaccional en backend con adaptador SMTP estándar, despacho asíncrono y plantillas HTML institucionales.
