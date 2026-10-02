import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '@core/services/auth.service';
import { firstValueFrom } from 'rxjs';

export const teacherGuard: CanActivateFn = async (_route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (!authService.isAuthenticated()) {
    const user = await firstValueFrom(authService.resolveCurrentUser());
    if (!user) {
      return router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
    }
  }

  const currentUser = authService.currentUser();
  if (currentUser?.role === 'teacher' || currentUser?.role === 'admin') {
    return true;
  }

  return router.createUrlTree(['/student']);
};
