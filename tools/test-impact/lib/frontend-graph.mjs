import fs from 'node:fs';
import path from 'node:path';
import { readJsonc, walkFiles } from './util.mjs';

const TS_IMPORT = /(?:import|export)\s+(?:[\s\S]*?\s+from\s+)?['"]([^'"]+)['"]/g;
const TS_DYNAMIC_IMPORT = /\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)/g;
const ANGULAR_TEMPLATE = /(?:templateUrl|styleUrl)\s*:\s*(['"`])(.+?)\1/g;
const ANGULAR_STYLES_ARRAY = /styleUrls\s*:\s*\[([\s\S]*?)\]/g;
const SCSS_DEPENDENCY = /@(?:use|import)\s+['"]([^'"]+)['"]/g;

function parseAliases(repoRoot) {
  const tsconfigPath = path.join(repoRoot, 'frontend', 'tsconfig.json');
  if (!fs.existsSync(tsconfigPath)) return [];
  const config = readJsonc(tsconfigPath);
  const paths = config.compilerOptions?.paths || {};
  const baseUrl = path.resolve(path.join(repoRoot, 'frontend'), config.compilerOptions?.baseUrl || '.');
  return Object.entries(paths)
    .map(([pattern, targets]) => ({
      pattern,
      target: path.resolve(baseUrl, targets[0])
    }))
    .sort((a, b) => b.pattern.length - a.pattern.length);
}

const RESOLVABLE_EXTENSIONS = new Set(['.ts', '.html', '.scss', '.json', '.mjs', '.js']);

function resolveCandidate(candidate) {
  const candidates = [candidate];
  const extension = path.extname(candidate);
  if (!RESOLVABLE_EXTENSIONS.has(extension)) {
    candidates.push(`${candidate}.ts`, `${candidate}.html`, `${candidate}.scss`, `${candidate}.json`);
    candidates.push(path.join(candidate, 'index.ts'));
  }
  return candidates.find((file) => fs.existsSync(file) && fs.statSync(file).isFile()) || null;
}

function resolveImport(fromFile, specifier, aliases) {
  if (specifier.startsWith('.')) {
    return resolveCandidate(path.resolve(path.dirname(fromFile), specifier));
  }
  for (const alias of aliases) {
    const starIndex = alias.pattern.indexOf('*');
    if (starIndex === -1) {
      if (alias.pattern === specifier) return resolveCandidate(alias.target);
      continue;
    }
    const prefix = alias.pattern.slice(0, starIndex);
    const suffix = alias.pattern.slice(starIndex + 1);
    if (!specifier.startsWith(prefix) || !specifier.endsWith(suffix)) continue;
    const wildcard = specifier.slice(prefix.length, specifier.length - suffix.length);
    return resolveCandidate(alias.target.replace('*', wildcard));
  }
  return null;
}

function parseScssDependencies(source) {
  return [...source.matchAll(SCSS_DEPENDENCY)].map((match) => match[1]);
}

export function buildFrontendGraph(repoRoot) {
  const frontendRoot = path.join(repoRoot, 'frontend');
  const sourceRoot = path.join(frontendRoot, 'src');
  const aliases = parseAliases(repoRoot);

  const files = walkFiles(sourceRoot, {
    skipDirs: ['node_modules', 'assets'],
    extensions: ['.ts', '.html', '.scss', '.json']
  });

  const nodes = new Map();
  const reverse = new Map();

  const addNode = (absolute) => {
    if (!nodes.has(absolute)) nodes.set(absolute, { imports: new Set(), kind: path.extname(absolute) });
    return nodes.get(absolute);
  };

  const addEdge = (from, to) => {
    addNode(from).imports.add(to);
    addNode(to);
    if (!reverse.has(to)) reverse.set(to, new Set());
    reverse.get(to).add(from);
  };

  for (const absolute of files) {
    addNode(absolute);
    if (!absolute.endsWith('.ts')) continue;
    const source = fs.readFileSync(absolute, 'utf8');

    for (const match of source.matchAll(TS_IMPORT)) {
      const resolved = resolveImport(absolute, match[1], aliases);
      if (resolved) addEdge(absolute, resolved);
    }
    for (const match of source.matchAll(TS_DYNAMIC_IMPORT)) {
      const resolved = resolveImport(absolute, match[1], aliases);
      if (resolved) addEdge(absolute, resolved);
    }
    for (const match of source.matchAll(ANGULAR_TEMPLATE)) {
      const resolved = resolveImport(absolute, match[2], aliases);
      if (resolved) addEdge(absolute, resolved);
    }
    for (const block of source.matchAll(ANGULAR_STYLES_ARRAY)) {
      for (const quoted of block[1].matchAll(/(['"`])(.+?)\1/g)) {
        const resolved = resolveImport(absolute, quoted[2], aliases);
        if (resolved) addEdge(absolute, resolved);
      }
    }
  }

  for (const absolute of files) {
    if (!absolute.endsWith('.scss')) continue;
    const source = fs.readFileSync(absolute, 'utf8');
    for (const dependency of parseScssDependencies(source)) {
      if (!dependency.startsWith('.')) continue;
      const resolved = resolveCandidate(path.resolve(path.dirname(absolute), dependency));
      if (resolved) addEdge(absolute, resolved);
    }
  }

  const specs = [...nodes.keys()]
    .filter((absolute) => absolute.endsWith('.spec.ts') || absolute.endsWith('.test.ts'))
    .sort();

  function relativeToFrontend(absolute) {
    return path.relative(frontendRoot, absolute).split(path.sep).join('/');
  }

  function absoluteFor(repoRelative) {
    return path.resolve(repoRoot, repoRelative);
  }

  function reverseClosure(startNodes) {
    const visited = new Set();
    const queue = [...startNodes];
    while (queue.length > 0) {
      const current = queue.shift();
      if (visited.has(current)) continue;
      visited.add(current);
      for (const importer of reverse.get(current) || []) {
        if (!visited.has(importer)) queue.push(importer);
      }
    }
    return visited;
  }

  /**
   * Devuelve las specs que cubren una ruta del repositorio. Para plantillas y
   * estilos resuelve primero el componente dueño; si no hay dueño conocido, la
   * ruta queda sin cobertura y el selector debe tratarla de forma conservadora.
   */
  function specsForPath(repoRelative) {
    const absolute = absoluteFor(repoRelative);
    const isSpec = absolute.endsWith('.spec.ts') || absolute.endsWith('.test.ts');
    if (isSpec) {
      return { specs: [relativeToFrontend(absolute)], owned: true, known: nodes.has(absolute) };
    }

    const startNodes = [absolute];
    if (!nodes.has(absolute)) {
      // Un archivo borrado o renombrado no está en el grafo actual. Se busca su
      // dueño por convención de nombre para no omitir la spec del componente.
      const directory = path.dirname(absolute);
      const basename = path.basename(absolute).replace(/\.(html|scss)$/, '');
      const sibling = path.join(directory, `${basename}.ts`);
      if (fs.existsSync(sibling)) startNodes.push(sibling);
    }

    const reachable = reverseClosure(startNodes);
    const covered = [...reachable]
      .filter((node) => node.endsWith('.spec.ts') || node.endsWith('.test.ts'))
      .map(relativeToFrontend)
      .sort();
    // `nodes.has` solo dice que el archivo está en el grafo, no que tenga un
    // componente dueño. Un HTML/SCSS sin importadores debe activar el fallback
    // conservador aunque exista como nodo.
    const hasImporter = (reverse.get(absolute)?.size || 0) > 0;
    return {
      specs: covered,
      owned: covered.length > 0 || hasImporter,
      known: nodes.has(absolute)
    };
  }

  return {
    frontendRoot,
    aliases,
    nodes,
    specs,
    relativeToFrontend,
    specsForPath,
    allSpecs: specs.map(relativeToFrontend),
    scssDependencyCount: files.filter((file) => file.endsWith('.scss')).length
  };
}
