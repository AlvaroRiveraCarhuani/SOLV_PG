# Technical Design: Centro de Supervisión y Auditoría Anti-Plagio AST

## Component Architecture

```
[TeacherPlagiarismComponent]
  ├── Control Bar (Course & Lab selectors, Rescan button)
  ├── KPI Summary Grid (Critical, Warnings, Students)
  ├── Dual Panel Layout:
  │    ├── Left: Match List (Pares con riesgo de plagio)
  │    └── Right: AST Inspector & Comparison View (Side-by-side AST subtrees & code links)
```

## State & Data Flow
- `courses`: Signal derivado de `TeacherDashboardService`.
- `labs`: Signal cargado dinámicamente vía `TeacherCourseService.getCourseLabs`.
- `report`: Signal con `PlagiarismReport` obtenido de `TeacherCourseService.analyzePlagiarism`.
- `selectedMatch`: Signal con `PlagiarismMatch` actualmente inspeccionado.
