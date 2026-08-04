# ADR-018: Automatización TLS Wildcard y Certificados SSL vía ACME DNS-01 (D2)

* **Estado:** Aceptado
* **Fecha:** 2026-08-04

## Contexto y Problema
Dado que cada laboratorio expone un subdominio opaco basado en UUID (ej. `abc123.solv.uab.edu.bo`), solicitar un certificado TLS individual HTTP-01 por cada estudiante generaría saturación en la API de Let's Encrypt, cuellos de botella y fallos en la provisión. Se requiere un certificado Wildcard universal (`*.solv.uab.edu.bo`) que proteja todos los subdominios de laboratorios sin modificar archivos de host locales ni exponer servicios en HTTP sin cifrar.

## Decisión Tomada
1. Adoptar la decisión de arquitectura **D2**: Integración de **Let's Encrypt Wildcard** gestionado automáticamente por Traefik v3 mediante el desafío **ACME DNS-01**.
2. Utilizar **desec.io** como proveedor de DNS administrado por API.
3. Configurar Traefik con la variable de entorno `DESEC_TOKEN` para que resuelva autónomamente las peticiones TXT de certificación Wildcard.

## Consecuencias
* **Positivas:**
  * Provisión instantánea de HTTPS cifrado para cualquier subdominio dinámico creado al vuelo.
  * Cero intervención manual del administrador o edición de archivos `hosts` en computadoras clientes.
  * Renovación transparente cada 60 días en segundo plano.
