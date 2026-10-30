# ADR 054: Catálogo de Entornos del Docente Centrado en el Stack

> **Estado:** Propuesto
> **Fecha:** 2026-10-30
> **Ámbito:** Docencia, Catálogo de plantillas
> **Relacionado:** ADR-030 (catálogo y aprobación), ADR-043 (modalidad dual), ADR-046 (constructor de entornos), ADR-048 (copiloto de arquitectura), ADR-052 (banco de ejercicios)

---

## Contexto

El catálogo del docente (`/teacher/templates`) consume `GET /api/v1/templates`, servido por el repositorio legacy `backend/internal/infrastructure/database/template_repository.go`. Esa consulta devuelve ocho columnas: `id`, `name`, `docker_image`, `base_ram_mb`, `target_environment`, `status`, `entrypoint` y `services_config`.

Con ese contrato la tarjeta no puede responder la pregunta que el docente trae: **"¿este entorno tiene las herramientas de mi práctica?"**. Solo puede mostrar el nombre, la imagen y los recursos.

El problema no es de datos: la información existe en tres niveles.

| Nivel | Tabla | Qué representa | Quién lo usa hoy |
| --- | --- | --- | --- |
| Pieza | `template_components` | Runtime, plataforma web, complemento o base de datos, con nombre, icono y compatibilidad | Administrador, en el constructor |
| Modelo | `template_models` | Stack curado con título humano, descripción, herramientas y categoría | Administrador, en la biblioteca de modelos |
| Plantilla | `lab_templates` | Imagen ejecutable con recursos, estado y auditoría | Docente, en el catálogo |

`lab_templates` ya guarda `tools_declared`, `stack_config` y `model_id`; `template_models` guarda `tools` y `stack_config`; `stack_config` referencia piezas por `runtime_id`, `web_platform_id`, `addon_ids` y `database_id`. El docente está viendo el artefacto operativo, no el entorno.

Además, la interfaz actual afirma cosas que el dato no sostiene:

- El mapeo del frontend hace `is_official: item.is_official ?? true` y el API no devuelve ese campo. Todas las plantillas se muestran como oficiales.
- El KPI "Plantillas Oficiales" cuenta esa bandera: siempre da el total del catálogo.
- La fila de KPIs (total, IDE, juez, oficiales) repite el inventario y no cambia ninguna decisión.
- RAM y CPU son los únicos números de la tarjeta, y son una consecuencia de la elección, no un criterio.

---

## Decisión

### 1. La unidad que navega el docente es el entorno curado

El catálogo del docente se arma con `template_models` de su tenant. Cada entorno se resuelve a sus piezas con `stack_config`, de modo que la tarjeta puede mostrar `Java 21`, `Spring Boot`, `Maven` y `PostgreSQL 16` con nombre e icono, en lugar de `eclipse-temurin:21-jdk-alpine`.

La plantilla ejecutable pasa a ser un dato de implementación: se muestra en el detalle, en "Requisitos técnicos", y decide si el entorno se puede usar.

### 2. La disponibilidad se declara, no se adivina

Un entorno está disponible si existe una `lab_template` aprobada vinculada, ya sea por `template_models.source_template_id` o por `lab_templates.model_id`. Si no existe, la acción principal es **Solicitar entorno**, no "Usar".

### 3. Las plantillas sin modelo no se ocultan

Los `lab_templates` con `model_id` nulo se agrupan como **Otros entornos**. Muestran el stack que declaren en `stack_config` o en `tools_declared`. Si no declaran nada, lo dicen: "Sin detalle declarado". Nunca se reconstruye el stack adivinando desde el nombre de la imagen en el servidor.

### 4. Un read model nuevo, sin escrituras

`GET /api/v1/teacher/environments` devuelve el catálogo resuelto. El endpoint legacy `/api/v1/templates` se mantiene para sus consumidores actuales hasta que migren; no se borra en esta etapa.

Contrato de referencia:

```json
{
  "data": [
    {
      "id": "d0000000-0000-0000-0000-000000000007",
      "kind": "curated",
      "name": "Java 21 + Spring Boot",
      "summary": "APIs REST y proyectos empresariales",
      "description": "Entorno OpenJDK 21 con Maven y PostgreSQL para prácticas de backend.",
      "environment_type": "IDE_PERSISTENTE",
      "category": "Backend",
      "components": [
        { "slug": "java-21", "name": "Java 21", "layer": "runtime", "icon_key": "java" },
        { "slug": "spring-maven", "name": "Maven", "layer": "addon", "icon_key": "spring" },
        { "slug": "postgres-16", "name": "PostgreSQL 16", "layer": "database", "icon_key": "postgres" }
      ],
      "available": true,
      "template_id": "a0000000-0000-0000-0000-000000000003",
      "official": true
    }
  ],
  "meta": { "total": 7 }
}
```

Parámetros: `q` busca en nombre, descripción y nombre de pieza; `component` filtra por slug de pieza; `environment_type` filtra por modalidad. El filtro por pieza es lo que permite que "PostgreSQL" encuentre todo lo que lo trae, en lugar de obligar a conocer la categoría del administrador.

### 5. La tarjeta decide, el detalle informa

- La tarjeta no muestra imagen, RAM ni CPU.
- El detalle muestra "Qué incluye" como árbol principal → complementos → base de datos, y "Requisitos técnicos" plegado: imagen, memoria recomendada y tiempo límite cuando aplica.
- La fila de KPIs desaparece. El conteo vive en la barra de filtros y cambia con el resultado.

### 6. "Oficial" es un flag explícito

Se agrega `is_official BOOLEAN NOT NULL DEFAULT FALSE` a `template_models` y se define el equivalente para `lab_templates`. Ninguna capa asume verdadero por defecto. La política de qué entornos son oficiales la fija el administrador; la interfaz solo muestra el flag.

---

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Agregar chips de tecnología a la tarjeta actual | La tarjeta sigue liderando con la imagen y los recursos; no resuelve la disponibilidad ni la bandera de oficial. Es un parche sobre la entidad equivocada. |
| Mostrar al docente el catálogo de administración | Expone auditoría, CVE, smoke test y vocabulario interno; su tarea y su jerarquía son otras. |
| Crear una tabla nueva de catálogo | Duplica `template_models`, `template_components` y `stack_config`, que ya existen y el administrador mantiene. |
| Deducir el stack desde el nombre de la imagen en el servidor | Heurística frágil y silenciosa; el stack es un dato declarado, no una adivinanza. El detector del frontend puede ayudar a curar, no a afirmar en producción. |
| Mantener la fila de KPIs y corregir solo la bandera | El conteo seguiría sin cambiar una decisión; el espacio se gana con acciones, no con simetría. |

---

## Consecuencias

**Positivas**

- El docente elige por reconocimiento ("Spring Boot") y no por decodificación ("temurin").
- La búsqueda por herramienta encuentra entornos aunque el docente no conozca la categoría.
- La disponibilidad y la bandera de oficial dejan de mentir.
- El administrador conserva una sola fuente de curaduría: la biblioteca de modelos y el catálogo de piezas.

**Costos y cuidados**

- Hay que curar `stack_config` en los siete modelos semilla; sin eso, las tarjetas muestran "Sin detalle declarado".
- Se agrega una consulta con resolución de piezas; conviene resolverla en una sola lectura con `jsonb_array_elements` y `JOIN` al catálogo, no con N+1.
- El endpoint legacy sigue vivo con su DTO duplicado; la deuda se documenta, no se esconde.
- La bandera de oficial cambia los conteos actuales: el catálogo dejará de mostrar 7 de 7.
- La portada pierde la fila de KPIs y las cifras de recursos; los tests de la vista actual deben actualizarse.

---

## Decisiones pendientes

1. **Etiqueta de navegación:** "Plantillas de Entorno" (actual) o "Entornos". El concepto interno es entorno; la etiqueta visible se decide con el usuario.
2. **Alcance de "Usar":** si abre el editor de práctica con el entorno preseleccionado o crea un laboratorio dentro de una materia existente. Cambia el flujo y el contrato de la acción.
3. **Plantillas huérfanas:** si se muestran siempre en "Otros entornos" o solo cuando no hay ningún modelo en la categoría.
4. **Política de oficiales:** qué criterio usa el administrador para marcar un entorno como oficial (revisión, origen institucional o ambos).
