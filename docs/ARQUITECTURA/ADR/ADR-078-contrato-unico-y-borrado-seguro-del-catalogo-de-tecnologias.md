# ADR-078: Contrato Único de Edición y Borrado Seguro en el Catálogo de Tecnologías

**Estado:** Aceptado
**Fecha:** 2026-11-13
**Área:** Backend (`Go`, `net/http.ServeMux`), API y catálogo de tecnologías
**Relacionado:** ADR-046 (catálogo de componentes y constructor de entornos), ADR-070 (desacople de navegación entre entornos, tecnologías y categorías)

---

## 1. Contexto

El catálogo de tecnologías (`template_components`) alimenta el constructor de entornos: cada plantilla y cada modelo referencian sus componentes desde `stack_config`, por id o por slug. Conviviían tres problemas de contrato:

1. **Seis rutas para el mismo recurso.** `PUT` y `PATCH` sobre `template-components`, `tech-components` y `components` apuntaban al mismo handler. El `PATCH` no era parcial: el DTO exige `name` y el repositorio escribe `description`, `setup_snippet` y `judge_entrypoint` con el valor recibido. Un cliente que enviaba solo `is_active` borraba esos tres campos.
2. **Borrado sin verificación de uso.** `DeleteTemplateComponent` solo comprobaba `is_system`. El compositor resuelve el runtime primario con error, pero omite en silencio los complementos y las bases de datos ausentes. Eliminar un complemento en uso recomponía las plantillas publicadas sin esa librería, sin error y sin aviso.
3. **Edición cruzada entre tenants.** El `UPDATE` filtraba por `tenant_id = $tenant OR tenant_id = global`, mientras que el `DELETE` exigía pertenencia. Un administrador de un tenant secundario podía modificar el catálogo compartido y no podía borrarlo.

---

## 2. Decisión

### 2.1 Un solo contrato de edición

`PUT /api/v1/admin/template-components/{id}` queda como única ruta de edición. Se retiran `PATCH` y los alias `tech-components` y `components`. `PUT` reemplaza el recurso completo: los campos de texto omitidos quedan vacíos, y eso es explícito en el contrato, no un efecto lateral de un método que se anunciaba como parcial.

### 2.2 Uso calculado al listar

`GET /api/v1/admin/template-components` devuelve `usage_count`: plantillas y modelos que referencian la tecnología por `runtime_id`, `web_platform_id`, `database_id` o `addon_ids`, comparando por id y por slug. Las plantillas `RECHAZADA` no cuentan porque nunca se aprovisionan. No es una columna persistida.

### 2.3 Borrado bloqueado por uso

El `DELETE` incorpora `NOT EXISTS` sobre `lab_templates` y `template_models` en la misma sentencia, de modo que no hay ventana entre comprobar y borrar. Si la fila existe, no es del sistema y pertenece al tenant, pero el borrado no afectó filas, la respuesta es `409 component_in_use` con la sugerencia de desactivar.

### 2.4 Propiedad por tenant

La edición y el borrado exigen `tenant_id = $tenant`. Las filas del catálogo global son de solo lectura para el resto de los tenants y responden `409 component_read_only`. Las semillas del catálogo pertenecen al tenant por defecto, así que el nodo único no cambia su operación.

### 2.5 La interfaz no ofrece lo que el servidor rechaza

La pantalla muestra el uso por tecnología y confirma el borrado. Cuando `usage_count > 0`, el diálogo no ofrece eliminar: ofrece desactivar, que es la acción que el servidor permite. La comprobación del servidor sigue siendo la que decide.

---

## 3. Consecuencias

- No hay migración: `usage_count` se calcula en la consulta.
- El conteo recorre `lab_templates` y `template_models` una vez por listado. Con el catálogo actual es despreciable; si crece, corresponde un índice GIN sobre `stack_config` o una vista materializada.
- `environment_requests.stack_config` no se cuenta: una solicitud pendiente que referencie una tecnología borrada queda en `missing_components`, que es un fallo visible y no un entorno alterado en silencio.
- Un cliente que dependía de `PATCH` o de los alias debe migrar a `PUT /template-components/{id}` con el recurso completo. No quedaban consumidores en el frontend.
