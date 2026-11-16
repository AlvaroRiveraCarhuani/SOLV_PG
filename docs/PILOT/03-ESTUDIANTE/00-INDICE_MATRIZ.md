# Matriz de Piloto — Estudiante

> **Rutas base:** `/student`, `/student/aula`, `/student/courses`, `/student/courses/:courseId`, `/student/evaluations`, `/student/history`, `/student/history/:submissionId`, `/student/exercises/:exerciseId`
> **Estado de ejecución:** `PENDIENTE` (matriz redactada, sin corrida registrada)
> **Referencias:** [Protocolo de piloto](../README.md), [Checklist pre-piloto](../../SECURITY/pre-pilot-checklist.md), [ADR-044](../../ARQUITECTURA/ADR/ADR-044-piloto-con-usuarios-reales.md)

---

## 1. Propósito

Validar que un estudiante pueda orientarse, entrar a clase, resolver una práctica de evaluación, trabajar en un entorno de desarrollo, entregar y consultar su historial sin depender del docente para cada paso. La matriz prioriza el resultado observable del estudiante y deja explícitos los límites que no debe poder cruzar.

## 2. Requisitos de entrada

| Requisito | Condición |
| :--- | :--- |
| Cuenta estudiante | Rol `student`, matriculada en `DOC-PILOTO-01` |
| Docente | Cuenta con aula abierta y prácticas publicadas |
| Práctica de evaluación | Casos públicos y privados, lenguaje disponible |
| Práctica de desarrollo | Entorno homologado aprobado y rúbrica configurada |
| Examen | Práctica con propósito `Examen / Calificado` y sesión de aula abierta |
| Operación | Checklist pre-piloto sin ítems `BLOQUEANTE` abiertos |

Los datos de prueba deben ser ficticios. El estudiante no recibe acceso a nombres, correos ni entregas de otros estudiantes.

## 3. Niveles

| Nivel | Archivo | Foco | Casos |
| :--- | :--- | :--- | :--- |
| **Nivel 1: Acceso y orientación** | [`01-NIVEL_1_ACCESO_Y_ORIENTACION.md`](01-NIVEL_1_ACCESO_Y_ORIENTACION.md) | Inicio, materias y aula activa | EST-1.1, EST-1.2, EST-1.3 |
| **Nivel 2: Evaluación algorítmica** | [`02-NIVEL_2_EVALUACION_ALGORITMICA.md`](02-NIVEL_2_EVALUACION_ALGORITMICA.md) | Casos públicos, envío, error y corrección | EST-2.1, EST-2.2, EST-2.3 |
| **Nivel 3: Entorno de desarrollo** | [`03-NIVEL_3_ENTORNO_DE_DESARROLLO.md`](03-NIVEL_3_ENTORNO_DE_DESARROLLO.md) | Apertura, persistencia, entrega y examen | EST-3.1, EST-3.2, EST-3.3 |
| **Nivel 4: Historial y resultados** | [`04-NIVEL_4_HISTORIAL_Y_RESULTADOS.md`](04-NIVEL_4_HISTORIAL_Y_RESULTADOS.md) | Historial, rúbrica y comentarios | EST-4.1, EST-4.2, EST-4.3 |
| **Nivel 5: Límites y casos borde** | [`05-NIVEL_5_LIMITES_Y_CASOS_BORDE.md`](05-NIVEL_5_LIMITES_Y_CASOS_BORDE.md) | Sin entrega, doble envío y aislamiento | EST-5.1, EST-5.2, EST-5.3 |

Cada nivel contiene tres subcasos: `Fácil`, `Medio` y `Difícil`. El subnivel describe la complejidad operativa, no la prioridad: los tres deben ejecutarse.

## 4. Reglas transversales

1. **Una acción por paso.** Cada fila produce un resultado observable antes de continuar.
2. **Evidencia mínima.** Captura de pantalla o registro de red con fecha, hora, usuario y ruta. Para envíos, el identificador de la entrega.
3. **Sin datos reales.** Nombres, correos y matrículas ficticios con el prefijo `EST-PILOTO`.
4. **Estados explícitos.** Un caso termina en `APROBADO`, `FALLA` o `BLOQUEANTE`. `PENDIENTE` solo mientras no se haya ejecutado.
5. **Un fallo bloqueante detiene la fase.** Si el estudiante no puede entrar a clase, enviar o abrir su entorno, no se avanza al siguiente nivel.
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

El rol Estudiante se considera validado cuando los 15 casos están en `APROBADO` y no queda ningún `BLOQUEANTE` abierto. Un `FALLA` no bloqueante se registra como hallazgo, se corrige y se vuelve a ejecutar el caso afectado; no se compensa con la aprobación de otro caso.
