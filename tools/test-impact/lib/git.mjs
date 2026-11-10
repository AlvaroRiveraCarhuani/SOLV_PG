import { execFileSync } from 'node:child_process';
import { isRelevantIgnoredSource } from './artifacts.mjs';

function git(repoRoot, args, { allowFailure = false } = {}) {
  try {
    return execFileSync('git', ['-C', repoRoot, ...args], {
      encoding: 'utf8',
      maxBuffer: 64 * 1024 * 1024,
      stdio: ['ignore', 'pipe', 'pipe']
    }).trim();
  } catch (error) {
    if (allowFailure) return null;
    const detail = error.stderr?.toString().trim() || error.message;
    throw new Error(`git ${args.join(' ')} falló: ${detail}`);
  }
}

export function resolveCommit(repoRoot, ref) {
  if (!ref || /^0+$/.test(ref)) return null;
  return git(repoRoot, ['rev-parse', '--verify', `${ref}^{commit}`], { allowFailure: true });
}

export function parseNameStatus(output) {
  if (!output) return [];
  const changes = [];
  for (const line of output.split(/\r?\n/)) {
    if (!line.trim()) continue;
    const parts = line.split('\t');
    const status = parts[0].charAt(0);
    if (status === 'R' || status === 'C') {
      changes.push({ status, oldPath: parts[1], path: parts[2] });
      continue;
    }
    changes.push({ status, path: parts[1] });
  }
  return changes;
}

function dedupe(changes) {
  const seen = new Set();
  const result = [];
  for (const change of changes) {
    const key = `${change.status}:${change.oldPath || ''}:${change.path}`;
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(change);
  }
  return result;
}

export function collectChanges({
  repoRoot,
  base = null,
  head = 'HEAD',
  files = null,
  includeWorkingTree = false
}) {
  if (files && files.length > 0) {
    const parsed = files.map((entry) => {
      const separator = entry.indexOf(':');
      if (separator > 0 && /^[ACDMRTUXB]$/.test(entry.slice(0, separator))) {
        return { status: entry.slice(0, separator), path: entry.slice(separator + 1) };
      }
      return { status: 'M', path: entry };
    });
    return { changes: dedupe(parsed), error: null };
  }

  if (!base) {
    return { changes: [], error: 'sin base para comparar' };
  }

  if (!resolveCommit(repoRoot, base)) {
    return { changes: [], error: `la base ${base} no existe en el repositorio` };
  }

  const output = git(repoRoot, ['diff', '--name-status', '-M', `${base}...${head}`], {
    allowFailure: true
  });
  if (output === null) {
    return { changes: [], error: `no se pudo comparar ${base}...${head}` };
  }

  const changes = parseNameStatus(output);
  if (includeWorkingTree) {
    const unstaged = git(repoRoot, ['diff', '--name-status', '-M'], { allowFailure: true });
    const staged = git(repoRoot, ['diff', '--cached', '--name-status', '-M'], { allowFailure: true });
    const untracked = git(repoRoot, ['ls-files', '--others', '--exclude-standard'], { allowFailure: true });
    const ignored = git(repoRoot, ['ls-files', '--others', '--ignored', '--exclude-standard'], {
      allowFailure: true
    });
    if (unstaged === null || staged === null || untracked === null || ignored === null) {
      return { changes, error: 'no se pudo leer el estado completo del working tree' };
    }
    changes.push(...parseNameStatus(unstaged));
    changes.push(...parseNameStatus(staged));
    for (const file of untracked.split(/\r?\n/)) {
      if (file.trim()) changes.push({ status: 'A', path: file.trim() });
    }
    // Un archivo ignorado que parece fuente no puede desaparecer del diff. Se
    // excluyen dependencias, builds, cachés y secretos locales.
    for (const file of ignored.split(/\r?\n/)) {
      const clean = file.trim();
      if (clean && isRelevantIgnoredSource(clean)) changes.push({ status: 'A', path: clean });
    }
  }

  return { changes: dedupe(changes), error: null };
}
