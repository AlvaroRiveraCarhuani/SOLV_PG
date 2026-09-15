import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '@core/services/auth.service';
import { firstValueFrom } from 'rxjs';

export const adminGuard: CanActivateFn = async (_route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (!authService.isAuthenticated()) {
    const user = await firstValueFrom(authService.resolveCurrentUser());
    if (!user) {
      return router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
    }
  }

  const currentUser = authService.currentUser();
  if (currentUser?.role === 'admin') {
    return true;
  }

  // Si no es admin, redirigir según su rol correspondiente
  if (currentUser?.role === 'teacher') {
    return router.createUrlTree(['/teacher']);
  }
  return router.createUrlTree(['/student']);
};
