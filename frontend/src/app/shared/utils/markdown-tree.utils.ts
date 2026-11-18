/**
 * Utilidades para detección y renderizado de estructuras de carpetas en Markdown.
 *
 * Transforma bloques de código que representan jerarquías de archivos (tanto con conectores
 * clásicos ├──/└── como listas de rutas indentadas) en árboles semánticos accesibles
 * idénticos a los del panel docente de solv-companion, con líneas guía CSS e íconos SVG.
 */

export interface TreeNode {
  name: string;
  isDir: boolean;
  children: TreeNode[];
}

export function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

const TREE_LINE_CONNECTOR = /^((?:│   |    |├── |└── |\|-- |\+-- |\\-- )*)(\S.*)$/u;
const CODE_PUNCTUATION = /[;{}]/;
const FORBIDDEN_CODE_SYNTAX = /\b(const|let|var|function|class|import|export|return|if|else|while|for|switch|case|def)\b|[;{}]|=>/;

const FILE_EXTENSIONS = /\.(ts|tsx|js|jsx|json|prisma|md|env|ya?ml|css|scss|html|sql|go|py|rs|java|c|cpp|sh|txt|lock|svg|png|jpg|jpeg|gif|ico|vue|svelte|astro)$/i;
const SPECIAL_FILES = /^(\.env(\.[a-z0-9_-]+)?|\.gitignore|\.dockerignore|README(\.[a-z]+)?|LICENSE|Makefile|Dockerfile|package\.json|tsconfig\.json|vite\.config\.[a-z0-9]+)$/i;

function cleanPathItem(raw: string): string {
  return raw.replace(/\s*\([^)]*\)\s*$/, '').trim();
}

function isFilePath(path: string): boolean {
  const clean = cleanPathItem(path);
  const basename = clean.split('/').filter(Boolean).pop() ?? clean;
  if (SPECIAL_FILES.test(basename)) {
    return true;
  }
  return FILE_EXTENSIONS.test(basename);
}

function isDirectoryName(name: string): boolean {
  const clean = cleanPathItem(name);
  if (clean.endsWith('/')) {
    return true;
  }
  return !isFilePath(clean);
}

/**
 * Parsea un bloque que usa conectores clásicos de árbol (├──, └──, │, +--, etc.)
 */
function parseConnectorTree(lines: string[]): TreeNode[] | null {
  const content = lines.filter(line => line.trim() !== '');
  if (content.length < 2) {
    return null;
  }

  const roots: TreeNode[] = [];
  const stack: { depth: number; node: TreeNode }[] = [];
  let connectors = 0;

  for (const [index, line] of content.entries()) {
    const match = TREE_LINE_CONNECTOR.exec(line);
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
    if (/[├└│|+|\\]/.test(prefix)) {
      connectors += 1;
    }

    const name = rawName.replace(/\/+$/, '');
    const isDir = rawName.endsWith('/') || isDirectoryName(rawName);
    const node: TreeNode = { name, isDir, children: [] };

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

/**
 * Parsea jerarquías de archivos expresadas con indentación y rutas (sin conectores ASCII).
 */
function parseIndentedTree(lines: string[]): TreeNode[] | null {
  const content = lines.filter(line => line.trim() !== '');
  if (content.length < 2) {
    return null;
  }
  if (content.some(line => FORBIDDEN_CODE_SYNTAX.test(line))) {
    return null;
  }

  let pathMatches = 0;
  for (const line of content) {
    const trimmed = cleanPathItem(line.trim());
    if (trimmed.startsWith('/') || trimmed.endsWith('/') || isFilePath(trimmed) || trimmed.includes('/')) {
      pathMatches++;
    }
  }

  if (pathMatches < Math.ceil(content.length * 0.7)) {
    return null;
  }

  const roots: TreeNode[] = [];
  const stack: { depth: number; node: TreeNode }[] = [];

  for (const line of content) {
    const indentMatch = line.match(/^(\s*)(.*)$/);
    const leadingSpaces = indentMatch ? indentMatch[1].length : 0;
    const rawName = (indentMatch ? indentMatch[2] : line).trim();
    if (!rawName) {
      continue;
    }

    const depth = leadingSpaces > 0 ? Math.max(1, Math.round(leadingSpaces / 2)) : 0;
    const isDir = rawName.endsWith('/') || isDirectoryName(rawName);
    const node: TreeNode = { name: rawName, isDir, children: [] };

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

  return roots.length > 0 ? roots : null;
}

/**
 * Parsea cualquier formato admitido de árbol de archivos.
 */
export function parseTree(lines: string[]): TreeNode[] | null {
  return parseConnectorTree(lines) ?? parseIndentedTree(lines);
}

/**
 * Determina si un bloque de texto representa una estructura de archivos.
 */
export function isTreeBlock(text: string): boolean {
  const lines = text.split('\n');
  return parseTree(lines) !== null;
}

function treeIcon(isDir: boolean): string {
  const paths = isDir
    ? '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>'
    : '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>';
  return `<svg class="tree-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths}</svg>`;
}

function renderTreeNode(node: TreeNode): string {
  const isDir = node.isDir || node.children.length > 0;
  const nested = node.children.length > 0
    ? `<ul class="tree-list">${node.children.map(renderTreeNode).join('')}</ul>`
    : '';
  return `<li class="tree-item${isDir ? ' tree-dir' : ' tree-file'}"><span class="tree-row">${treeIcon(isDir)}<span class="tree-name">${escapeHtml(node.name)}</span></span>${nested}</li>`;
}

/**
 * Renderiza el bloque completo de árbol en un contenedor HTML semántico idéntico al de solv-companion.
 */
export function renderTreeBlock(text: string): string {
  const lines = text.split('\n');
  const tree = parseTree(lines);
  if (tree && tree.length > 0) {
    return `<div class="tree" role="group" aria-label="Estructura de archivos y carpetas"><ul class="tree-list">${tree.map(renderTreeNode).join('')}</ul></div>`;
  }
  return `<pre><code>${escapeHtml(text)}</code></pre>`;
}
