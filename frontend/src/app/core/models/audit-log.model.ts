/**
 * Modelo de audit logs (ADR-027) y enriquecimiento semántico del
 * wireframe AUDIT_LOGS.md: traducción de verbos HTTP a eventos legibles,
 * resolución de actores institucionales y status semántico.
 */

export interface AuditLog {
  id: string;
  tenant_id: string;
  actor_id: string;
  /** Email institucional resuelto por el backend; fallback actor_id. */
  actor_email: string;
  action: string;
  resource_type: string;
  resource_id?: string | null;
  status_code: number;
  metadata?: Record<string, unknown> | null;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export type AuditActionKind = 'create' | 'update' | 'delete' | 'other';

export interface EnrichedAction {
  label: string;
  kind: AuditActionKind;
}

export interface StatusInfo {
  label: string;
  kind: 'success' | 'info' | 'error';
}

/** Traduce la acción técnica (verbo + ruta o constante) a evento legible. */
export function enrichAction(action: string): EnrichedAction {
  if (action.startsWith('POST')) {
    return { label: 'Creación', kind: 'create' };
  }
  if (action.startsWith('PUT') || action.startsWith('PATCH')) {
    return { label: 'Actualización', kind: 'update' };
  }
  if (action.startsWith('DELETE')) {
    return { label: 'Eliminación', kind: 'delete' };
  }
  return { label: humanizeConstant(action), kind: 'other' };
}

/** Convierte constantes de dominio (EMERGENCY_*) a texto legible. */
export function humanizeConstant(action: string): string {
  return action
    .toLowerCase()
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

/**
 * Estado semántico del código HTTP (colores del wireframe AUDIT_LOGS.md).
 * El color depende de la combinación verbo + status: PUT exitoso se muestra
 * azul/neutro (Actualización), no verde; 403 en DELETE rojo, etc.
 */
export function describeStatus(statusCode: number, actionKind?: AuditActionKind): StatusInfo {
  if (statusCode >= 200 && statusCode < 300) {
    if (actionKind === 'update') {
      return { label: `${statusCode} OK`, kind: 'info' };
    }
    const label =
      statusCode === 201 ? 'Creado' :
      statusCode === 204 ? 'Eliminado' :
      'OK';
    return { label: `${statusCode} ${label}`, kind: 'success' };
  }
  if (statusCode < 400) {
    return { label: `${statusCode} Redirección`, kind: 'info' };
  }
  const label =
    statusCode === 400 ? 'Inválido' :
    statusCode === 401 ? 'No autenticado' :
    statusCode === 403 ? 'Denegado' :
    statusCode === 404 ? 'No encontrado' :
    statusCode === 409 ? 'Conflicto' :
    statusCode === 422 ? 'No procesable' :
    statusCode === 429 ? 'Límite' :
    'Error';
  return { label: `${statusCode} ${label}`, kind: 'error' };
}

/** Extrae el motivo legible del metadata cuando existe. */
export function metadataReason(log: AuditLog): string | null {
  const meta = log.metadata;
  if (!meta) {
    return null;
  }
  const reason = (meta['reason'] ?? meta['motivo']) as unknown;
  return typeof reason === 'string' && reason.trim() ? reason.trim() : null;
}
