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
    redirectTo: 'evaluaciones'
  },
  {
    path: 'exercises',
    redirectTo: 'dashboard'
  },
  {
    path: 'templates',
    redirectTo: 'dashboard'
  }
];
