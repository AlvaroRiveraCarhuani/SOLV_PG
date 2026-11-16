# red-del-examen Specification

## Purpose

Cerrar la salida a internet en el entorno de examen sin dejar al alumno sin editor, con una
excepción declarada y verificable, y con la garantía de que el proyecto base funciona en esas
condiciones.

## Requirements

### Requirement: Salida cerrada por defecto

El entorno de un ejercicio de examen MUST NOT tener salida a internet. El sistema MUST
conectar el contenedor a una red interna sin puerta de enlace y MUST NOT usar
`NetworkMode: none`, porque el editor MUST seguir siendo alcanzable por Traefik.

#### Scenario: Alumno abre el examen
- GIVEN un examen con `exam_network = none`
- WHEN el alumno abre su entorno
- THEN el editor carga normalmente
- AND una consulta a un dominio externo falla por falta de ruta.

#### Scenario: El editor sigue disponible
- GIVEN un examen con `exam_network = none`
- WHEN Traefik sirve el editor
- THEN el alumno alcanza el puerto del editor
- AND no obtiene salida hacia internet.

---

### Requirement: Excepción declarada

El docente MUST poder declarar `exam_network = egress` para un ejercicio, con un motivo
obligatorio. La excepción MUST quedar auditada y MUST mostrarse al alumno como parte de las
condiciones del examen. Sin motivo, el sistema MUST rechazar la excepción.

#### Scenario: Excepción con motivo
- GIVEN un ejercicio que consume una API externa
- WHEN el docente declara `egress` con el motivo
- THEN el examen se guarda con la excepción
- AND el alumno ve que este examen tiene acceso a internet.

#### Scenario: Excepción sin motivo
- GIVEN un pedido de `egress` sin motivo
- WHEN se valida el ejercicio
- THEN el sistema rechaza el cambio
- AND explica que la excepción exige un motivo.

---

### Requirement: Verificación del proyecto base sin red

Al publicar una práctica de examen con `exam_network = none`, el sistema MUST compilar y
ejecutar el proyecto base en un contenedor sin salida. Si el proyecto necesita red, la
publicación MUST rechazarse indicando qué dependencia o comando la necesitó.

#### Scenario: Proyecto que funciona sin red
- GIVEN un proyecto base con dependencias incluidas en la imagen
- WHEN el docente publica la práctica de examen
- THEN la verificación compila y ejecuta el proyecto
- AND la publicación se acepta.

#### Scenario: Proyecto que necesita red
- GIVEN un proyecto base cuyo arranque ejecuta una instalación remota
- WHEN el docente publica la práctica
- THEN la verificación falla
- AND la publicación se rechaza indicando el paso que necesitó red.

---

### Requirement: Satélites accesibles sin salida

Cuando el examen declare un motor gestionado, el sistema MUST adjuntar el satélite por una
red interna, de modo que el alumno lo alcance sin obtener salida a internet.

#### Scenario: Examen con base de datos
- GIVEN un examen con `exam_network = none` y un satélite PostgreSQL
- WHEN el alumno ejecuta su proyecto
- THEN la conexión al satélite funciona
- AND la salida a internet sigue cerrada.
