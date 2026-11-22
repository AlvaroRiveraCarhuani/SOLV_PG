import { marked } from 'marked';
import { escapeHtml, parseTree, type TreeNode } from '@shared/utils/markdown-tree.utils';

/**
 * Fila del árbol. Angular descarta los `<svg>` al sanitizar `[innerHTML]`, así
 * que la fila deja un marcador y el ícono lo dibuja el SCSS con una máscara:
 * el árbol conserva el diseño del panel docente sin abrir la consigna a HTML
 * sin sanitizar.
 */
function renderTreeNode(node: TreeNode): string {
  const isDir = node.isDir || node.children.length > 0;
  const nested = node.children.length > 0
    ? `<ul class="tree-list">${node.children.map(renderTreeNode).join('')}</ul>`
    : '';
  const kind = isDir ? 'tree-dir' : 'tree-file';
  return (
    `<li class="tree-item ${kind}">` +
    `<span class="tree-row"><span class="tree-icon" aria-hidden="true"></span>` +
    `<span class="tree-name">${escapeHtml(node.name)}</span></span>` +
    `${nested}</li>`
  );
}

/**
 * Consigna pública renderizada. Un bloque que representa una estructura de
 * carpetas se resuelve con el parser compartido y se dibuja como árbol
 * semántico, en lugar de mostrar los conectores ASCII como código plano. El
 * resto del código queda en un `<pre>` normal.
 */
export function renderStatementMarkdown(raw: string): string {
  if (!raw || !raw.trim()) {
    return '<p>Sin descripción disponible.</p>';
  }

  // Normalizar saltos de línea escapados (\n literales) procedentes de serialización JSON
  const normalized = raw.includes('\\n') ? raw.replace(/\\n/g, '\n') : raw;

  const renderer = new marked.Renderer();
  renderer.code = ({ text, lang }) => {
    const tree = parseTree(text.split('\n'));
    if (tree && tree.length > 0) {
      return (
        `<div class="tree" role="group" aria-label="Estructura de archivos y carpetas">` +
        `<ul class="tree-list">${tree.map(renderTreeNode).join('')}</ul></div>`
      );
    }

    const language = (lang ?? '').trim().toLowerCase();
    const languageClass = language ? ` class="language-${escapeHtml(language)}"` : '';
    return `<pre><code${languageClass}>${escapeHtml(text)}</code></pre>`;
  };

  return marked.parse(normalized, { renderer, async: false }) as string;
}
