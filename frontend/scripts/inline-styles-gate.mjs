// Gate D8: prohíbe estilos inline (styles: [`...`]) en componentes Angular.
// Todo CSS debe vivir en archivos .scss gobernados por stylelint
// (color-no-hex, tipografía por tokens, allowed-lists).
// Se ejecuta como parte de `npm run lint:styles` (gate de CI).
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const ROOT = join(process.cwd(), 'src', 'app');

function walk(dir, out = []) {
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    const st = statSync(p);
    if (st.isDirectory()) walk(p, out);
    else if (p.endsWith('.ts') && !p.endsWith('.spec.ts')) out.push(p);
    // acepta .d.ts automáticamente por el filtro anterior (no terminan en .spec.ts,
    // pero no contienen el patrón así que es inofensivo)
  }
  return out;
}

// Detecta styles: [ ] con cualquier contenido, incluyendo backticks multi-línea.
// Evita falsos positivos con styleUrls/styleUrl porque exige [ y ] literal.
const PATTERN = /styles\s*:\s*\[/;

const offenders = [];
for (const file of walk(ROOT)) {
  const content = readFileSync(file, 'utf8');
  if (PATTERN.test(content)) offenders.push(relative(process.cwd(), file));
}

if ( offenders.length) {
  console.error(
    `\n✖ inline-styles-gate: ${offenders.length} componente(s) con estilos inline (styles: [...]).\n` +
      '  Todo CSS debe vivir en archivos .scss gobernados por stylelint.\n' +
      offenders.map((f) => `  - ${f}`).join('\n') +
      '\n'
  );
  process.exit(1);
}

console.log('inline-styles-gate: OK — 0 componentes con styles:[]');
