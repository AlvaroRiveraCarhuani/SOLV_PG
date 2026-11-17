/**
 * Renderizado del Markdown que llega en la consigna.
 *
 * El texto se escribe en Markdown desde la plataforma y el panel lo recibe
 * completo: títulos, listas, bloques de código y énfasis. Se convierte acá, en
 * el host de la extensión, sin traer una librería: el paquete de la extensión
 * excluye node_modules, así que una dependencia de ejecución no viajaría en el
 * .vsix ni en la imagen del laboratorio.
 *
 * Todo el texto se escapa antes de interpretar el marcado: la consigna nunca
 * puede inyectar etiquetas ni atributos.
 */

export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

const LINK_PATTERN = /\[([^\]\n]+)\]\((https?:\/\/[^\s)]+)\)/g;

/** Marcado que vive dentro de una línea: código, énfasis y enlaces. */
function inline(value: string): string {
  return escapeHtml(value)
    .replace(/`([^`\n]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
    .replace(/__([^_\n]+)__/g, '<strong>$1</strong>')
    .replace(/(^|[\s(])\*([^*\n]+)\*(?=[\s.,;:)]|$)/g, '$1<em>$2</em>')
    .replace(LINK_PATTERN, (_match, text: string, href: string) => `<a href="${href}">${text}</a>`);
}

function headingLevel(hashes: string): number {
  // El panel ya usa h2 para el título de la sección: la consigna arranca en h3.
  return Math.min(hashes.length + 2, 6);
}

function renderList(items: string[], ordered: boolean): string {
  const tag = ordered ? 'ol' : 'ul';
  const body = items.map(item => `<li>${inline(item)}</li>`).join('');
  return `<${tag}>${body}</${tag}>`;
}

interface TreeNode {
  name: string;
  isDir: boolean;
  children: TreeNode[];
}

// Un nivel de árbol ocupa cuatro columnas: `│   `, `    `, `├── ` o `└── `.
const TREE_LINE = /^((?:│   |    |├── |└── )*)(\S.*)$/;
const CODE_PUNCTUATION = /[;{}]/;

/**
 * Reconoce un bloque de código que en realidad dibuja un árbol de carpetas.
 * Exige al menos dos líneas con conectores y que la primera esté en la raíz, de
 * modo que un fragmento de código normal nunca se tome por estructura.
 */
function parseTree(lines: string[]): TreeNode[] | null {
  const content = lines.filter(line => line.trim() !== '');
  if (content.length < 2) {
    return null;
  }

  const roots: TreeNode[] = [];
  const stack: { depth: number; node: TreeNode }[] = [];
  let connectors = 0;

  for (const [index, line] of content.entries()) {
    const match = TREE_LINE.exec(line);
    if (!match) {
      return null;
    }
    const prefix = match[1];
    const rawName = match[2].trim();
    if (!rawName || CODE_PUNCTUATION.test(rawName) || CODE_PUNCTUATION.test(prefix)) {
      return null;
    }
    const depth = prefix.length / 4;
    if (!Number.isInteger(depth) || (index === 0 && depth !== 0)) {
      return null;
    }
    if (/[├└│]/.test(prefix)) {
      connectors += 1;
    }

    const name = rawName.replace(/\/+$/, '');
    const node: TreeNode = { name, isDir: rawName.endsWith('/'), children: [] };
    while (stack.length > 0 && stack[stack.length - 1].depth >= depth) {
      stack.pop();
    }
    if (stack.length === 0) {
      roots.push(node);
    } else {
      stack[stack.length - 1].node.children.push(node);
    }
    stack.push({ depth, node });
  }

  return connectors >= 1 && roots.length === 1 ? roots : null;
}

function treeIcon(isDir: boolean): string {
  const paths = isDir
    ? '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>'
    : '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>';
  return `<svg class="tree-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`;
}

function renderTreeNode(node: TreeNode): string {
  const isDir = node.isDir || node.children.length > 0;
  const nested = node.children.length > 0 ? `<ul class="tree-list">${node.children.map(renderTreeNode).join('')}</ul>` : '';
  return `<li class="tree-item${isDir ? ' tree-dir' : ' tree-file'}"><span class="tree-row">${treeIcon(isDir)}<span class="tree-name">${escapeHtml(node.name)}</span></span>${nested}</li>`;
}

/** Renderiza el bloque: árbol cuando su forma lo es, código en cualquier otro caso. */
function renderCodeBlock(lines: string[]): string {
  const tree = parseTree(lines);
  if (tree) {
    return `<div class="tree"><ul class="tree-list">${tree.map(renderTreeNode).join('')}</ul></div>`;
  }
  return `<pre><code>${escapeHtml(lines.join('\n'))}</code></pre>`;
}

const UNORDERED_ITEM = /^\s*[-*+]\s+(.*)$/;
const ORDERED_ITEM = /^\s*\d+[.)]\s+(.*)$/;
const FENCE = /^\s*```/;
const HEADING = /^(#{1,6})\s+(.*)$/;
const RULE = /^\s*(-{3,}|\*{3,}|_{3,})\s*$/;
const QUOTE = /^\s*>\s?(.*)$/;

export function renderMarkdown(source: string): string {
  const lines = source.replace(/\r\n?/g, '\n').split('\n');
  const blocks: string[] = [];
  let paragraph: string[] = [];
  let list: { ordered: boolean; items: string[] } | null = null;
  let quote: string[] = [];

  const flushParagraph = (): void => {
    if (paragraph.length > 0) {
      blocks.push(`<p>${inline(paragraph.join(' '))}</p>`);
      paragraph = [];
    }
  };
  const flushList = (): void => {
    if (list) {
      blocks.push(renderList(list.items, list.ordered));
      list = null;
    }
  };
  const flushQuote = (): void => {
    if (quote.length > 0) {
      blocks.push(`<blockquote><p>${inline(quote.join(' '))}</p></blockquote>`);
      quote = [];
    }
  };
  const flushAll = (): void => {
    flushParagraph();
    flushList();
    flushQuote();
  };

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];

    if (FENCE.test(line)) {
      flushAll();
      const code: string[] = [];
      for (index += 1; index < lines.length && !FENCE.test(lines[index]); index += 1) {
        code.push(lines[index]);
      }
      blocks.push(renderCodeBlock(code));
      continue;
    }

    if (line.trim() === '') {
      flushAll();
      continue;
    }

    const heading = HEADING.exec(line);
    if (heading) {
      flushAll();
      const level = headingLevel(heading[1]);
      blocks.push(`<h${level}>${inline(heading[2].trim())}</h${level}>`);
      continue;
    }

    if (RULE.test(line)) {
      flushAll();
      blocks.push('<hr>');
      continue;
    }

    const quoteMatch = QUOTE.exec(line);
    if (quoteMatch) {
      flushParagraph();
      flushList();
      quote.push(quoteMatch[1].trim());
      continue;
    }

    const unordered = UNORDERED_ITEM.exec(line);
    const ordered = unordered ? null : ORDERED_ITEM.exec(line);
    if (unordered || ordered) {
      flushParagraph();
      flushQuote();
      const isOrdered = ordered !== null;
      const content = (ordered ?? unordered)![1].trim();
      if (!list || list.ordered !== isOrdered) {
        flushList();
        list = { ordered: isOrdered, items: [content] };
      } else {
        list.items.push(content);
      }
      continue;
    }

    flushList();
    flushQuote();
    paragraph.push(line.trim());
  }

  flushAll();
  return blocks.join('\n');
}
