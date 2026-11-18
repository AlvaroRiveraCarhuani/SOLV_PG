/** Avance del docente tal como lo devuelve GET /api/v1/class-advances/pending. */
export type AdvanceRecipientStatus = 'pending' | 'applied' | 'dismissed' | 'failed' | 'expired';

export interface AdvanceDelivery {
  id: string;
  session_id: string;
  exercise_id: string;
  title?: string;
  scope: string;
  snapshot_sha256: string;
  snapshot_size_bytes: number;
  file_count: number;
  created_by?: string | null;
  created_at: string;
  status: AdvanceRecipientStatus;
  decided_at?: string;
  subject_id: string;
  session_status: 'open' | 'closed';
  grace_until?: string;
  purpose: string;
}

/** Confirmación previa de GET /api/v1/class-advances/{id}/files. */
export interface AdvancePreview {
  advance_id: string;
  exercise_id: string;
  file_count: number;
  added: string[];
  overwritten: string[];
  /** Contenido emitido por archivo; alimenta el diff en memoria. */
  files: Record<string, string>;
}

export interface AdvanceApplyResult {
  advance_id: string;
  files_written: number;
  overwritten: string[];
  added: string[];
  backup_path?: string;
}

export interface AdvanceRollbackResult {
  advance_id: string;
  files_restored: number;
  restored: string[];
  /** Archivos que siguen respaldados: mientras haya, el avance sigue aplicado. */
  remaining: string[];
  fully_restored: boolean;
  backup_path: string;
}

/** Archivos del respaldo que todavía se pueden volver a su versión anterior. */
export interface AdvanceRollbackInventory {
  advance_id: string;
  files: string[];
}

/** Estado de un archivo del avance frente al proyecto del alumno. */
export type AdvanceFileStatus = 'new' | 'modified' | 'unchanged';

export interface AdvanceFileEntry {
  path: string;
  status: AdvanceFileStatus;
  selected: boolean;
}

export interface EmittedAdvance {
  advance: {
    id: string;
    session_id: string;
    exercise_id: string;
    scope: string;
    file_count: number;
    created_at: string;
  };
  recipients: string[];
}

export type AdvanceScope = 'class' | 'selected';
