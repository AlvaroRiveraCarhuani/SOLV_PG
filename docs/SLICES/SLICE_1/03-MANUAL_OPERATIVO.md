# Manual Operativo - SLICE 1: Inicialización de la Base del Sistema

## 1. Requisitos del Entorno
* Go version 1.22+
* PostgreSQL 15+ o contenedor de PostgreSQL
* Docker Engine 24.0+

## 2. Variables de Entorno (.env)
```env
DATABASE_URL=postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable
PORT=3000
```

## 3. Comandos de Ejecución
```bash
# Compilar y levantar la API en desarrollo
cd backend
go run cmd/api/main.go
```
