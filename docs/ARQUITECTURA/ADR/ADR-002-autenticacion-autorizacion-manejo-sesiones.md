# Estado

**Aceptado**

---

## 1. Contexto y Origen de la Necesidad

Para asignar correctamente los entornos y volúmenes persistentes, el sistema SOLV requiere identificar unívocamente a los usuarios.

Además, debido a que docentes y estudiantes comparten el mismo dominio institucional (`@uab.edu.bo`), el sistema necesita un mecanismo robusto de Autorización (AuthZ) para diferenciar privilegios (por ejemplo, crear laboratorios vs. unirse a laboratorios), sin depender de un acceso directo a la base de datos central de la universidad (Kardex).

---

## 2. Decisión Arquitectónica

Se implementará un sistema de identidad descentralizado y de **Control de Acceso Basado en Roles (RBAC)** híbrido a dos niveles, compuesto por los siguientes componentes:

### Autenticación (AuthN - Identidad)

- Single Sign-On (SSO) mediante OAuth 2.0 utilizando cuentas institucionales de Google Workspace.

### Autorización de Plataforma (AuthZ Global - Roles)

La asignación del rol de privilegio (**DOCENTE / ADMIN**) estará estrictamente aislada de plataformas externas para evitar ataques de escalada de privilegios. Se gestionará localmente mediante:

- **Pre-aprovisionamiento (Allowlist):**  
  Una lista blanca estática de correos institucionales pre-aprobados (*Zero-Click Onboarding*).

- **Autorización de Respaldo (Magic Links):**  
  Enlaces criptográficos de invitación de un solo uso para docentes que no se encuentren en la lista blanca inicial.

### Autorización Contextual (AuthZ de Entornos)

- Sincronización dinámica de inscripciones delegada a la API de Google Classroom.  
- El sistema cruzará la identidad del estudiante validada por SSO con las listas oficiales de las clases para otorgar acceso a los contenedores.

### Gestión de Sesión

- Emisión de **JSON Web Tokens (JWT)** sin estado para manejar sesiones de usuario de forma escalable.

---

## 3. Justificación y Eficiencia (Primeros Principios)

### Delegación de Identidad vs. Soberanía de Roles

- Al usar Google SSO, el sistema SOLV elimina completamente la necesidad de almacenar contraseñas locales, reduciendo significativamente la superficie de ataque.
- Se garantiza que solo usuarios institucionales activos puedan acceder al sistema.

Para mitigar el riesgo de **Privilege Escalation** (por ejemplo, estudiantes creando clases en Classroom para obtener permisos):

- SOLV mantiene la soberanía del rol.
- La base de datos local actúa como autoridad suprema:
  - Si un usuario no está en la Allowlist o no fue invitado → su rol será **ESTUDIANTE**.

---

### Automatización vs. Vendor Lock-in

- El uso de la API de Google Classroom se limita a consultar inscripciones.
- Esto permite automatizar el acceso a laboratorios sin intervención manual.

Para evitar dependencia total de Google:

- El núcleo de SOLV implementará el **Patrón Adaptador (Adapter Pattern)**.
- Esto abstrae el proveedor externo y permite migrar en el futuro (por ejemplo, a Moodle).

---

### Seguridad de Administración (Zero Trust)

El registro de docentes excepcionales se realizará mediante enlaces criptográficos de un solo uso que:

- expiran automáticamente  
- se invalidan al consumirse  

Esto evita filtraciones o accesos no autorizados.

Medidas adicionales:

- Mitigación de **Forced Browsing**:
  - Responder `404 Not Found` en lugar de `403 Forbidden`.
- Protección de endpoints administrativos:
  - Reglas de **IP Allowlisting** en el proxy inverso (Traefik).

---

## 4. Consecuencias y Trade-offs (Análisis de Costo)

### Costo

El desarrollo inicial requiere implementar:

- adaptadores para consumir APIs externas  
- archivo de configuración de Allowlist  
- módulo de generación de tokens criptográficos  
- sistema de invitaciones seguras  

Esto incrementa la complejidad del backend (FastAPI).

---

### Ventaja

El sistema final ofrece:

- autonomía frente a plataformas externas en el control de roles  
- escalabilidad para miles de usuarios  
- mínima carga administrativa para el docente  
- seguridad empresarial basada en **Defensa en Profundidad**  

Además, la arquitectura es resistente a ataques internos de escalada de privilegios.