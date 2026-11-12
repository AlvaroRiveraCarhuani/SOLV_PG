# ADR-021: Registro Abierto OpenVSX para Entornos Interactivos (D5)

* **Estado:** Aceptado
* **Fecha:** 2026-08-04

## Contexto y Problema
El uso de registros de extensiones propietarios de Microsoft en entornos OpenVSCode Server puede generar incompatibilidades de licencias. Además, se requiere permitir la instalación controlada de extensiones educativas de lenguajes (Go, Python, C/C++) sin depender de servicios privativos.

## Decisión Tomada
1. Adoptar la decisión de arquitectura **D5**: Integrar el registro abierto de extensiones **OpenVSX** (`open-vsx.org`) en la configuración de la plantilla base de OpenVSCode Server.
2. Permitir la precarga de extensiones validadas directamente en las imágenes Docker de laboratorios.

## Consecuencias
* **Positivas:**
  * Cumplimiento estricto con licencias de código abierto.
  * Autonomía en la selección e instalación de herramientas de desarrollo para los estudiantes.

## Enmienda 2026-11-11: Aislamiento de Capas y Ciclo de Vida de Extensiones

Para evitar la corrupción de extensiones del sistema, colisiones de permisos y sobreconsumo de almacenamiento por volúmenes globales compartidos, se formaliza la separación estricta de dos capas de extensiones:

1. **Capa del Sistema y Plantilla Docente (Imagen Docker inmutable):**
   - Reside en `/home/.openvscode-server/extensions` (`builtin-extensions-dir`).
   - Aloja las 91 extensiones base del núcleo de OpenVSCode Server, la extensión de plataforma `solv-companion` (ADR-062) y las herramientas o servidores de lenguaje (LSP) oficiales preconfigurados por el docente en la plantilla compuesta (`solv-local/stack:*`).
   - Se distribuye en capas de solo lectura compartidas por el host mediante el sistema de capas de Docker (*overlayfs*), con coste nulo de almacenamiento por alumno y arranque inmediato sin depender de descargas en vivo de registros externos.

2. **Capa de Usuario y Práctica (Volumen de Práctica aislado):**
   - OpenVSCode Server se ejecuta con `--extensions-dir /home/workspace/.vscode/extensions`.
   - Cualquier extensión adicional instalada por el estudiante desde Open VSX queda confinada en su volumen de trabajo (`solv_workspace_<studentID>_<subjectID>_<exerciseID>`), montado en `/home/workspace`.
   - Se descarta la creación de un volumen global por estudiante (`solv_extensions_<alumno>`) por generar enmascaramiento de capas (*volume masking* en Docker) que oculta actualizaciones de plantillas, contaminación cruzada de servidores de lenguaje entre materias y falta de aislamiento por tenant (D2).
   - En situaciones de examen o entrega congelada (ADR-005), la capa de extensiones de usuario pasa automáticamente a solo lectura junto con el workspace.
   - El snapshot de entrega (`ADVANCE_SNAPSHOT_EXCLUDES`) ya excluye `.vscode/`, asegurando que las extensiones instaladas por el alumno no engorden los envíos de código.
