# ADR-044: Piloto con Usuarios Reales

## Estado
Aceptado

## Contexto
Tras completar la implementación del modelo de datos dual, el editor condicional por modalidad y el sistema de rúbricas cualitativas para IDE Persistente, el sistema SOLV está funcionalmente completo, observable, escalable, recuperable y seguro. Es momento de validar el sistema en producción con un curso real antes del lanzamiento oficial.

## Decisión
Ejecutar un piloto de 2 semanas con un curso de prueba (30-50 estudiantes) que incluya:
- 5 ejercicios de JUEZ_EFIMERO (algoritmos y estructuras de datos)
- 3 ejercicios de IDE_PERSISTENTE (desarrollo web y bases de datos)

## Criterios de Éxito
- Tasa de disponibilidad > 99.5% durante el piloto
- Latencia P95 de submissions < 3 segundos
- Cero incidentes críticos (pérdida de datos, corrupción)
- Feedback positivo de >80% de docentes y estudiantes

## Alcance del Piloto
- **Incluido:** ejercicios de ambos modos, rúbricas, mapa curricular, sugerencias de refuerzo, SpeedGrader forense
- **Excluido:** ejercicios de exámenes con semilla por estudiante (se validará en fase posterior)

## Plan de Rollback
Si el piloto falla:
1. Detener el backend
2. Restaurar desde backup diario
3. Revertir a versión anterior si es necesario
4. Analizar causa raíz y corregir

## Próximos Pasos Post-Piloto
- Analizar métricas de uso real vs. prueba de carga
- Recopilar feedback cualitativo de usuarios
- Planificar siguiente iteración basada en hallazgos

## Consecuencias
- El sistema se expone a usuarios reales con datos académicos
- El equipo debe estar en modo on-call durante el piloto
- Se documentarán lecciones aprendidas para futuras iteraciones
