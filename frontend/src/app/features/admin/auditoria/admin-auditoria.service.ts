import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { AuditLog } from '@core/models/audit-log.model';

export interface AuditLogListResponse {
  tenant_id: string;
  limit: number;
  offset: number;
  data: AuditLog[];
}

export interface ActorTimelineResponse {
  tenant_id: string;
  actor_id: string;
  count: number;
  data: AuditLog[];
}

export interface EmergencyActionResult {
  action: string;
  affected_count: number;
  executed_by: string;
  message: string;
}

/** Acciones de emergencia reales del backend (ADR-032). */
export const EMERGENCY_ACTIONS = [
  {
    id: 'terminate_all_workspaces',
    label: 'Terminar todos los workspaces',
    phrase: 'TERMINAR TODOS LOS WORKSPACES',
    impact: 'destructiva' as const,
    description: 'Detiene y destruye todos los contenedores activos del tenant.'
  },
  {
    id: 'hibernate_all_workspaces',
    label: 'Hibernar todos los workspaces',
    phrase: 'HIBERNAR TODOS LOS WORKSPACES',
    impact: 'operativa' as const,
    description: 'Pausa todos los contenedores activos preservando su estado.'
  },
  {
    id: 'kill_zombies',
    label: 'Limpiar zombies Docker',
    phrase: 'LIMPIAR ZOMBIES DOCKER',
    impact: 'operativa' as const,
    description: 'Elimina contenedores huérfanos o zombies del daemon Docker.'
  },
  {
    id: 'docker_prune',
    label: 'Purgar imágenes y volúmenes huérfanos',
    phrase: 'PURGAR CAPAS HUERFANAS',
    impact: 'operativa' as const,
    description: 'Libera espacio en disco eliminando capas huérfanas sin tocar volúmenes nombrados.'
  },
  {
    id: 'reset_pools',
    label: 'Reiniciar pools y circuit breakers',
    phrase: 'REINICIAR POOLS',
    impact: 'operativa' as const,
    description: 'Reinicia el pool de conexiones de PostgreSQL y contadores de circuit breakers.'
  }
] as const;

export type EmergencyActionId = (typeof EMERGENCY_ACTIONS)[number]['id'];

/**
 * Servicio del submódulo 14.7 (Auditoría y Emergencias).
 * Auditoría: GET /api/v1/admin/audit-logs y timeline por actor.
 * Emergencias: POST /api/v1/admin/emergency/{action} (ADR-032).
 */
@Injectable({ providedIn: 'root' })
export class AdminAuditoriaService {
  private readonly http = inject(HttpClient);

  readonly isExecuting = signal(false);

  listAuditLogs(page: number, limit = 20, filters: { action?: string; actorId?: string } = {}): Observable<AuditLogListResponse> {
    let params = new HttpParams()
      .set('limit', limit)
      .set('offset', (page - 1) * limit);
    if (filters.action) {
      params = params.set('action', filters.action);
    }
    if (filters.actorId) {
      params = params.set('actor_id', filters.actorId);
    }
    return this.http.get<AuditLogListResponse>('/api/v1/admin/audit-logs', { params });
  }

  getActorTimeline(actorId: string, limit = 200): Observable<ActorTimelineResponse> {
    return this.http.get<ActorTimelineResponse>(
      `/api/v1/admin/audit-logs/actors/${actorId}/timeline`,
      { params: new HttpParams().set('limit', limit) }
    );
  }

  executeEmergencyAction(action: EmergencyActionId, reason: string): Observable<EmergencyActionResult> {
    this.isExecuting.set(true);
    return this.http.post<EmergencyActionResult>(`/api/v1/admin/emergency/${action}`, { reason }).pipe();
  }
}
