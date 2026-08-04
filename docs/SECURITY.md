# Seguridad y Aislamiento de Red (Zero Trust)

## Introducción y Desafío con Iptables y Docker

Por defecto, Docker inyecta reglas directamente en la cadena `FORWARD` de iptables y crea enlaces de traducción de direcciones (NAT). Esto significa que **cualquier regla definida en las cadenas estándar INPUT o FORWARD de Linux es completamente ignorada (bypaseada) por el tráfico de Docker**.

Los contenedores de los estudiantes, al estar en la red de tipo bridge, tienen acceso de red de salida y pueden alcanzar la IP del Gateway del host (generalmente `172.17.0.1` o el gateway de la red personalizada). Esto expone servicios críticos del host como:
* PostgreSQL (`5432`)
* Prometheus (`9090`)
* Grafana (`3000`)
* API Go (`8080`)

---

## Solución: La cadena DOCKER-USER

Para resolver esta vulnerabilidad, Docker expone una cadena especial de iptables llamada **`DOCKER-USER`**. 

Docker pasa todo el tráfico de los contenedores por esta cadena **antes** de evaluar sus propias reglas internas. Al insertar reglas de tipo `DROP` al inicio de `DOCKER-USER`, garantizamos el aislamiento completo del host frente a los contenedores sin interferir con la comunicación estándar de Docker.

---

## Cómo ejecutar el script de firewall

El script automatiza y valida de forma idempotente las reglas necesarias para aislar el host en la red por defecto (`docker0`) y la red personalizada del proyecto (`solv_net`).

### Instrucciones de ejecución:

1. Asegúrate de otorgar permisos de ejecución al script:
   ```bash
   chmod +x infra/firewall/docker-user-rules.sh
   ```

2. Ejecuta el script con privilegios de administrador (`sudo`):
   ```bash
   sudo ./infra/firewall/docker-user-rules.sh
   ```

3. El script es seguro para ejecutarse múltiples veces ya que realiza una validación previa (`iptables -C`) antes de insertar cada regla, evitando la duplicación.
