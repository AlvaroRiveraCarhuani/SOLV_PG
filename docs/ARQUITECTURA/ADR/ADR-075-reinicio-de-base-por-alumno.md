# ADR 075: Reinicio de la Base de una Práctica por Alumno

> **Estado:** Implementado
> **Fecha:** 2026-11-13
> **Ámbito:** Docencia, Servicios satélite
> **Relacionado:** ADR-051 (PostgreSQL de aula como satélite dedicado), ADR-059 (entornos acotados a una práctica), ADR-071 (alcance del script), ADR-073 (script en almacenamiento propio), ADR-074 (aplicación en una transacción)

---

## Contexto

El script de inicialización se aplica una sola vez por alumno y por práctica. El sello vive en
`_solv_exercise_migrations`, dentro de la base aislada de ese alumno. Es lo que evita reaplicar el
script en cada arranque, pero también lo que deja a los alumnos con la base vieja cuando el docente
corrige el script después de que ya abrieron la práctica.

Hasta ahora no había forma de resolverlo desde la interfaz. El docente podía editar el script, pero
no volver a aplicarlo en un alumno puntual sin intervención manual sobre el satélite.

## Decisión

El docente ve, en la página de la práctica, qué alumnos ya tienen base y puede reiniciar una.

- `GET /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/student-databases` lista los
  entornos de la práctica con nombre y correo del alumno.
- `POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/students/{studentId}/database/reset`
  borra la base aislada del alumno y la vuelve a crear con el script vigente.

El reinicio ejecuta `DROP DATABASE ... WITH (FORCE)`, recrea rol, base y permisos, y aplica el script
con el mismo camino que usa el arranque del laboratorio. Al desaparecer la base desaparece el sello,
así que el script se aplica de nuevo. **El volumen del alumno no se toca**: su código, su repositorio
y sus archivos quedan intactos.

### Reglas de acceso

- Solo docentes y administradores.
- El docente debe ser titular de la materia, con la misma comprobación que la prueba de entorno.
- Solo se reinician alumnos que ya abrieron la práctica: no se crean bases al azar.
- La operación pide confirmación explícita en la interfaz, porque borra datos.

### Alcance por motor

- PostgreSQL y MySQL/MariaDB: soportado.
- Motores sin base aislada por alumno o con host externo: se rechaza con un mensaje explícito en
  lugar de improvisar un borrado.

## Consecuencias

- El docente resuelve el caso sin tocar el servidor ni pedir soporte.
- El reinicio es destructivo por diseño: se pierden los datos que el alumno hubiera cargado en su
  base. El código no se pierde. La confirmación y el alcance acotado al alumno son la mitigación.
- Si el script nuevo falla, el alumno queda con una base vacía. La prueba de entorno permite
  detectarlo antes de reiniciar; el error del script no se revierte de forma automática.
- La lista crece con la cantidad de alumnos que abrieron la práctica. Es una lectura por práctica,
  acotada por materia y práctica, con orden por nombre.

## Estimación de tiempo en el editor

La misma pantalla del script muestra una estimación por alumno. Está calibrada con mediciones reales
sobre el PostgreSQL de aula con la configuración del satélite (`postgres:18-alpine`, 384 MB,
`shared_buffers=96MB`) y la transacción de ADR-074 ya aplicada:

| Sentencias | Mediana medida |
| ---: | ---: |
| 100 | 0,14 s |
| 500 | 0,18 s |
| 1.000 | 0,22 s |
| 5.000 | 0,64 s |
| 20.000 | 2,05 s |

La curva es `0,1 s + sentencias / 10.000`, con tres corridas por tamaño. Es una estimación, no una
promesa: el disco y la carga del servidor pueden moverla.

## Verificación

- `backend/internal/core/services/practice_env_test_databases_test.go`: listado, reinicio de un
  alumno que abrió la práctica, rechazo de un alumno que no abrió y rechazo de un docente ajeno.
- `go build ./...`, `go vet ./...` y `go test ./internal/core/services/ -run PracticeEnvTest`.
- `npx ng build` para el tipo de las plantillas y `npm run lint:styles` para los estilos.
