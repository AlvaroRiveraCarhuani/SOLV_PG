# SLICE 13 — Contratos de API (Capa Docente BaaS)

## Resumen de Endpoints Implementados

| # | Endpoint | Método | Descripción |
| :-: | :--- | :---: | :--- |
| 1 | `/api/v1/exercises` | `POST` | Creación de ejercicio en borrador (`draft`) |
| 2 | `/api/v1/exercises/{id}` | `PUT` | Actualización de ejercicio |
| 3 | `/api/v1/exercises/{id}/test-cases/bulk` | `POST` | Importación masiva de casos de prueba (CSV / JSON) |
| 4 | `/api/v1/exercises/{id}/publish` | `POST` | Transición `draft` -> `published` (valida $\ge 1$ caso público) |
| 5 | `/api/v1/teacher/courses` | `GET` | Listado de materias con métricas agregadas (`students`, `active`, `pending`, `at_risk`) |
| 6 | `/api/v1/teacher/attention` | `GET` | Widget de atención prioritaria por severidad (`critical`, `warning`, `standard`) |
| 7 | `/api/v1/teacher/courses/{id}/labs` | `GET` | Estadísticas por laboratorio de un curso |
| 8 | `/api/v1/teacher/courses/{id}/submissions` | `GET` | Cola de entregas con filtros por veredicto y ejercicio |
| 9 | `/api/v1/teacher/submissions/{id}/review` | `GET` | Detalle SpeedGrader con casos privados desenmascarados y punteros prev/next |
| 10 | `/api/v1/submissions/{id}/override` | `POST` | Override manual de calificación con validación $\ge 10$ caracteres |
| 11 | `/api/v1/teacher/submissions/{id}/comments` | `POST` | Agregar comentario pedagógico anclado a línea de código |
| 12 | `/api/v1/teacher/submissions/{id}/comments` | `GET` | Listar comentarios in-line de una entrega |
| 13 | `/api/v1/teacher/submissions/{id}/run-ephemeral` | `POST` | Runner efímero en sandbox sin persistir submission |
| 14 | `/api/v1/teacher/courses/{id}/grades/export` | `GET` | Exportación de calificaciones en CSV con UTF-8 BOM |

Consulte la especificación completa en [`docs/API_CONTRACTS.md`](../../API_CONTRACTS.md).
