import { marked } from 'marked';

/**
 * Línea de un árbol de carpetas escrito a mano: guías, rama y nombre.
 * `proyecto/`, `│   ├── src/`, `    └── main.ts`.
 */
const TREE_BRANCH = /^([\s│]*)([├└]── )(.+)$/u;
const TREE_GLYPH = /[├└]── /u;

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * Un bloque es un árbol cuando tiene al menos dos ramas: una sola línea con
 * `├──` puede ser un ejemplo cualquiera y se deja como código.
 */
function isTreeBlock(text: string): boolean {
  return text.split('\n').filter((line) => TREE_GLYPH.test(line)).length >= 2;
}

function renderTreeRow(line: string): string {
  const match = line.match(TREE_BRANCH);
  if (!match) {
    const name = line.trim();
    if (!name) {
      return '';
    }
    const kind = name.endsWith('/') ? 'is-dir' : 'is-file';
    return `<div class="tree-row"><span class="tree-node ${kind}">${escapeHtml(name)}</span></div>`;
  }

  const [, guides, branch, name] = match;
  const kind = name.trim().endsWith('/') ? 'is-dir' : 'is-file';
  return (
    `<div class="tree-row">` +
    `<span class="tree-guide">${escapeHtml(guides + branch)}</span>` +
    `<span class="tree-node ${kind}">${escapeHtml(name)}</span>` +
    `</div>`
  );
}

/**
 * Consigna pública renderizada. Marked entrega un `<pre>` plano para la
 * estructura de carpetas; acá cada rama se marca como fila y se distingue
 * carpeta de archivo, para que el árbol se lea como el del entorno.
 */
export function renderStatementMarkdown(raw: string): string {
  if (!raw.trim()) {
    return '<p>Sin descripción disponible.</p>';
  }

  const renderer = new marked.Renderer();
  renderer.code = ({ text, lang }) => {
    if (isTreeBlock(text)) {
      const rows = text
        .replace(/\n+$/, '')
        .split('\n')
        .map(renderTreeRow)
        .join('');
      return `<div class="file-tree" role="group">${rows}</div>`;
    }

    const language = (lang ?? '').trim().toLowerCase();
    const languageClass = language ? ` class="language-${escapeHtml(language)}"` : '';
    return `<pre><code${languageClass}>${escapeHtml(text)}</code></pre>`;
  };

  return marked.parse(raw, { renderer, async: false }) as string;
}
