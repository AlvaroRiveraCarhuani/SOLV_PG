# Design: Experiencia y Flujo de Trabajo del Docente

## Technical Approach

Implementar el módulo `/teacher` en Angular 22 standalone y zoneless utilizando Angular Signals para la reactividad. La interfaz se divide en tres vistas principales conectadas a los contratos BaaS (ADR-026, ADR-012):
1. **Teacher Dashboard (`/teacher/dashboard`):** Monitor reactivo de incidencias por severidad y alumnos `at_risk`.
2. **Teacher Course View (`/teacher/cursos/:id`):** Cuaderno docente, gestión de ejercicios (`draft`/`published`), subida masiva de casos de prueba vía CSV y exportación de matriz de calificaciones.
3. **SpeedGrader (`/teacher/revision/:submissionId`):** Visor de código con comentarios in-line anclados a líneas, desenmascaramiento de casos de prueba privados y navegación fluida entre entregas continuas.

## Architecture Decisions

| Decision | Choice | Alternatives Considered | Rationale |
|---|---|---|---|
| **State Management en SpeedGrader** | Servicio dedicado con Signals (`TeacherGradingService`) | Component input/output drilling o NgRx Store | Mantiene el cursor de navegación (`prev_id`/`next_id`), el cache del código y el estado de los comentarios desacoplado del template. |
| **Visor de Código y Comentarios** | Componente presentacional custom con marcado de líneas semántico | Librerías pesadas externas (Monaco Editor / Ace) | Ligero, sin dependencias externas complejas, compatible con SSR/zoneless y estilizable al 100% con tokens SCSS. |
| **Subida CSV de Casos de Prueba** | Parseo y validación de RFC 4180 con feedback de línea previo a envío API | Subida ciega multipart sin validación frontend | Brinda feedback instantáneo al docente indicando la fila exacta con error. |

## Data Flow

```
[ Teacher UI Components ]
         │ (Signals / User Actions)
         ▼
[ Teacher Services (Signals Store) ]
         │ (HTTP with HttpParams & Tenant Headers)
         ▼
[ Backend Go BaaS Endpoints (/api/v1/teacher/*) ]
         │
         ▼
[ PostgreSQL / Judge Engine ]
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `frontend/src/app/features/teacher/teacher.routes.ts` | Create | Rutas hijas del módulo docente |
| `frontend/src/app/features/teacher/layout/teacher-layout.component.ts` | Create | Shell con Sidebar y Topbar docente |
| `frontend/src/app/features/teacher/dashboard/teacher-dashboard.component.ts` | Create | Dashboard de incidentes y estado de materias |
| `frontend/src/app/features/teacher/courses/course-detail/teacher-course-detail.component.ts` | Create | Vista de cuaderno, lista de ejercicios y matriz de notas |
| `frontend/src/app/features/teacher/courses/exercise-editor/exercise-editor-modal.component.ts` | Create | Modal de creación/edición de ejercicios y carga de CSV |
| `frontend/src/app/features/teacher/grading/speed-grader/speed-grader.component.ts` | Create | SpeedGrader con visor de código, comentarios y navegación |
| `frontend/src/app/features/teacher/services/teacher-dashboard.service.ts` | Create | Servicio API para métricas e incidentes |
| `frontend/src/app/features/teacher/services/teacher-course.service.ts` | Create | Servicio API para materias, ejercicios y notas |
| `frontend/src/app/features/teacher/services/teacher-grading.service.ts` | Create | Servicio API y state para SpeedGrader y comentarios |
| `frontend/src/app/app.routes.ts` | Modify | Registrar ruta lazy `/teacher` |

## Interfaces / Contracts

```typescript
export interface TeacherIncident {
  id: string;
  courseId: string;
  courseName: string;
  studentName: string;
  severity: 'critical' | 'warning' | 'standard';
  type: 'OOM_KILLED' | 'AST_BLOCKED' | 'PENDING_REVIEW';
  createdAt: string;
}

export interface TeacherExercise {
  id: string;
  subjectId: string;
  title: string;
  description: string;
  state: 'draft' | 'published';
  testCasesCount: number;
  publicTestCasesCount: number;
  dueDate?: string;
}

export interface SubmissionReviewDetail {
  id: string;
  studentId: string;
  studentName: string;
  exerciseId: string;
  exerciseTitle: string;
  code: string;
  score: number;
  maxScore: number;
  verdict: string;
  feedback?: string;
  submittedAt: string;
  prevSubmissionId?: string;
  nextSubmissionId?: string;
  testCases: Array<{
    id: string;
    input: string;
    expectedOutput: string;
    actualOutput?: string;
    passed: boolean;
    isHidden: boolean;
  }>;
  comments: Array<{
    id: string;
    lineNumber: number;
    content: string;
    authorName: string;
    createdAt: string;
  }>;
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (Frontend) | Servicios de datos y reactividad de Signals | Vitest (`ng test`) aislando llamadas HTTP con `HttpTestingController` |
| Component (Frontend) | Renderizado de incidentes, validación de draft/published en formulario y navegación SpeedGrader | Pruebas de integración de componentes en jsdom |
| Lint & Style | Presupuesto SCSS y cero selectores `solv-` | `npm run lint:styles` |

## Migration / Rollout

No requiere migración de base de datos ni feature flags. Se despliega como módulo de frontend nuevo conectado al backend existente.

## Open Questions

- None
