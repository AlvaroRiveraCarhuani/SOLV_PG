import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '@core/services/auth.service';

export const authGuard: CanActivateFn = async (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  // Si ya tenemos el usuario en memoria
  if (authService.isAuthenticated()) {
    return true;
  }

  // Si no está en memoria, intentar resolver vía cookie HttpOnly
  const user = await firstValueFrom(authService.resolveCurrentUser());
  if (user) {
    return true;
  }

  // No autenticado -> Redirigir a login preservando la URL solicitada
  return router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
};
