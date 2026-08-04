# Diseño Técnico - SLICE 4: Entornos Interactivos y Traefik v3

## 1. Flujo de Enrutamiento Dinámico con Traefik v3

```
[Navegador del Alumno]
         │
         ▼ (http://<uuid>.solv.local)
  [Traefik v3 Proxy]
         │
         ├── Lee Docker Labels dinámicos:
         │   ├── traefik.enable=true
         │   ├── traefik.http.routers.<uuid>.rule=Host(`<uuid>.solv.local`)
         │   └── traefik.http.services.<uuid>.loadbalancer.server.port=8443
         │
         ▼ (Red solv-traefik-net, enable_icc=false)
[Contenedor code-server /workspace]
```

## 2. Idempotencia en la Creación de Workspaces (`POST /api/v1/workspaces/start`)

1. El backend recibe la petición con el `student_id` y `subject_id`.
2. Verifica en PostgreSQL si ya existe un entorno para ese alumno y materia.
3. Si existe y el contenedor está corriendo, devuelve la `access_url` existente (Idempotencia).
4. Si no existe, genera un **UUID v4 opaco**, crea el volumen nombrado `/workspace`, inyecta los labels de Traefik v3 y arranca el contenedor.
