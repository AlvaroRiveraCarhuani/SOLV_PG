import path from 'node:path';

const IMAGE_SIGNATURES = [
  { extension: '.png', check: (buffer) => buffer.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])) },
  { extension: '.jpg', check: (buffer) => buffer.subarray(0, 3).equals(Buffer.from([0xff, 0xd8, 0xff])) },
  { extension: '.jpeg', check: (buffer) => buffer.subarray(0, 3).equals(Buffer.from([0xff, 0xd8, 0xff])) },
  { extension: '.gif', check: (buffer) => buffer.subarray(0, 4).toString('latin1') === 'GIF8' },
  {
    extension: '.webp',
    check: (buffer) =>
      buffer.subarray(0, 4).toString('latin1') === 'RIFF' &&
      buffer.subarray(8, 12).toString('latin1') === 'WEBP'
  },
  { extension: '.ico', check: (buffer) => buffer.subarray(0, 4).equals(Buffer.from([0x00, 0x00, 0x01, 0x00])) },
  { extension: '.woff', check: (buffer) => buffer.subarray(0, 4).toString('latin1') === 'wOFF' },
  { extension: '.woff2', check: (buffer) => buffer.subarray(0, 4).toString('latin1') === 'wOF2' },
  { extension: '.ttf', check: (buffer) => buffer.subarray(0, 4).equals(Buffer.from([0x00, 0x01, 0x00, 0x00])) }
];

function balanced(source, open, close) {
  let depth = 0;
  for (const char of source) {
    if (char === open) depth += 1;
    if (char === close) depth -= 1;
    if (depth < 0) return false;
  }
  return depth === 0;
}

/**
 * Valida el contenido mínimo que puede comprobarse sin compilar ni instalar
 * dependencias. No sustituye a una compilación: para formatos sin validador
 * disponible solo se comprueba que el archivo no esté vacío.
 */
export function validateAssetContent(relative, buffer) {
  const extension = path.posix.extname(relative).toLowerCase();
  const text = () => buffer.toString('utf8');

  if (extension === '.svg') {
    const source = text().trim();
    if (!source.startsWith('<svg') && !source.startsWith('<?xml')) {
      return { ok: false, error: 'un SVG debe empezar por <svg> o <?xml' };
    }
    if (!/<svg[\s>]/.test(source)) return { ok: false, error: 'falta el elemento raíz <svg>' };
    if (!/<\/svg>\s*$/.test(source)) return { ok: false, error: 'falta el cierre </svg>' };
    if (source.includes('{{')) return { ok: false, error: 'el SVG contiene una plantilla sin resolver' };
    return { ok: true };
  }

  if (extension === '.json') {
    try {
      JSON.parse(text());
      return { ok: true };
    } catch (error) {
      return { ok: false, error: `JSON inválido: ${error.message}` };
    }
  }

  if (extension === '.css' || extension === '.scss') {
    if (!balanced(text(), '{', '}')) return { ok: false, error: 'llaves desbalanceadas' };
    return { ok: true };
  }

  if (extension === '.html') {
    const source = text().toLowerCase();
    if (!source.includes('<html') && !source.includes('<!doctype')) {
      return { ok: false, error: 'un HTML debe declarar <!doctype> o <html>' };
    }
    return { ok: true };
  }

  const signature = IMAGE_SIGNATURES.find((entry) => entry.extension === extension);
  if (signature) {
    if (!signature.check(buffer)) {
      return { ok: false, error: `firma de ${extension} inválida` };
    }
    return { ok: true };
  }

  return { ok: true, limit: `sin validador semántico para ${extension || 'archivo sin extensión'}` };
}
