import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '@core/services/auth.service';
import { User } from '@core/models/user.model';

export const authGuard: CanActivateFn = async (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  // 1. Si viene el token en los queryParams al redirigir desde OAuth
  const tokenFromQuery = route.queryParams['token'];
  if (tokenFromQuery && typeof window !== 'undefined') {
    sessionStorage.setItem('solv_token', tokenFromQuery);
  }

  // 2. Si ya tenemos el usuario en memoria
  if (authService.isAuthenticated()) {
    return true;
  }

  // 3. Si no está en memoria, intentar resolver vía Bearer / cookie
  const user = await firstValueFrom(authService.resolveCurrentUser());
  if (user) {
    return true;
  }

  // 4. Modo desarrollo local: si el backend no está disponible y estamos en localhost,
  // proveer usuario de diseño según la ruta solicitada para evaluar la interfaz
  if (typeof window !== 'undefined' && (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')) {
    const requestedRole = state.url.startsWith('/admin') ? 'admin' : (state.url.startsWith('/teacher') ? 'teacher' : 'student');
    const devUser: User = {
      id: '00000000-0000-0000-0000-000000000001',
      tenant_id: '00000000-0000-0000-0000-000000000001',
      email: 'admin.dev@uab.edu.bo',
      first_name: 'Administrador',
      last_name: 'General',
      role: requestedRole,
      created_at: new Date().toISOString()
    };
    authService.currentUser.set(devUser);
    return true;
  }

  // No autenticado -> Redirigir a login preservando la URL solicitada
  return router.createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });
};
