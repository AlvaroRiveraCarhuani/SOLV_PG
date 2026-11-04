// Gate de movimiento: protege el estándar de animaciones.
// Falla si:
//   1) un componente declara un @keyframes que duplica una primitiva global
//      (por nombre `solv-*` o por cuerpo idéntico a un keyframe del catálogo),
//   2) un @keyframes local usa un nombre de giro (`spin`, `solvSpin`, ...) en vez
//      de la clase .motion-spin,
//   3) un @keyframes local de entrada (no es bucle infinito) no está justificado
//      en MOTION_EXCEPTIONS,
//   4) un archivo anima sin su bloque @media (prefers-reduced-motion: reduce), o
//   5) una declaración animation referencia un keyframe que no existe.
// Se ejecuta como parte de `npm run lint:styles` (gate de CI).
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const SRC = join(process.cwd(), 'src');
const PRIMITIVES_REL = join('src', 'styles', '_animations.scss');
const GLOBAL_STYLE_FILES = [PRIMITIVES_REL, join('src', 'styles.scss')];

// Excepciones justificadas: cada entrada necesita archivo y motivo.
// Un keyframe local de entrada (que no sea bucle) solo pasa si está aquí.
export const MOTION_EXCEPTIONS = {
  bannerSlideDown: {
    file: 'src/app/features/admin/dashboard/admin-dashboard.component.scss',
    reason: 'Aviso superior que baja 6px desde el borde; no hay primitiva de aviso superior.',
  },
};

const CSS_WIDE_KEYWORDS = new Set(['none', 'initial', 'inherit', 'unset', 'revert', 'revert-layer']);
const ANIMATION_KEYWORDS = new Set([
  ...CSS_WIDE_KEYWORDS,
  'normal',
  'reverse',
  'alternate',
  'alternate-reverse',
  'forwards',
  'backwards',
  'both',
  'infinite',
  'paused',
  'running',
  'linear',
  'ease',
  'ease-in',
  'ease-out',
  'ease-in-out',
  'step-start',
  'step-end',
  'jump-start',
  'jump-end',
  'jump-none',
  'jump-both',
]);
const FUNCTION_TOKEN = /^(?:var|cubic-bezier|steps|linear|min|max|clamp|calc|env)\(/;

// Reemplaza comentarios por espacios conservando las líneas, para que los
// números de línea reportados sigan siendo los del archivo original.
function stripComments(source) {
  const withoutBlocks = source.replace(/\/\*[\s\S]*?\*\//g, (comment) =>
    comment.replace(/[^\n]/g, ' ')
  );
  return withoutBlocks
    .split('\n')
    .map((line) => (line.trimStart().startsWith('//') ? line.replace(/[^\n]/g, ' ') : line))
    .join('\n');
}

function lineAt(source, index) {
  let line = 1;
  for (let i = 0; i < index && i < source.length; i += 1) {
    if (source[i] === '\n') line += 1;
  }
  return line;
}

function balancedBlock(source, openIndex) {
  let depth = 0;
  for (let i = openIndex; i < source.length; i += 1) {
    if (source[i] === '{') depth += 1;
    else if (source[i] === '}') {
      depth -= 1;
      if (depth === 0) return source.slice(openIndex + 1, i);
    }
  }
  return source.slice(openIndex + 1);
}

export function extractKeyframes(source) {
  const keyframes = [];
  const pattern = /@keyframes\s+([\w-]+)\s*\{/g;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    const openIndex = match.index + match[0].length - 1;
    keyframes.push({
      name: match[1],
      body: balancedBlock(source, openIndex),
      line: lineAt(source, match.index),
    });
  }
  return keyframes;
}

export function extractAnimations(source) {
  const declarations = [];
  const pattern = /animation(?:-name)?\s*:\s*([^;{}]+)/g;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    declarations.push({ value: match[1].trim(), line: lineAt(source, match.index) });
  }
  return declarations;
}

function splitTokens(value) {
  const tokens = [];
  let current = '';
  let depth = 0;
  for (const char of value) {
    if (char === '(') depth += 1;
    if (char === ')') depth -= 1;
    if (/\s/.test(char) && depth === 0) {
      if (current) tokens.push(current);
      current = '';
      continue;
    }
    current += char;
  }
  if (current) tokens.push(current);
  return tokens;
}

export function keyframeName(value) {
  for (const token of splitTokens(value)) {
    if (FUNCTION_TOKEN.test(token)) continue;
    if (/^[-+]?[\d.]+m?s$/i.test(token)) continue;
    if (ANIMATION_KEYWORDS.has(token.toLowerCase())) continue;
    return token;
  }
  return null;
}

function normalizeBody(body) {
  return body.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\s+/g, '').toLowerCase();
}

function findStructuralDuplicate(keyframe, primitives) {
  const normalized = normalizeBody(keyframe.body);
  for (const [name, body] of Object.entries(primitives)) {
    if (normalizeBody(body) === normalized) return name;
  }
  return null;
}

export function validateExceptionRegistry(exceptions = MOTION_EXCEPTIONS) {
  const errors = [];
  for (const [name, entry] of Object.entries(exceptions)) {
    if (!entry || typeof entry !== 'object') {
      errors.push(`scripts/motion-gate.mjs: la excepción "${name}" necesita { file, reason }.`);
      continue;
    }
    if (!entry.file) errors.push(`scripts/motion-gate.mjs: la excepción "${name}" no declara archivo.`);
    if (!entry.reason || entry.reason.trim().length < 20) {
      errors.push(`scripts/motion-gate.mjs: la excepción "${name}" necesita un motivo explicado.`);
    }
  }
  return errors;
}

export function validateMotionScss(file, source, options = {}) {
  const exceptions = options.exceptions ?? MOTION_EXCEPTIONS;
  const primitives = options.primitives ?? {};
  const classes = options.classes ?? {};
  const knownGlobals = options.knownGlobals ?? new Set(Object.keys(primitives));
  const isPrimitivesFile = options.isPrimitivesFile ?? file.endsWith('_animations.scss');
  // Los archivos de catálogo (styles.scss y _animations.scss) son parte del
  // sistema, no consumidores: sus keyframes no exigen justificación.
  const isCatalogFile = (options.catalogFiles ?? []).includes(file) || isPrimitivesFile;
  const text = stripComments(source);
  const errors = [];

  const keyframes = extractKeyframes(text);
  const declarations = extractAnimations(text);

  if (isPrimitivesFile) return errors;

  const localNames = new Set(keyframes.map((keyframe) => keyframe.name));

  for (const keyframe of keyframes) {
    if (keyframe.name.startsWith('solv-')) {
      errors.push(
        `${file}:${keyframe.line}: @keyframes "${keyframe.name}" redefine una primitiva del sistema. ` +
          `Usar la clase ${classes[keyframe.name] ?? 'del catálogo de _animations.scss'}.`
      );
      continue;
    }

    if (/spin/i.test(keyframe.name)) {
      errors.push(
        `${file}:${keyframe.line}: @keyframes "${keyframe.name}" repite el giro del sistema. Usar la clase .motion-spin.`
      );
      continue;
    }

    const duplicate = findStructuralDuplicate(keyframe, primitives);
    if (duplicate) {
      errors.push(
        `${file}:${keyframe.line}: @keyframes "${keyframe.name}" repite el movimiento de "${duplicate}". ` +
          `Usar la primitiva ${classes[duplicate] ?? duplicate}.`
      );
      continue;
    }

    const isLoop = declarations.some(
      (declaration) =>
        keyframeName(declaration.value) === keyframe.name && /\binfinite\b/i.test(declaration.value)
    );
    const exception = exceptions[keyframe.name];
    const justified = exception && exception.file === file;

    if (!isCatalogFile && !isLoop && !justified) {
      errors.push(
        `${file}:${keyframe.line}: @keyframes "${keyframe.name}" es una entrada local sin justificar. ` +
          'Usar .motion-fade, .motion-rise, .motion-panel, .motion-drawer, .motion-backdrop o .animate-dropdown, ' +
          'o registrarlo en MOTION_EXCEPTIONS (scripts/motion-gate.mjs) con archivo y motivo.'
      );
    }
  }

  for (const declaration of declarations) {
    const name = keyframeName(declaration.value);
    if (!name) continue;
    if (!localNames.has(name) && !knownGlobals.has(name)) {
      errors.push(
        `${file}:${declaration.line}: la animación "${name}" no está definida en el archivo ni en el catálogo global ` +
          '(_animations.scss / styles.scss).'
      );
    }
  }

  const hasActiveAnimation = declarations.some(
    (declaration) => !CSS_WIDE_KEYWORDS.has(declaration.value.trim().toLowerCase())
  );
  const hasReducedMotion = /@media[^{]*prefers-reduced-motion\s*:\s*reduce/.test(text);

  if (hasActiveAnimation && !hasReducedMotion) {
    errors.push(
      `${file}: anima sin bloque @media (prefers-reduced-motion: reduce). ` +
        'Cada archivo que declara animación necesita su versión reducida.'
    );
  }

  return errors;
}

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const path = join(dir, entry);
    const info = statSync(path);
    if (info.isDirectory()) walk(path, out);
    else if (path.endsWith('.scss')) out.push(path);
  }
  return out;
}

function stripKeyframeBlocks(source) {
  let out = '';
  let cursor = 0;
  const pattern = /@keyframes\s+[\w-]+\s*\{/g;
  let match;
  while ((match = pattern.exec(source)) !== null) {
    const openIndex = match.index + match[0].length - 1;
    let depth = 0;
    let end = source.length;
    for (let i = openIndex; i < source.length; i += 1) {
      if (source[i] === '{') depth += 1;
      else if (source[i] === '}') {
        depth -= 1;
        if (depth === 0) {
          end = i + 1;
          break;
        }
      }
    }
    out += source.slice(cursor, match.index);
    cursor = end;
    pattern.lastIndex = end;
  }
  return out + source.slice(cursor);
}

function readIfExists(path) {
  try {
    return readFileSync(path, 'utf8');
  } catch {
    return '';
  }
}

export function runMotionGate() {
  const primitivesSource = readIfExists(join(process.cwd(), PRIMITIVES_REL));
  const primitiveKeyframes = extractKeyframes(stripComments(primitivesSource));

  // Catálogo nombre → clase utilitaria (según la regla que lo usa en _animations.scss).
  const primitiveRules = stripKeyframeBlocks(stripComments(primitivesSource));
  const catalog = {};
  for (const keyframe of primitiveKeyframes) {
    const selectorPattern = new RegExp(`([^{}]+)\\{[^{}]*animation\\s*:\\s*${keyframe.name}\\b[^{}]*\\}`);
    const selector = selectorPattern.exec(primitiveRules)?.[1];
    catalog[keyframe.name] = selector
      ? selector
          .split(',')
          .map((part) => part.trim())
          .filter((part) => part.startsWith('.'))
          .join(', ')
      : 'primitiva del sistema';
  }

  const primitives = Object.fromEntries(
    primitiveKeyframes.map((keyframe) => [keyframe.name, keyframe.body])
  );
  const knownGlobals = new Set(primitiveKeyframes.map((keyframe) => keyframe.name));
  for (const file of GLOBAL_STYLE_FILES) {
    for (const keyframe of extractKeyframes(stripComments(readIfExists(join(process.cwd(), file))))) {
      knownGlobals.add(keyframe.name);
    }
  }

  const errors = [...validateExceptionRegistry()];

  for (const path of walk(SRC)) {
    const file = relative(process.cwd(), path);
    errors.push(
      ...validateMotionScss(file, readFileSync(path, 'utf8'), {
        primitives,
        classes: catalog,
        knownGlobals,
        catalogFiles: GLOBAL_STYLE_FILES,
        exceptions: MOTION_EXCEPTIONS,
      })
    );
  }

  if (errors.length) {
    console.error(
      `\n✖ motion-gate: ${errors.length} incumplimiento(s) del estándar de animaciones.\n` +
        '  Las entradas usan las primitivas .motion-*; los bucles de estado se apagan con prefers-reduced-motion.\n\n' +
        errors.map((error) => `  - ${error}`).join('\n') +
        '\n'
    );
    process.exitCode = 1;
    return;
  }

  console.log('motion-gate: OK — animaciones sobre el catálogo compartido');
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  runMotionGate();
}
