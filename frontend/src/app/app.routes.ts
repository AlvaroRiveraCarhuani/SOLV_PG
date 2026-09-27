import { Routes } from '@angular/router';
import { authGuard } from '@core/guards/auth.guard';
import { adminGuard } from '@core/guards/admin.guard';

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
    path: 'admin',
    canActivate: [authGuard, adminGuard],
    loadComponent: () =>
      import('@features/admin/layout/admin-layout.component').then((m) => m.AdminLayoutComponent),
    children: [
      {
        path: '',
        pathMatch: 'full',
        redirectTo: 'dashboard'
      },
      {
        path: 'dashboard',
        loadComponent: () =>
          import('@features/admin/dashboard/admin-dashboard.component').then((m) => m.AdminDashboardComponent)
      },
      {
        path: 'docentes',
        loadComponent: () =>
          import('@features/admin/teachers/admin-teachers.component').then((m) => m.AdminTeachersComponent)
      },
      {
        path: 'cursos',
        loadComponent: () =>
          import('@features/admin/courses/admin-courses.component').then((m) => m.AdminCoursesComponent)
      },
      {
        path: 'estudiantes',
        loadComponent: () =>
          import('@features/admin/students/admin-students.component').then((m) => m.AdminStudentsComponent)
      },
      {
        path: 'plantillas',
        loadComponent: () =>
          import('@features/admin/templates/admin-templates.component').then((m) => m.AdminTemplatesComponent)
      },
      {
        path: 'modelos-categorias',
        loadComponent: () =>
          import('@features/admin/templates/components/model-library/model-library.component').then((m) => m.ModelLibraryComponent)
      },
      {
        path: 'configuracion',
        loadComponent: () =>
          import('@features/admin/configuracion/admin-configuracion.component').then((m) => m.AdminConfiguracionComponent)
      },
      {
        path: 'manual',
        loadComponent: () =>
          import('@features/admin/manual/admin-manual.component').then((m) => m.AdminManualComponent)
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
