export interface WorkspaceProjectFile {
  path: string;
  content: string;
}

export interface WorkspaceProjectChangeSummary {
  added: string[];
  modified: string[];
  removed: string[];
}

export const WORKSPACE_PROJECT_MAX_BYTES = 1024 * 1024;
export const WORKSPACE_PROJECT_MAX_FILES = 300;
export const WORKSPACE_PROJECT_MAX_FILE_BYTES = 512 * 1024;
export const WORKSPACE_PROJECT_MAX_ZIP_BYTES = 10 * 1024 * 1024;
export const WORKSPACE_PROJECT_MAX_PATH_BYTES = 1024;

const SNAPSHOT_HEADER_PREFIX = '# === FILE: ';
const SNAPSHOT_HEADER_SUFFIX = ' ===';
function byteLength(value: string): number {
  return new TextEncoder().encode(value).length;
}

function hasControlCharacters(value: string, allowTextWhitespace: boolean): boolean {
  for (const character of value) {
    const code = character.charCodeAt(0);
    if (code === 0x7f) return true;
    if (code < 0x20 && !(allowTextWhitespace && (character === '\n' || character === '\r' || character === '\t'))) {
      return true;
    }
  }
  return false;
}

/**
 * Valida y normaliza una ruta relativa. La interfaz no puede confiar en el
 * navegador: el backend repite estas reglas antes de tocar el borrador.
 */
export function normalizeWorkspacePath(raw: string): { path: string; error?: undefined } | { path?: undefined; error: string } {
  const value = (raw || '').normalize('NFC');
  if (value === '') return { error: 'La ruta no puede estar vacía.' };
  if (byteLength(value) > WORKSPACE_PROJECT_MAX_PATH_BYTES) {
    return { error: `La ruta supera ${WORKSPACE_PROJECT_MAX_PATH_BYTES} bytes.` };
  }
  if (value.includes('\\')) return { error: 'La ruta usa un separador no permitido.' };
  if (value.startsWith('/')) return { error: 'La ruta debe ser relativa.' };
  if (hasControlCharacters(value, false)) return { error: 'La ruta contiene caracteres de control.' };
  const segments = value.split('/');
  if (segments.some(segment => segment === '' || segment === '.' || segment === '..' || segment.trim() !== segment)) {
    return { error: 'La ruta contiene segmentos ambiguos.' };
  }
  return { path: value };
}

/** El snapshot usa LF y un único salto final en archivos con contenido. */
export function canonicalizeProjectContent(content: string): string {
  const normalized = (content || '').replace(/\r\n/g, '\n').replace(/\r/g, '\n').replace(/^\uFEFF/, '');
  if (normalized.trim() === '') return '';
  return `${normalized.replace(/\n+$/, '')}\n`;
}

function contentIsRepresentable(filePath: string, content: string): string | null {
  if (hasControlCharacters(content, true)) {
    return `El archivo ${filePath} contiene caracteres de control.`;
  }
  for (const line of content.split('\n')) {
    if (line.startsWith(SNAPSHOT_HEADER_PREFIX) && line.endsWith(SNAPSHOT_HEADER_SUFFIX)) {
      return `El archivo ${filePath} contiene una cabecera de snapshot ambigua.`;
    }
  }
  return null;
}

export function validateProjectFiles(files: WorkspaceProjectFile[]): string | null {
  if (files.length === 0) return 'El proyecto no contiene archivos.';
  if (files.length > WORKSPACE_PROJECT_MAX_FILES) {
    return `El proyecto supera el máximo de ${WORKSPACE_PROJECT_MAX_FILES} archivos.`;
  }
  const seen = new Set<string>();
  let total = 0;
  for (const file of files) {
    const normalized = normalizeWorkspacePath(file.path);
    if (normalized.error) return normalized.error;
    const path = normalized.path as string;
    if (seen.has(path)) return `La ruta ${path} está duplicada.`;
    seen.add(path);
    const content = canonicalizeProjectContent(file.content);
    const contentError = contentIsRepresentable(path, content);
    if (contentError) return contentError;
    const size = byteLength(content);
    if (size > WORKSPACE_PROJECT_MAX_FILE_BYTES) {
      return `El archivo ${path} supera ${WORKSPACE_PROJECT_MAX_FILE_BYTES} bytes.`;
    }
    total += size;
  }
  if (total > WORKSPACE_PROJECT_MAX_BYTES) {
    return `El proyecto supera ${WORKSPACE_PROJECT_MAX_BYTES} bytes de contenido.`;
  }
  return null;
}

/**
 * Lee el formato de cabeceras compartido con las entregas IDE. Devuelve error
 * ante rutas inseguras o duplicadas en vez de recortar silenciosamente.
 */
export function parseWorkspaceSnapshot(raw: string): { files: WorkspaceProjectFile[]; error?: undefined } | { files?: undefined; error: string } {
  if (!raw || raw.trim() === '') return { error: 'El proyecto está vacío.' };
  if (byteLength(raw) > WORKSPACE_PROJECT_MAX_BYTES) return { error: 'El proyecto supera el tamaño permitido.' };
  const text = raw.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  const files: WorkspaceProjectFile[] = [];
  const seen = new Set<string>();
  let current: { path: string; contentStart: number } | null = null;

  const push = (contentEnd: number, trimSeparator: boolean): string | null => {
    if (!current) return null;
    let content = text.slice(current.contentStart, contentEnd);
    if (trimSeparator && content.endsWith('\n')) content = content.slice(0, -1);
    if (content.trim() === '') content = '';
    if (seen.has(current.path)) return `La ruta ${current.path} está duplicada.`;
    seen.add(current.path);
    files.push({ path: current.path, content });
    return null;
  };

  let lineStart = 0;
  while (lineStart <= text.length) {
    const newline = text.indexOf('\n', lineStart);
    const lineEnd = newline < 0 ? text.length : newline;
    const line = text.slice(lineStart, lineEnd);
    if (line.startsWith(SNAPSHOT_HEADER_PREFIX) && line.endsWith(SNAPSHOT_HEADER_SUFFIX)) {
      if (current) {
        const duplicateError = push(lineStart, true);
        if (duplicateError) return { error: duplicateError };
      }
      const path = line.slice(SNAPSHOT_HEADER_PREFIX.length, -SNAPSHOT_HEADER_SUFFIX.length).trim();
      const normalized = normalizeWorkspacePath(path);
      if (normalized.error) return { error: normalized.error };
      current = { path: normalized.path as string, contentStart: newline < 0 ? text.length : newline + 1 };
    }
    if (newline < 0) break;
    lineStart = newline + 1;
  }
  if (!current) return { error: 'El proyecto no tiene cabeceras de archivo reconocibles.' };
  const lastError = push(text.length, false);
  if (lastError) return { error: lastError };
  if (files.length > WORKSPACE_PROJECT_MAX_FILES) return { error: 'El proyecto supera el máximo de archivos.' };
  files.sort((a, b) => compareWorkspacePaths(a.path, b.path));
  return { files };
}

export function encodeWorkspaceSnapshot(files: WorkspaceProjectFile[]): { text: string; error?: undefined } | { text?: undefined; error: string } {
  const validationError = validateProjectFiles(files);
  if (validationError) return { error: validationError };
  const sorted = [...files]
    .map(file => ({ path: normalizeWorkspacePath(file.path).path as string, content: file.content }))
    .sort((a, b) => compareWorkspacePaths(a.path, b.path));
  const blocks = sorted.map(file => {
    const content = canonicalizeProjectContent(file.content);
    return `${SNAPSHOT_HEADER_PREFIX}${file.path}${SNAPSHOT_HEADER_SUFFIX}\n${content}`;
  });
  const text = blocks.join('\n');
  const parsed = parseWorkspaceSnapshot(text);
  if (parsed.error) return { error: parsed.error };
  const parsedFiles = parsed.files ?? [];
  for (const file of sorted) {
    const expected = canonicalizeProjectContent(file.content);
    const stored = parsedFiles.find(candidate => candidate.path === file.path);
    if (!stored || stored.content !== expected) {
      return { error: `El archivo ${file.path} no se conserva completo en el snapshot.` };
    }
  }
  return { text };
}

function compareWorkspacePaths(a: string, b: string): number {
  return a < b ? -1 : a > b ? 1 : 0;
}

export function summarizeProjectChanges(base: WorkspaceProjectFile[], current: WorkspaceProjectFile[]): WorkspaceProjectChangeSummary {
  const baseMap = new Map(base.map(file => [file.path, file.content]));
  const currentMap = new Map(current.map(file => [file.path, file.content]));
  const added: string[] = [];
  const modified: string[] = [];
  const removed: string[] = [];
  for (const [path, content] of currentMap) {
    if (!baseMap.has(path)) added.push(path);
    else if (baseMap.get(path) !== content) modified.push(path);
  }
  for (const path of baseMap.keys()) {
    if (!currentMap.has(path)) removed.push(path);
  }
  added.sort();
  modified.sort();
  removed.sort();
  return { added, modified, removed };
}

export function languageForProjectPath(path: string): string {
  const extension = path.includes('.') ? path.slice(path.lastIndexOf('.') + 1).toLowerCase() : '';
  switch (extension) {
    case 'py': return 'python';
    case 'ts':
    case 'tsx': return 'typescript';
    case 'js':
    case 'jsx': return 'javascript';
    case 'json': return 'json';
    case 'java': return 'java';
    case 'c':
    case 'h':
    case 'cpp':
    case 'cc':
    case 'hpp': return 'cpp';
    case 'sh':
    case 'bash': return 'shell';
    case 'dockerfile': return 'dockerfile';
    default: return 'plaintext';
  }
}