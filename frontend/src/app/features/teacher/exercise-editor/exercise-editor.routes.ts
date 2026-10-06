import { Routes } from '@angular/router';

export const EXERCISE_EDITOR_ROUTES: Routes = [
  {
    path: 'new',
    loadComponent: () =>
      import('./exercise-editor.component').then(m => m.ExerciseEditorComponent)
  },
  {
    path: ':id/edit',
    loadComponent: () =>
      import('./exercise-editor.component').then(m => m.ExerciseEditorComponent)
  }
];
