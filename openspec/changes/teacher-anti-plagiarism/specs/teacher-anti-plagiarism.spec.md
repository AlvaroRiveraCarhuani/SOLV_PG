# Specification: Centro de Supervisión y Auditoría Anti-Plagio AST

## Requirements

### R1: Vista Global de Supervisión (`/teacher/anti-plagio`)
1. El componente `TeacherPlagiarismComponent` debe cargar la lista de materias del docente y permitir seleccionar una materia específica o ver el consolidado.
2. Si se selecciona una materia, debe cargar dinámicamente sus laboratorios y permitir filtrar por ejercicio.
3. Debe proveer un botón "Re-escanear Materia" para forzar el análisis de similitud AST.

### R2: KPIs de Integridad y Riesgo
1. Debe calcular y mostrar:
   - Total de Pares Comparados
   - Coincidencias Críticas (Riesgo alto / similitud >= 80%)
   - Coincidencias en Advertencia (Riesgo moderado / similitud 50-79%)
   - Alumnos Afectados

### R3: Explorador y Comparador Lado a Lado
1. La columna izquierda debe listar los pares coincidentes mostrando:
   - Nombres de los dos estudiantes.
   - Porcentaje de similitud AST y nivel de riesgo (`critical` vs `warning`).
   - Ejercicio asociado y fecha de análisis.
2. La columna derecha debe renderizar el detalle del par seleccionado:
   - Similitud exacta de tokens y nodos AST idénticos.
   - Comparación de fragmentos de código de ambos estudiantes.
   - Botones directos para abrir la entrega de cada estudiante en SpeedGrader (`/teacher/revision/:id`).

### R4: Estándares de Diseño y Calidad
1. Los valores técnicos visibles deben usar `[machineData]` o la clase estática `font-mono`.
2. Las fechas deben usar `<time>` con el pipe `dateText`.
3. Estilos en SCSS sin valores hexadecimales hardcodeados.
4. Cobertura de pruebas unitarias en `teacher-plagiarism.component.spec.ts`.
