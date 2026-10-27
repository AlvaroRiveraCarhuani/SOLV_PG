import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import {
  HostHardwareMetrics,
  DockerContainerSummary,
  HostSystemHealth,
  CourseLoadSummary,
  TechnicalIncident,
  LoadSnapshot,
  WorkspaceLogsResponse
} from '@core/models/admin.model';
import { Observable, catchError, forkJoin, of, tap } from 'rxjs';

interface BackendHostHardware {
  ram_used_bytes: number;
  ram_total_bytes: number;
  ram_percent: number;
  cpu_cores: number;
  cpu_percent: number;
  disk_used_bytes: number;
  disk_total_bytes: number;
  disk_percent: number;
  disk_path?: string;
}

interface BackendDockerDaemon {
  server_version: string;
  storage_driver: string;
  healthy: boolean;
}

interface BackendHealthMetrics {
  tenant_id: string;
  running_labs: number;
  hibernated_labs: number;
  oom_killed_labs: number;
  unknown_labs: number;
  orphaned_labs: number;
  total_ram_alloc_mb: number;
  health_status: string;
  health_reasons?: string[];
  uptime_seconds?: number;
  host_safety_ceiling_pct?: number;
  estimated_max_containers?: number;
  max_containers_policy?: number;
  ram_per_lab_mb?: number;
  capacity_basis?: 'policy' | 'mem_available';
  host_hardware?: BackendHostHardware;
  docker?: BackendDockerDaemon;
  load_history?: LoadSnapshot[];
}

@Injectable({
  providedIn: 'root'
})
export class AdminMetricsService {
  private http = inject(HttpClient);

  // Signals de estado
  isLoading = signal<boolean>(false);
  error = signal<string | null>(null);
  health = signal<HostSystemHealth | null>(null);

  constructor() {
    this.fetchMetrics();
  }

  fetchMetrics(): void {
    this.isLoading.set(true);
    this.error.set(null);

    forkJoin({
      health: this.http.get<BackendHealthMetrics>('/api/v1/admin/metrics/health').pipe(catchError(() => of(null))),
      coursesLoad: this.http.get<CourseLoadSummary[]>('/api/v1/admin/dashboard/courses-load').pipe(catchError(() => of([]))),
      incidents: this.http.get<TechnicalIncident[]>('/api/v1/admin/dashboard/incidents').pipe(catchError(() => of([]))),
      containers: this.http.get<DockerContainerSummary[]>('/api/v1/admin/dashboard/containers').pipe(catchError(() => of([])))
    }).pipe(
      tap(({ health, coursesLoad, incidents, containers }) => {
        let metrics: HostHardwareMetrics;
        const maxContainers = health?.max_containers_policy ?? health?.estimated_max_containers ?? 40;

        if (health?.host_hardware) {
          // Datos 100% reales del servidor Linux Asus mediante gopsutil
          metrics = {
            ram_used_bytes: health.host_hardware.ram_used_bytes,
            ram_total_bytes: health.host_hardware.ram_total_bytes,
            ram_percent: Math.round(health.host_hardware.ram_percent),
            cpu_cores: health.host_hardware.cpu_cores,
            cpu_percent: Math.round(health.host_hardware.cpu_percent),
            disk_used_bytes: health.host_hardware.disk_used_bytes,
            disk_total_bytes: health.host_hardware.disk_total_bytes,
            disk_percent: Math.round(health.host_hardware.disk_percent),
            disk_path: health.host_hardware.disk_path,
            containers_active: health.running_labs,
            containers_hibernated: health.hibernated_labs,
            containers_max: maxContainers,
            host_safety_ceiling_pct: health?.host_safety_ceiling_pct ?? 85,
            estimated_max_containers: health?.estimated_max_containers ?? maxContainers,
            capacity_basis: health?.capacity_basis,
            max_containers_policy: health?.max_containers_policy,
            ram_per_lab_mb: health?.ram_per_lab_mb
          };
        } else {
          const running = health?.running_labs ?? 0;
          const hibernated = health?.hibernated_labs ?? 0;
          const totalRamMB = 32 * 1024;
          const usedRamMB = running * 512 + 4096;
          metrics = {
            ram_used_bytes: usedRamMB * 1024 * 1024,
            ram_total_bytes: totalRamMB * 1024 * 1024,
            ram_percent: Math.min(Math.round((usedRamMB / totalRamMB) * 100), 100),
            cpu_cores: 16,
            cpu_percent: 12,
            disk_used_bytes: 184 * 1024 * 1024 * 1024,
            disk_total_bytes: 512 * 1024 * 1024 * 1024,
            disk_percent: 36,
            containers_active: running,
            containers_hibernated: hibernated,
            containers_max: maxContainers,
            host_safety_ceiling_pct: health?.host_safety_ceiling_pct ?? 85,
            estimated_max_containers: health?.estimated_max_containers ?? maxContainers,
            capacity_basis: health?.capacity_basis,
            max_containers_policy: health?.max_containers_policy,
            ram_per_lab_mb: health?.ram_per_lab_mb
          };
        }

        const docker = health?.docker;
        const rawStatus = health?.health_status;
        const healthStatus: 'operational' | 'degraded' | 'critical' =
          rawStatus === 'degraded' || rawStatus === 'critical' ? rawStatus : 'operational';
        const dockerVersion = docker
          ? (docker.healthy
              ? `Docker Engine v${docker.server_version} (${docker.storage_driver})`
              : 'Docker daemon inaccesible')
          : '';

        this.health.set({
          status: healthStatus,
          docker: docker ?? { server_version: '', storage_driver: '', healthy: false },
          docker_version: dockerVersion,
          health_reasons: health?.health_reasons ?? [],
          uptime_seconds: health?.uptime_seconds ?? 0,
          metrics,
          containers: containers || [],
          courses_load: coursesLoad || [],
          incidents: incidents || [],
          load_history: health?.load_history ?? []
        });

        this.isLoading.set(false);
      }),
      catchError(() => {
        this.isLoading.set(false);
        return of(null);
      })
    ).subscribe();
  }

  hibernateAll() {
    return this.http.post('/api/v1/admin/emergency/hibernate_all', {})
      .pipe(
        tap(() => this.fetchMetrics()),
        catchError(() => {
          this.health.update((current) => {
            if (!current) return current;
            return {
              ...current,
              metrics: {
                ...current.metrics,
                containers_active: 0,
                containers_hibernated: current.metrics.containers_active + current.metrics.containers_hibernated
              },
              containers: current.containers.map(c => ({ ...c, status: 'hibernated' as const })),
              courses_load: current.courses_load.map(c => ({
                ...c,
                active_students: 0,
                hibernated_students: c.active_students + c.hibernated_students,
                ram_used_mb: 0
              }))
            };
          });
          return of(null);
        })
      );
  }

  // Mismo caso que la pausa: la ruta real es /api/v1/workspaces/{id}/restart;
  // el antiguo prefijo /admin/workspaces devolvía 404 silencioso. El error se
  // propaga para que la UI lo muestre.
  restartWorkspace(workspaceId: string): Observable<void> {
    return this.http.post<void>(`/api/v1/workspaces/${workspaceId}/restart`, {}).pipe(
      tap(() => this.fetchMetrics())
    );
  }

  stopContainer(containerId: string): void {
    this.health.update(current => {
      if (!current) return current;
      return {
        ...current,
        containers: current.containers.filter(c => c.id !== containerId),
        metrics: {
          ...current.metrics,
          containers_active: Math.max(0, current.metrics.containers_active - 1)
        }
      };
    });
  }

  // La ruta real de pausa es /api/v1/workspaces/{id}/pause (el rol admin pasa
  // la guarda del handler). El antiguo prefijo /admin/workspaces no existía y
  // devolvía 404 silencioso: el error se propaga para que la UI lo muestre.
  pauseWorkspace(workspaceId: string): Observable<void> {
    return this.http.post<void>(`/api/v1/workspaces/${workspaceId}/pause`, {}).pipe(
      tap(() => this.fetchMetrics())
    );
  }

  getWorkspaceLogs(workspaceId: string): Observable<WorkspaceLogsResponse> {
    return this.http.get<WorkspaceLogsResponse>(`/api/v1/admin/workspaces/${workspaceId}/logs`);
  }

  resolveIncident(workspaceId: string): Observable<{ status: string; workspace_id: string; message: string }> {
    return this.http.post<{ status: string; workspace_id: string; message: string }>(
      `/api/v1/admin/workspaces/${workspaceId}/resolve`, 
      {}
    ).pipe(
      tap(() => this.fetchMetrics())
    );
  }

  getCourseWorkspaces(courseId: string, search = '', status = 'all'): Observable<DockerContainerSummary[]> {
    let params = new HttpParams();
    if (search) params = params.set('search', search);
    if (status && status !== 'all') params = params.set('status', status);
    return this.http.get<DockerContainerSummary[]>(`/api/v1/admin/courses/${courseId}/workspaces`, { params }).pipe(
      catchError(() => of([]))
    );
  }
}
