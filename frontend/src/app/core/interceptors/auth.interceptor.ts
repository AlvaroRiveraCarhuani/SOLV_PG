import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { TenantService } from '@core/services/tenant.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);
  const tenantService = inject(TenantService);

  // Headers dinámicos contextuales
  const tenantId = tenantService.config()?.tenant_id;
  let headers = req.headers;

  if (tenantId && !headers.has('X-Tenant-Id')) {
    headers = headers.set('X-Tenant-Id', tenantId);
  }

  // Token de sesión para entorno local (Bearer fallback ante aislamiento de cookies cross-port)
  const token = typeof window !== 'undefined' ? sessionStorage.getItem('solv_token') : null;
  if (token && !headers.has('Authorization')) {
    headers = headers.set('Authorization', `Bearer ${token}`);
  }

  // Clonar request asegurando withCredentials: true para transportar la cookie HttpOnly solv_session
  const authReq = req.clone({
    withCredentials: true,
    headers
  });

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // 401 Unauthorized: sesión expirada o ausente
      if (error.status === 401) {
        const currentUrl = router.url;
        if (!currentUrl.includes('/login')) {
          router.navigate(['/login'], { queryParams: { returnUrl: currentUrl } });
        }
      }
      return throwError(() => error);
    })
  );
};
