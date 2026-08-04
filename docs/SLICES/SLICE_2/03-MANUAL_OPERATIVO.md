# Manual Operativo - SLICE 2: Autenticación y Gestión de Volúmenes

## 1. Configuración del Entorno (.env)
```env
GOOGLE_CLIENT_ID=800667972597-...apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
GOOGLE_REDIRECT_URL=http://localhost:3000/auth/google/callback
JWT_SECRET=una_cadena_aleatoria_y_muy_larga_para_solv
```

## 2. Comandos cURL para Pruebas Manuales
```bash
# 1. Iniciar login en navegador
# Navegar a: http://localhost:3000/auth/google/login

# 2. Copiar token devuelto e invocar endpoint protegido
curl -H "Authorization: Bearer <TOKEN>" http://localhost:3000/api/v1/workspaces/me
```
