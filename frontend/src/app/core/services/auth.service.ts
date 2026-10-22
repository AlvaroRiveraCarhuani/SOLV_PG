import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, tap, catchError, of, switchMap, map } from 'rxjs';
import { User, UserRole } from '@core/models/user.model';
import { TenantService } from '@core/services/tenant.service';

/** Respuesta de GET /api/v1/users/me: puede venir envuelta en { data: ... } o cruda. */
type MeResponse = { data?: User } | User;

function unwrapUser(resp: MeResponse | null | undefined): User | null {
  if (!resp) return null;
  if ('data' in resp) {
    return resp.data ?? null;
  }
  return resp as User;
}

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
   * Canjea un token transaccional de invitación docente (ADR-025).
   */
  acceptTeacherInvitation(token: string): Observable<{ status: string; message: string }> {
    return this.http.post<{ status: string; message: string }>(
      '/api/v1/invitations/teachers/accept',
      { token },
      { withCredentials: true }
    );
  }

  /**
   * Resuelve la identidad del usuario actual vía GET /api/v1/users/me.
   * El backend lee la cookie 'solv_session' o el header ForwardAuth.
   * Si existe una invitación pendiente en la sesión, la canjea automáticamente.
   */
  resolveCurrentUser(): Observable<User | null> {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      const urlToken = params.get('token');
      if (urlToken) {
        sessionStorage.setItem('solv_token', urlToken);
        const cleanUrl = window.location.pathname;
        window.history.replaceState({}, document.title, cleanUrl);
      }
      const inviteToken = params.get('invite');
      if (inviteToken) {
        sessionStorage.setItem('solv_invite_token', inviteToken);
      }
    }

    this.isLoading.set(true);
    return this.http.get<MeResponse>('/api/v1/users/me', { withCredentials: true }).pipe(
      switchMap((resp) => {
        const user = unwrapUser(resp);
        const pendingInvite = typeof window !== 'undefined' ? sessionStorage.getItem('solv_invite_token') : null;
        if (pendingInvite && user) {
          return this.acceptTeacherInvitation(pendingInvite).pipe(
            switchMap(() => {
              sessionStorage.removeItem('solv_invite_token');
              return this.http.get<MeResponse>('/api/v1/users/me', { withCredentials: true }).pipe(
                map((uResp) => unwrapUser(uResp) as User)
              );
            }),
            catchError((err) => {
              console.warn('[AuthService] No se pudo canjear la invitación docente:', err);
              sessionStorage.removeItem('solv_invite_token');
              return of(user);
            })
          );
        }
        return of(user);
      }),
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
    if (typeof window !== 'undefined') {
      sessionStorage.removeItem('solv_token');
    }
    this.http.post('/api/v1/auth/logout', {}).pipe(
      catchError(() => of(null))
    ).subscribe(() => {
      this.currentUser.set(null);
      this.router.navigate(['/login']);
    });
  }
}
