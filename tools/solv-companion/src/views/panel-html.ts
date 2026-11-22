import * as vscode from 'vscode';
import { escapeHtml, renderMarkdown } from './markdown';
import {
  AdvanceView,
  PanelState,
  advanceFileActionText,
  excerptText,
  rubricBadgeText,
  rubricTitleText
} from './panel-state';

const TIMESTAMP_FORMAT = new Intl.DateTimeFormat('es', {
  day: '2-digit',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit'
});

export function formatTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return TIMESTAMP_FORMAT.format(date);
}

type IconName =
  | 'book'
  | 'file'
  | 'diff'
  | 'upload'
  | 'undo'
  | 'comment'
  | 'broadcast'
  | 'users'
  | 'freeze'
  | 'external'
  | 'warning'
  | 'check'
  | 'clock'
  | 'pause'
  | 'play'
  | 'chevron-down';

const ICON_PATHS: Record<IconName, string> = {
  book: '<path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H19v18H6.5A2.5 2.5 0 0 1 4 18.5z"/><path d="M8 3v18"/>',
  file: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>',
  diff: '<path d="M6 3v12"/><circle cx="6" cy="18" r="2.5"/><circle cx="6" cy="8" r="2.5"/><path d="M18 21V9"/><circle cx="18" cy="6" r="2.5"/><path d="M6 10.5v5"/>',
  upload: '<path d="M12 16V4"/><path d="m7 9 5-5 5 5"/><path d="M5 20h14"/>',
  undo: '<path d="M4 9h9a5 5 0 0 1 0 10H8"/><path d="m8 5-4 4 4 4"/>',
  comment: '<path d="M20 15a2 2 0 0 1-2 2H8l-4 4V6a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2z"/>',
  broadcast: '<circle cx="12" cy="12" r="2.5"/><path d="M7.7 7.7a6 6 0 0 0 0 8.6"/><path d="M16.3 16.3a6 6 0 0 0 0-8.6"/><path d="M4.6 4.6a10 10 0 0 0 0 14.8"/><path d="M19.4 19.4a10 10 0 0 0 0-14.8"/>',
  users: '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20a6.5 6.5 0 0 1 13 0"/><path d="M16 5.5a3.5 3.5 0 0 1 0 5"/><path d="M17.5 20a6.5 6.5 0 0 0-3-5.5"/>',
  freeze: '<path d="M12 2v20"/><path d="m4 6 16 12"/><path d="M20 6 4 18"/><path d="m9 4 3 2 3-2"/><path d="m9 20 3-2 3 2"/>',
  pause: '<rect width="4" height="16" x="6" y="4"/><rect width="4" height="16" x="14" y="4"/>',
  play: '<polygon points="6 3 20 12 6 21 6 3"/>',
  external: '<path d="M14 4h6v6"/><path d="M20 4 11 13"/><path d="M18 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h5"/>',
  warning: '<path d="M12 3 2.5 20h19z"/><path d="M12 9v5"/><path d="M12 17v.5"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7"/>',
  clock: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>',
  'chevron-down': '<path d="m6 9 6 6 6-6"/>'
};

function icon(name: IconName): string {
  return `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICON_PATHS[name]}</svg>`;
}

/** Indicador del botón que disparó la acción en curso. */
function spinner(): string {
  return '<span class="spinner" aria-hidden="true"></span>';
}

/**
 * Contenido de un botón de acción: el que está en curso muestra un indicador y
 * el texto que describe lo que se está haciendo; el resto vuelve a su rótulo.
 */
function actionContent(
  state: PanelState,
  action: string,
  iconName: IconName | undefined,
  idleLabel: string
): string {
  if (state.busy?.action === action) {
    return `${spinner()}${escapeHtml(state.busy.label)}`;
  }
  return `${iconName ? icon(iconName) : ''}${escapeHtml(idleLabel)}`;
}

/** Con una acción en curso se bloquean todos los botones del panel. */
function busyAttributes(state: PanelState): string {
  return state.busy ? ' disabled aria-disabled="true"' : '';
}

function nonce(): string {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let value = '';
  for (let index = 0; index < 32; index += 1) {
    value += alphabet.charAt(Math.floor(Math.random() * alphabet.length));
  }
  return value;
}

function deliveryLabel(state: PanelState): { text: string; tone: string } {
  const { delivery, score, maxScore } = state.course;
  if (delivery === 'graded') {
    return { text: `Calificada ${score ?? 0}/${maxScore ?? 100}`, tone: 'graded' };
  }
  if (delivery === 'submitted') {
    return { text: 'Entregada', tone: 'submitted' };
  }
  return { text: 'Borrador', tone: 'draft' };
}

function badgeFor(status: 'new' | 'modified' | 'unchanged'): string {
  if (status === 'new') {
    return '<span class="badge badge-new">Nuevo</span>';
  }
  if (status === 'modified') {
    return '<span class="badge badge-modified">Modificado</span>';
  }
  return '<span class="badge badge-neutral">Sin cambios</span>';
}

function connectionLabel(state: PanelState): string {
  if (state.connection === 'connected') {
    return 'Conexión activa';
  }
  if (state.connection === 'connecting') {
    return 'Conectando';
  }
  return 'Sin conexión';
}

/**
 * Aviso del avance aplicado. Con inventario el alumno elige qué archivos
 * vuelven atrás; sin inventario (consulta fallida o avance aplicado antes de
 * este cambio) se mantiene el deshacer completo.
 */
function appliedAdvanceNotice(advance: AdvanceView): string {
  if (advance.appliedFiles === undefined) {
    return `<p class="notice notice-info">${icon('check')}Avance aplicado a tu entorno. Puedes deshacerlo para volver a la versión anterior.</p>`;
  }
  if (advance.appliedFiles.length === 0) {
    return `<p class="notice notice-info">${icon('check')}Avance aplicado. El avance agregó archivos nuevos: no hay versión anterior para restaurar.</p>`;
  }
  return `<p class="notice notice-info">${icon('check')}Avance aplicado a tu entorno. Elegí qué archivos volver a su versión anterior; el resto queda como está.</p>`;
}

/** Checklist del avance aplicado: lo que el respaldo todavía puede restaurar. */
function appliedAdvanceList(advance: AdvanceView): string {
  const files = advance.appliedFiles ?? [];
  if (files.length === 0) {
    return '';
  }
  const rows = files
    .map(
      path => `
        <li class="file-row">
          <label class="file-check">
            <input type="checkbox" data-path="${escapeHtml(path)}" checked>
            <span class="file-icon">${icon('file')}</span>
            <code class="file-path">${escapeHtml(path)}</code>
          </label>
        </li>`
    )
    .join('');
  return `<ul class="file-list">${rows}</ul>`;
}

function appliedAdvanceActions(state: PanelState, advance: AdvanceView): string {
  if (advance.appliedFiles === undefined) {
    return `<button class="btn btn-secondary" data-action="rollback"${busyAttributes(state)}>${actionContent(state, 'rollback', 'undo', 'Deshacer avance')}</button>`;
  }
  if (advance.appliedFiles.length === 0) {
    return '';
  }
  return `<button class="btn btn-secondary" data-action="rollback"${busyAttributes(state)}>${actionContent(state, 'rollback', 'undo', 'Deshacer seleccionados')}</button>`;
}

function renderAdvance(state: PanelState): string {
  const advance = state.advance;
  if (!advance) {
    return '';
  }

  const rows = advance.files
    .map(
      file => `
        <li class="file-row">
          <label class="file-check">
            <input type="checkbox" data-path="${escapeHtml(file.path)}" ${file.selected ? 'checked' : ''}>
            <span class="file-icon">${icon('file')}</span>
            <code class="file-path">${escapeHtml(file.path)}</code>
          </label>
          <span class="file-meta">
            ${badgeFor(file.status)}
            ${
              file.status === 'unchanged'
                ? ''
                : `<button class="link-button" data-action="diff" data-path="${escapeHtml(file.path)}">${icon('diff')}${advanceFileActionText(file.status)}</button>`
            }
          </span>
        </li>`
    )
    .join('');

  return `
    <section class="section" aria-labelledby="advance-title">
      <div class="section-head">
        <h2 id="advance-title">${escapeHtml(advance.title || 'Avance del docente')}</h2>
        <time class="muted" datetime="${escapeHtml(advance.createdAt)}">${escapeHtml(formatTimestamp(advance.createdAt))}</time>
      </div>

      ${
        advance.applied
          ? appliedAdvanceNotice(advance)
          : `<p class="notice">El docente compartió una actualización. Revisa el checklist y elige qué archivos incorporar.</p>`
      }

      ${advance.applied ? appliedAdvanceList(advance) : `<ul class="file-list">${rows}</ul>`}

      <div class="actions">
        ${
          advance.applied
            ? appliedAdvanceActions(state, advance)
            : `<button class="btn btn-primary" data-action="apply"${busyAttributes(state)}>${actionContent(state, 'apply', 'upload', 'Aplicar archivos seleccionados')}</button>
               <button class="btn btn-ghost" data-action="dismiss"${busyAttributes(state)}>${actionContent(state, 'dismiss', undefined, 'Descartar avance')}</button>`
        }
      </div>
    </section>`;
}

function renderComments(state: PanelState): string {
  if (state.role !== 'student') {
    return '';
  }
  if (state.comments.length === 0) {
    return `
      <section class="section doc-section" id="comments" data-doc-section="comments" aria-labelledby="comments-title">
        <h2 id="comments-title">Observaciones del docente</h2>
        <p class="muted">Todavía no hay observaciones sobre esta entrega.</p>
      </section>`;
  }

  const grouped = new Map<string, typeof state.comments>();
  for (const comment of state.comments) {
    const list = grouped.get(comment.file) ?? [];
    list.push(comment);
    grouped.set(comment.file, list);
  }

  const groups = Array.from(grouped.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(
      ([file, comments]) => `
        <div class="comment-group">
          <p class="comment-group-title"><code>${escapeHtml(file)}</code></p>
          <ul class="comment-list">
            ${comments
              .map(
                comment => `
              <li>
                <button class="comment" data-action="open-comment" data-file="${escapeHtml(comment.file)}" data-line="${comment.line}" data-comment="${escapeHtml(comment.id)}">
                  <span class="comment-head">
                    <span class="comment-line">Línea ${comment.line}</span>
                    <span class="comment-author">${escapeHtml(comment.author)}</span>
                  </span>
                  <span class="comment-body">${escapeHtml(comment.text)}</span>
                </button>
              </li>`
              )
              .join('')}
          </ul>
        </div>`
    )
    .join('');

  return `
    <section class="section doc-section" id="comments" data-doc-section="comments" aria-labelledby="comments-title">
      <h2 id="comments-title">Observaciones del docente</h2>
      ${groups}
    </section>`;
}

function formatRemainingTime(endsAt?: string | null): string | null {
  if (!endsAt) {
    return null;
  }
  const end = new Date(endsAt).getTime();
  if (Number.isNaN(end)) {
    return null;
  }
  const diffMs = end - Date.now();
  if (diffMs <= 0) {
    return 'Finalizada';
  }
  const totalMin = Math.round(diffMs / 60000);
  if (totalMin < 60) {
    return `${totalMin} min`;
  }
  const hours = Math.floor(totalMin / 60);
  const mins = totalMin % 60;
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
}

function renderTeacher(state: PanelState): string {
  const teacher = state.teacher;
  if (!teacher) {
    return '';
  }

  const isExam = state.assignment?.purpose === 'exam';
  const ratio =
    teacher.enrolled > 0 ? Math.min(100, Math.round((teacher.present / teacher.enrolled) * 100)) : 0;
  const remaining = teacher.sessionOpen ? formatRemainingTime(teacher.endsAt) : null;

  return `
    <section class="section" aria-labelledby="class-title">
      <div class="section-head">
        <h2 id="class-title">Aula</h2>
        <span class="badge ${
          teacher.sessionOpen
            ? teacher.frozen
              ? 'badge-stuck'
              : 'badge-new'
            : 'badge-neutral'
        }">${
          teacher.sessionOpen
            ? teacher.frozen
              ? isExam
                ? 'Examen en pausa'
                : 'Clase en pausa'
              : remaining
                ? `Sesión abierta · ${remaining}`
                : 'Sesión abierta'
            : 'Sin sesión'
        }</span>
      </div>

      <div class="metrics">
        <div class="metric">
          <span class="metric-label">${icon('users')}Presentes</span>
          <span class="metric-value">${teacher.present} / ${teacher.enrolled}</span>
        </div>
        <div class="metric">
          <span class="metric-label">${icon('upload')}Entregas</span>
          <span class="metric-value">${teacher.submitted}</span>
        </div>
        ${
          teacher.sessionOpen && remaining
            ? `<div class="metric">
                <span class="metric-label">${icon('clock')}Restante</span>
                <span class="metric-value">${escapeHtml(remaining)}</span>
              </div>`
            : ''
        }
      </div>

      <div class="presence" role="img" aria-label="Presencia ${teacher.present} de ${teacher.enrolled} alumnos">
        <span class="presence-fill" style="width:${ratio}%"></span>
      </div>

      <div class="actions"${state.busy ? ' aria-busy="true"' : ''}>
        ${
          teacher.sessionOpen
            ? isExam
              ? `<button class="btn btn-secondary" data-action="toggle-pause"${busyAttributes(state)}>${actionContent(
                  state,
                  'toggle-pause',
                  teacher.frozen ? 'play' : 'pause',
                  teacher.frozen ? 'Reanudar examen' : 'Pausar examen'
                )}</button>
                 <button class="btn btn-ghost" data-action="close-aula"${busyAttributes(state)}>${actionContent(state, 'close-aula', undefined, 'Cerrar examen')}</button>`
              : `<button class="btn btn-primary" data-action="broadcast"${busyAttributes(state)}>${actionContent(state, 'broadcast', 'broadcast', 'Emitir avance a la clase')}</button>
                 <button class="btn btn-secondary" data-action="toggle-pause"${busyAttributes(state)}>${actionContent(
                   state,
                   'toggle-pause',
                   teacher.frozen ? 'play' : 'pause',
                   teacher.frozen ? 'Reanudar clase' : 'Pausar clase'
                 )}</button>
                 <button class="btn btn-ghost" data-action="close-aula"${busyAttributes(state)}>${actionContent(state, 'close-aula', undefined, 'Cerrar clase')}</button>`
            : `<button class="btn btn-primary" data-action="open-aula" style="width: 100%; justify-content: center;"${busyAttributes(state)}>${actionContent(state, 'open-aula', 'users', 'Iniciar clase')}</button>`
        }
      </div>

      ${
        teacher.lastAdvance
          ? `<p class="muted">Último avance emitido: <time datetime="${escapeHtml(teacher.lastAdvance.createdAt)}">${escapeHtml(formatTimestamp(teacher.lastAdvance.createdAt))}</time> a ${teacher.lastAdvance.recipients} alumnos.</p>`
          : ''
      }
    </section>`;
}

/**
 * Acceso al SpeedGrader. Vive al final del panel y del documento: el aula se
 * opera arriba y la evaluación es la salida al terminar la clase.
 */
function renderEvaluation(state: PanelState): string {
  if (state.role === 'student') {
    return '';
  }
  return `
    <section class="section" aria-labelledby="evaluation-title">
      <h2 id="evaluation-title">Evaluación</h2>
      <p class="muted">Revisa las entregas del aula y califícalas con la rúbrica.</p>
      <button class="link-button" data-action="speedgrader">${icon('external')}Abrir evaluación en la plataforma</button>
    </section>`;
}

function renderRubric(assignment: PanelState['assignment']): string {
  const rubric = assignment.rubricCriteria;
  if (!rubric || rubric.length === 0) {
    if (!assignment.criteria || assignment.criteria.length === 0) {
      return '';
    }
    return `<div class="section-head"><h2 id="rubric-title">${escapeHtml(rubricTitleText(assignment.purpose))}</h2></div><ul class="criteria">${assignment.criteria
      .map(item => `<li>${escapeHtml(item)}</li>`)
      .join('')}</ul>`;
  }

  const totalPoints =
    assignment.totalPoints ??
    rubric.reduce((sum, item) => sum + (item.weight || 0), 0);

  // 1. Vista Compacta (< 600px, Sidebar): Acordeón con <details>
  const compactCards = rubric
    .map((criterion, index) => {
      const pts = criterion.weight;
      const title = criterion.name;
      const desc = criterion.description || '';
      const levels = criterion.levels || [];

      const levelsHtml =
        levels.length > 0
          ? `<div class="rubric-levels-chips">
              ${levels
                .map(lvl => {
                  const calculated = Math.round((lvl.score / 100) * pts * 10) / 10;
                  return `<div class="level-chip">
                    <div class="level-chip-top">
                      <span class="level-name">${escapeHtml(lvl.name)}</span>
                      <span class="level-score">${lvl.score}% · ${calculated} pts</span>
                    </div>
                    ${lvl.description ? `<p class="level-desc">${escapeHtml(lvl.description)}</p>` : ''}
                  </div>`;
                })
                .join('')}
            </div>`
          : '';

      return `<details class="rubric-accordion-item" data-detail-key="rubric-criterion-${index}"${index === 0 ? ' open' : ''}>
        <summary class="rubric-summary">
          <div class="summary-left">
            <span class="rubric-weight-badge">${pts} pts</span>
            <span class="criterion-name" title="${escapeHtml(title)}">${escapeHtml(title)}</span>
          </div>
          <span class="chevron" aria-hidden="true">${icon('chevron-down')}</span>
        </summary>
        <div class="rubric-accordion-body">
          ${desc ? `<p class="criterion-description">${escapeHtml(desc)}</p>` : ''}
          ${levelsHtml}
        </div>
      </details>`;
    })
    .join('');

  // 2. Vista Matriz Ancha (>= 600px, Pestaña completa)
  const matrixRows = rubric
    .map(criterion => {
      const pts = criterion.weight;
      const title = criterion.name;
      const desc = criterion.description || '';
      const levels = criterion.levels || [];
      const hasLevels = levels.length > 0;

      if (!hasLevels) {
        return `<div class="matrix-row matrix-row-simple">
          <div class="matrix-criterion-head">
            <span class="matrix-weight-pill">${pts} pts</span>
            <strong class="matrix-criterion-title">${escapeHtml(title)}</strong>
          </div>
          ${desc ? `<p class="matrix-criterion-desc">${escapeHtml(desc)}</p>` : ''}
        </div>`;
      }

      const levelsGrid = `<div class="matrix-levels-grid">
        ${levels
          .map(lvl => {
            const calculated = Math.round((lvl.score / 100) * pts * 10) / 10;
            return `<div class="matrix-level-card">
              <div class="matrix-level-header">
                <span class="matrix-level-name">${escapeHtml(lvl.name)}</span>
                <span class="matrix-level-pts">${lvl.score}% · ${calculated} pts</span>
              </div>
              ${lvl.description ? `<p class="matrix-level-desc">${escapeHtml(lvl.description)}</p>` : ''}
            </div>`;
          })
          .join('')}
      </div>`;

      return `<div class="matrix-row">
        <div class="matrix-criterion-col">
          <div class="matrix-criterion-head">
            <span class="matrix-weight-pill">${pts} pts</span>
            <strong class="matrix-criterion-title">${escapeHtml(title)}</strong>
          </div>
          ${desc ? `<p class="matrix-criterion-desc">${escapeHtml(desc)}</p>` : ''}
        </div>
        <div class="matrix-levels-col">
          ${levelsGrid}
        </div>
      </div>`;
    })
    .join('');

  const rubricHeader = rubricTitleText(assignment.purpose);

  return `<details class="rubric-details" data-detail-key="rubric" open>
    <summary class="rubric-summary-header">
      <div class="rubric-header-left">
        <span class="rubric-chevron" aria-hidden="true">${icon('chevron-down')}</span>
        <h2 class="rubric-header-title" id="rubric-title">${escapeHtml(rubricHeader)}</h2>
      </div>
      <span class="rubric-total-badge">Total: ${totalPoints} pts</span>
    </summary>

    <div class="rubric-body">
      <!-- Modo compacto (panel lateral < 600px) -->
      <div class="rubric-compact-view">
        ${compactCards}
      </div>

      <!-- Modo matriz ancha (pestaña del editor >= 600px) -->
      <div class="rubric-matrix-view">
        ${matrixRows}
      </div>
    </div>
  </details>`;
}

function hasRubric(assignment: PanelState['assignment']): boolean {
  return Boolean(assignment.rubricCriteria?.length || assignment.criteria?.length);
}

function consignaBadge(state: PanelState): string {
  if (!state.assignment.rubricCriteria || state.assignment.rubricCriteria.length === 0) {
    return '';
  }
  return `<span class="badge ${state.assignment.purpose === 'exam' ? 'badge-exam' : 'badge-submitted'}">${escapeHtml(rubricBadgeText(state.assignment.purpose))}</span>`;
}

function consignaSummary(state: PanelState): string {
  return (
    state.assignment.summary ||
    (state.assignment.purpose === 'exam'
      ? 'La consigna del examen se publica desde la plataforma.'
      : 'La consigna de la práctica se publica desde la plataforma.')
  );
}

/**
 * Lateral: primero lo que exige una decisión (avance del aula u observaciones)
 * y la consigna al final como referencia, recortada a un extracto.
 */
function renderSidebar(state: PanelState): string {
  const action = state.role === 'student' ? renderAdvance(state) : renderTeacher(state);
  const comments = state.role === 'student' ? renderComments(state) : '';
  return `${renderPauseNotice(state)}${action}${comments}${renderConsignaExcerpt(state)}${renderEvaluation(state)}`;
}

/**
 * Aviso de aula pausada en el panel del alumno. El entorno ya está en solo
 * lectura: esto explica por qué el editor no guarda, sin ofrecer una acción que
 * el alumno no puede tomar.
 */
function renderPauseNotice(state: PanelState): string {
  if (state.role !== 'student' || !state.student?.paused) {
    return '';
  }
  const isExam = state.assignment?.purpose === 'exam';
  const since = state.student.pausedSince;
  return `
    <section class="section" aria-labelledby="pause-title" role="status">
      <div class="section-head">
        <h2 id="pause-title">${isExam ? 'Examen en pausa' : 'Clase en pausa'}</h2>
        <span class="badge badge-stuck">Solo lectura</span>
      </div>
      <p class="notice notice-warning">${icon('pause')}El docente pausó ${isExam ? 'el examen' : 'la clase'}. Tu entorno no acepta escritura hasta que lo reanude.</p>
      ${
        since
          ? `<p class="muted">Pausado desde <time datetime="${escapeHtml(since)}">${escapeHtml(formatTimestamp(since))}</time>.</p>`
          : ''
      }
    </section>`;
}

function renderConsignaExcerpt(state: PanelState): string {
  // Sin resumen configurado se muestra el aviso de la plataforma: la sección
  // nunca desaparece, porque es la única entrada al documento de la pestaña.
  const excerpt = excerptText(consignaSummary(state));
  return `
    <section class="section" aria-labelledby="assignment-title">
      <div class="section-head">
        <h2 id="assignment-title">Consigna</h2>
        ${consignaBadge(state)}
      </div>
      <button class="link-button" data-action="open-tab">${icon('external')}Leer consigna completa en una pestaña</button>
      <p class="excerpt">${escapeHtml(excerpt)}</p>
    </section>`;
}

/**
 * Aviso de avance en la pestaña: el checklist y sus botones viven en el panel
 * lateral, así no hay dos lugares donde aplicar lo mismo.
 */
function renderAdvanceBanner(state: PanelState): string {
  const advance = state.advance;
  if (state.role !== 'student' || !advance || advance.applied) {
    return '';
  }
  return `
    <section class="section doc-banner">
      <p class="notice">${icon('broadcast')}El docente compartió un avance y está pendiente.</p>
      <button class="btn btn-primary" data-action="focus-panel">${icon('upload')}Revisar en el panel lateral</button>
    </section>`;
}

function renderDocNav(state: PanelState): string {
  const entries: Array<{ id: string; label: string }> = [];
  // El docente trabaja desde la pestaña: el aula va primero en el índice.
  if (state.role === 'teacher') {
    entries.push({ id: 'aula', label: 'Aula' });
  }
  entries.push({ id: 'assignment', label: 'Consigna' });
  if (hasRubric(state.assignment)) {
    entries.push({ id: 'rubric', label: 'Rúbrica' });
  }
  if (state.role === 'student') {
    entries.push({ id: 'comments', label: 'Observaciones' });
  } else {
    entries.push({ id: 'evaluation', label: 'Evaluación' });
  }
  return `<nav class="doc-nav" aria-label="Secciones de la práctica">${entries
    .map(
      entry =>
        `<a class="doc-nav-link" href="#${entry.id}" data-nav-target="${entry.id}">${escapeHtml(entry.label)}</a>`
    )
    .join('')}</nav>`;
}

/**
 * Documento de la pestaña: consigna completa, rúbrica plegada y observaciones,
 * con un índice de anclas que aparece cuando hay ancho para dos columnas.
 */
function renderDocument(state: PanelState, rubricContent: string): string {
  const sections: string[] = [];

  // La pestaña es la pantalla de trabajo del docente: sin esta tarjeta, abrir
  // el documento lo dejaba sin emitir, congelar ni cerrar el aula.
  if (state.role === 'teacher') {
    sections.push(`
    <div class="doc-section" id="aula" data-doc-section="aula">
      ${renderTeacher(state)}
    </div>`);
  }

  sections.push(`
    <section class="section doc-section" id="assignment" data-doc-section="assignment">
      <div class="section-head">
        <h2>Consigna</h2>
        ${consignaBadge(state)}
      </div>
      <div class="summary markdown">${renderMarkdown(consignaSummary(state))}</div>
    </section>`);

  if (hasRubric(state.assignment) && rubricContent) {
    sections.push(`
      <section class="section doc-section" id="rubric" data-doc-section="rubric" aria-labelledby="rubric-title">
        ${rubricContent}
      </section>`);
  }

  if (state.role === 'student') {
    sections.push(renderComments(state));
  } else {
    sections.push(`
      <div class="doc-section" id="evaluation" data-doc-section="evaluation">
        ${renderEvaluation(state)}
      </div>`);
  }

  return `
    <div class="doc-layout">
      ${renderDocNav(state)}
      <article class="doc-body">
        ${renderAdvanceBanner(state)}
        ${sections.join('')}
      </article>
    </div>`;
}

/**
 * HTML del panel. Usa solo variables de tema de VS Code y SVG en línea: se
 * adapta a tema claro, oscuro y alto contraste sin librerías externas.
 */
export function renderPanelHtml(
  webview: vscode.Webview,
  state: PanelState,
  options?: { isTab?: boolean }
): string {
  const scriptNonce = nonce();
  const delivery = deliveryLabel(state);
  const rubricContent = renderRubric(state.assignment);
  const message = state.message
    ? `<p class="notice notice-${state.message.kind}">${icon(
        state.message.kind === 'info' ? 'check' : 'warning'
      )}${escapeHtml(state.message.text)}</p>`
    : '';

  return `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${scriptNonce}'; img-src ${webview.cspSource} data:;">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SOLV Companion</title>
<style>
  :root {
    color-scheme: light dark;
    --panel-gap: 14px;
    --panel-radius: 6px;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    padding: 0 0 24px;
    font-family: var(--vscode-font-family);
    font-size: var(--vscode-font-size);
    line-height: 1.45;
    color: var(--vscode-sideBar-foreground, var(--vscode-foreground));
    background-color: var(--vscode-sideBar-background, var(--vscode-editor-background));
  }
  .icon {
    width: 15px;
    height: 15px;
    flex: 0 0 auto;
    vertical-align: -2px;
  }
  .masthead {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
    padding: var(--panel-gap) var(--panel-gap) 10px;
    border-bottom: 1px solid var(--vscode-panel-border, transparent);
  }
  .masthead-main { display: flex; gap: 10px; min-width: 0; }
  .masthead-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--panel-radius);
    background: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
  }
  .course-code {
    margin: 0;
    font-family: var(--vscode-editor-font-family);
    font-size: 0.82em;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--vscode-descriptionForeground);
  }
  h1 {
    margin: 1px 0 0;
    font-size: 1.02em;
    font-weight: 600;
    line-height: 1.3;
    overflow-wrap: anywhere;
  }
  .teacher { margin: 2px 0 0; color: var(--vscode-descriptionForeground); font-size: 0.92em; }
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 7px;
    border-radius: 10px;
    border: 1px solid transparent;
    font-size: 0.78em;
    font-weight: 500;
    white-space: nowrap;
  }
  .badge-new { color: var(--vscode-charts-green); border-color: var(--vscode-charts-green); }
  .badge-modified { color: var(--vscode-charts-orange); border-color: var(--vscode-charts-orange); }
  .badge-neutral { color: var(--vscode-descriptionForeground); border-color: var(--vscode-panel-border, var(--vscode-descriptionForeground)); }
  .badge-draft { color: var(--vscode-descriptionForeground); border-color: var(--vscode-panel-border, var(--vscode-descriptionForeground)); }
  .badge-submitted { color: var(--vscode-charts-blue); border-color: var(--vscode-charts-blue); }
  .badge-graded { color: var(--vscode-charts-green); border-color: var(--vscode-charts-green); }
  .badge-exam { color: var(--vscode-charts-orange, #f59e0b); border-color: var(--vscode-charts-orange, #f59e0b); }
  .summary { margin: 0; }
  .markdown > :first-child { margin-top: 0; }
  .markdown > :last-child { margin-bottom: 0; }
  .markdown p { margin: 0 0 8px; }
  .markdown h3,
  .markdown h4,
  .markdown h5,
  .markdown h6 {
    margin: 12px 0 6px;
    font-size: 0.95em;
    font-weight: 600;
    line-height: 1.3;
    color: var(--vscode-foreground);
  }
  .markdown ul,
  .markdown ol { margin: 0 0 8px; padding-left: 18px; }
  .markdown li { margin-bottom: 3px; }
  .markdown code {
    padding: 1px 4px;
    border-radius: 3px;
    font-family: var(--vscode-editor-font-family);
    font-size: 0.92em;
    background: var(--vscode-textCodeBlock-background);
  }
  .markdown pre {
    margin: 2px 0 10px;
    padding: 10px 12px;
    max-width: 100%;
    overflow-x: auto;
    overscroll-behavior-x: contain;
    border: 1px solid var(--vscode-panel-border, transparent);
    border-radius: var(--panel-radius);
    background: var(--vscode-textCodeBlock-background);
    font-size: 0.86em;
    line-height: 1.5;
    tab-size: 2;
    white-space: pre;
  }
  .markdown pre code { padding: 0; background: transparent; font-size: inherit; }
  .markdown pre::-webkit-scrollbar { height: 8px; }
  .markdown pre::-webkit-scrollbar-thumb {
    border-radius: 8px;
    background: var(--vscode-scrollbarSlider-background);
  }
  .markdown pre::-webkit-scrollbar-thumb:hover {
    background: var(--vscode-scrollbarSlider-hoverBackground);
  }
  .markdown pre::-webkit-scrollbar-track { background: transparent; }
  .markdown .tree {
    margin: 2px 0 10px;
    padding: 8px 12px;
    border: 1px solid var(--vscode-panel-border, transparent);
    border-radius: var(--panel-radius);
    background: var(--vscode-textCodeBlock-background);
  }
  .markdown .tree-list {
    margin: 0;
    padding-left: 12px;
    list-style: none;
    border-left: 1px solid var(--vscode-panel-border, transparent);
  }
  .markdown .tree > .tree-list { padding-left: 0; border-left: none; }
  .markdown .tree-item { margin: 0; }
  .markdown .tree-row { display: flex; align-items: center; gap: 6px; padding: 1px 0; }
  .markdown .tree-icon { width: 13px; height: 13px; flex: 0 0 auto; }
  .markdown .tree-dir > .tree-row .tree-icon { color: var(--vscode-charts-blue); }
  .markdown .tree-file > .tree-row .tree-icon { color: var(--vscode-descriptionForeground); }
  .markdown .tree-name {
    font-family: var(--vscode-editor-font-family);
    font-size: 0.9em;
    line-height: 1.6;
  }
  .markdown .tree-dir > .tree-row .tree-name { font-weight: 600; }
  .markdown hr {
    margin: 12px 0;
    border: none;
    border-top: 1px solid var(--vscode-panel-border, transparent);
  }
  .markdown blockquote {
    margin: 0 0 8px;
    padding-left: 10px;
    border-left: 2px solid var(--vscode-panel-border, transparent);
    color: var(--vscode-descriptionForeground);
  }
  .markdown blockquote p { margin: 0; }
  .markdown a { color: var(--vscode-textLink-foreground); }
  .section { padding: 12px var(--panel-gap) 4px; }
  .section + .section { border-top: 1px solid var(--vscode-panel-border, transparent); }
  .section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 8px; }
  h2 {
    margin: 0 0 6px;
    font-size: 0.82em;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--vscode-descriptionForeground);
  }
  p { margin: 0 0 8px; }
  .muted { color: var(--vscode-descriptionForeground); font-size: 0.88em; }
  time { font-family: var(--vscode-editor-font-family); font-variant-numeric: tabular-nums; }
  code { font-family: var(--vscode-editor-font-family); font-size: 0.92em; }
  .criteria { margin: 0; padding-left: 18px; }
  .criteria li { margin-bottom: 3px; }
  .rubric-details { margin-top: 10px; }
  .rubric-summary-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
    cursor: pointer;
    user-select: none;
    list-style: none;
  }
  .rubric-summary-header::-webkit-details-marker {
    display: none;
  }
  .rubric-header-left {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .rubric-chevron {
    color: var(--vscode-descriptionForeground);
    display: flex;
    align-items: center;
    transition: transform 0.15s ease;
  }
  .rubric-details[open] .rubric-chevron {
    transform: rotate(180deg);
  }
  .rubric-header-title {
    margin: 0;
    font-size: 0.85em;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--vscode-foreground);
  }
  .rubric-total-badge {
    background-color: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
    padding: 2px 8px;
    border-radius: 10px;
    font-family: var(--vscode-editor-font-family);
    font-variant-numeric: tabular-nums;
    font-size: 0.82em;
    font-weight: 600;
  }
  .rubric-compact-view {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .rubric-matrix-view {
    display: none;
  }
  .rubric-accordion-item {
    border: 1px solid var(--vscode-panel-border, rgba(128, 128, 128, 0.2));
    border-radius: var(--panel-radius);
    background-color: var(--vscode-editor-background, var(--vscode-sideBar-background));
    color: var(--vscode-sideBar-foreground, var(--vscode-foreground));
    overflow: hidden;
  }
  .rubric-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px 10px;
    cursor: pointer;
    user-select: none;
    list-style: none;
    gap: 8px;
  }
  .rubric-summary::-webkit-details-marker {
    display: none;
  }
  .summary-left {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }
  .rubric-weight-badge {
    background: var(--vscode-charts-blue);
    color: #ffffff;
    font-family: var(--vscode-editor-font-family);
    font-variant-numeric: tabular-nums;
    font-size: 0.76em;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 4px;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .criterion-name {
    font-size: 0.9em;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .chevron {
    color: var(--vscode-descriptionForeground);
    display: flex;
    align-items: center;
    transition: transform 0.15s ease;
    flex-shrink: 0;
  }
  .rubric-accordion-item[open] .chevron {
    transform: rotate(180deg);
  }
  .rubric-accordion-body {
    padding: 8px 10px 10px;
    border-top: 1px solid var(--vscode-panel-border, rgba(128, 128, 128, 0.15));
    font-size: 0.88em;
  }
  .criterion-description {
    margin: 0 0 8px;
    color: var(--vscode-foreground);
    line-height: 1.4;
  }
  .rubric-levels-chips {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .level-chip {
    padding: 6px 8px;
    border-radius: 4px;
    background: var(--vscode-textBlockQuote-background, rgba(128, 128, 128, 0.08));
    border-left: 3px solid var(--vscode-charts-blue);
  }
  .level-chip-top {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    font-size: 0.86em;
    margin-bottom: 2px;
  }
  .level-name {
    font-weight: 600;
  }
  .level-score {
    color: var(--vscode-descriptionForeground);
    font-family: var(--vscode-editor-font-family);
    font-variant-numeric: tabular-nums;
    font-size: 0.82em;
  }
  .level-desc {
    margin: 0;
    color: var(--vscode-descriptionForeground);
    font-size: 0.82em;
    line-height: 1.35;
  }

  @media (min-width: 600px) {
    .rubric-compact-view {
      display: none;
    }
    .rubric-matrix-view {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .matrix-row {
      display: grid;
      grid-template-columns: 280px 1fr;
      gap: 14px;
      padding: 12px 14px;
      border: 1px solid var(--vscode-panel-border, rgba(128, 128, 128, 0.2));
      border-radius: var(--panel-radius);
      background-color: var(--vscode-editor-background, var(--vscode-sideBar-background));
      color: var(--vscode-sideBar-foreground, var(--vscode-foreground));
    }
    .matrix-row.matrix-row-simple {
      display: block;
      padding: 12px 14px;
    }
    .matrix-row.matrix-row-simple .matrix-criterion-head {
      margin-bottom: 4px;
    }
    .matrix-criterion-col {
      border-right: 1px solid var(--vscode-panel-border, rgba(128, 128, 128, 0.15));
      padding-right: 14px;
    }
    .matrix-criterion-head {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 6px;
    }
    .matrix-weight-pill {
      background: var(--vscode-charts-blue);
      color: #ffffff;
      font-family: var(--vscode-editor-font-family);
      font-variant-numeric: tabular-nums;
      font-size: 0.78em;
      font-weight: 700;
      padding: 2px 7px;
      border-radius: 4px;
    }
    .matrix-criterion-title {
      font-size: 0.96em;
    }
    .matrix-criterion-desc {
      margin: 0;
      font-size: 0.86em;
      color: var(--vscode-descriptionForeground);
      line-height: 1.4;
    }
    .matrix-levels-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 10px;
    }
    .matrix-level-card {
      padding: 8px 10px;
      border-radius: 4px;
      background: var(--vscode-textBlockQuote-background, rgba(128, 128, 128, 0.08));
      border-top: 2px solid var(--vscode-charts-blue);
      display: flex;
      flex-direction: column;
      gap: 4px;
    }
    .matrix-level-header {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      gap: 6px;
    }
    .matrix-level-name {
      font-weight: 600;
      font-size: 0.86em;
    }
    .matrix-level-pts {
      font-family: var(--vscode-editor-font-family);
      font-variant-numeric: tabular-nums;
      font-size: 0.8em;
      color: var(--vscode-descriptionForeground);
      white-space: nowrap;
    }
    .matrix-level-desc {
      margin: 0;
      font-size: 0.82em;
      color: var(--vscode-descriptionForeground);
      line-height: 1.35;
    }
  }
  .notice {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    margin: 0 0 10px;
    padding: 8px 10px;
    border-radius: var(--panel-radius);
    border-left: 3px solid var(--vscode-charts-blue);
    background: var(--vscode-textBlockQuote-background, transparent);
  }
  .notice-icon { flex: 0 0 auto; }
  .notice-info { border-left-color: var(--vscode-charts-green); }
  .notice-warning { border-left-color: var(--vscode-charts-orange); }
  .notice-error { border-left-color: var(--vscode-errorForeground); }
  .file-list, .comment-list { list-style: none; margin: 0 0 10px; padding: 0; }
  .file-row {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 7px 0;
    border-bottom: 1px solid var(--vscode-panel-border, transparent);
  }
  .file-row:last-child { border-bottom: none; }
  .file-check { display: flex; align-items: center; gap: 7px; cursor: pointer; }
  .file-check input { margin: 0; }
  .file-icon { color: var(--vscode-descriptionForeground); display: flex; }
  .file-path { overflow-wrap: anywhere; }
  .file-meta { display: flex; align-items: center; gap: 10px; padding-left: 22px; }
  .actions { display: flex; flex-wrap: wrap; gap: 8px; margin: 10px 0 6px; }
  .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 5px 12px;
    border: 1px solid transparent;
    border-radius: var(--panel-radius);
    font-family: var(--vscode-font-family);
    font-size: 0.92em;
    cursor: pointer;
  }
  .btn:focus-visible { outline: 1px solid var(--vscode-focusBorder); outline-offset: 2px; }
  .btn:disabled { opacity: 0.65; cursor: default; }
  .btn:disabled:hover { background: inherit; }
  .spinner {
    width: 13px;
    height: 13px;
    flex: 0 0 auto;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) {
    .spinner { animation-duration: 2.5s; }
  }
  .btn-primary {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
  }
  .btn-primary:hover { background: var(--vscode-button-hoverBackground); }
  .btn-secondary {
    background: var(--vscode-button-secondaryBackground);
    color: var(--vscode-button-secondaryForeground);
  }
  .btn-secondary:hover { background: var(--vscode-button-secondaryHoverBackground); }
  .btn-ghost {
    background: transparent;
    color: var(--vscode-foreground);
    border-color: var(--vscode-panel-border, var(--vscode-descriptionForeground));
  }
  .btn-ghost:hover { background: var(--vscode-toolbar-hoverBackground, transparent); }
  .link-button {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 0;
    border: none;
    background: none;
    color: var(--vscode-textLink-foreground);
    font-family: var(--vscode-font-family);
    font-size: 0.88em;
    cursor: pointer;
  }
  .link-button:hover { color: var(--vscode-textLink-activeForeground); text-decoration: underline; }
  .comment-group { margin-bottom: 12px; }
  .comment-group-title { margin: 0 0 4px; color: var(--vscode-descriptionForeground); }
  .comment {
    display: block;
    width: 100%;
    padding: 8px 10px;
    margin-bottom: 6px;
    text-align: left;
    border: 1px solid var(--vscode-panel-border, rgba(128, 128, 128, 0.2));
    border-radius: var(--panel-radius);
    background-color: var(--vscode-editor-background, var(--vscode-sideBar-background));
    color: var(--vscode-sideBar-foreground, var(--vscode-foreground));
    font-family: var(--vscode-font-family);
    font-size: 0.92em;
    cursor: pointer;
  }
  .comment:hover { border-color: var(--vscode-focusBorder); }
  .comment-head { display: flex; justify-content: space-between; gap: 8px; margin-bottom: 3px; }
  .comment-line { font-family: var(--vscode-editor-font-family); color: var(--vscode-descriptionForeground); }
  .comment-author { color: var(--vscode-descriptionForeground); }
  .comment-body { display: block; }
  .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(88px, 1fr)); gap: 8px; margin-bottom: 10px; }
  .metric {
    padding: 8px 10px;
    border: 1px solid var(--vscode-panel-border, rgba(128, 128, 128, 0.2));
    border-radius: var(--panel-radius);
    background-color: var(--vscode-editor-background, var(--vscode-sideBar-background));
    color: var(--vscode-sideBar-foreground, var(--vscode-foreground));
  }
  .metric-label {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 0.8em;
    color: var(--vscode-descriptionForeground);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .metric-value {
    display: block;
    margin-top: 2px;
    font-family: var(--vscode-editor-font-family);
    font-variant-numeric: tabular-nums;
    font-size: 1.06em;
  }
  .presence {
    height: 4px;
    margin: 0 0 12px;
    border-radius: 2px;
    background: var(--vscode-progressBar-background, var(--vscode-panel-border));
    opacity: 0.6;
    overflow: hidden;
  }
  .presence-fill {
    display: block;
    height: 100%;
    background: var(--vscode-charts-green);
  }
  footer {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px var(--panel-gap) 0;
    color: var(--vscode-descriptionForeground);
    font-size: 0.82em;
  }

  /* Extracto de consigna del panel lateral: seis líneas y el resto en la pestaña. */
  .excerpt {
    margin: 0 0 8px;
    color: var(--vscode-descriptionForeground);
    display: -webkit-box;
    -webkit-line-clamp: 6;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  /* Documento de la pestaña: índice de anclas solo cuando hay ancho. */
  html { scroll-behavior: smooth; }
  @media (prefers-reduced-motion: reduce) {
    html { scroll-behavior: auto; }
  }
  .doc-layout { display: block; }
  .doc-nav { display: none; }
  .doc-section { scroll-margin-top: 8px; }
  .doc-body > .doc-section + .doc-section {
    border-top: 1px solid var(--vscode-panel-border, transparent);
  }
  .doc-banner { padding-bottom: 10px; }
  @media (min-width: 900px) {
    .doc-layout {
      display: grid;
      grid-template-columns: 176px minmax(0, 1fr);
      align-items: start;
    }
    .doc-nav {
      display: flex;
      flex-direction: column;
      gap: 2px;
      position: sticky;
      top: 0;
      padding: 14px 6px 14px 10px;
    }
    .doc-nav-link {
      padding: 5px 8px;
      border-left: 2px solid transparent;
      border-radius: 0 var(--panel-radius) var(--panel-radius) 0;
      color: var(--vscode-descriptionForeground);
      text-decoration: none;
      font-size: 0.9em;
    }
    .doc-nav-link:hover {
      color: var(--vscode-foreground);
      background: var(--vscode-list-hoverBackground);
    }
    .doc-nav-link.is-active {
      color: var(--vscode-foreground);
      border-left-color: var(--vscode-focusBorder);
      background: var(--vscode-list-inactiveSelectionBackground);
      font-weight: 600;
    }
  }
</style>
</head>
<body>
  <header class="masthead">
    <div class="masthead-main">
      <span class="masthead-icon">${icon('book')}</span>
      <div>
        ${
          state.assignment.title
            ? `<p class="course-code">${escapeHtml(state.course.code ? (state.course.title ? `${state.course.code} · ${state.course.title}` : state.course.code) : (state.course.title || ''))}</p>
               <h1>${escapeHtml(state.assignment.title)}</h1>`
            : `${state.course.code ? `<p class="course-code">${escapeHtml(state.course.code)}</p>` : ''}
               <h1>${escapeHtml(state.course.title || 'Laboratorio SOLV')}</h1>`
        }
        ${state.course.teacher ? `<p class="teacher">${escapeHtml(state.course.teacher)}</p>` : ''}
      </div>
    </div>
    ${
      state.role === 'student'
        ? `<span class="badge badge-${delivery.tone}">${escapeHtml(delivery.text)}</span>`
        : ''
    }
  </header>

  ${message}

  ${options?.isTab ? renderDocument(state, rubricContent) : renderSidebar(state)}

  <footer>
    ${icon(state.connection === 'connected' ? 'check' : 'warning')}
    <span>${connectionLabel(state)}</span>
    <span aria-hidden="true">·</span>
    <span>SOLV Companion</span>
  </footer>

  <script nonce="${scriptNonce}">
    const host = acquireVsCodeApi();

    // El panel se reescribe en cada refresco, así que la apertura de la rúbrica
    // y de cada criterio vive en el estado de la webview: sin esto, cualquier
    // acción volvía a abrir un acordeón que la persona acababa de plegar.
    const savedDetails = (host.getState() || {}).details || {};
    document.querySelectorAll('details[data-detail-key]').forEach(details => {
      const key = details.getAttribute('data-detail-key');
      if (Object.prototype.hasOwnProperty.call(savedDetails, key)) {
        details.open = savedDetails[key];
      }
      details.addEventListener('toggle', () => {
        const next = {};
        document.querySelectorAll('details[data-detail-key]').forEach(item => {
          next[item.getAttribute('data-detail-key')] = item.open;
        });
        const state = host.getState() || {};
        state.details = next;
        host.setState(state);
      });
    });

    // Índice del documento: salto suave y sección activa según lo que se ve.
    // No reemplaza contenido, así que la consigna sigue completa al bajar.
    const navLinks = Array.from(document.querySelectorAll('[data-nav-target]'));
    if (navLinks.length > 0) {
      const markActive = id => {
        navLinks.forEach(link => {
          link.classList.toggle('is-active', link.getAttribute('data-nav-target') === id);
        });
      };
      navLinks.forEach(link => {
        link.addEventListener('click', event => {
          event.preventDefault();
          const target = document.getElementById(link.getAttribute('data-nav-target'));
          if (!target) {
            return;
          }
          target.scrollIntoView({ behavior: 'smooth', block: 'start' });
          markActive(target.id);
        });
      });
      const sections = navLinks
        .map(link => document.getElementById(link.getAttribute('data-nav-target')))
        .filter(Boolean);
      if (sections.length > 0) {
        const updateActiveSection = () => {
          const scrollBottom = window.innerHeight + window.scrollY;
          const isAtBottom = scrollBottom >= document.documentElement.scrollHeight - 30;
          if (isAtBottom) {
            markActive(sections[sections.length - 1].id);
            return;
          }
          const readingLine = 140;
          let currentId = sections[0].id;
          for (let i = 0; i < sections.length; i++) {
            const rect = sections[i].getBoundingClientRect();
            if (rect.top <= readingLine) {
              currentId = sections[i].id;
            } else {
              break;
            }
          }
          markActive(currentId);
        };

        let scrollTicking = false;
        window.addEventListener(
          'scroll',
          () => {
            if (!scrollTicking) {
              window.requestAnimationFrame(() => {
                updateActiveSection();
                scrollTicking = false;
              });
              scrollTicking = true;
            }
          },
          { passive: true }
        );
        updateActiveSection();
      }
    }

    document.addEventListener('click', event => {
      const target = event.target.closest('[data-action]');
      if (!target) {
        return;
      }
      const action = target.getAttribute('data-action');
      const path = target.getAttribute('data-path');
      if (action === 'apply') {
        const paths = Array.from(document.querySelectorAll('input[data-path]:checked'))
          .map(input => input.getAttribute('data-path'));
        host.postMessage({ type: 'apply', paths });
        return;
      }
      if (action === 'diff') {
        host.postMessage({ type: 'diff', path });
        return;
      }
      if (action === 'rollback') {
        // Con checklist se manda la selección; la vista sin inventario no tiene
        // casillas y pide el deshacer completo del avance.
        if (document.querySelectorAll('input[data-path]').length === 0) {
          host.postMessage({ type: 'rollback' });
          return;
        }
        const paths = Array.from(document.querySelectorAll('input[data-path]:checked')).map(input =>
          input.getAttribute('data-path')
        );
        host.postMessage({ type: 'rollback', paths });
        return;
      }
      if (action === 'open-comment') {
        host.postMessage({
          type: 'open-comment',
          file: target.getAttribute('data-file'),
          line: Number(target.getAttribute('data-line')),
          commentId: target.getAttribute('data-comment')
        });
        return;
      }
      host.postMessage({ type: action });
    });
  </script>
</body>
</html>`;
}
