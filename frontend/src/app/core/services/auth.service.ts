import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, catchError, of } from 'rxjs';
import { User, UserRole } from '@core/models/user.model';
import { TenantService } from '@core/services/tenant.service';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly tenantService = inject(TenantService);

  // Signals reactivos de sesión
  readonly currentUser = signal<User | null>(null);
  readonly isLoading = signal<boolean>(false);

  // Computeds derivados
  readonly isAuthenticated = computed(() => !!this.currentUser());
  readonly role = computed<UserRole | null>(() => this.currentUser()?.role ?? null);
  readonly isStudent = computed(() => this.role() === 'student');
  readonly isTeacher = computed(() => this.role() === 'teacher');
  readonly isAdmin = computed(() => this.role() === 'admin');

  /**
   * Inicia el flujo OAuth2 con Google Workspace redirigiendo al backend.
   * Traefik y Go gestionan la verificación de dominio permitido (@uab.edu.bo)
   * y depositan la cookie HttpOnly 'solv_session' (ADR-017).
   */
  loginWithGoogle(): void {
    const tenant = this.tenantService.config();
    const tenantId = tenant?.tenant_id || '00000000-0000-0000-0000-000000000001';
    
    // Redirección directa al endpoint de autenticación de Go
    window.location.href = `/auth/google/login?tenant_id=${encodeURIComponent(tenantId)}`;
  }

  /**
   * Resuelve la identidad del usuario actual vía GET /api/v1/users/me.
   * El backend lee la cookie 'solv_session' o el header ForwardAuth.
   */
  resolveCurrentUser(): Observable<User | null> {
    this.isLoading.set(true);
    return this.http.get<User>('/api/v1/users/me').pipe(
      tap((user) => {
        this.currentUser.set(user);
        this.isLoading.set(false);
      }),
      catchError(() => {
        this.currentUser.set(null);
        this.isLoading.set(false);
        return of(null);
      })
    );
  }

  /**
   * Cierra la sesión activa revocando la cookie HttpOnly y redirigiendo a login.
   */
  logout(): void {
    this.http.post('/api/v1/auth/logout', {}).pipe(
      catchError(() => of(null))
    ).subscribe(() => {
      this.currentUser.set(null);
      this.router.navigate(['/login']);
    });
  }
}
