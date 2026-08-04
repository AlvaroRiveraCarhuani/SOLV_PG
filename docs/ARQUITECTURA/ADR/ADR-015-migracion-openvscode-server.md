# ADR-015: Migración a la Distribución Oficial OpenVSCode Server

* **Estado:** Aceptado (Evoluciona ADR-004)
* **Fecha:** 2026-07-20

## Contexto y Problema
El uso de distribuciones personalizadas o parches profundos en el IDE web (como `code-server`) generaba deuda técnica, fragilidad ante actualizaciones oficiales de VS Code y puertos no estándar (puerto `8443`). Además, el backend en Go y el proxy Traefik v3 ya garantizan la autenticación perimetral (Zero Trust), por lo que no es necesario imponer tokens de conexión internos dentro del contenedor del laboratorio.

## Decisión Tomada
1. Sustituir `code-server` por la imagen oficial **`gitpod/openvscode-server:latest`**.
2. Actualizar el puerto de enrutamiento dinámico en Traefik v3 al puerto estándar **3000**.
3. Inyectar la bandera de arranque `--without-connection-token --host 0.0.0.0`, eliminando tokens internos del contenedor.
4. Mapear el volumen nombrado del estudiante al directorio oficial `/home/workspace:rw`.

## Consecuencias
* **Positivas:**
  * Paridad total con VS Code oficial en el navegador y eliminación de deuda técnica.
  * Cero tokens internos innecesarios en el contenedor.
  * Enrutamiento estandarizado en puerto 3000.
