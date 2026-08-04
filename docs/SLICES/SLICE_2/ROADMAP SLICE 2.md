### Paso 1: Configuración de Infraestructura en GCP

- **Acción:** Crear el proyecto en Google Cloud Console.
    
- **Configuración:** Configurar la pantalla de consentimiento y generar el Client ID y Secret.
    
- **Restricción:** Aplicar el filtro estricto para aceptar únicamente correos del dominio `@uab.edu.bo`.
    

### Paso 2: Backend (Go) - Flujo SSO y Emisión de JWT

- **Endpoints de Autenticación:**
    
    - `GET /auth/google/login`: Al entrar aquí desde cualquier navegador, Go nos redirigirá a la pantalla de login de Google.
        
    - `GET /auth/google/callback`: Google nos devuelve aquí. Go extrae el correo, lo guarda en PostgreSQL y genera el JWT. **En lugar de redirigir a un frontend, Go simplemente imprimirá en el navegador un JSON crudo con el token:** `{"token": "eyJhb..."}`.
        
- **Middleware:** Implementar un middleware que intercepte peticiones a `/labs/*`, lea la cabecera `Authorization: Bearer <token>`, valide la firma del JWT y pase el `user_id` al contexto de la petición.
    

### Paso 3: Backend (Go) - Gestión de Volúmenes (ADR 001)

- **Creación/Vinculación:** En `lab_service.go`, antes de llamar a `container.Create`, usar el SDK de Docker para verificar si existe el volumen `solv_vol_{user_id}_{template_id}`. Si no, crearlo.
    
- **Montaje:** Añadir la configuración de `HostConfig.Binds` para montar ese volumen directamente en el `/workspace` del contenedor.
    
- **Ciclo de Destrucción:** Crear el endpoint `DELETE /labs/{id}` que apague y borre el contenedor, libere la regla de Traefik, pero garantice que el volumen de almacenamiento quede intacto.
    

### Paso 4: Validación Empírica (Modo CLI / Scripting)

El ciclo de pruebas que confirma que el Slice 2 es un éxito total, operado 100% como ingeniero DevOps:

1. Abrimos el navegador en `http://localhost:3000/auth/google/login`, elegimos nuestra cuenta institucional y copiamos el token JWT que escupe la pantalla.
    
2. Vamos a la terminal y lanzamos un `curl` o un script de Python, pasando el token en el header, para solicitar la creación del laboratorio.
    
3. Ingresamos al contenedor (vía URL de Traefik o por CLI con `docker exec`) y creamos un archivo `database_test.sql` en `/workspace`.
    
4. Lanzamos otro `curl` con el método `DELETE` para destruir el contenedor.
    
5. Volvemos a ejecutar el paso 2 para instanciar el laboratorio nuevamente, entramos, y comprobamos que `database_test.sql` sigue existiendo.a