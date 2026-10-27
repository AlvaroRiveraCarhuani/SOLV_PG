# Verificación — Slice 17: Telemetría Verdadera

Matriz ejecutada al cierre del feature (2026-10-27).

| Gate | Resultado |
|------|-----------|
| `go vet ./...` | OK |
| `go test ./internal/...` | 12 paquetes OK (incluye health 9, sampler 4, reconciler 7, handler 7) |
| `go test ./tests/integration/` | OK contra `solv_test`, incluida la migración 00028 y el precheck semgrep con escaneos reales |
| `npm run lint:styles` | OK (gates de hex y máquina de datos) |
| `npm run lint:ts` | OK |
| `CI=true npm run build` | OK |
| `npm run test:ci` | 105 archivos / 785 tests + 26 de gates |

## Criterios de aceptación verificados

- Pausar un laboratorio incrementa `hibernated_labs` (conteo real, no lista de running).
- Matar un contenedor a mano produce `orphaned` en el siguiente ciclo; detener el daemon produce `unknown` sin destruir registros.
- `docker stop` con la BD limpia lleva el badge a `critical` con motivo `docker_unreachable`.
- Reiniciar el backend no inventa historia: el gráfico arranca con muestras reales persistidas.
- Con 5.8 GB usados de 7.6 GB el panel deja de informar capacidad imposible (`13`) y explica el límite con `capacity_basis`.