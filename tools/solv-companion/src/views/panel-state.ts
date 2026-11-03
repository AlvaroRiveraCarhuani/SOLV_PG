import { AdvanceFileEntry, AdvancePreview } from '../domain/advance';

export type Role = 'student' | 'teacher';
export type ConnectionState = 'connected' | 'connecting' | 'offline';
export type DeliveryState = 'draft' | 'submitted' | 'graded';

export interface CourseContext {
  code: string;
  title: string;
  teacher: string;
  delivery: DeliveryState;
  score?: number;
  maxScore?: number;
}

export interface RubricLevelView {
  name: string;
  score: number;
  description: string;
}

export interface RubricCriterionView {
  id?: string;
  name: string;
  weight: number;
  description?: string;
  levels?: RubricLevelView[];
}

export interface AssignmentContext {
  title: string;
  summary: string;
  criteria: string[];
  rubricCriteria?: RubricCriterionView[];
  totalPoints?: number;
  purpose?: 'class' | 'exam' | string;
}

export interface AdvanceView {
  id: string;
  title: string;
  createdAt: string;
  files: AdvanceFileEntry[];
  applied: boolean;
  canRollback: boolean;
}

export interface CommentView {
  id: string;
  file: string;
  line: number;
  author: string;
  text: string;
}

export interface TeacherView {
  sessionOpen: boolean;
  frozen: boolean;
  sessionId?: string;
  present: number;
  enrolled: number;
  submitted: number;
  lastAdvance?: { id: string; createdAt: string; recipients: number };
}

export interface PanelState {
  role: Role;
  connection: ConnectionState;
  course: CourseContext;
  assignment: AssignmentContext;
  advance?: AdvanceView;
  comments: CommentView[];
  teacher?: TeacherView;
  canSubmit: boolean;
  message?: { kind: 'info' | 'warning' | 'error'; text: string };
}

/**
 * Estado de la entrega a partir de la nota. Una entrega registrada sin nota
 * está entregada; con nota, calificada.
 */
export function deriveDeliveryState(score: number | null | undefined): DeliveryState {
  return typeof score === 'number' ? 'graded' : 'submitted';
}

/**
 * Arma el checklist del avance comparando lo emitido con lo que el alumno tiene
 * en disco. Un archivo que no existe localmente es nuevo; uno que existe y
 * cambia es modificado; uno idéntico queda sin cambios y desmarcado.
 */
export function buildChecklist(
  preview: AdvancePreview,
  localContents: Record<string, string | undefined>
): AdvanceFileEntry[] {
  const added = new Set(preview.added);
  return Object.keys(preview.files)
    .sort((a, b) => a.localeCompare(b))
    .map(path => {
      if (added.has(path)) {
        return { path, status: 'new' as const, selected: true };
      }
      const local = localContents[path];
      if (local === undefined || local !== preview.files[path]) {
        return { path, status: 'modified' as const, selected: true };
      }
      return { path, status: 'unchanged' as const, selected: false };
    });
}

/** Rutas que el alumno dejó marcadas en el checklist. */
export function selectedPaths(files: AdvanceFileEntry[]): string[] {
  return files.filter(file => file.selected).map(file => file.path);
}

/** Aplica la selección que llega de la webview sin inventar rutas. */
export function applySelection(files: AdvanceFileEntry[], selected: string[]): AdvanceFileEntry[] {
  const chosen = new Set(selected);
  return files.map(file => ({ ...file, selected: chosen.has(file.path) }));
}

/** Texto del badge de la rúbrica según el propósito pedagógico. */
export function rubricBadgeText(purpose?: string): string {
  return purpose === 'exam' ? 'Rúbrica de evaluación' : 'Rúbrica de práctica';
}

/** Título del encabezado de la rúbrica según el propósito pedagógico. */
export function rubricTitleText(purpose?: string): string {
  return purpose === 'exam' ? 'Rúbrica de Evaluación' : 'Rúbrica de Práctica';
}

/** Texto del botón de entrega en la barra de estado según el propósito pedagógico. */
export function submissionActionText(purpose?: string): string {
  return purpose === 'exam' ? '$(cloud-upload) Entregar Examen' : '$(cloud-upload) Entregar Práctica';
}
