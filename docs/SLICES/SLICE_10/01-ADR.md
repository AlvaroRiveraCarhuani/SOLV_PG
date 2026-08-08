# ADR-018: Proveedor DNS desec.io y Certificados TLS Wildcard con ACME DNS-01

## Estado
Aprobado e Implementado (Slice 10).

## Contexto
La plataforma SOLV requiere terminación TLS segura para el dominio apex `solv.dedyn.io` y todos los subdominios dinámicos de los laboratorios virtuales de los estudiantes (`*.solv.dedyn.io`). Dado que se necesita soporte para subdominios wildcard dinámicos, el protocolo de verificación HTTP-01 de Let's Encrypt resulta insuficiente, requiriéndose la validación de desafíos DNS-01 a través de la API del proveedor de DNS.

## Decisión
1. **Proveedor DNS:** Adoptar `desec.io` como proveedor de DNS administrado para el dominio `solv.dedyn.io`.
2. **ACME Resolver:** Utilizar el `certResolver` de Traefik v3 (`letsencrypt`) configurado con el proveedor `desec` y el desafío `dnsChallenge`.
3. **Gestión de Credenciales:** Inyectar la credencial del token dedicado `traefik-acme` como la variable de entorno `DESEC_TOKEN` desde `.env` al contenedor `solv_traefik`.
4. **Dominio & Wildcard:** Aprovisionar los dominios en Traefik especificando `main: solv.dedyn.io` y `sans: [*.solv.dedyn.io]`.

## Consecuencias
- **Positivas:**
  - Renovación de certificados 100% automatizada e ininterrumpida sin exponer puertos ni interrumpir tráfico.
  - Soporte nativo para subdominios wildcard dinámicos ilimitados para los entornos de laboratorios.
  - Cero costo operativo mediante la integración con la infraestructura desec.io.
- **Seguridad:**
  - Las llaves privadas del certificado se almacenan exclusivamente en `letsencrypt/acme.json` con permisos estrictos `chmod 600`, fuera del control de versiones (`.gitignore`).
