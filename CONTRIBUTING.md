# Guía de Contribución y Gobernanza — Juez Virtual SOLV

¡Bienvenido al proyecto SOLV! Para mantener la estabilidad enterprise del Juez Virtual, la arquitectura limpia y la trazabilidad técnica, todo desarrollo debe ajustarse estrictamente a las reglas y protocolos definidos en este documento.

---

## 1. Protocolo de desarrollo por Órdenes Atómicas (OA)

El desarrollo en SOLV no se realiza mediante tareas genéricas, sino mediante **Órdenes Atómicas (OA)** autocorrectivas y cerradas. Cada OA define un alcance acotado y no negociable.

### Estructura de una Orden Atómica
1. **Contexto:** Estado actual del sistema e integración con OAs anteriores.
2. **Decisiones Cerradas (Invariantes):** Reglas pedagógicas, arquitectónicas o de UX que NO se rediscuten.
3. **Especificación Técnica:** Contrato de API, esquemas relacionales, componentes UI y lógica de negocio.
4. **Criterios de Aceptación:** Lista de verificación automatizable en CI/CD.
5. **Entregable de Retorno:** Log de pruebas, artefactos generados y lista explícita de cambios.

---

## 2. Convenciones de Código y Arquitectura

### Backend (Go 1.26)
- **Arquitectura Hexagonal:**
  - `core/domain`: Entidades, contratos e interfaces.
  - `core/services`: Casos de uso y lógica de negocio pura.
  - `infrastructure/`: Adaptadores de base de datos, Docker, cgroups y semgrep.
  - `delivery/http`: Router, DTOs y handlers HTTP usando `net/http.ServeMux` (SIN frameworks como Gin).
- **Base de Datos & Migraciones:**
  - PostgreSQL 18.
  - Toda modificación de esquema debe incluir script de migración reversible en `backend/migrations/` usando `goose` (soporte `Up` y `Down`).
- **Tests:**
  - Pruebas unitarias en `*_test.go` utilizando mocks de repositorio.
  - Pruebas de integración HTTP usando `httptest.NewServer` o `httptest.NewRecorder`.

### Frontend (Angular 22 Standalone & Signals)
- **Estándares:**
  - Angular 22 standalone, zoneless y reactividad basada en Signals.
  - Selectores semánticos limpios en kebab-case sin prefijo `solv-` (PROHIBIDO `selector: 'solv-...'`).
- **Sistema de Diseño y SCSS:**
  - PROHIBIDO literales hex (colores `#...`) o fallbacks rem en CSS/SCSS de componentes.
  - Usar tokens semánticos: `var(--tenant-primary)`, `var(--color-text-primary)`, `var(--border-color)`.
  - Gate de estilo `npm run lint:styles` exigido en CI.
- **Formateo de Fechas y Datos Técnicos:**
  - Toda fecha o timestamp se renderiza obligatoriamente con el pipe compartido `dateText` dentro de un `<time>` semántico.
  - Valores técnicos (códigos, IDs, métricas, tags, veredictos) usan el atributo `[machineData]` o la clase `font-mono`.

---

## 3. Estándar de Commits (Conventional Commits)

Los mensajes de commit deben seguir la especificación **Conventional Commits**:
```txt
<tipo>(<alcance>): <descripción corta en minúsculas>

<cuerpo obligatorio de 1 a 2 líneas explicando Qué, Por qué, Cómo e Impacto>
```

### Reglas para mensajes de commit:
- Usar primera persona del plural o impersonal técnico: `"Se agregó X"`, `"Agregamos X"`.
- **PROHIBIDO** incluir atribución de IA (ej: `Co-Authored-By: AI...`).
- **PROHIBIDO** incluir etiquetas de coordinación interna (`OA-XX`, `cajones`, `slices`). Usar lenguaje de dominio puro (ej. `feat(exercises): implementar importacion masiva`).

---

## 4. Flujo de Trabajo y Pull Requests

1. **Ramas:** Crear ramas descriptivas partiendo de `main` (ej: `feat/bulk-import-exercises` o `fix/submission-keystroke-event`).
2. **Verificación Local:** Antes de abrir un PR, ejecutar:
   - `cd backend && go vet ./... && go test ./...`
   - `cd frontend && npm run lint:styles && CI=true npm run build`
3. **Pull Request:** Abrir un Pull Request hacia `main` usando el template `.github/PULL_REQUEST_TEMPLATE.md`.
4. **Revisión y CI:** El pipeline de CI debe finalizar en verde (6 stages) y contar con la aprobación de al menos un CODEOWNER (`@AlvaroRiveraCarhuani`).
