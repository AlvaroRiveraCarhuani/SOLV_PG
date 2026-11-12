# Modelo de Seguridad y Aislamiento de Contenedores (CRIT-09 & CRIT-15)

## 1. Terminación TLS y Seguridad de Red
- **Cifrado en Tránsito:** Todo el tráfico HTTP hacia la plataforma `solv.dedyn.io` y los subdominios `*.solv.dedyn.io` es redirigido automáticamente a HTTPS (puerto 443) mediante Traefik v3.
- **Certificados TLS:** Certificado Wildcard renovado automáticamente vía desafío ACME DNS-01 utilizando la API de `desec.io`.
- **Aislamiento Inter-Contenedor (ICC):** Redes Docker dedicadas y segmentación por tipo de entorno:
  - Los contenedores de evaluación de código arbitrario del Juez Virtual (`JUEZ_EFIMERO`) operan bajo sandbox estricto sin acceso a red (`NetworkMode: "none"`).
  - Los entornos interactivos de desarrollo (`IDE_PERSISTENTE`) se conectan a la red interna para publicar su servidor web exclusivamente a través del proxy inverso y comunicarse con la API de la plataforma (`host.docker.internal:host-gateway`), manteniendo activo el aislamiento de privilegios.

## 2. Politicas de Endurecimiento en Runtimes Docker
- **No-New-Privileges:** Todos los contenedores levantados por la plataforma (`StartContainer` y `StartEnvironmentContainer`) incluyen la directiva de seguridad:
  ```json
  "SecurityOpt": ["no-new-privileges:true"]
  ```
  Esto evita la escalada de privilegios mediante binarios setuid/setgid dentro del contenedor.
- **Usuario No-Root:** Los contenedores de laboratorios virtuales (OpenVSCode Server) se ejecutan bajo el UID/GID no privilegiado `1000:1000`.
- **Perfil Seccomp Default:** Se mantiene activo el filtro de llamadas al sistema por defecto de Docker para bloquear syscalls peligrosas (`reboot`, `kexec_load`, etc.).
