import { Routes } from '@angular/router';
import { authGuard } from '@core/guards/auth.guard';
import { adminGuard } from '@core/guards/admin.guard';
import { teacherGuard } from '@core/guards/teacher.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('@features/auth/login/login.component').then((m) => m.LoginComponent)
  },
  {
    path: 'teacher',
    canActivate: [authGuard, teacherGuard],
    loadComponent: () =>
      import('@features/teacher/layout/teacher-layout.component').then((m) => m.TeacherLayoutComponent),
    loadChildren: () =>
      import('@features/teacher/teacher.routes').then((m) => m.TEACHER_ROUTES)
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
      },
      {
        path: 'courses',
        loadComponent: () =>
          import('@features/student/courses/course-list/student-courses.component').then(
            (m) => m.StudentCoursesComponent
          )
      },
      {
        path: 'courses/:courseId',
        loadComponent: () =>
          import('@features/student/courses/course-detail/student-course-detail.component').then(
            (m) => m.StudentCourseDetailComponent
          )
      },
      {
        path: 'workspaces',
        loadComponent: () =>
          import('@features/student/workspaces/workspace-list/student-workspaces.component').then(
            (m) => m.StudentWorkspacesComponent
          )
      },
      {
        path: 'workspaces/:subjectId',
        loadComponent: () =>
          import('@features/student/workspaces/active-lab/student-active-lab.component').then(
            (m) => m.StudentActiveLabComponent
          )
      },
      {
        path: 'evaluations',
        loadComponent: () =>
          import('@features/student/evaluations/student-evaluations.component').then(
            (m) => m.StudentEvaluationsComponent
          )
      },
      {
        path: 'history',
        loadComponent: () =>
          import('@features/student/history/student-history.component').then(
            (m) => m.StudentHistoryComponent
          )
      },
      {
        path: 'cursos/:courseId',
        redirectTo: 'courses/:courseId'
      },
      {
        path: 'exercises/:exerciseId',
        loadComponent: () =>
          import('@features/student/exercises/solver/student-exercise-solver.component').then(
            (m) => m.StudentExerciseSolverComponent
          )
      },
      {
        path: 'ejercicios/:exerciseId',
        redirectTo: 'exercises/:exerciseId'
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
        path: 'plantillas/nueva',
        loadComponent: () =>
          import('@features/admin/templates/studio/template-studio.component').then((m) => m.TemplateStudioComponent)
      },
      {
        path: 'plantillas/:id/editar',
        loadComponent: () =>
          import('@features/admin/templates/studio/template-studio.component').then((m) => m.TemplateStudioComponent)
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
        path: 'auditoria',
        loadComponent: () =>
          import('@features/admin/auditoria/admin-auditoria.component').then((m) => m.AdminAuditoriaComponent)
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
