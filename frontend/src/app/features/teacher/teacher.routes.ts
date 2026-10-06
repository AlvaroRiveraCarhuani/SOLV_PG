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
    path: 'cursos/:id',
    loadComponent: () =>
      import('./courses/course-detail/teacher-course-detail.component').then(m => m.TeacherCourseDetailComponent)
  },
  {
    path: 'courses/:courseId/exercises/new',
    loadComponent: () =>
      import('./exercise-editor/exercise-editor.component').then(m => m.ExerciseEditorComponent)
  },
  {
    path: 'courses/:courseId/exercises/:id/edit',
    loadComponent: () =>
      import('./exercise-editor/exercise-editor.component').then(m => m.ExerciseEditorComponent)
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
    path: 'revision/:submissionId',
    loadComponent: () =>
      import('./grading/speed-grader/speed-grader.component').then(m => m.SpeedGraderComponent)
  },
  {
    path: 'evaluaciones',
    loadComponent: () =>
      import('./evaluations/teacher-evaluations.component').then(m => m.TeacherEvaluationsComponent)
  },
  {
    path: 'anti-plagio',
    loadComponent: () =>
      import('./supervision/anti-plagio/teacher-plagiarism.component').then(m => m.TeacherPlagiarismComponent)
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
  }
];
