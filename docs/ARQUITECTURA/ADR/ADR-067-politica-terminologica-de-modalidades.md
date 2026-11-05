# ADR-067: Política terminológica de modalidades

**Estado:** Aceptado  
**Fecha:** 2026-11-05  
**Relacionado:** ADR-052 (discriminador de modalidad), ADR-065 (vocabulario de entorno de práctica), `AGENTS.md`

## Contexto

La terminología de modalidades estaba escrita a mano en plantillas, utilidades y documentación. Conviven tres capas distintas:

- El identificador de contrato: `IDE_PERSISTENTE`, `JUEZ_EFIMERO` y `AMBOS`.
- El nombre visible de la modalidad: «Entorno de Desarrollo», «Entorno de Evaluación» y «Entorno de Desarrollo y Evaluación».
- El nombre del motor: «Juez Virtual».

Mezclar las capas produjo etiquetas que presentaban el motor o una herramienta como si fueran la modalidad, y variantes que solo cambiaban mayúsculas o acentos.

El gate anterior comparaba contra una lista de nombres conocidos. Esa lista no podía detectar un sinónimo nuevo, una variante de caja o una etiqueta en un encabezado, y eximía archivos completos.

## Decisión

### Catálogo único

Los nombres visibles, el texto corto, la descripción y el nombre del motor viven en `frontend/src/app/shared/constants/environment-types.constants.ts`. La interfaz los consume con el pipe `environmentLabel`, con `environmentTypeOptions({ include })` en filtros y con `environmentBadge` para presentar un valor guardado. El nombre del motor es `EVALUATION_ENGINE_NAME` y no se escribe en plantillas.

### Clasificación por forma

`frontend/scripts/terminology-gate.mjs` clasifica cada texto normalizado (minúsculas, sin acentos, sin comillas ni puntuación de borde) en cuatro clases:

| Clase | Ejemplo | Regla |
| :--- | :--- | :--- |
| `official` | Entorno de Evaluación | Solo el catálogo lo declara; fuera de él es un literal prohibido. |
| `engine` | Juez Virtual | Válido en prosa y en encabezado; nunca como etiqueta de modalidad. |
| `non-official` | «Entorno de Programación», «IDE Persistente», «Evaluador Automático» | Frase con forma de modalidad o de motor que no es nombre válido. |
| `other` | Entorno de trabajo, Espacio de trabajo, prosa | No es una etiqueta. |

La forma se decide por el núcleo (`entorno`, `ambiente`, `modalidad`, `espacio`, `juez`, `evaluador`, `ide`) y por las palabras que lo acompañan. Un núcleo sin calificador de modalidad («Entorno interactivo web») queda en `other`. Los conceptos ajenos conocidos («trabajo», «práctica», «laboratorio», «ejecución», «revisión», «entrega») quedan en `other` aunque compartan el núcleo.

### Reglas documentales

En toda la documentación (`docs/**`) el gate revisa celdas de tabla, encabezados, términos de glosario y la pareja identificador ↔ etiqueta. La prosa que describe el motor se conserva. Los bloques de código cercados no se revisan.

La prosa vigente también se revisa: un nombre de modalidad escrito como etiqueta dentro de una frase («…se corrige en el Entorno de Programación…») falla aunque no ocupe la celda entera. Solo cuenta como inicio del tramo una palabra con mayúscula inicial, de modo que el uso común («el entorno de desarrollo local») no se reporta, y el tramo no atraviesa comillas, código en línea ni marcas de formato. Un nombre entrecomillado o dentro de un enlace a un ADR anterior se toma como cita; un nombre permitido seguido de más palabras no se reporta.

### Excepciones

Una excepción histórica debe ser explícita y verificable:

- **ADR histórico:** solo si su número es menor que el de esta decisión y el texto referencia `ADR-067` con una línea «Terminología vigente: ADR-067» o «Sustituido por ADR-067».
- **Registro de gobernanza:** solo los documentos declarados en `GOVERNANCE_RECORDS`, con ruta exacta, motivo escrito y referencia a `ADR-067`. Dentro de ellos se exime la mención de nombres antiguos del motor y los nombres de modalidad entrecomillados; un sinónimo de modalidad en celda, encabezado o glosario y toda desalineación de identificador siguen detectándose. La prosa de un documento histórico no se clasifica, porque su redacción es parte del registro.
- Las rutas se comparan por igualdad, sin comodines ni sufijos. Máximo cuatro registros de gobernanza.

### Exclusiones del escaneo de código

- Los archivos `*.spec.ts` no se escanean: sus literales son datos de prueba y no llegan a la interfaz.
- El archivo del catálogo no se escanea como consumidor porque es la fuente; en su lugar se validan `label`, `short`, `description`, la clave declarada y el nombre del motor.
- Los bloques de código cercados de la documentación no se escanean: muestran comandos, diagramas y salidas de ejemplo.

### AMBOS

`AMBOS` sigue siendo una capacidad de plantilla, modelo y solicitud: no es una tercera modalidad de ejercicio ni una opción para estudiantes. `environmentTypeOptions` exige `include` para que ningún selector lo ofrezca por descuido. Una plantilla dual aparece en los filtros de Entorno de Desarrollo y de Entorno de Evaluación, porque sirve a las dos.

### Valores desconocidos

Un `environment_type` que no se reconoce se presenta con el estado neutro del catálogo («Modalidad sin reconocer»). Ningún consumidor lo convierte en Desarrollo ni en Evaluación, y los modales de plantilla no proponen presets de RAM para él.

## Consecuencias

- El gate deja de depender de una lista de palabras: un nombre nuevo con forma de modalidad falla sin necesidad de actualizar el script.
- Los documentos históricos se conservan, pero deben declarar qué decisión los sustituye.
- `environmentTypeOptions` obliga a decidir qué tipos ofrece cada selector.
- Un valor desconocido se ve como tal en la interfaz en lugar de heredar una modalidad.

## Lo que no cambia

- Los identificadores de contrato, la API y los valores de base de datos.
- Las restricciones de `exercises.environment_type` y las migraciones.
- El funcionamiento del motor de evaluación y el veredicto académico.
- Los ADRs y documentos históricos que declaran su fecha.
