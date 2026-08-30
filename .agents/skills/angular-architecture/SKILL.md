---
name: angular-architecture
description: Reglas y estándares para el desarrollo en Angular 22 standalone, zoneless y reactividad basada en Signals.
---

# Angular 22 — Arquitectura y Estándares

## Core Principios
- Angular 22 Standalone por defecto: PROHIBIDO crear o usar `NgModules`.
- Zoneless architecture: reactividad pura con Signals.
- Control flow nativo: usar estrictamente `@if`, `@for`, `@switch`. PROHIBIDO `*ngIf`, `*ngFor`, `*ngSwitch`.
- Carga diferida obligatoria: `@defer` obligatorio para editor Monaco, visualizadores de código e componentes pesados.
- Inyección de dependencias: usar función `inject()` en lugar de DI por constructor.
- Enrutamiento: rutas lazy con `loadComponent()` y guards funcionales (`CanActivateFn`).

## Reactividad y Estado
- Usar únicamente Signals como modelo de estado: `signal()`, `computed()`, `input()`, `output()`, `model()`.
- Comunicaciones de componentes mediante `input()` y `output()` nativas de Signals.
- Lecturas de API mediante `httpResource()` / `resource()` o RxJS interoperable con `toSignal()`.

## Estructura de Directorios (Feature-Based)
```text
src/app/
├── core/            # Servicios singleton (Auth, HTTP Interceptors, Guards, Models)
├── shared/          # UI componentes reutilizables, directivas, pipes
└── features/        # Módulos de funcionalidad (student, teacher, admin)
```

## Styling y Componentes UI
- Styling: Vanilla SCSS + CSS Custom Properties (variables) para tokens de diseño.
- PROHIBIDO instalar o usar TailwindCSS.
- Librerías UI autorizadas: Angular Material 3 + Lucide Icons.
- HttpClient estrictamente tipado contra interfaces de `app/core/models/`.
