# ADR-081: Alta de Institución, Catálogo Base y Fin del Nombre Hardcodeado

## Estado
Aprobado

## Contexto

Hasta ahora, abrir SOLV en una universidad distinta era: entrar a PostgreSQL, hacer un `INSERT` en `tenants`, editar a mano el `allowed_domains`, crear el primer administrador con otro `INSERT`, y rezar para que el panel no asumiera filas que nadie había creado. Además, el código y las migraciones nombraban a una universidad concreta: el baseline sembraba su nombre, su dominio y su correo de soporte, y el alta de docentes mandaba correos firmados con ese nombre como constante en dos handlers.

Tres consecuencias, en orden de gravedad:

1. Una segunda institución no se podía instalar sin que alguien con acceso a la base la armara a mano. Eso convierte "instalación" en "servicio que solo yo puedo dar".
2. Todo correo de bienvenida salía con el nombre de otra universidad, aunque la instalación fuera de otra.
3. El dominio público tenía un valor por defecto ajeno (`solv.dedyn.io`) en compose. Un despliegue con el `.env` incompleto publicaba el sitio en un dominio que no era del cliente, en vez de detenerse.

## Decisión

### 1. El alta es un comando de operación, no un botón del panel

`solv-tenant` (imagen del backend, `cmd/tenant`) crea la institución completa:

```bash
docker compose exec backend solv-tenant -action create \
  -name "Universidad Ejemplo" -slug uejemplo \
  -admin-email admin@uejemplo.edu -domains uejemplo.edu \
  -period-code 2026-I -period-name "Primer Semestre 2026" \
  -period-start 2026-02-01 -period-end 2026-06-30

docker compose exec backend solv-tenant -action list
```

Por qué no el panel: el panel pertenece a una institución. Quien crea otra no tiene sesión en ninguna, y darle a un `admin` de una institución la capacidad de crear tenants convierte un token de administrador en una llave de plataforma. Si algún día hay un rol de plataforma, este ADR se revisa; hoy no existe y no se inventa para una pantalla.

El alta es idempotente por slug: reejecutar el comando informa que la institución ya existe y no cambia nada. Por eso puede formar parte de un procedimiento de instalación repetible.

### 2. El catálogo base es mínimo y explícito

Una transacción crea, en este orden:

| Qué | Por qué |
| :--- | :--- |
| `tenants` (nombre, slug, dominios, config de marca) | La institución y su identidad |
| Primer `users` con rol `admin` | Sin administrador nadie puede configurar nada |
| `academic_periods` con un periodo activo | Materias y cursos necesitan periodo |
| `subjects` "Materia General" (`GEN-101`) | Los entornos anteriores al modelo académico necesitan una materia ancla |
| `subject_catalog` con `GEN-101` | La materia ancla se enlaza al catálogo, no queda suelta |

Nada más. Un catálogo precargado con materias que la institución no dicta es peor que uno vacío: hay que borrarlo antes de empezar. Las plantillas, categorías y tecnologías son globales y no se duplican por institución.

Si cualquier paso falla, la transacción revierte completa. Una institución a medio crear no permite iniciar sesión y no hay forma de arreglarla desde la interfaz.

### 3. Ningún nombre institucional vive en el código

- El baseline siembra una institución neutra ("Institución", slug `default`, sin dominios, sin logo, sin correo de soporte). Conserva el UUID fijo porque varias filas ancla lo referencian; el nombre y el dominio dejan de ser de una universidad.
- Los correos de alta docente resuelven el nombre desde `tenants.config.institution_name` mediante `ResolveBranding()`, con `tenants.name` como respaldo y `SOLV` como último recurso. `TeacherInvitationHandler.SetTenantRepository` habilita esa resolución sin cambiar el constructor que ya usan los tests.
- `compose.yml` exige `SOLV_HOST`: `${SOLV_HOST:?...}` detiene el arranque con un mensaje. Ya no hay dominio por defecto.
- `example.env` usa `solv.midominio.edu` y `midominio.edu`, que se leen como lo que son: marcadores.
- `infra/traefik/dynamic_conf.yml` ya no publica el sitio; solo conserva el middleware de autenticación de los entornos.

## Consecuencias

**A favor**

- Instalar una segunda institución deja de requerir acceso a la base y pasa a ser un comando repetible, con salida legible.
- Una imagen sirve para cualquier universidad: el nombre sale de la configuración, no del binario.
- Un `.env` incompleto falla al arrancar en vez de publicar el sitio en el dominio de otro.
- Una instalación nueva nace sin identidad y no arrastra el nombre de la universidad donde se desarrolló el proyecto.

**En contra**

- El alta requiere acceso al host o al contenedor del backend. Es una decisión: quien puede crear instituciones ya puede leer la base, así que no se agrega una superficie nueva.
- El baseline cambió después de estar aplicado en instalaciones existentes. No las afecta (goose no reejecuta migraciones aplicadas), pero deja dos historias: la de quien instaló antes y la de quien instala desde ahora. Queda registrado acá para que nadie se sorprenda al comparar bases.
- El primer administrador se crea con rol `admin` sin pasar por Google. Su primer inicio de sesión es por OAuth con ese correo; si el dominio no está en `allowed_domains`, no entra. El comando no valida esa coherencia: conviene pasar `-domains` con el dominio real.

## Verificación

- `cmd/tenant` compila dentro de la imagen (`/usr/local/bin/solv-tenant`) y como binario suelto para desarrollo.
- El alta se probó sobre PostgreSQL 18 con una base vacía migrada: crea tenant, administrador, periodo, materia ancla y entrada de catálogo, y una segunda ejecución con el mismo slug no duplica nada.
- El catálogo de errores del comando distingue slug tomado, correo ya usado y datos inválidos, con código de salida distinto de cero salvo en el caso idempotente.

## Referencias

- ADR-079 — identificador externo de las personas espejadas
- ADR-080 — propiedad de los datos
- ADR-029 — periodos académicos y archivado
- `backend/cmd/tenant/main.go`, `backend/internal/core/services/tenant_provisioning.go`
