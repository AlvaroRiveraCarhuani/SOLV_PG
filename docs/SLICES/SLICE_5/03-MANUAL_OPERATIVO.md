# Manual Operativo - SLICE 5: Auto-Tuner y Monitoreo de Recursos

## 1. Ejecución del Calibrador Empírico de Hardware (Asus)
```bash
# Compilar binario estático en desarrollo (Fedora)
cd backend
GOOS=linux GOARCH=amd64 go build -o bin/autotune ./cmd/autotune

# Copiar ejecutable al servidor físico Asus
scp bin/autotune asus:~/autotune

# Ejecutar el auto-tuner vía SSH en la Asus
ssh asus "chmod +x ~/autotune && ~/autotune"
```

## 2. Inyección del Resultado en el Archivo .env
```env
# Agregar la cifra calibrada empíricamente en el archivo .env
OOM_GUARD_MB=1408
```

## 3. Endpoints HTTP de Monitoreo e Interacción
```bash
# Registrar latido HTTP de intención de uso
curl -X POST http://localhost:3000/api/v1/workspaces/<UUID>/heartbeat

# Reiniciar entorno tras caída por OOMKilled
curl -X POST http://localhost:3000/api/v1/workspaces/<UUID>/restart
```
