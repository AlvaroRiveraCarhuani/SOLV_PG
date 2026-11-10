import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { isGeneratedArtifact, isRelevantIgnoredSource } from './artifacts.mjs';

export const CACHE_SCHEMA = 4;

let toolchainCache = null;

function git(repoRoot, args) {
  const result = spawnSync('git', ['-C', repoRoot, ...args], {
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024
  });
  if (result.status !== 0) {
    throw new Error(`git ${args.join(' ')} falló; no se puede calcular una clave de caché segura`);
  }
  return result.stdout;
}

function sha256(value) {
  return crypto.createHash('sha256').update(value).digest('hex');
}

function fileHash(file) {
  try {
    return sha256(fs.readFileSync(file));
  } catch {
    return 'missing';
  }
}

function stablePlan(plan) {
  const { metrics, generatedAt, ...rest } = plan;
  return rest;
}

function commandVersion(command, args) {
  const result = spawnSync(command, args, { encoding: 'utf8' });
  if (result.status !== 0) return 'missing';
  return String(result.stdout || '').trim().split(/\r?\n/)[0] || 'unknown';
}

function environmentFingerprint() {
  const names = Object.keys(process.env)
    .filter((name) =>
      /^(CI|NODE_ENV|TZ|DATABASE_URL|TEST_DB_DSN|TEST_DATABASE_URL|JWT_SECRET|SOLV_[A-Z0-9_]+)$/.test(
        name
      )
    )
    // La identidad de la ejecución no cambia el resultado de una suite; si se
    // incluyera, la caché de CI quedaría invalidada en cada corrida.
    .filter((name) => name !== 'SOLV_TEST_EXECUTION_ID' && name !== 'SOLV_TEST_RUN_ID')
    .sort();
  return Object.fromEntries(names.map((name) => [name, sha256(process.env[name] || '')]));
}

/**
 * Identidad de las herramientas que producen el resultado. Cambiar de versión
 * de Node, Go o npm puede cambiar el resultado de una suite cacheada.
 */
export function toolchainInfo() {
  if (toolchainCache) return toolchainCache;
  toolchainCache = {
    node: process.version,
    go: commandVersion('go', ['version']),
    npm: commandVersion('npm', ['--version'])
  };
  return toolchainCache;
}

/**
 * Hash del código del motor (runner, CLI y librerías). Una corrección del
 * runner invalida los resultados anteriores aunque el plan sea idéntico.
 */
export function runnerDigest(repoRoot) {
  const root = path.join(repoRoot, 'tools', 'test-impact');
  const files = [];
  const visit = (directory) => {
    let entries;
    try {
      entries = fs.readdirSync(directory, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries.sort((left, right) => left.name.localeCompare(right.name))) {
      if (entry.name === '.cache' || entry.name === 'node_modules') continue;
      const absolute = path.join(directory, entry.name);
      if (entry.isDirectory()) {
        visit(absolute);
        continue;
      }
      if (!entry.name.endsWith('.mjs')) continue;
      files.push(`${path.relative(root, absolute)}:${fileHash(absolute)}`);
    }
  };
  visit(root);
  return sha256(files.join('\n'));
}

/**
 * La clave incluye el contenido versionado y el working tree, no solo la lista
 * de rutas; además fija política, versión del runner, comando, toolchain y
 * entorno. Un cambio de bytes en cualquiera de esos inputs invalida la entrada.
 */
export function computeCacheKey(repoRoot, plan, context = {}) {
  const head = git(repoRoot, ['rev-parse', 'HEAD']).trim() || 'sin-head';
  const tracked = git(repoRoot, ['diff', '--binary', 'HEAD']);
  const staged = git(repoRoot, ['diff', '--cached', '--binary']);
  const untracked = git(repoRoot, ['ls-files', '--others', '--exclude-standard'])
    .split(/\r?\n/)
    .filter(Boolean)
    .filter((file) => !isGeneratedArtifact(file))
    .map((file) => `${file}:${fileHash(path.join(repoRoot, file))}`)
    .join('\n');
  // Un archivo fuente ignorado también puede alterar el resultado de una suite
  // cacheable. Se incluye solo si pasa la política acotada de fuentes.
  const ignored = git(repoRoot, ['ls-files', '--others', '--ignored', '--exclude-standard'])
    .split(/\r?\n/)
    .filter(Boolean)
    .filter((file) => isRelevantIgnoredSource(file))
    .map((file) => `${file}:${fileHash(path.join(repoRoot, file))}`)
    .join('\n');
  // Estado instalado de dependencias: no se recorre node_modules completo; el
  // lockfile instalado por npm refleja el árbol resuelto y cambia si se instala
  // otra versión sin tocar el lockfile de raíz.
  const dependencies = [
    'frontend/package-lock.json',
    'backend/go.sum',
    'frontend/node_modules/.package-lock.json'
  ]
    .map((file) => `${file}:${fileHash(path.join(repoRoot, file))}`)
    .join('\n');
  const environment = context.env || environmentFingerprint();
  return sha256(
    JSON.stringify({
      schema: CACHE_SCHEMA,
      key: stablePlan(plan),
      head,
      tracked: sha256(tracked),
      staged: sha256(staged),
      untracked: sha256(untracked),
      ignored: sha256(ignored),
      dependencies: sha256(dependencies),
      policy: fileHash(path.join(repoRoot, 'tools', 'test-impact', 'policy.json')),
      runner: runnerDigest(repoRoot),
      command: context.command || null,
      toolchain: context.toolchain || toolchainInfo(),
      environment
    })
  );
}

export function cacheRoot(repoRoot, override = null) {
  return override ? path.resolve(override) : path.join(repoRoot, 'tools', 'test-impact', '.cache');
}

function canonical(value) {
  if (Array.isArray(value)) return value.map(canonical);
  if (value !== null && typeof value === 'object') {
    return Object.fromEntries(
      Object.keys(value)
        .sort()
        .map((entry) => [entry, canonical(value[entry])])
    );
  }
  return value;
}

function cacheSignature(entry, receiptKey) {
  const payload = { ...entry };
  delete payload.signature;
  return crypto
    .createHmac('sha256', receiptKey)
    .update(JSON.stringify(canonical(payload)))
    .digest('hex');
}

export function readCache(repoRoot, key, override = null, receiptKey = null) {
  const file = path.join(cacheRoot(repoRoot, override), `${key}.json`);
  try {
    const entry = JSON.parse(fs.readFileSync(file, 'utf8'));
    if (entry.schema !== CACHE_SCHEMA || entry.key !== key) return null;
    if (entry.status !== 0) return null;
    if (!receiptKey || !entry.signature) return null;
    const expected = cacheSignature(entry, receiptKey);
    const left = Buffer.from(expected, 'hex');
    const right = Buffer.from(String(entry.signature), 'hex');
    if (left.length !== right.length || !crypto.timingSafeEqual(left, right)) return null;
    return entry;
  } catch {
    return null;
  }
}

export function writeCache(repoRoot, key, value, override = null, receiptKey = null) {
  // Nunca se cachea un fallo ni una simulación: solo llegan aquí resultados
  // reales con estado 0 desde el runner.
  if (value.status !== 0) return;
  const directory = cacheRoot(repoRoot, override);
  fs.mkdirSync(directory, { recursive: true });
  const file = path.join(directory, `${key}.json`);
  const temporary = `${file}.${process.pid}.tmp`;
  if (!receiptKey) throw new Error('no hay clave para firmar la entrada de caché');
  const entry = { schema: CACHE_SCHEMA, key, createdAt: new Date().toISOString(), ...value };
  entry.signature = cacheSignature(entry, receiptKey);
  fs.writeFileSync(temporary, `${JSON.stringify(entry, null, 2)}\n`);
  fs.renameSync(temporary, file);
}
