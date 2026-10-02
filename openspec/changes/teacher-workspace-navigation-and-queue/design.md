# Technical Design: Navegación Docente y Cola Centralizada de Entregas

## Architecture Overview

El diseño se apoya en el patrón de arquitectura por capas y la reactividad basada en Angular Signals (Zoneless).

```
┌───────────────────────────────────────────────────────────┐
│                    TeacherLayoutComponent                 │
│  ┌───────────────────────┐   ┌─────────────────────────┐  │
│  │    SidebarComponent   │   │      RouterOutlet       │  │
│  │   (TEACHER_SECTIONS)  │   │  (TeacherEvaluations)   │  │
│  └───────────────────────┘   └─────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
                                           │
                                           ▼
                            TeacherEvaluationsComponent
                        (KPIs + Filtros + Tabla Reactiva)
                                           │
                                           ▼
                                TeacherCourseService
                             (loadAllSubmissions, courses)
```

## Component Breakdown

### 1. `TeacherLayoutComponent` & `TEACHER_NAV_SECTIONS`
Actualización de las secciones del menú de navegación lateral:
- `DOCENCIA`:
  - Panel de Control: `/teacher/dashboard` (icon: `activity`)
  - Evaluaciones & Entregas: `/teacher/evaluaciones` (icon: `award`)
- `CONTENIDOS`:
  - Banco de Ejercicios: `/teacher/exercises` (icon: `boxes`)
  - Plantillas Docker: `/teacher/templates` (icon: `layers`)
- `SUPERVISIÓN`:
  - Anti-Plagio AST: `/teacher/anti-plagio` (icon: `shield-alert`)

### 2. `TeacherEvaluationsComponent` (`/teacher/evaluaciones`)
- **Signals**:
  - `courseFilter = signal<string>('all')`
  - `verdictFilter = signal<string>('all')`
  - `statusFilter = signal<'all' | 'pending' | 'graded'>('all')`
  - `searchTerm = signal<string>('')`
- **Computed**:
  - `filteredSubmissions`: Filtro reactivo combinando materia, veredicto, estado y búsqueda por nombre de alumno o laboratorio.
  - `kpiTotalSubmissions`, `kpiPendingReview`, `kpiAutoGraded`, `kpiAtRisk`.

## UI & Design System Adherence
- Uso estricto de tokens de diseño (`var(--tenant-*)`, `var(--color-*)`, `var(--bg-*)`, `var(--border-*)`).
- Toda fecha formateada mediante `<time>{{ date | dateText:'compact-datetime' }}</time>`.
- Datos técnicos y métricas marcadas con la directiva `[machineData]`.
