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
  kind: 'neutral' | 'info' | 'warning' | 'error' | 'success';
}

/**
 * Catálogo de acciones de dominio emitidas por el backend.
 * La clave es el identificador técnico en mayúsculas que guarda el backend;
 * sólo se traduce el texto que ve el usuario en la interfaz.
 */
export const AUDIT_ACTION_LABELS: Record<string, string> = {
  // Avance de clase y congelamiento
  CLASS_ADVANCE_APPLIED: 'Avance de clase aplicado',
  CLASS_ADVANCE_EMITTED: 'Avance de clase emitido',
  CLASS_ADVANCE_DISMISSED: 'Avance de clase descartado',
  CLASS_ADVANCE_GRACE_SET: 'Plazo de gracia de clase fijado',
  CLASS_ADVANCE_GRACE_EXTENDED: 'Plazo de gracia de clase extendido',
  CLASS_ADVANCE_GRACE_DEADLINE_SET: 'Límite de gracia de clase fijado',
  CLASS_ADVANCE_GRACE_CLOSED: 'Plazo de gracia de clase cerrado',
  CLASS_ADVANCE_ROLLBACK: 'Reversión de avance de clase',
  CLASS_FROZEN: 'Sesión de clase congelada',
  CLASS_UNFROZEN: 'Sesión de clase descongelada',

  // Exámenes y entornos de evaluación
  EXAM_PRACTICE_ENVIRONMENT_FROZEN: 'Práctica de examen congelada',
  EXAM_PRACTICE_ENVIRONMENT_UNFROZEN: 'Práctica de examen descongelada',
  EXAM_NETWORK_EGRESS_GRANTED: 'Acceso a red en examen concedido',

  // Ejercicios y entregas
  EXERCISE_CREATED: 'Ejercicio creado',
  EXERCISE_PUBLISHED: 'Ejercicio publicado',
  EXERCISE_PURPOSE_CHANGED: 'Propósito de ejercicio modificado',
  EXERCISE_GRADE_SELECTED: 'Calificación de ejercicio seleccionada',
  EVALUATOR_RUN: 'Evaluador ejecutado',
  GRADES_EXPORTED: 'Notas de laboratorio exportadas',
  SUBMISSION_REVIEWED: 'Entrega revisada',
  SUBMISSION_OVERRIDDEN: 'Nota de entrega modificada',
  SUBMISSION_COMMENTED: 'Comentario en entrega',
  SUBMISSION_REVIEW_UPDATED: 'Revisión de entrega actualizada',

  // Plantillas y tecnologías
  TEMPLATE_CATEGORY_DELETED: 'Categoría de plantilla eliminada',
  TEMPLATE_MODEL_DEACTIVATED: 'Modelo de plantilla desactivado',
  TEMPLATE_MODEL_REACTIVATED: 'Modelo de plantilla reactivado',
  TEMPLATE_PROMOTED_TO_MODEL: 'Plantilla promovida a modelo',
  TEMPLATE_COMPONENT_CREATED: 'Tecnología creada',
  TEMPLATE_REVIEWED: 'Plantilla revisada',
  TEMPLATE_REGISTERED: 'Plantilla registrada',
  TEMPLATE_QUOTA_UPDATED: 'Cuota de plantilla actualizada',
  TEMPLATE_DUPLICATED: 'Plantilla duplicada',
  IMAGE_VERIFICATION_FORCE_BYPASS: 'Omisión forzada de verificación de imagen',

  // Gobernanza y administración
  ADMINISTRATOR_CREATED: 'Administrador creado',
  ADMINISTRATOR_REVOKED: 'Administrador revocado',
  ORPHAN_VOLUME_RELEASED: 'Volumen huérfano liberado',

  // Reconciliación de entornos
  PRACTICE_ENVIRONMENT_RECONCILED_ORPHANED: 'Entorno reconciliado como huérfano',
  PRACTICE_ENVIRONMENT_RECONCILED_EXITED: 'Entorno reconciliado como detenido',
  PRACTICE_ENVIRONMENT_RECONCILED_RECOVERED: 'Entorno reconciliado y recuperado',
  PRACTICE_ENVIRONMENT_RECONCILED_UNKNOWN_PERSISTENT: 'Entorno persistente desconocido detectado',
  PRACTICE_ENVIRONMENT_HIBERNATED: 'Entorno hibernado',
  COURSE_WORKSPACES_PAUSED: 'Espacios de materia pausados',
  SUBJECT_CREATED: 'Materia creada',
  CREATE_SUBJECT: 'Materia creada',

  // Mantenimiento
  MAINTENANCE_ENABLED: 'Mantenimiento activado',
  MAINTENANCE_DISABLED: 'Mantenimiento desactivado',
  MAINTENANCE_AUTO_DISABLED: 'Mantenimiento desactivado automáticamente',

  // Emergencias
  EMERGENCY_HIBERNATE_ALL: 'Emergencia: hibernación total',
  EMERGENCY_TERMINATE_ALL: 'Emergencia: terminación de todos los entornos',
  EMERGENCY_KILL_ZOMBIES: 'Emergencia: limpieza de contenedores huérfanos',
  EMERGENCY_PRUNE_DOCKER: 'Emergencia: depuración de Docker',
  EMERGENCY_RESET_POOLS: 'Emergencia: reinicio de pools'
};

/** Catálogo de traducción para tipos de recursos en auditoría. */
export const AUDIT_RESOURCE_LABELS: Record<string, string> = {
  class_session: 'Sesión de clase',
  template_component: 'Tecnología',
  lab_template: 'Plantilla',
  template_category: 'Categoría',
  template_model: 'Modelo',
  emergency_action: 'Acción de emergencia',
  environment: 'Entorno',
  environment_instance: 'Entorno',
  user: 'Usuario',
  docker_volume: 'Volumen Docker',
  docker_image: 'Imagen Docker',
  exercise: 'Ejercicio',
  submission: 'Entrega',
  maintenance: 'Mantenimiento',
  subject: 'Materia',
  subjects: 'Materias',
  course: 'Curso'
};

/** Traduce el identificador técnico de un recurso a español. */
export function auditResourceLabel(resourceType: string): string {
  if (!resourceType) return '';
  const key = resourceType.trim().toLowerCase();
  return AUDIT_RESOURCE_LABELS[key] ?? humanizeConstant(resourceType);
}

/** Traduce una constante de dominio conocida; conserva el crudo humanizado como respaldo. */
export function auditActionLabel(action: string): string {
  return AUDIT_ACTION_LABELS[action] ?? humanizeConstant(action);
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
  return { label: auditActionLabel(action), kind: 'other' };
}

/** Convierte constantes de dominio a texto legible capitalizado como respaldo. */
export function humanizeConstant(action: string): string {
  return action
    .toLowerCase()
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

/**
 * Estado semántico del código HTTP con silencio visual en 2xx.
 * 2xx se presenta en tono neutro discreto; ámbar/rojo se reservan para 4xx/5xx.
 */
export function describeStatus(statusCode: number, _actionKind?: AuditActionKind): StatusInfo {
  if (statusCode >= 200 && statusCode < 300) {
    const label =
      statusCode === 201 ? '201 Creado' :
      statusCode === 204 ? '204 Eliminado' :
      statusCode === 200 ? '200 Exitoso' :
      `${statusCode} Exitoso`;
    return { label, kind: 'neutral' };
  }
  if (statusCode >= 300 && statusCode < 400) {
    return { label: `${statusCode} Redirección`, kind: 'info' };
  }
  if (statusCode >= 400 && statusCode < 500) {
    const text =
      statusCode === 400 ? 'Inválido' :
      statusCode === 401 ? 'No autenticado' :
      statusCode === 403 ? 'Denegado' :
      statusCode === 404 ? 'No encontrado' :
      statusCode === 409 ? 'Conflicto' :
      statusCode === 422 ? 'No procesable' :
      statusCode === 429 ? 'Límite superado' :
      'Rechazado';
    return { label: `${statusCode} ${text}`, kind: 'warning' };
  }
  const text =
    statusCode === 500 ? 'Error interno' :
    statusCode === 502 ? 'Error de pasarela' :
    statusCode === 503 ? 'Servicio no disponible' :
    statusCode === 504 ? 'Tiempo de espera de pasarela' :
    'Error de servidor';
  return { label: `${statusCode} ${text}`, kind: 'error' };
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
