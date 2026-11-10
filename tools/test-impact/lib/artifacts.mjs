import path from 'node:path';

/**
 * Rutas de salida del motor y del CI. La exclusión NO puede depender del
 * basename: un fuente real llamado `impact-report.json` debe seguir siendo un
 * cambio. Solo los nombres de artefacto en la raíz y los directorios de salida
 * conocidos son artefactos.
 */
export function isGeneratedArtifact(filePath) {
  const normalized = path.posix.normalize(String(filePath || '').replace(/\\/g, '/'));
  if (!normalized || normalized === '.' || normalized.startsWith('../')) return false;
  return (
    /^impact-(plan|report)[^/]*\.json$/.test(normalized) ||
    normalized.startsWith('tools/test-impact/.cache/') ||
    normalized === 'impact-reports' ||
    normalized.startsWith('impact-reports/') ||
    /^(backend|frontend)-e2e\.(log|pid)$/.test(normalized) ||
    normalized.startsWith('frontend/playwright-report/') ||
    normalized.startsWith('frontend/e2e/test-results/') ||
    normalized.startsWith('frontend/test-results/')
  );
}

const SOURCE_EXTENSIONS = new Set([
  '.go',
  '.ts',
  '.tsx',
  '.js',
  '.mjs',
  '.cjs',
  '.json',
  '.scss',
  '.css',
  '.html',
  '.sql',
  '.yaml',
  '.yml',
  '.toml',
  '.xml',
  '.svg',
  '.sh',
  '.py',
  '.java',
  '.rs',
  '.c',
  '.cc',
  '.cpp',
  '.h',
  '.hpp',
  '.proto',
  '.graphql'
]);

const RELEVANT_ROOTS = [
  'backend/',
  'frontend/src/',
  'frontend/public/',
  'frontend/scripts/',
  'frontend/e2e/',
  'e2e/',
  'tools/test-impact/',
  'infra/'
];

const HEAVY_PREFIXES = [
  'node_modules/',
  '.git/',
  '.venv/',
  'venv/',
  'vendor/',
  'dist/',
  'build/',
  'coverage/',
  '.angular/',
  'tmp/',
  'uploads/',
  'backend/bin/',
  'frontend/.angular/',
  'frontend/dist/',
  'frontend/coverage/',
  'frontend/node_modules/',
  'tests/stress/venv/'
];

/**
 * Un archivo ignorado solo entra al diff si parece una fuente versionable. Se
 * excluyen dependencias, builds, cachés y secretos locales; no se recorre ni se
 * incorpora `node_modules` indiscriminadamente.
 */
export function isRelevantIgnoredSource(filePath) {
  const normalized = path.posix.normalize(String(filePath || '').replace(/\\/g, '/'));
  if (!normalized || normalized === '.' || normalized.startsWith('../')) return false;
  if (isGeneratedArtifact(normalized)) return false;
  if (/(^|\/)\.env(\.|$)/.test(normalized)) return false;
  if (normalized.split('/').some((part) => part === '.git' || part === '.cache')) return false;
  if (HEAVY_PREFIXES.some((prefix) => normalized.startsWith(prefix))) return false;
  if (normalized.includes('/node_modules/') || normalized.startsWith('node_modules/')) return false;
  if (!RELEVANT_ROOTS.some((root) => normalized.startsWith(root))) return false;
  const extension = path.posix.extname(normalized).toLowerCase();
  return SOURCE_EXTENSIONS.has(extension);
}
