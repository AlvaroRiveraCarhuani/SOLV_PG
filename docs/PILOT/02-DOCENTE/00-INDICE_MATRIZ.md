# Matriz de Piloto — Docente

> **Rutas base:** `/teacher/dashboard`, `/teacher/aula`, `/teacher/por-calificar`, `/teacher/cursos/:id`, `/teacher/cursos/:courseId/ejercicios/nuevo`, `/teacher/revision/:submissionId`, `/teacher/exercises`, `/teacher/templates`
> **Estado de ejecución:** `PENDIENTE` (matriz redactada, sin corrida registrada)
> **Referencias:** [Protocolo de piloto](../README.md), [Checklist pre-piloto](../../SECURITY/pre-pilot-checklist.md), [ADR-044](../../ARQUITECTURA/ADR/ADR-044-piloto-con-usuarios-reales.md)

---

## 1. Propósito

Validar que un docente pueda abrir una clase, publicar una práctica, acompañar a los estudiantes durante la sesión, revisar entregas y cerrar el ciclo de calificación sin asistencia técnica permanente. La matriz separa la operación normal de los límites del sistema: un caso no se aprueba por “verse bien”, sino por producir el resultado observable indicado.

## 2. Requisitos de entrada

| Requisito | Condición |
| :--- | :--- |
| Cuenta docente | Rol `teacher`, asignada a la materia de prueba y con acceso al período activo |
| Materia | `DOC-PILOTO-01`, con al menos un estudiante matriculado |
| Estudiante | Cuenta `student` de prueba, matriculada en `DOC-PILOTO-01` |
| Plantilla | Un entorno homologado aprobado en el catálogo |
| Práctica | Una práctica de evaluación y una de desarrollo disponibles para publicar |
| Operación | Checklist pre-piloto sin ítems `BLOQUEANTE` abiertos |

Los datos de prueba deben ser ficticios. No se usan correos, matrículas ni nombres de estudiantes reales.

## 3. Niveles

| Nivel | Archivo | Foco | Casos |
| :--- | :--- | :--- | :--- |
| **Nivel 1: Operación básica** | [`01-NIVEL_1_OPERACION_BASICA.md`](01-NIVEL_1_OPERACION_BASICA.md) | Aula, práctica de evaluación y práctica de desarrollo | DOC-1.1, DOC-1.2, DOC-1.3 |
| **Nivel 2: Evaluación y revisión** | [`02-NIVEL_2_EVALUACION_Y_REVISION.md`](02-NIVEL_2_EVALUACION_Y_REVISION.md) | Cola de revisión, rúbrica y ajuste manual | DOC-2.1, DOC-2.2, DOC-2.3 |
| **Nivel 3: Aula y seguimiento** | [`03-NIVEL_3_AULA_Y_SEGUIMIENTO.md`](03-NIVEL_3_AULA_Y_SEGUIMIENTO.md) | Estudiantes trabados, avances y cierre de examen | DOC-3.1, DOC-3.2, DOC-3.3 |
| **Nivel 4: Contenido y reutilización** | [`04-NIVEL_4_CONTENIDO_Y_REUTILIZACION.md`](04-NIVEL_4_CONTENIDO_Y_REUTILIZACION.md) | Banco de prácticas, solicitud de entorno y libreta | DOC-4.1, DOC-4.2, DOC-4.3 |
| **Nivel 5: Límites y recuperación** | [`05-NIVEL_5_LIMITES_Y_RECUPERACION.md`](05-NIVEL_5_LIMITES_Y_RECUPERACION.md) | Aislamiento, revisión concurrente y entorno no disponible | DOC-5.1, DOC-5.2, DOC-5.3 |

Cada nivel contiene tres subcasos: `Fácil`, `Medio` y `Difícil`. El subnivel describe la complejidad operativa, no la prioridad: los tres deben ejecutarse.

## 4. Reglas transversales

1. **Una acción por paso.** Cada fila de la tabla de pasos produce un resultado observable antes de continuar.
2. **Evidencia mínima.** Captura de pantalla o registro de red con fecha, hora, usuario y ruta. Para operaciones de servidor, el identificador devuelto por la API.
3. **Sin datos reales.** Nombres, correos y matrículas deben ser ficticios y consistentes con el prefijo `DOC-PILOTO`.
4. **Estados explícitos.** Un caso termina en `APROBADO`, `FALLA` o `BLOQUEANTE`. `PENDIENTE` solo mientras no se haya ejecutado.
5. **Un fallo bloqueante detiene la fase.** Si el docente no puede abrir el aula, publicar una práctica o calificar, no se avanza al siguiente nivel.
6. **Terminología visible.** En la interfaz se usan los nombres oficiales `Entorno de Evaluación` y `Entorno de Desarrollo`; los identificadores `JUEZ_EFIMERO` e `IDE_PERSISTENTE` quedan para la API y la configuración.

## 5. Registro de evidencia

Cada caso incluye una tabla de cierre. El responsable del piloto la completa al ejecutar:

| Campo | Contenido |
| :--- | :--- |
| **Estado** | `PENDIENTE`, `APROBADO`, `FALLA` o `BLOQUEANTE` |
| **Evidencia** | Ruta o nombre del archivo de captura, registro o respuesta |
| **Ejecutado por** | Nombre y rol |
| **Fecha y hora** | Marca temporal con zona horaria |
| **Observaciones** | Fricción de interfaz, error, comportamiento inesperado o “—” |

## 6. Criterio de cierre del rol

El rol Docente se considera validado cuando los 15 casos están en `APROBADO` y no queda ningún `BLOQUEANTE` abierto. Un `FALLA` no bloqueante se registra como hallazgo, se corrige y se vuelve a ejecutar el caso afectado; no se compensa con la aprobación de otro caso.
