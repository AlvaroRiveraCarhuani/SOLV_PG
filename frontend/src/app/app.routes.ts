import { Routes } from '@angular/router';
import { authGuard } from '@core/guards/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('@features/auth/login/login.component').then((m) => m.LoginComponent)
  },
  {
    path: 'student',
    canActivate: [authGuard],
    loadComponent: () =>
      import('@features/student/layout/student-layout.component').then((m) => m.StudentLayoutComponent),
    children: [
      {
        path: '',
        loadComponent: () =>
          import('@features/student/dashboard/student-dashboard.component').then((m) => m.StudentDashboardComponent)
      }
    ]
  },
  {
    path: '',
    pathMatch: 'full',
    redirectTo: 'login'
  },
  {
    path: '**',
    redirectTo: 'login'
  }
];
