import fs from 'node:fs';
import path from 'node:path';

export function toPosix(value) {
  return value.split(path.sep).join('/');
}

export function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Convierte un glob de ruta en una expresión regular anclada.
 * Soporta `**`, `*`, `?` y alternativas `{a,b}`. No usa dependencias externas
 * para que el selector funcione en el runner sin instalar nada.
 */
export function globToRegExp(glob) {
  let regex = '';
  for (let index = 0; index < glob.length; index += 1) {
    const char = glob[index];
    if (char === '*') {
      if (glob[index + 1] === '*') {
        const isSegmentStart = index === 0 || glob[index - 1] === '/';
        const next = glob[index + 2];
        if (isSegmentStart && next === '/') {
          regex += '(?:[^/]+/)*';
          index += 2;
          continue;
        }
        if (isSegmentStart && next === undefined) {
          regex += '.*';
          index += 1;
          continue;
        }
        regex += '.*';
        index += 1;
        continue;
      }
      regex += '[^/]*';
      continue;
    }
    if (char === '?') {
      regex += '[^/]';
      continue;
    }
    if (char === '{') {
      const end = glob.indexOf('}', index);
      if (end === -1) {
        regex += '\\{';
        continue;
      }
      const options = glob
        .slice(index + 1, end)
        .split(',')
        .map((option) => escapeRegExp(option));
      regex += `(?:${options.join('|')})`;
      index = end;
      continue;
    }
    regex += escapeRegExp(char);
  }
  return new RegExp(`^${regex}$`);
}

export function matchesAny(globs, value) {
  return globs.some((glob) => globToRegExp(glob).test(value));
}

export function walkFiles(root, { skipDirs = [], extensions = null } = {}) {
  const files = [];
  const skip = new Set(skipDirs);
  const visit = (dir) => {
    let entries;
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      if (entry.name === '.git' || entry.name === 'node_modules') continue;
      const absolute = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        if (skip.has(entry.name)) continue;
        visit(absolute);
        continue;
      }
      if (!entry.isFile()) continue;
      if (extensions && !extensions.includes(path.extname(entry.name))) continue;
      files.push(absolute);
    }
  };
  visit(root);
  return files;
}

export function stripJsonComments(raw) {
  let output = '';
  let inString = false;
  let escaped = false;
  for (let index = 0; index < raw.length; index += 1) {
    const char = raw[index];
    const next = raw[index + 1];
    if (inString) {
      output += char;
      if (escaped) {
        escaped = false;
      } else if (char === '\\') {
        escaped = true;
      } else if (char === '"') {
        inString = false;
      }
      continue;
    }
    if (char === '"') {
      inString = true;
      output += char;
      continue;
    }
    if (char === '/' && next === '/') {
      while (index < raw.length && raw[index] !== '\n') index += 1;
      output += '\n';
      continue;
    }
    if (char === '/' && next === '*') {
      index += 2;
      while (index < raw.length && !(raw[index] === '*' && raw[index + 1] === '/')) {
        index += 1;
      }
      index += 1;
      continue;
    }
    output += char;
  }
  return output;
}

export function readJsonc(file) {
  return JSON.parse(stripJsonComments(fs.readFileSync(file, 'utf8')));
}

export function uniqueSorted(values) {
  return [...new Set(values)].sort();
}
