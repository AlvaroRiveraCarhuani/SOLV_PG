import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, catchError, map, throwError } from 'rxjs';

export interface Administrator {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  status: string;
  origin: string;
  created_at: string;
  last_login_at?: string | null;
}

export interface CreateAdministratorPayload {
  first_name: string;
  last_name: string;
  email: string;
}

export interface ApiEnvelope<T> {
  data: T;
  message?: string;
  error?: string;
}

@Injectable({ providedIn: 'root' })
export class AdminConfigAdministradoresService {
  private readonly http = inject(HttpClient);

  readonly administrators = signal<Administrator[]>([]);
  readonly isLoading = signal<boolean>(false);
  readonly isSubmitting = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  loadAdministrators(): Observable<Administrator[]> {
    this.isLoading.set(true);
    this.error.set(null);

    return this.http.get<ApiEnvelope<Administrator[]>>('/api/v1/admin/administrators').pipe(
      map((res) => res.data || []),
      tap((admins) => {
        this.administrators.set(admins);
        this.isLoading.set(false);
      }),
      catchError((err) => {
        const errorMsg = err?.error?.message || err?.error?.error || 'Error al cargar la lista de administradores.';
        this.error.set(errorMsg);
        this.isLoading.set(false);
        return throwError(() => err);
      })
    );
  }

  createAdministrator(payload: CreateAdministratorPayload): Observable<Administrator> {
    this.isSubmitting.set(true);
    this.error.set(null);

    return this.http.post<ApiEnvelope<Administrator>>('/api/v1/admin/administrators', payload).pipe(
      map((res) => res.data),
      tap((newAdmin) => {
        this.administrators.update((list) => [...list, newAdmin]);
        this.isSubmitting.set(false);
      }),
      catchError((err) => {
        this.isSubmitting.set(false);
        const errorMsg = err?.error?.message || err?.error?.error || 'No se pudo registrar el administrador.';
        this.error.set(errorMsg);
        return throwError(() => err);
      })
    );
  }

  revokeAdministrator(id: string): Observable<void> {
    this.isSubmitting.set(true);
    this.error.set(null);

    return this.http.post<ApiEnvelope<{ id: string; status: string }>>(`/api/v1/admin/administrators/${id}/revoke`, {}).pipe(
      map(() => void 0),
      tap(() => {
        this.administrators.update((list) => list.filter((a) => a.id !== id));
        this.isSubmitting.set(false);
      }),
      catchError((err) => {
        this.isSubmitting.set(false);
        const errorMsg = err?.error?.message || err?.error?.error || 'No se pudo revocar al administrador.';
        this.error.set(errorMsg);
        return throwError(() => err);
      })
    );
  }
}
