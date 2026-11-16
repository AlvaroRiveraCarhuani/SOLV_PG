# Protocolo de Validación Piloto y Matrices de Prueba — SOLV

> **Estado Inicial del Entorno:** Base de datos sin semillas pre-cargadas de plantillas ni componentes (`Zero-Seed State`).
> **Propósito:** Validar el sistema de punta a punta simulando escenarios académicos reales de distintos semestres y especialidades universitarias, registrando hallazgos técnicos, fricciones de interfaz y correcciones aplicadas.

---

## Estructura de Dominios de Prueba

El protocolo sigue el orden causal estricto de operación de la plataforma:

1. **[`01-ADMIN/`](01-ADMIN/)** — Gobernanza, Catálogo de Componentes, Constructor de Plantillas, Cuotas y Docentes.
   - **[`01-CONSTRUCTOR_PLANTILLAS/`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/00-INDICE_MATRIZ.md)**:
     - [`00-INDICE_MATRIZ.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/00-INDICE_MATRIZ.md) — Tablero general y hallazgos de interfaz/arquitectura.
     - **[`01-LABORATORIO_PRACTICA/`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/01-LABORATORIO_PRACTICA/)** *(Entornos `IDE_PERSISTENTE` con Editor Visual)*:
       - [`01-NIVEL_1_FUNDAMENTOS.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/01-LABORATORIO_PRACTICA/01-NIVEL_1_FUNDAMENTOS.md) (Fácil, Medio, Difícil)
       - [`02-NIVEL_2_MONOLENGUAJE_COMPLEMENTOS.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/01-LABORATORIO_PRACTICA/02-NIVEL_2_MONOLENGUAJE_COMPLEMENTOS.md) (Fácil, Medio, Difícil)
       - [`03-NIVEL_3_FULLSTACK_POLIGLOTA.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/01-LABORATORIO_PRACTICA/03-NIVEL_3_FULLSTACK_POLIGLOTA.md) (Fácil, Medio, Difícil)
       - [`04-NIVEL_4_ESPECIALIDADES_ACADEMICAS.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/01-LABORATORIO_PRACTICA/04-NIVEL_4_ESPECIALIDADES_ACADEMICAS.md) (Fácil, Medio, Difícil)
       - [`05-NIVEL_5_EDGE_CASES_SEGURIDAD.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/01-LABORATORIO_PRACTICA/05-NIVEL_5_EDGE_CASES_SEGURIDAD.md) (Fácil, Medio, Difícil)
     - **[`02-EVALUADOR_AUTOMATICO/`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/02-EVALUADOR_AUTOMATICO/)** *(Entornos `JUEZ_EFIMERO` sin interfaz gráfica)*:
       - [`01-NIVEL_1_FUNDAMENTOS_JUEZ.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/02-EVALUADOR_AUTOMATICO/01-NIVEL_1_FUNDAMENTOS_JUEZ.md) (Fácil, Medio, Difícil)
       - [`02-NIVEL_2_LIBRERIAS_Y_ESTRUCTURAS.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/02-EVALUADOR_AUTOMATICO/02-NIVEL_2_LIBRERIAS_Y_ESTRUCTURAS.md) (Fácil, Medio, Difícil)
       - [`03-NIVEL_3_EDGE_CASES_AISLAMIENTO.md`](01-ADMIN/01-CONSTRUCTOR_PLANTILLAS/02-EVALUADOR_AUTOMATICO/03-NIVEL_3_EDGE_CASES_AISLAMIENTO.md) (Fácil, Medio, Difícil)
2. **[`02-DOCENTE/`](02-DOCENTE/)** — Apertura de aulas, publicación de prácticas, revisión, seguimiento y libreta.
   - [`00-INDICE_MATRIZ.md`](02-DOCENTE/00-INDICE_MATRIZ.md) — Alcance, requisitos, reglas de evidencia y criterio de cierre del rol.
   - [`01-NIVEL_1_OPERACION_BASICA.md`](02-DOCENTE/01-NIVEL_1_OPERACION_BASICA.md) — Aula, práctica de evaluación y práctica de desarrollo.
   - [`02-NIVEL_2_EVALUACION_Y_REVISION.md`](02-DOCENTE/02-NIVEL_2_EVALUACION_Y_REVISION.md) — Cola de revisión, rúbrica y ajuste manual.
   - [`03-NIVEL_3_AULA_Y_SEGUIMIENTO.md`](02-DOCENTE/03-NIVEL_3_AULA_Y_SEGUIMIENTO.md) — Estudiantes trabados, avances y cierre de examen.
   - [`04-NIVEL_4_CONTENIDO_Y_REUTILIZACION.md`](02-DOCENTE/04-NIVEL_4_CONTENIDO_Y_REUTILIZACION.md) — Banco de prácticas, solicitud de entorno y libreta.
   - [`05-NIVEL_5_LIMITES_Y_RECUPERACION.md`](02-DOCENTE/05-NIVEL_5_LIMITES_Y_RECUPERACION.md) — Aislamiento, revisión concurrente y entorno no disponible.
3. **[`03-ESTUDIANTE/`](03-ESTUDIANTE/)** — Inicio, materias, evaluación algorítmica, entorno de desarrollo, historial y límites.
   - [`00-INDICE_MATRIZ.md`](03-ESTUDIANTE/00-INDICE_MATRIZ.md) — Alcance, requisitos, reglas de evidencia y criterio de cierre del rol.
   - [`01-NIVEL_1_ACCESO_Y_ORIENTACION.md`](03-ESTUDIANTE/01-NIVEL_1_ACCESO_Y_ORIENTACION.md) — Inicio, materias y aula activa.
   - [`02-NIVEL_2_EVALUACION_ALGORITMICA.md`](03-ESTUDIANTE/02-NIVEL_2_EVALUACION_ALGORITMICA.md) — Casos públicos, envío, error y corrección.
   - [`03-NIVEL_3_ENTORNO_DE_DESARROLLO.md`](03-ESTUDIANTE/03-NIVEL_3_ENTORNO_DE_DESARROLLO.md) — Apertura, persistencia, entrega y examen.
   - [`04-NIVEL_4_HISTORIAL_Y_RESULTADOS.md`](03-ESTUDIANTE/04-NIVEL_4_HISTORIAL_Y_RESULTADOS.md) — Historial, rúbrica y comentarios.
   - [`05-NIVEL_5_LIMITES_Y_CASOS_BORDE.md`](03-ESTUDIANTE/05-NIVEL_5_LIMITES_Y_CASOS_BORDE.md) — Sin entrega, doble envío y aislamiento.
4. **[`smoke-test-guide.md`](smoke-test-guide.md)** — Guía rápida de verificación pre-despliegue.

> **Estado de las matrices:** Docente y Estudiante tienen 15 casos cada una, distribuidos en cinco niveles. La ejecución está `PENDIENTE`; los estados se registran en la tabla de cierre de cada caso. El piloto no inicia hasta cerrar los bloqueos del [checklist pre-piloto](../SECURITY/pre-pilot-checklist.md).
