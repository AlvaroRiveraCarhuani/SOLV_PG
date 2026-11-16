# ADR-082: Préstamo de Memoria en la Admisión y Techo Único de Cuota por Entorno

## Estado
Aceptado

## Contexto

Dos reglas distintas compartían el mismo número. `HardQuotaMemoryMB` valía 2048 MB y a la vez era el techo que el auto-bursting podía alcanzar (`qos_worker.go`) y el máximo del perfil EWMA (`lab_template_repository.go`). La cuota de un entorno nuevo no lo miraba: `startPracticeEnvironment` toma `template.BaseRamMB`, y desde ADR-043 la pantalla del administrador ofrece perfiles de hasta 4096 MB.

El resultado eran tres respuestas para la misma pregunta: una plantilla de 4 GB corría con 4 GB, el perfilador declaraba 2 GB como máximo y el auto-bursting se detenía en 2 GB.

Al mismo tiempo, el nodo no repartía lo que sobraba. Con cuotas de 512 MB y dos alumnos conectados sobre un servidor de 12 GB quedaban más de 8 GB sin usar. El auto-bursting no lo resolvía: reacciona cuando un contenedor ya está al 80% de *su* límite, suma 256 MB por ciclo y se detiene en el techo. Un entorno que necesita 1.5 GB desde el arranque se quedaba corto antes de que el worker lo mirara.

La consecuencia era de lectura: el sistema se veía estático ("siempre 512 MB") aunque el nodo tuviera margen para dar más.

## Decisión

### 1. Un techo por contenedor, un presupuesto para el nodo

`HardQuotaMemoryMB` sube a 4096 MB: es el mayor perfil que el administrador puede definir en una plantilla y el máximo que el auto-bursting puede alcanzar. La protección del servidor no la da ese número, la da el presupuesto de admisión sobre la RAM total medida.

### 2. La admisión concede base más préstamo

`loanQuotaMB` calcula:

```text
presupuesto = RAM total × techo de seguridad − comprometido − satélites
```

La base la fijan la política del tenant y la plantilla, y es un piso: nadie recibe menos. Sobre esa base se concede el perfil más alto de la lista (512, 768, 1024, 1536, 2048, 4096) que cumpla dos condiciones: no supera el techo duro y deja sitio para otro entorno del mismo tamaño (2 × perfil ≤ presupuesto). Si ni la base entra, el entorno se rechaza con `host_memory_exhausted`, igual que antes.

Efecto en un nodo de 12 GB: el primer entorno recibe 4096 MB, el segundo 2048, el tercero 1536 y, a medida que entra la clase, la cuota vuelve a la base. En uno de 8 GB el primero recibe 2048.

### 3. El préstamo no sobrevive a la hibernación

Al despertar, `reactivatePracticeEnvironment` vuelve a decidir la cuota con el presupuesto del momento (`loanQuotaForReactivation`). Un entorno que se durmió con 4096 MB puede volver con 512 si la clase llenó el nodo: la memoria prestada se devuelve al hibernar. Nunca se le quita memoria a un contenedor en ejecución, porque bajar el límite en caliente hace que el kernel mate el proceso que se pase.

El entorno que despierta no se cuenta dos veces: se excluye de lo comprometido, porque al hibernar dejó de ocupar cupo.

### 4. La decisión se serializa

`reserveAdmissionQuota` toma un mutex y aparta el monto concedido mientras el entorno no tiene fila. Sin eso, treinta aperturas en el primer minuto de clase leen el mismo hueco y todas se quedan con el préstamo. La reserva se libera cuando el registro existe o cuando el alta falla.

### 5. La reserva de plataforma no es una compuerta

`PlatformReserveMB` (2048 MB) se mantiene como supuesto de planificación del panel de capacidad (`RecommendMaxContainers`), no como resta en la admisión. Descontarla en la compuerta negaría entornos que hoy entran; el techo de seguridad ya deja ese margen libre.

## Consecuencias

**A favor**

- La cuota deja de ser un número fijo: con el nodo vacío el alumno recibe hasta ocho veces la base.
- Un solo techo (4096) en código, perfilador y auto-bursting. La pantalla del administrador deja de prometer algo que el backend recortaba.
- El criterio de rechazo no cambia: si la base no entra, el entorno se rechaza como antes.
- El préstamo tiene una salida natural (la hibernación), así que la clase se reequilibra sola sin intervención del administrador.

**En contra**

- El que abre primero recibe más. Es una consecuencia del préstamo, no un olvido: quien llega con la clase llena recibe la base, que es la cuota que el sistema promete a todos.
- La cuota concedida ya no es igual a la que declara la plantilla. El panel muestra el valor real por entorno (`memory_limit_mb`), pero comparar plantilla contra entorno exige leer ese campo.
- El auto-bursting sigue mirando solo el margen libre del host (15%), no el presupuesto de admisión. Con el techo en 4096, un contenedor puede crecer por encima del préstamo si el host tiene aire. Se acepta: el guard del host es la última barrera y el crecimiento es de 256 MB por ciclo.
- La serialización es en proceso. Un despliegue con más de un backend sobre el mismo daemon necesitaría una exclusión en base de datos. Hoy el producto es de un nodo y un proceso.

## Verificación

- `TestLoanQuotaMB` fija la regla en trece escenarios: nodo vacío, clase llena, satélites, décimo alumno, sin host, techo por defecto, base de 4096 y base de 1536.
- `TestReserveAdmissionQuota_NoReparteElMismoHuecoDosVeces` abre dos entornos seguidos y comprueba que el segundo ve tomado lo que el primero apartó; liberar dos veces no devuelve el monto dos veces.
- `TestReserveAdmissionQuota_RechazaSinHuecoParaLaBase` mantiene el rechazo cuando ni la base entra.
- `TestLoanQuotaForReactivation_CaducaElPrestamo` comprueba que un entorno vuelve con la base con la clase llena, con 4096 con el nodo vacío, y que no se cuenta dos veces a sí mismo.

## Referencias

- ADR-008 — asignación y limitación de recursos
- ADR-011 — hibernación y control de admisión
- ADR-043 — gobernanza de recursos vía catálogo de plantillas
- ADR-049 — telemetría del host persistida
- `docs/OPERATIONS/capacidad.md`
- `backend/internal/core/services/practice_environment_capacity.go`
