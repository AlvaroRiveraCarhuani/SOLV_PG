import { Routes } from '@angular/router';

export const TEACHER_ROUTES: Routes = [
  {
    path: '',
    pathMatch: 'full',
    redirectTo: 'dashboard'
  },
  {
    path: 'dashboard',
    loadComponent: () =>
      import('./dashboard/teacher-dashboard.component').then(m => m.TeacherDashboardComponent)
  },
  {
    path: 'aula',
    loadComponent: () => import('./aula/aula.component').then(m => m.AulaComponent)
  },
  {
    // "Mi entorno" se absorbió en la ficha de cada práctica (ADR-059). La ruta
    // queda como redirección para los enlaces y marcadores anteriores.
    path: 'entorno',
    pathMatch: 'full',
    redirectTo: 'dashboard'
  },
  {
    // Una sola familia de rutas: la materia vive en "cursos" y la práctica en
    // "practicas". "ejercicios" queda solo para el editor y la previsualización.
    path: 'cursos/:id',
    loadComponent: () =>
      import('./courses/course-detail/teacher-course-detail.component').then(m => m.TeacherCourseDetailComponent)
  },
  {
    path: 'cursos/:courseId/analytics',
    loadComponent: () =>
      import('./courses/analytics/analytics-dashboard.component').then(m => m.AnalyticsDashboardComponent)
  },
  {
    path: 'cursos/:courseId/ejercicios/nuevo',
    loadComponent: () =>
      import('./exercise-editor/exercise-editor.component').then(m => m.ExerciseEditorComponent)
  },
  {
    path: 'cursos/:courseId/ejercicios/:id/editar',
    loadComponent: () =>
      import('./exercise-editor/exercise-editor.component').then(m => m.ExerciseEditorComponent)
  },
  {
    path: 'cursos/:courseId/ejercicios/:id/probar',
    loadComponent: () =>
      import('./exercise-preview/exercise-preview.component').then(m => m.ExercisePreviewComponent)
  },
  {
    // La auditoría vive dentro de la práctica: la ruta fija materia y práctica.
    // Va antes del detalle para que el segmento extra no se pierda.
    path: 'cursos/:courseId/practicas/:exerciseId/similitud',
    loadComponent: () =>
      import('./supervision/anti-plagio/teacher-plagiarism.component').then(m => m.TeacherPlagiarismComponent)
  },
  {
    path: 'cursos/:courseId/practicas/:id',
    loadComponent: () =>
      import('./courses/practice-detail/practice-detail.component').then(m => m.PracticeDetailComponent)
  },
  {
    path: 'revision/:submissionId',
    loadComponent: () =>
      import('./grading/speed-grader/speed-grader.component').then(m => m.SpeedGraderComponent)
  },
  {
    path: 'por-calificar',
    loadComponent: () =>
      import('./reviews/teacher-reviews.component').then(m => m.TeacherReviewsComponent)
  },
  {
    // El centro global se retiró: la calificación vive en "Por calificar" y en
    // la materia. Los enlaces y marcadores viejos vuelven a la cola de revisión.
    path: 'evaluaciones',
    redirectTo: 'por-calificar'
  },
  {
    // El listado global se retiró: los enlaces viejos vuelven a la cola de revisión.
    path: 'anti-plagio',
    redirectTo: 'por-calificar'
  },
  {
    path: 'exercises',
    loadComponent: () =>
      import('./exercises/teacher-exercises.component').then(m => m.TeacherExercisesComponent)
  },
  {
    path: 'templates',
    loadComponent: () =>
      import('./templates/teacher-templates.component').then(m => m.TeacherTemplatesComponent)
  },
  {
    path: 'configuracion',
    loadComponent: () =>
      import('./config/teacher-ai-settings.component').then(m => m.TeacherAiSettingsComponent)
  },
  {
    path: 'notifications',
    loadComponent: () =>
      import('@features/shared/notifications/notifications-page.component').then(
        m => m.NotificationsPageComponent
      )
  }
];
