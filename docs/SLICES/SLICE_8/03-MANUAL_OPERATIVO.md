# Manual Operativo y Guía de Despliegue — Slice 8

Este manual describe los procedimientos operativos necesarios para desplegar, configurar y verificar el entorno de desarrollo y producción del **Slice 8**.

---

## 1. Configuración de Variables de Entorno (.env)

Copia el archivo de ejemplo e inyecta los valores correspondientes en la raíz del proyecto:

```bash
cp example.env .env
```

Asegúrate de definir las siguientes variables sin hardcodear dominios:

```ini
# Autenticación Google OAuth2
GOOGLE_CLIENT_ID="tu-client-id-google"
GOOGLE_CLIENT_SECRET="tu-client-secret-google"
GOOGLE_REDIRECT_URL="http://localhost:3000/auth/google/callback"

# Seguridad y Cookies
JWT_SECRET="clave-super-secreta-jwt-min-32-caracteres"
COOKIE_DOMAIN=".solv.uab.edu.bo"

# Base de Datos PostgreSQL
DATABASE_URL="postgres://solv_user:solv_password@localhost:5432/solv_db?sslmode=disable"

# Proveedor DNS desec.io (ACME DNS-01)
DESEC_TOKEN="tu-token-desec-io"
```

---

## 2. Aplicación de Reglas de Firewall `DOCKER-USER` (CRIT-01)

Ejecuta el script de blindaje de red con privilegios de superusuario en el anfitrión Linux:

```bash
chmod +x infra/firewall/docker-user-rules.sh
sudo ./infra/firewall/docker-user-rules.sh
```

> [!IMPORTANT]
> Este paso es obligatorio para evitar que Docker Engine exponga el puerto 5432 de PostgreSQL o los contenedores de laboratorios directamente al público sin pasar por Traefik v3.

---

## 3. Arranque de Infraestructura y Servicios (Make)

### Paso A: Levantar Contenedores de Soporte (Traefik v3 + PostgreSQL)

```bash
docker compose down && docker compose up -d
```

### Paso B: Compilar y Levantar el Backend en Go

```bash
make run
```

---

## 4. Verificación Operativa de Puertos y Servicios

Verifica el estado de los puertos expuestos en el servidor anfitrión mediante `nmap` o `netstat`:

```bash
nmap -p 1-65535 localhost
```

**Resultado Esperado de Puertos Expuestos Públicamente:**
* **80/tcp (HTTP):** Abierto (Traefik v3)
* **443/tcp (HTTPS):** Abierto (Traefik v3)
* **8080/tcp (Traefik Dashboard):** Abierto (Modo seguro / interno)
* **5432/tcp (PostgreSQL):** Enlazado a `127.0.0.1` (Filtrado / No accesible externamente)
* **3000/tcp (OpenVSCode Server):** Enlazado internamente a la red `solv_net` (Sin exposición pública directa)
