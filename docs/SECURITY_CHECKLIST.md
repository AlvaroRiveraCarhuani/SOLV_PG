# Checklist de Verificación de Seguridad (Zero Trust)

Sigue estos pasos para auditar que el host esté completamente protegido frente a accesos maliciosos desde los contenedores de los estudiantes.

---

## 🛠️ Procedimiento de Auditoría

### 1. Entrar al contenedor de un estudiante
Obtén el ID o nombre del contenedor activo del estudiante y conéctate a su terminal interactiva:
```bash
docker exec -it <nombre-contenedor-estudiante> sh
```

### 2. Instalar herramientas de diagnóstico (si es necesario)
Si el contenedor es una imagen Alpine o Debian/Ubuntu, puedes instalar `nmap` y `curl` para la prueba:
* **Alpine:** `apk add --no-cache nmap curl`
* **Debian/Ubuntu:** `apt-get update && apt-get install -y nmap curl`

### 3. Escanear el Gateway del Host con Nmap
Ejecuta el escaneo contra la IP del Gateway del Host.
* Si el contenedor está en la red bridge por defecto:
  ```bash
  nmap -p 5432,9090,3000,8080 172.17.0.1
  ```
* Si el contenedor está en la red personalizada `solv_net`:
  ```bash
  nmap -p 5432,9090,3000,8080 172.25.0.1
  ```

#### 📌 Resultado Esperado:
Todos los puertos escaneados (`5432`, `9090`, `3000`, `8080`) deben aparecer estrictamente con el estado **`filtered`** (bloqueados por firewall) o **`closed`**.

---

### 4. Validar Intentos de Conexión Directos

Ejecuta los siguientes comandos desde dentro del contenedor y confirma que todos fallen por expiración de tiempo (Timeout) o rechazo de conexión:

```bash
# Intentar conectar a PostgreSQL
curl -m 5 http://172.17.0.1:5432

# Intentar conectar a Prometheus
curl -m 5 http://172.17.0.1:9090

# Intentar conectar a Grafana
curl -m 5 http://172.17.0.1:3000

# Intentar conectar a la API del Backend de Go
curl -m 5 http://172.17.0.1:8080
```

#### 📌 Resultado Esperado:
Todos los comandos `curl` deben retornar un error de conexión (ej. `Connection timed out` o `Connection refused`) y ninguno debe obtener respuesta del host.
