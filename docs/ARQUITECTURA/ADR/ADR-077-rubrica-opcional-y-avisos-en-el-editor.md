# ADR 077: Rúbrica Opcional y Avisos en el Editor de Prácticas

> **Estado:** Implementado
> **Fecha:** 2026-11-13
> **Ámbito:** Docencia, Editor de prácticas
> **Relacionado:** ADR-071 (alcance del script), ADR-074 (script en una transacción), ADR-075 (reinicio de base por alumno), ADR-076 (despublicar)

---

## Contexto

La rúbrica se exigía para publicar en el entorno de desarrollo. No todas las prácticas la usan:
algunas se califican a mano o solo sirven como espacio de trabajo. El bloqueo obligaba a inventar
criterios para poder publicar.

El docente tampoco veía el estado de la prueba del entorno desde el editor, y al editar una práctica
ya publicada no había ningún aviso de que cada guardado impacta a los alumnos.

## Decisión

**Rúbrica opcional.** Deja de ser requisito de publicación. Sin criterios, la práctica se publica y
se califica a mano; el checklist lo informa como dato, no como advertencia. Si la rúbrica existe y
los pesos no suman 100%, el checklist lo advierte, pero tampoco bloquea.

**Avisos en el editor.**

- Con la práctica publicada se muestra un banner: cada cambio guardado se aplica a los alumnos.
- Junto al script de base de datos se avisa que quienes ya abrieron conservan su base y que el
  reinicio por alumno está en la página de la práctica.
- El editor incorpora "Probar entorno", que guarda el borrador y ejecuta la misma prueba real que la
  página de la práctica.
- La confirmación de publicación resume el resultado de la última prueba, incluido el error del
  script cuando falló.

## Consecuencias

- Publicar deja de exigir una rúbrica que no todos usan; los pesos se advierten sin bloquear.
- El error del script se ve antes de publicar por la prueba real, no por una validación sintáctica.
- Los cambios sobre una práctica publicada dejan de ser silenciosos.

## Alternativas descartadas

- **Exigir rúbrica solo si hay entregas.** Se define antes de publicar; condicionarla a entregas
  existentes llegaba tarde.
- **Validar el SQL al guardar.** Solo la ejecución real detecta errores de sintaxis y de permisos;
  hacerlo en cada autoguardado costaría un contenedor por cambio. Se resuelve con la prueba manual.
- **Bloquear la edición de prácticas publicadas.** Impedía corregir erratas; el aviso y la prueba
  previa cubren el riesgo.

## Verificación

- Tests de frontend (1.122) y build de producción.
- La prueba de entorno y el despublicar requieren el stack completo para su verificación manual.
