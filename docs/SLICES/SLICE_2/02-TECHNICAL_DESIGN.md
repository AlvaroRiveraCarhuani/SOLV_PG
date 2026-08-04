# Diseño Técnico - SLICE 2: Autenticación SSO Google, Middleware JWT y Volúmenes

## 1. Flujo de Autenticación SSO Google OAuth2

```
[Navegador] ──> GET /auth/google/login ──> [Redirección a Google OAuth]
                                                    │
[Google Auth] ──> Callback con Código OAuth ────────┘
     │
     ▼
[GET /auth/google/callback]
     │
     ├── 1. Extrae token y valida correo institucional (@uab.edu.bo)
     ├── 2. Registra/obtiene usuario en PostgreSQL
     ├── 3. Emite JWT firmado con JWT_SECRET
     └── 4. Deuelve JSON con token {"token": "eyJhb..."}
```

## 2. Middleware de Autenticación JWT

```
[Request] ──> [Authorization: Bearer <token>] ──> [JWTMiddleware]
                                                        │
                 ┌──────────────────────────────────────┴────────────────────────────────┐
                 ▼                                                                       ▼
         Firma o Expiración Inválida                                             Token Válido
         [HTTP 401 Unauthorized]                                      [Inyecta user_id en context.Context]
                                                                                         │
                                                                                         ▼
                                                                           [Handler Destino /labs/*]
```
