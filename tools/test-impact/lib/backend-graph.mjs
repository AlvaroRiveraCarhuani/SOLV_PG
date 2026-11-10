import fs from 'node:fs';
import path from 'node:path';
import { walkFiles } from './util.mjs';

const IMPORT_BLOCK = /import\s*\(([\s\S]*?)\)/g;
const IMPORT_LINE = /^\s*import\s+(?:[\w.]+\s+)?"([^"]+)"/gm;
const QUOTED = /"([^"]+)"/g;

function parseImports(source) {
  const imports = new Set();
  for (const block of source.matchAll(IMPORT_BLOCK)) {
    for (const quoted of block[1].matchAll(QUOTED)) {
      imports.add(quoted[1]);
    }
  }
  for (const line of source.matchAll(IMPORT_LINE)) {
    imports.add(line[1]);
  }
  return [...imports];
}

export function readGoModule(repoRoot) {
  const goMod = path.join(repoRoot, 'backend', 'go.mod');
  const source = fs.readFileSync(goMod, 'utf8');
  const match = source.match(/^module\s+(\S+)/m);
  return match ? match[1] : null;
}

export function buildBackendGraph(repoRoot) {
  const backendRoot = path.join(repoRoot, 'backend');
  const moduleName = readGoModule(repoRoot);
  const goFiles = walkFiles(backendRoot, {
    skipDirs: ['bin', 'uploads', 'testdata', 'vendor'],
    extensions: ['.go']
  });

  const packages = new Map();
  for (const absolute of goFiles) {
    const dir = path.dirname(absolute);
    const relDir = path.relative(repoRoot, dir).split(path.sep).join('/');
    const key = `./${path.posix.relative('backend', relDir)}`;
    if (!packages.has(key)) {
      packages.set(key, { key, dir, files: [], imports: new Set(), hasTests: false });
    }
    const entry = packages.get(key);
    entry.files.push(absolute);
    if (absolute.endsWith('_test.go')) entry.hasTests = true;
    const source = fs.readFileSync(absolute, 'utf8');
    for (const imported of parseImports(source)) {
      if (moduleName && imported.startsWith(`${moduleName}/`)) {
        entry.imports.add(`./${imported.slice(moduleName.length + 1)}`);
      }
    }
  }

  const reverseDeps = new Map();
  for (const entry of packages.values()) {
    for (const dependency of entry.imports) {
      if (!packages.has(dependency)) continue;
      if (!reverseDeps.has(dependency)) reverseDeps.set(dependency, new Set());
      reverseDeps.get(dependency).add(entry.key);
    }
  }

  const unitPackages = [...packages.values()]
    .filter((entry) => entry.hasTests)
    .filter((entry) => !entry.key.startsWith('./tests/'))
    .map((entry) => entry.key)
    .sort();

  const integrationPackages = [...packages.values()]
    .filter((entry) => entry.hasTests && entry.key.startsWith('./tests/'))
    .map((entry) => entry.key)
    .sort();

  const packageDirs = [...packages.values()]
    .map((entry) => ({ key: entry.key, relDir: entry.key === './' ? '' : entry.key.slice(2) }))
    .sort((a, b) => b.relDir.length - a.relDir.length);

  function packageForPath(filePath) {
    const normalized = filePath.split(path.sep).join('/');
    if (!normalized.startsWith('backend/')) return null;
    const relative = normalized.slice('backend/'.length);
    const match = packageDirs.find(
      (entry) => entry.relDir === '' || relative === entry.relDir || relative.startsWith(`${entry.relDir}/`)
    );
    return match ? match.key : null;
  }

  function reverseClosure(startPackages) {
    const selected = new Set();
    const queue = [...startPackages];
    while (queue.length > 0) {
      const current = queue.shift();
      if (selected.has(current)) continue;
      selected.add(current);
      for (const dependent of reverseDeps.get(current) || []) {
        if (!selected.has(dependent)) queue.push(dependent);
      }
    }
    return [...selected];
  }

  return {
    moduleName,
    packages,
    unitPackages,
    integrationPackages,
    packageForPath,
    reverseClosure,
    packagesWithTests: new Set(
      [...packages.values()].filter((entry) => entry.hasTests).map((entry) => entry.key)
    )
  };
}
