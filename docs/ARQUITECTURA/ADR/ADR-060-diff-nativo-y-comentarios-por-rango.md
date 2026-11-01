# ADR 060: Diff Nativo y Comentarios por Rango en SpeedGrader

> **Estado:** Implementado
> **Fecha:** 2026-11-01
> **Ámbito:** Docencia, SpeedGrader, Revisión por modalidad
> **Relacionado:** ADR-043 (modalidad dual), ADR-052 (discriminador de entorno), ADR-053 (revisión docente), ADR-058 (avance del docente), [REVISION_POR_MODALIDAD](../../UI/DOCENTE/REVISION_POR_MODALIDAD.md)

---

## Contexto

La revisión por modalidad ya separa el SpeedGrader de juez y el de IDE, muestra el árbol del snapshot
y traduce los comentarios a la línea global de `code`. Quedaban dos huecos pedagógicos:

1. El diff de juez era una tabla propia que reconstruía líneas con una heurística de diff, no una
   vista nativa del editor. En IDE no había diff: el payload enviaba `boilerplate` vacío y no existía
   una base con la que comparar el proyecto.
2. Un comentario solo podía apuntar a una línea. Los errores de arquitectura, transacciones SQL o
   bloques `try/catch` abarcan varias líneas; obligar a una sola línea produce texto como “desde acá
   hasta la 18” en lugar de un ancla real.

## Decisión

### 1. Diff nativo con CodeMirror

Se agrega `@codemirror/merge` y un componente compartido `code-diff-viewer` que usa
`unifiedMergeView` en modo solo lectura. El docente lee los bloques eliminados y agregados; no acepta
ni rechaza cambios porque no está editando la entrega.

- **Juez:** compara `exercises.boilerplate` contra `submissions.code`.
- **IDE:** compara el archivo activo contra su homónimo en `base_files`.
- El tema del diff usa los tokens del sistema (`--color-success*`, `--color-danger*`); los nombres
  internos de CodeMirror viven en el tema de TypeScript, no en SCSS de componente.

### 2. El proyecto base de IDE se guarda en `exercises.boilerplate`

El paso de publicación de un ejercicio IDE muestra un editor `ide-base-snapshot` con el mismo formato
del snapshot de entrega:

```text
# === FILE: main.py ===
def fetch_items():
    return []
```

No se agrega una tabla ni un segundo formato. El read model de revisión separa `boilerplate` con
`ParseProjectSnapshot` y devuelve `base_files[]` y `base_files_available`. Si el ejercicio no tiene
base reconocible, la vista oculta el diff; no inventa una comparación contra la plantilla Docker ni
contra la primera entrega.

### 3. Comentarios por rango con `end_line_number`

La migración `00043_submission_comment_ranges.sql` agrega `end_line_number INT NULL` y una restricción
`end_line_number >= line_number`. No hay backfill: los comentarios existentes quedan como comentarios
de una línea.

- `line_number` sigue siendo la línea global inicial sobre `code`; `end_line_number` es opcional.
- En IDE, el visor traduce el bloque local a líneas globales y el backend exige que ambas caigan en
  el mismo archivo. Un rango que cruza archivos o cabeceras responde `422 line_range_spans_files`; uno
  invertido responde `422 line_range_invalid`; uno fuera de `code`, `422 line_out_of_range`.
- Un comentario de una sola línea puede seguir cayendo en una cabecera (por ejemplo, la nota general
  de la rúbrica en la línea 1).
- La UI muestra `L12` o `L12–L18` según el rango y conserva el ancla global, así que no requiere
  migrar la tabla ni reescribir los comentarios viejos.

## Consecuencias

- El docente ve exactamente qué cambió el alumno respecto del proyecto base, sin una tabla de diff
  paralela.
- Un ejercicio IDE sin proyecto base no tiene diff. Es un estado visible y advertido en el paso de
  publicación, no un fallo silencioso.
- El payload de revisión crece con `base_files`; queda acotado por el mismo límite de 2 MiB y 500
  archivos del parser.
- Los comentarios de bloque se pueden renderizar en la extensión de VS Code con el mismo contrato
  (`line_number` + `end_line_number`) sin un campo nuevo por archivo.

## Alternativas descartadas

| Alternativa | Motivo |
| --- | --- |
| Seguir con la tabla de diff propia | Duplica el motor de diff, no resalta bloques como el editor y obliga a mantener dos representaciones |
| Comparar IDE contra la imagen Docker de la plantilla | La imagen no expone el árbol de archivos del proyecto base y no es un dato de la práctica |
| Comparar contra la primera entrega del alumno | La primera entrega no es una base pedagógica; castiga al alumno que corrige |
| Guardar `file_path` y línea local en los comentarios | Requiere migración y backfill; la línea global ya funciona y la traducción vive en la vista. Se puede hacer después si la extensión lo necesita |
| Codificar el rango dentro de `comment` | Contamina el texto pedagógico y rompe la lectura de comentarios existentes |
