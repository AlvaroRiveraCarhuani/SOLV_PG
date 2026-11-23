import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap, catchError, of, forkJoin } from 'rxjs';

// ---------------------------------------------------------------------------
// Contratos alineados al backend real
// ---------------------------------------------------------------------------

// Supuestos con los que el backend calcula cuántos entornos sostiene este host.
// capacity_basis = 'unavailable' cuando no pudo medir la RAM del servidor.
export interface CapacityRecommendation {
  recommended_max_containers: number;
  host_total_mb: number;
  platform_reserve_mb: number;
  ram_per_lab_mb: number;
  host_safety_ceiling_pct: number;
  capacity_basis: 'host_ram' | 'unavailable';
}

export interface ServerPolicies {
  ram_limit_mb?: number;
  inactivity_minutes: number;
  max_containers?: number;
  host_safety_ceiling_pct?: number;
  updated_at?: string;
  recommendation?: CapacityRecommendation;
}

export interface UpdateServerPoliciesPayload {
  host_safety_ceiling_pct?: number;
  inactivity_minutes?: number;
  ram_limit_mb?: number;
  max_containers?: number;
}

export interface MaintenanceStatus {
  maintenance_mode: boolean;
  maintenance_until?: string | null;
  maintenance_reason?: string;
}

export interface BackupConfigItem {
  id: string;
  tenant_id: string;
  local_frequency_hours: number;
  local_retention_days: number;
  remote_enabled: boolean;
  remote_provider: string;
  remote_bucket_name: string;
  remote_endpoint: string;
  remote_retention_days: number;
  is_active: boolean;
  updated_at: string;
  storage_path?: string;
  disk_free_bytes?: number;
  next_scheduled_at?: string | null;
}

export type BackupConfig = BackupConfigItem;

export interface BackupExecutionItem {
  id: string;
  file_name: string;
  file_size_bytes: number;
  sha256_checksum: string;
  storage_tier: 'local' | 'remote' | 'both';
  status: 'in_progress' | 'success' | 'failed';
  error_message?: string;
  started_at: string;
  completed_at?: string | null;
  last_verify_ok?: boolean | null;
  last_verify_at?: string | null;
}

/** Respuesta GlobalResponse del backend: { data, error, message } */
interface ApiEnvelope<T> {
  data: T;
  message?: string;
}

/** Volumen que el inventario decidió conservar, con el motivo. */
export interface KeptVolumeItem {
  name: string;
  size_mb: number;
  reason: string;
}

export interface OrphanVolumePlan {
  orphans: { name: string; size_mb: number }[];
  kept: KeptVolumeItem[];
  total_orphan_mb: number;
  total_kept_mb: number;
}

export interface OrphanVolumePruneResult {
  deleted: { name: string; size_mb: number }[];
  skipped: KeptVolumeItem[];
  freed_mb: number;
  remaining_orphan_mb: number;
}

/** Resultado tipado de verifyBackup: { data: { is_valid, ... } } */
export interface VerifyBackupResult {
  execution_id: string;
  file_name: string;
  database_checksum: string;
  computed_checksum: string;
  is_valid: boolean;
  message: string;
}

@Injectable({ providedIn: 'root' })
export class AdminConfigServidorService {
  private readonly http = inject(HttpClient);

  readonly policies = signal<ServerPolicies | null>(null);
  readonly maintenance = signal<MaintenanceStatus | null>(null);
  readonly backupConfig = signal<BackupConfigItem | null>(null);
  readonly backups = signal<BackupExecutionItem[]>([]);
  readonly isLoading = signal(false);
  readonly error = signal<string | null>(null);
  readonly orphanPlan = signal<OrphanVolumePlan | null>(null);
  readonly orphansLoading = signal(false);
  readonly orphansError = signal<string | null>(null);
  /** Momento de la última lectura del daemon que sí terminó. */
  readonly orphansCheckedAt = signal<string | null>(null);

  loadAll(): void {
    this.isLoading.set(true);
    this.error.set(null);

    forkJoin({
      policies: this.http.get<ApiEnvelope<ServerPolicies>>('/api/v1/admin/server/policies').pipe(
        tap((res) => this.policies.set(res.data)),
        catchError(() => {
          this.error.set('No se pudo cargar la configuración del servidor.');
          return of(null);
        })
      ),
      maintenance: this.http.get<MaintenanceStatus>('/api/v1/admin/maintenance/status').pipe(
        tap((status) => this.maintenance.set(status)),
        catchError(() => of(null))
      ),
      backupConfig: this.http.get<ApiEnvelope<BackupConfigItem>>('/api/v1/admin/backups/config').pipe(
        tap((res) => this.backupConfig.set(res.data)),
        catchError(() => of(null))
      ),
      backups: this.http.get<ApiEnvelope<BackupExecutionItem[]> | BackupExecutionItem[]>('/api/v1/admin/backups').pipe(
        tap((res) => {
          const data = (res as ApiEnvelope<BackupExecutionItem[]>).data ?? (res as BackupExecutionItem[]);
          this.backups.set(Array.isArray(data) ? data : []);
        }),
        catchError(() => of(null))
      )
    }).subscribe({
      next: () => this.isLoading.set(false),
      error: () => this.isLoading.set(false)
    });
  }

  /**
   * Lee el daemon y cruza los volúmenes sin contenedor contra los entornos en base.
   * El callback opcional avisa que la lectura terminó, para que quien la pidió
   * pueda confirmar el resultado en vez de dejar al usuario sin respuesta.
   */
  loadOrphanVolumes(onDone?: () => void): void {
    this.orphansLoading.set(true);
    this.orphansError.set(null);
    this.http.get<ApiEnvelope<OrphanVolumePlan>>('/api/v1/admin/server/storage/orphans').subscribe({
      next: (res) => {
        this.orphanPlan.set(res.data);
        this.orphansCheckedAt.set(new Date().toISOString());
        this.orphansLoading.set(false);
        onDone?.();
      },
      error: (err) => {
        this.orphansError.set(this.resolveError(err));
        this.orphansLoading.set(false);
      }
    });
  }

  pruneOrphanVolumes(volumeNames: string[], confirmationPhrase: string): Observable<ApiEnvelope<OrphanVolumePruneResult>> {
    return this.http.post<ApiEnvelope<OrphanVolumePruneResult>>('/api/v1/admin/server/storage/orphans/prune', {
      volume_names: volumeNames,
      confirmation_phrase: confirmationPhrase
    });
  }

  updatePolicies(dto: UpdateServerPoliciesPayload | Partial<ServerPolicies>): Observable<ApiEnvelope<ServerPolicies>> {
    return this.http.put<ApiEnvelope<ServerPolicies>>('/api/v1/admin/server/policies', dto).pipe(
      tap((res) => this.policies.set(res.data))
    );
  }

  enableMaintenance(until: string, reason: string, confirmPhrase = ''): Observable<unknown> {
    return this.http.post('/api/v1/admin/maintenance/enable', { until, reason, confirm_phrase: confirmPhrase }).pipe(
      tap(() => this.maintenance.set({ maintenance_mode: true, maintenance_until: until || null, maintenance_reason: reason }))
    );
  }

  disableMaintenance(): Observable<unknown> {
    return this.http.post('/api/v1/admin/maintenance/disable', {}).pipe(
      tap(() => this.maintenance.set({ maintenance_mode: false }))
    );
  }

  updateBackupConfig(dto: Partial<BackupConfigItem>): Observable<ApiEnvelope<BackupConfigItem>> {
    return this.http.put<ApiEnvelope<BackupConfigItem>>('/api/v1/admin/backups/config', dto).pipe(
      tap((res) => this.backupConfig.set(res.data))
    );
  }

  triggerBackup(): Observable<ApiEnvelope<BackupExecutionItem>> {
    return this.http.post<ApiEnvelope<BackupExecutionItem>>('/api/v1/admin/backups/trigger', {}).pipe(
      tap((res) => this.backups.update((prev) => [res.data, ...prev]))
    );
  }

  verifyBackup(id: string): Observable<ApiEnvelope<VerifyBackupResult>> {
    return this.http.post<ApiEnvelope<VerifyBackupResult>>(`/api/v1/admin/backups/${id}/verify`, {});
  }

  /**
   * El archivo se pide por HttpClient, no por navegación directa: el endpoint
   * exige Authorization y ese encabezado lo agrega el interceptor, cosa que un
   * href no hace (el navegador abre una pestaña sin credenciales y responde 401).
   */
  downloadBackup(id: string): Observable<Blob> {
    return this.http.get(`/api/v1/admin/backups/${id}/download`, { responseType: 'blob' });
  }

  resolveError(err: unknown): string {
    const e = err as { error?: { message?: string; error?: string }; message?: string };
    return e?.error?.message ?? e?.message ?? 'Ocurrió un error inesperado. Intente nuevamente.';
  }
}
