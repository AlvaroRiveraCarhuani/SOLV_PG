// Gate de fechas: la plataforma renderiza fechas SOLO vía el formateador
// canónico shared/pipes/date-text.pipe.ts (formato determinista) y con fuente
// mono por semántica (<time> usa la tipografía mono global).
// Falla si:
//   1) aparece toLocaleDateString / toLocaleTimeString / Intl.DateTimeFormat
//      fuera del formateador canónico (locale del runtime = salida ambigua), o
//   2) dateText aparece fuera de <time>, o
//   3) src/styles.scss deja de aplicar tipografía mono tabular a <time>.
// Se ejecuta como parte de `npm run lint:styles` (gate de CI).
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const SRC_APP = join(process.cwd(), 'src', 'app');
const CANONICAL_REL = join('src', 'app', 'shared', 'pipes', 'date-text.pipe.ts');
const STYLES_REL = join('src', 'styles.scss');

function walk(dir, extensions, out = []) {
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    const st = statSync(p);
    if (st.isDirectory()) walk(p, extensions, out);
    else if (extensions.some((extension) => p.endsWith(extension)) && !p.endsWith('.spec.ts')) out.push(p);
  }
  return out;
}

const DATE_PIPE_PATTERN = /\|\s*(dateText|date)\b/g;
const TAG_PATTERN = /<!--(?:[\s\S]*?)-->|<\/?([a-z][\w:-]*)\b[^>]*>/gi;
const VOID_ELEMENTS = new Set(['area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'param', 'source', 'track', 'wbr']);

function isInsideTimeElement(template, offset) {
  const stack = [];
  const prefix = template.slice(0, offset);
  let match;

  TAG_PATTERN.lastIndex = 0;
  while ((match = TAG_PATTERN.exec(prefix)) !== null) {
    const token = match[0];
    const tagName = match[1]?.toLowerCase();
    if (!tagName) continue;

    if (token.startsWith('</')) {
      const index = stack.lastIndexOf(tagName);
      if (index !== -1) stack.splice(index);
    } else if (!token.endsWith('/>') && !VOID_ELEMENTS.has(tagName)) {
      stack.push(tagName);
    }
  }

  return stack.includes('time');
}

export function validateTemplate(file, template) {
  const errors = [];
  DATE_PIPE_PATTERN.lastIndex = 0;
  let match;

  while ((match = DATE_PIPE_PATTERN.exec(template)) !== null) {
    if (match[1] === 'date') {
      errors.push(`${file}: no usar el pipe Angular "date"; usar el pipe compartido dateText.`);
    } else if (!isInsideTimeElement(template, match.index)) {
      errors.push(`${file}: dateText debe renderizarse dentro de un elemento <time>.`);
    }
  }

  return errors;
}

export function validateGlobalTimeTypography(styles) {
  const match = /(?:^|})\s*time\s*\{([^{}]*)\}/m.exec(styles);
  if (!match) return ['src/styles.scss: falta la regla global time.'];

  const declarations = match[1];
  const errors = [];
  if (!/font-family\s*:\s*var\(\s*--font-mono\s*\)/.test(declarations)) {
    errors.push('src/styles.scss: la regla global time debe usar var(--font-mono).');
  }
  if (!/font-variant-numeric\s*:\s*tabular-nums\b/.test(declarations)) {
    errors.push('src/styles.scss: la regla global time debe usar números tabulares.');
  }
  return errors;
}

export function validateGlobalControlTypography(styles) {
  const match = /(?:^|})\s*input\s*,\s*select\s*,\s*textarea\s*,\s*optgroup\s*,\s*option\s*,\s*button\s*\{([^{}]*)\}/m.exec(styles);
  if (!match) return ['src/styles.scss: falta la regla global de tipografía para controles.'];

  if (!/font-family\s*:\s*var\(\s*--font-sans\s*\)/.test(match[1])) {
    return ['src/styles.scss: los controles deben usar var(--font-sans).'];
  }
  return [];
}

export function runDateFormatGate() {
  const errors = [];
  const canonicalAbs = join(process.cwd(), CANONICAL_REL);

  // TypeScript may use the canonical formatter but cannot format dates locally.
  for (const file of walk(SRC_APP, ['.ts'])) {
    if (file === canonicalAbs) continue;
    if (/toLocaleDateString|toLocaleTimeString|Intl\.DateTimeFormat/.test(readFileSync(file, 'utf8'))) {
      errors.push(
        `${relative(process.cwd(), file)}: formatea fechas por su cuenta. ` +
          `Usar el pipe dateText (@shared/pipes/date-text.pipe) o formatSolvDate en TS.`
      );
    }
  }

  // Date values in Angular templates must use the canonical pipe and semantic time element.
  for (const file of walk(SRC_APP, ['.html'])) {
    errors.push(...validateTemplate(relative(process.cwd(), file), readFileSync(file, 'utf8')));
  }

  errors.push(...validateGlobalTimeTypography(readFileSync(join(process.cwd(), STYLES_REL), 'utf8')));
  errors.push(...validateGlobalControlTypography(readFileSync(join(process.cwd(), STYLES_REL), 'utf8')));

  if (errors.length) {
    console.error(
      `\n✖ date-format-gate: ${errors.length} violación(es) del canon de fechas.\n` +
        `  Usar dateText dentro de <time>, y formatSolvDate/parseDateValue para lógica TS.\n\n` +
        errors.map((error) => `  - ${error}`).join('\n') +
        '\n'
    );
    process.exitCode = 1;
    return;
  }

  console.log('date-format-gate: OK — fechas canónicas y semánticas');
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  runDateFormatGate();
}
