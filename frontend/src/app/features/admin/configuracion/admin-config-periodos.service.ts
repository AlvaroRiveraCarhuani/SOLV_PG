import { Injectable, inject, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, map, catchError, of, switchMap } from 'rxjs';

export interface AcademicPeriodConfig {
  id: string;
  tenant_id: string;
  name: string;
  code: string;
  start_date: string;
  end_date: string;
  is_active: boolean;
  is_archived?: boolean;
  archived_at?: string | null;
  archived_by?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface CreatePeriodPayload {
  name: string;
  code: string;
  start_date: string;
  end_date: string;
  is_active?: boolean;
}

export type PeriodLifecycle = 'activo' | 'proximo' | 'archivado';

interface ApiErrorResponse {
  error?: string;
  message?: string;
}

/**
 * Ciclo de vida del período (ADR-029 con archivado formal en BD):
 * - activo:    is_active = true
 * - proximo:   planificado, sin archivar, con vigencia vigente o futura
 * - archivado: is_archived = true (congelamiento formal e irreversible) o
 *              vencido que el sweep formalizó con is_archived
 */
export function periodLifecycle(period: AcademicPeriodConfig, today: Date = new Date()): PeriodLifecycle {
  if (period.is_active) return 'activo';
  if (period.is_archived) return 'archivado';
  const end = new Date(period.end_date + 'T23:59:59');
  return end >= today ? 'proximo' : 'archivado';
}

@Injectable({ providedIn: 'root' })
export class AdminConfigPeriodosService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api/v1/admin/academic-periods';

  readonly periods = signal<AcademicPeriodConfig[]>([]);
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<string | null>(null);

  readonly activePeriod = computed<AcademicPeriodConfig | null>(
    () => this.periods().find((p) => p.is_active) ?? null
  );

  readonly kpis = computed(() => {
    const today = new Date();
    const list = this.periods();
    return {
      total: list.length,
      activeCourses: 0,
      nextUp: list.filter((p) => periodLifecycle(p, today) === 'proximo').length,
      archived: list.filter((p) => periodLifecycle(p, today) === 'archivado').length
    };
  });

  fetchPeriods(): Observable<AcademicPeriodConfig[]> {
    this.isLoading.set(true);
    this.error.set(null);
    return this.http
      .get<{ data?: AcademicPeriodConfig[] } | AcademicPeriodConfig[]>(this.baseUrl)
      .pipe(
        map((res) => (Array.isArray(res) ? res : res.data ?? [])),
        tap((list) => this.periods.set(list)),
        catchError(() => {
          this.error.set('No se pudo conectar con el servicio de períodos académicos.');
          return of([]);
        }),
        tap(() => this.isLoading.set(false))
      );
  }

  createPeriod(payload: CreatePeriodPayload): Observable<AcademicPeriodConfig> {
    return this.http
      .post<{ data?: AcademicPeriodConfig } | AcademicPeriodConfig>(this.baseUrl, payload)
      .pipe(
        map((res) => (res as { data?: AcademicPeriodConfig }).data ?? (res as AcademicPeriodConfig)),
        tap((period) => this.periods.update((prev) => [...prev, period]))
      );
  }

  /**
   * Actualiza campos del período. Cuando setActive es true el backend garantiza
   * exclusividad: desactiva el resto de períodos del tenant en la misma transacción.
   */
  updatePeriod(
    id: string,
    dto: { name?: string; code?: string; start_date?: string; end_date?: string; is_active?: boolean }
  ): Observable<AcademicPeriodConfig> {
    return this.http
      .put<{ data?: AcademicPeriodConfig } | AcademicPeriodConfig>(`${this.baseUrl}/${id}`, dto)
      .pipe(
        map((res) => (res as { data?: AcademicPeriodConfig }).data ?? (res as AcademicPeriodConfig)),
        tap((updated) =>
          this.periods.update((prev) => prev.map((p) => (p.id === id ? updated : p)))
        )
      );
  }

  /**
   * Archivo formal con confirmación fuerte (ADR-029): congela el período y
   * sella sus materias en modo solo lectura. Irreversible desde la API.
   */
  archivePeriod(id: string, confirmationCode: string): Observable<{ status: string }> {
    return this.http.post<{ status: string }>(`${this.baseUrl}/${id}/archive`, {
      confirmation_code: confirmationCode
    }).pipe(
      tap(() =>
        this.periods.update((prev) =>
          prev.map((p) =>
            p.id === id
              ? { ...p, is_active: false, is_archived: true, archived_at: new Date().toISOString() }
              : p
          )
        )
      )
    );
  }

  deletePeriod(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`).pipe(
      tap(() => this.periods.update((prev) => prev.filter((p) => p.id !== id)))
    );
  }

  /**
   * Alta + activación inmediata como unidad transaccional para el usuario
   * (evita ventanas con doble período activo). Los errores de activación
   * (p. ej. 422 period_expired) se propagan al llamador.
   */
  createAndActivate(payload: CreatePeriodPayload): Observable<AcademicPeriodConfig> {
    return this.createPeriod({ ...payload, is_active: false }).pipe(
      switchMap((created) =>
        payload.is_active
          ? this.updatePeriod(created.id, { is_active: true }).pipe(map(() => created))
          : of(created)
      )
    );
  }

  /** Errores de negocio del backend (422/409) traducidos a mensajes accionables. */
  resolveError(err: unknown): string {
    const e = err as ApiErrorResponse;
    const code = e?.error;
    if (code === 'invalid_date_range') {
      return 'La fecha de fin debe ser posterior o igual a la fecha de inicio.';
    }
    if (code === 'period_expired') {
      return 'No se puede activar un período académico cuya fecha ya finalizó.';
    }
    if (code === 'conflict_associated_subjects') {
      return 'No se puede eliminar un período con materias asociadas. Desasócialas primero.';
    }
    if (code === 'period_archived') {
      return 'El período está formalmente archivado y es inmutable (ADR-029).';
    }
    if (code === 'confirmation_failed') {
      return 'El código de confirmación no coincide con el código del período.';
    }
    return e?.message || 'Ocurrió un error inesperado. Intenta nuevamente.';
  }
}
