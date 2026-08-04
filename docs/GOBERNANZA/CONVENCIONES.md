# Convenciones de Código y Mensajes de Commit — SOLV BaaS

Este documento especifica las normas obligatorias para contribuciones al repositorio de **SOLV**.

---

## 1. Convención de Commits (Conventional Commits + Gentleman Dots)

Todos los commits registrados en el proyecto deben cumplir con el estándar **Conventional Commits** estructurado en español neutro, incluyendo cuerpo descriptivo de 4 secciones y footers obligatorios de trazabilidad:

```text
<tipo>(<alcance>): <descripción corta en imperativo>

- Qué se hizo: <descripción concisa de la modificación técnica>
- Por qué se hizo: <motivación del cambio o requerimiento resuelto>
- Cómo lo soluciona: <implementación concreta y mecanismos utilizados>
- Impacto: <efectos en el sistema, rendimiento o mantenibilidad>

Slice: <número de slice>
ADRs: <lista de ADRs asociados>
```

### Tipos Permitidos:
* `feat`: Nueva funcionalidad o capacidad técnica.
* `fix`: Corrección de errores o desvíos.
* `refactor`: Cambios de código que no modifican comportamiento externo.
* `docs`: Cambios o reestructuración en la documentación.
* `infra`: Modificaciones en infraestructura, Docker o firewalls.
* `test`: Adición o actualización de suites de prueba.

---

## 2. Regla de Gobernanza y Limpieza Pública

> [!CAUTION]
> **Prohibición de Nomenclatura Interna:**
> Queda estrictamente prohibido incluir términos o códigos de control interno (como "CRIT-XX") dentro de mensajes de commit públicos, nombres de archivos de documentación oficial o cabeceras públicas. Toda referencia técnica debe expresarse mediante títulos descriptivos, identificadores de Slice o números de ADR.
