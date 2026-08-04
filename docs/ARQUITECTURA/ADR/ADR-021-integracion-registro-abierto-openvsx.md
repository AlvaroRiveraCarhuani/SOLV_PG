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
