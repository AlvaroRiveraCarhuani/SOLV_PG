# ADR-023: Protección Zero Trust del Host con Cadena `DOCKER-USER` de `iptables` y Enlace Localhost

* **Estado:** Aceptado
* **Fecha:** 2026-08-04

## Contexto y Problema
Docker Engine por defecto manipula las reglas de `iptables` en Linux insertando reglas NAT que sobreescriben los firewalls tradicionales del sistema anfitrión (ej. `ufw` o cadenas `INPUT`). Esto permitía que servicios como PostgreSQL (puerto 5432) o los contenedores de laboratorios fueran accesibles directamente desde redes externas si se especificaban mapeos de puertos, puenteando la seguridad perimetral de Traefik v3.

## Decisión Tomada
1. Crear el script de infraestructura [infra/firewall/docker-user-rules.sh](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/infra/firewall/docker-user-rules.sh) para interceptar el tráfico en la cadena `DOCKER-USER` antes de que Docker aplique sus reglas NAT.
2. Forzar el enlace de PostgreSQL exclusivamente a `127.0.0.1:5432` en [compose.yml](file:///home/alvarorivera/Documentos/Desarrollo/SOLV_PG/compose.yml).
3. Bloquear en `DOCKER-USER` cualquier tráfico no autorizado dirigido a puertos internos, garantizando que el único punto de entrada expuesto públicamente sea Traefik v3 en los puertos 80 y 443.

## Consecuencias
* **Positivas:**
  * Inmunidad total contra el bypass de firewall predeterminado de Docker Engine.
  * Protección Zero Trust del servidor anfitrión On-Premise.
* **Negativas:**
  * Requiere la ejecución del script con privilegios de superusuario (`sudo`) al iniciar el servidor anfitrión.
