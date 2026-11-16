# dimensionamiento-practica Specification

## Purpose

Definir el límite de CPU por Entorno de Desarrollo, los perfiles de memoria elegidos con
medición del nodo piloto y una lectura única de la cuota de cada entorno.

## Requirements

### Requirement: Límite de CPU configurable por entorno

El sistema MUST aplicar un límite de CPU absoluto (`NanoCPUs`) al crear el contenedor de un
Entorno de Desarrollo. El valor MUST venir de la política del servidor y MUST NOT estar
escrito en el código de infraestructura. El valor `0` MUST significar sin límite y MUST
reproducir el comportamiento anterior. El sistema MUST NOT modificar el límite de CPU de un
contenedor en ejecución.

#### Scenario: Límite aplicado al crear
- GIVEN una política con `practice_cpu_millis = 1000`
- WHEN se crea un Entorno de Desarrollo
- THEN el contenedor tiene `NanoCPUs = 1_000_000_000`
- AND la respuesta del entorno informa `cpu_millis = 1000`.

#### Scenario: Límite apagado
- GIVEN una política con `practice_cpu_millis = 0`
- WHEN se crea un Entorno de Desarrollo
- THEN el contenedor no define `NanoCPUs`
- AND el comportamiento es el anterior al cambio.

#### Scenario: Entorno existente conserva su límite
- GIVEN un entorno creado con `NanoCPUs = 1`
- WHEN la política cambia a 2
- THEN el contenedor en ejecución conserva 1
- AND el valor nuevo se aplica al recrearlo.

---

### Requirement: Límite de procesos

El sistema MUST fijar `PidsLimit` en el contenedor de un Entorno de Desarrollo, con el
mismo valor que usa el entorno de revisión, salvo que la medición demuestre que corta una
compilación del proyecto de referencia.

#### Scenario: Compilación dentro del límite
- GIVEN un entorno con `PidsLimit` vigente
- WHEN el alumno compila el proyecto de referencia
- THEN la compilación termina
- AND no se registra un error de procesos.

---

### Requirement: Perfiles de memoria elegidos con medición

Los perfiles del préstamo y la base de memoria MUST fijarse a partir de la medición de
`medicion.md`, con la regla declarada: base = p95 del consumo en régimen más 25% de margen,
redondeada al perfil inmediato superior. El sistema MUST NOT cambiar esos valores sin una
medición registrada.

#### Scenario: Base ajustada tras medir
- GIVEN una medición que registra p95 de 610 MB en régimen
- WHEN se fija la base
- THEN la base es el perfil inmediato superior a 762 MB
- AND la planilla de medición queda adjunta al ADR.

#### Scenario: Sin medición no se cambia el número
- GIVEN un pedido de subir la base sin planilla
- WHEN se revisa el cambio
- THEN el cambio se rechaza por falta de medición.

---

### Requirement: Una sola lectura de la cuota

La interfaz MUST mostrar, para cada entorno, el perfil declarado por la plantilla, la cuota
concedida por la admisión y el límite real del contenedor, cada uno con su nombre. El
sistema MUST NOT presentar el perfil de la plantilla como si fuera la cuota aplicada.

#### Scenario: Cuota menor a la declarada
- GIVEN una plantilla de 4096 MB y un entorno con `memory_limit_mb = 512`
- WHEN el docente abre la ficha del entorno
- THEN ve "perfil declarado 4096 MB" y "cuota concedida 512 MB"
- AND ve un aviso de que la cuota se ajustó por el espacio del nodo.

#### Scenario: Cuota igual a la declarada
- GIVEN una plantilla de 1024 MB y un entorno con `memory_limit_mb = 1024`
- WHEN el docente abre la ficha
- THEN no aparece ningún aviso de ajuste.

---

### Requirement: Medición obligatoria antes de aplicar

El cambio MUST completar la planilla de `medicion.md` en el nodo piloto con tres corridas
por escenario y valores p50 y p95 antes de fijar cualquier número. Los valores no medidos
MUST declararse como no medidos.

#### Scenario: Medición incompleta
- GIVEN una planilla con escenarios sin corridas
- WHEN se intenta aplicar el dimensionamiento
- THEN el ADR registra el escenario como no medido
- AND el número correspondiente no se cambia.
