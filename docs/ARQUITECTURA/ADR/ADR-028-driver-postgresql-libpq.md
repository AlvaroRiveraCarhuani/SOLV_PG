# ADR-028: Selección del Driver PostgreSQL (`lib/pq`) frente a `pgx`

## Estado
Aprobado e Implementado.

## Contexto
El driver oficial `github.com/lib/pq` para PostgreSQL en Go ha sido clasificado por la comunidad en estado de mantenimiento pasivo, recomendando la migración hacia `jackc/pgx` para nuevos proyectos comerciales. Sin embargo, la plataforma SOLV utiliza la abstracción `jmoiron/sqlx` combinada con `lib/pq` desde las primeras fases de desarrollo.

## Decisión
Mantener la combinación **`github.com/lib/pq` + `jmoiron/sqlx`** para la versión actual de la plataforma (MVP BaaS e implementación para defensa de tesis académica).

## Justificación Técnica
1. **Estabilidad del Código Probado:** Toda la capa de persistencia (repositorios de materias, entregas, discriminador multitenant, invitaciones docentes transaccionales y audit logs) cuenta con conjuntos de pruebas de integración validadas bajo `sqlx` y `lib/pq`.
2. **Cero Riesgo de Regresión:** La migración hacia `pgx` implicaría adaptar la sintaxis de consultas nombradas (`sqlx.NamedExecContext`) y el manejo de conectores nulos, introduciendo riesgos inaceptables en fases avanzadas del proyecto.
3. **Desempeño Suficiente:** Las pruebas de estrés confirman que la combinación `lib/pq` + `sqlx` responde en submilisegundos en la base de datos local PostgreSQL 18.

## Perspectivas Futuras (Post-Tesis)
La migración transparente hacia `jackc/pgx/v5` (aprovechando sus mejoras de binary protocol y conexión nativa a pool) se programa como una tarea de refactorización para la fase comercial multi-nodo posterior a la defensa académica.
