import * as vscode from 'vscode';
import { PanelState, rubricBadgeText, rubricTitleText } from './panel-state';

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
  external: '<path d="M14 4h6v6"/><path d="M20 4 11 13"/><path d="M18 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h5"/>',
  warning: '<path d="M12 3 2.5 20h19z"/><path d="M12 9v5"/><path d="M12 17v.5"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7"/>',
  'chevron-down': '<path d="m6 9 6 6 6-6"/>'
};

function icon(name: IconName): string {
  return `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICON_PATHS[name]}</svg>`;
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
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
                : `<button class="link-button" data-action="diff" data-path="${escapeHtml(file.path)}">${icon('diff')}Ver diferencias</button>`
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
          ? `<p class="notice notice-info">${icon('check')}Avance aplicado a tu entorno. Puedes deshacerlo para volver a la versión anterior.</p>`
          : `<p class="notice">El docente compartió una actualización. Revisa el checklist y elige qué archivos incorporar.</p>`
      }

      <ul class="file-list">${rows}</ul>

      <div class="actions">
        ${
          advance.applied
            ? `<button class="btn btn-secondary" data-action="rollback">${icon('undo')}Deshacer último avance</button>`
            : `<button class="btn btn-primary" data-action="apply">${icon('upload')}Comparar y aplicar selección</button>
               <button class="btn btn-ghost" data-action="dismiss">Descartar avance</button>`
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
      <section class="section" aria-labelledby="comments-title">
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
    <section class="section" aria-labelledby="comments-title">
      <h2 id="comments-title">Observaciones del docente</h2>
      ${groups}
    </section>`;
}

function renderTeacher(state: PanelState): string {
  const teacher = state.teacher;
  if (!teacher) {
    return '';
  }

  const ratio =
    teacher.enrolled > 0 ? Math.min(100, Math.round((teacher.present / teacher.enrolled) * 100)) : 0;

  return `
    <section class="section" aria-labelledby="class-title">
      <div class="section-head">
        <h2 id="class-title">Aula</h2>
        <span class="badge ${teacher.sessionOpen ? 'badge-new' : 'badge-neutral'}">${teacher.sessionOpen ? 'Sesión abierta' : 'Sin sesión'}</span>
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
        <div class="metric">
          <span class="metric-label">${icon('freeze')}Emisión</span>
          <span class="metric-value">${teacher.frozen ? 'Congelada' : 'Habilitada'}</span>
        </div>
      </div>

      <div class="presence" role="img" aria-label="Presencia ${teacher.present} de ${teacher.enrolled} alumnos">
        <span class="presence-fill" style="width:${ratio}%"></span>
      </div>

      <div class="actions">
        ${
          teacher.sessionOpen
            ? `<button class="btn btn-primary" data-action="broadcast">${icon('broadcast')}Emitir avance a la clase</button>
               <button class="btn btn-secondary" data-action="freeze">${icon('freeze')}${teacher.frozen ? 'Descongelar clase' : 'Congelar clase'}</button>
               <button class="btn btn-ghost" data-action="close-aula">Cerrar clase</button>`
            : `<button class="btn btn-primary" data-action="open-aula" style="width: 100%; justify-content: center;">${icon('users')}Iniciar clase</button>`
        }
      </div>
      <button class="link-button" data-action="speedgrader">${icon('external')}Abrir evaluación en la plataforma</button>

      ${
        teacher.lastAdvance
          ? `<p class="muted">Último avance emitido: <time datetime="${escapeHtml(teacher.lastAdvance.createdAt)}">${escapeHtml(formatTimestamp(teacher.lastAdvance.createdAt))}</time> a ${teacher.lastAdvance.recipients} alumnos.</p>`
          : ''
      }
    </section>`;
}

function renderRubric(assignment: PanelState['assignment']): string {
  const rubric = assignment.rubricCriteria;
  if (!rubric || rubric.length === 0) {
    if (!assignment.criteria || assignment.criteria.length === 0) {
      return '';
    }
    return `<ul class="criteria">${assignment.criteria
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

      return `<details class="rubric-accordion-item"${index === 0 ? ' open' : ''}>
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

      const levelsGrid =
        levels.length > 0
          ? `<div class="matrix-levels-grid">
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
            </div>`
          : '<span class="muted">Sin niveles específicos definidos</span>';

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

  return `<div class="rubric-container">
    <div class="rubric-header">
      <span class="rubric-header-title">${escapeHtml(rubricHeader)}</span>
      <span class="rubric-total-badge">Total: ${totalPoints} pts</span>
    </div>

    <!-- Modo compacto (panel lateral < 600px) -->
    <div class="rubric-compact-view">
      ${compactCards}
    </div>

    <!-- Modo matriz ancha (pestaña del editor >= 600px) -->
    <div class="rubric-matrix-view">
      ${matrixRows}
    </div>
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
  .rubric-container { margin-top: 10px; }
  .rubric-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  .rubric-header-title {
    font-size: 0.85em;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--vscode-descriptionForeground);
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
        ${
          options?.isTab
            ? ''
            : `<button type="button" class="link-button" data-action="open-tab" style="margin-top: 4px;">${icon('external')} Abrir en pestaña</button>`
        }
      </div>
    </div>
    ${
      state.role === 'student'
        ? `<span class="badge badge-${delivery.tone}">${escapeHtml(delivery.text)}</span>`
        : ''
    }
  </header>

  ${message}

  <section class="section" aria-labelledby="assignment-title">
    <div class="section-head">
      <h2 id="assignment-title">Consigna</h2>
      ${
        state.assignment.rubricCriteria && state.assignment.rubricCriteria.length > 0
          ? `<span class="badge ${state.assignment.purpose === 'exam' ? 'badge-exam' : 'badge-submitted'}">${escapeHtml(rubricBadgeText(state.assignment.purpose))}</span>`
          : ''
      }
    </div>
    <p class="summary">${escapeHtml(state.assignment.summary || (state.assignment.purpose === 'exam' ? 'La consigna del examen se publica desde la plataforma.' : 'La consigna de la práctica se publica desde la plataforma.'))}</p>
    ${rubricContent}
  </section>

  ${state.role === 'student' ? renderAdvance(state) : renderTeacher(state)}
  ${state.role === 'student' ? renderComments(state) : ''}

  <footer>
    ${icon(state.connection === 'connected' ? 'check' : 'warning')}
    <span>${connectionLabel(state)}</span>
    <span aria-hidden="true">·</span>
    <span>SOLV Companion</span>
  </footer>

  <script nonce="${scriptNonce}">
    const host = acquireVsCodeApi();
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
