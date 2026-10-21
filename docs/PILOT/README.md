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
2. **`02-DOCENTE/`** *(Próxima fase)* — Solicitud de entornos, creación de laboratorios, configuración del Juez Virtual, rúbricas y SpeedGrader.
3. **`03-ESTUDIANTE/`** *(Próxima fase)* — Conexión a contenedores OpenVSCode Server, persistencia de volúmenes, envíos al Juez Virtual y límites de memoria.
4. **[`smoke-test-guide.md`](smoke-test-guide.md)** — Guía rápida de verificación pre-despliegue.
