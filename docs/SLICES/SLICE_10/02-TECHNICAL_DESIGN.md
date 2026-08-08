# Diseño Técnico — Slice 10: TLS Wildcard & Container Hardening

## Flujo de Validación ACME DNS-01 con desec.io

```mermaid
sequenceDiagram
    autonumber
    participant Traefik as Traefik v3 (solv_traefik)
    participant LE as Let's Encrypt CA
    participant Desec as desec.io API
    participant DNS as DNS Público desec.io

    Traefik->>LE: Solicitud de Certificado Wildcard (*.solv.dedyn.io + solv.dedyn.io)
    LE-->>Traefik: Desafío ACME DNS-01 (_acme-challenge.solv.dedyn.io)
    Traefik->>Desec: PUT /api/v1/domains/solv.dedyn.io/rrsets/_acme-challenge/TXT/
    Note over Traefik,Desec: Autenticado vía Token DESEC_TOKEN
    Desec->>DNS: Propagación de Registro TXT
    Traefik->>Traefik: Espera de propagación (delayBeforeCheck: 10s)
    LE->>DNS: Consulta Registro TXT _acme-challenge
    DNS-->>LE: Registro TXT verificado exitosamente
    LE-->>Traefik: Emisión de Certificado TLS Wildcard (Válido 90 días)
    Traefik->>Traefik: Almacenamiento seguro en /letsencrypt/acme.json (chmod 600)
```

## Arquitectura de Aislamiento de Contenedores (Hardening)

```text
+-----------------------------------------------------------------------+
|                              HOST ASUS                                |
|                                                                       |
|  +------------------------+          +-----------------------------+  |
|  |   Contenedores IDE     |          |  Contenedores Juez (BD/Lang)|  |
|  | (openvscode-server)    |          |   (ephemeral execution)     |  |
|  +------------------------+          +-----------------------------+  |
|  | - SecurityOpt:         |          | - SecurityOpt:              |  |
|  |   no-new-privileges:true|          |   no-new-privileges:true    |  |
|  | - User: "1000:1000"    |          | - Memory & CPU Caps         |  |
|  | - Seccomp default      |          | - Seccomp default           |  |
|  | - Network: solv_net    |          | - Network: none (sin red)   |  |
|  +------------------------+          +-----------------------------+  |
+-----------------------------------------------------------------------+
```
