# Manual Operativo - SLICE 4: Despliegue de Workspaces Interactivos

## 1. Preparación de la Red Aislada de Docker
```bash
# Crear la red compartida para Traefik y los entornos
docker network create --driver bridge --opt com.docker.network.bridge.enable_icc=false solv-traefik-net
```

## 2. Invocación de la API de Inicio de Entorno
```bash
curl -X POST http://localhost:3000/api/v1/workspaces/start \
  -H "Content-Type: application/json" \
  -d '{"student_id":"alumno_101", "subject_id":"programacion_1"}'
```
* **Respuesta Esperada:**
```json
{
  "workspace_id": "b4b134de-deeb-4d69-9fbd-24b9a8a3760f",
  "access_url": "http://b4b134de-deeb-4d69-9fbd-24b9a8a3760f.solv.local",
  "status": "running"
}
```
