# Proposal: Dimensionamiento medido de RAM y CPU del Entorno de Desarrollo

## Contexto y Motivación

ADR-082 ya resolvió la parte de memoria: `HardQuotaMemoryMB` vale 4096 MB, la admisión
concede base más préstamo (`loanQuotaMB`) con perfiles 512/768/1024/1536/2048/4096, la
decisión se serializa y el préstamo se devuelve al hibernar. `docs/OPERATIONS/capacidad.md`
§8 lo describe.

Quedan tres huecos:

1. **CPU sin techo.** `StartPracticeEnvironmentContainer` fija memoria y
   `MemoryReservation` al 75%, pero no fija `NanoCPUs` ni `PidsLimit`. El contenedor de
   revisión del docente sí los fija (1 núcleo, 512 procesos). Un alumno que compila en su
   Entorno de Desarrollo puede ocupar todos los núcleos del nodo y degradar la clase
   entera, incluida la API.
2. **Números elegidos sin medir.** La regla `2 × perfil ≤ presupuesto` y los perfiles del
   préstamo salieron de aritmética, no de una curva de consumo. Nadie midió cuánta memoria
   necesita el servidor de IDE al arrancar con un proyecto real, ni cuánto pide una
   instalación de dependencias, ni qué pasa con 30 entornos abiertos en menos de un minuto.
3. **Tres números para la misma cuota.** La plantilla declara 4096, el entorno puede
   recibir 512 y Docker tiene el valor concedido. Solo el tercero es real, y ninguna
   pantalla muestra los tres juntos, así que el docente compara contra un número que no se
   aplicó.

## Alcance

- Documentar y ejecutar el **protocolo de medición** del nodo piloto antes de fijar
  cualquier número (`medicion.md`).
- Agregar el **límite de CPU por contenedor** al Entorno de Desarrollo, con el valor como
  política del servidor y no como constante de código.
- Ajustar los **perfiles del préstamo y la base** con las reglas de decisión que salen de la
  medición, no con estimaciones.
- **Reconciliar los tres números** de cuota en la interfaz: perfil de la plantilla, cuota
  concedida al entorno y límite real del contenedor.

## Fuera de Alcance

- El préstamo en sí y la serialización de la admisión: ya existen (ADR-082).
- Cuota de disco e I/O: cambio aparte.
- Separación de perfiles práctica/examen: cambio aparte
  (`separar-practica-y-examen`). Este cambio solo dimensiona el perfil de práctica.
- Ajuste en caliente del límite de memoria de un contenedor en ejecución: la decisión de
  ADR-082 de no bajar memoria en caliente se mantiene.
- Migración de entornos existentes: el nuevo límite de CPU se aplica al crear.

## Capabilities

### New Capabilities

- `dimensionamiento-practica`: límite de CPU por entorno, perfiles de memoria elegidos con
  medición del nodo y una sola lectura de cuota por entorno en la interfaz.

## Riesgos y Plan de Rollback

| Riesgo | Impacto | Control |
|---|---|---|
| Un límite de CPU demasiado bajo alarga la compilación y el alumno lo lee como lentitud de la plataforma | Alto académico | El valor sale de la curva de contención; si el tiempo de compilación con el límite supera 1.5× la referencia sin límite, se sube un escalón |
| Subir la base de memoria reduce la cantidad de entornos simultáneos | Medio | Se recalcula la capacidad recomendada con la fórmula existente y se publica el número nuevo antes de aplicarlo |
| Un contenedor llega al nuevo límite de CPU durante la instalación de dependencias | Medio | El límite se aplica por cuota de CPU, no por `cpu_shares`, y la instalación se mide como caso propio |
| La interfaz muestra la cuota concedida y el docente cree que perdió el perfil de su plantilla | Medio | La pantalla muestra los tres valores con su nombre, y el perfil de la plantilla queda como techo declarado |

**Rollback:** el límite de CPU se apaga con `PRACTICE_CPU_LIMIT=off` (o el valor `0` en la
política), que deja el comportamiento actual: sin `NanoCPUs`. Los perfiles del préstamo se
devuelven al listado anterior en una sola constante. No hay migración de datos ni cambios
de esquema.

## Impacto Documental

- ADR nuevo: "Límite de CPU por entorno y dimensionamiento medido", con referencia cruzada
  a ADR-082; entrada en `docs/ARQUITECTURA/ADR/INDEX.md`.
- `docs/OPERATIONS/capacidad.md`: reemplazar la tabla de supuestos por la tabla medida y
  corregir §4 (hoy admite que no se mide CPU ni arranque simultáneo).
- `openspec/changes/dimensionamiento-practica-ram-cpu/medicion.md`: planilla del nodo
  piloto, con los valores obtenidos.
