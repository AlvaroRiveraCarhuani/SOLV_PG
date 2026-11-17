import assert from 'node:assert/strict';
import { test } from 'node:test';
import { renderMarkdown } from './markdown';

test('renderiza títulos, listas y énfasis de una consigna', () => {
  const html = renderMarkdown(
    ['## 1. Modelo de datos', '', '- id (UUID)', '- **title** obligatorio', '', 'Texto con `codigo` y *énfasis*.'].join('\n')
  );

  assert.match(html, /<h4>1\. Modelo de datos<\/h4>/);
  assert.match(html, /<ul><li>id \(UUID\)<\/li><li><strong>title<\/strong> obligatorio<\/li><\/ul>/);
  assert.match(html, /<p>Texto con <code>codigo<\/code> y <em>énfasis<\/em>\.<\/p>/);
});

test('convierte el bloque que dibuja un árbol de carpetas en un árbol real', () => {
  const html = renderMarkdown(
    [
      '```',
      'proyecto/',
      '├── backend/',
      '│   ├── src/',
      '│   │   └── index.ts',
      '│   └── package.json',
      '└── README.md',
      '```',
    ].join('\n')
  );

  assert.match(html, /<div class="tree"><ul class="tree-list">/);
  assert.ok(!html.includes('├──'));
  assert.ok(!html.includes('<pre>'));
  assert.match(html, /tree-dir"><span class="tree-row"><svg[^>]*>.*?<\/svg><span class="tree-name">backend<\/span>/);
  assert.match(html, /tree-name">index\.ts<\/span>/);
  // index.ts queda anidado dentro de src, que a su vez está dentro de backend
  assert.equal((html.match(/tree-list/g) || []).length, 4);
});

test('un bloque de código normal sigue siendo código', () => {
  const html = renderMarkdown(['```', 'const total = items.length;', 'console.log(total);', '```'].join('\n'));

  assert.match(html, /^<pre><code>/);
  assert.ok(!html.includes('tree-list'));
});

test('respeta los bloques de código y no interpreta su contenido', () => {
  const html = renderMarkdown(['```', 'const **x** = 1;', '## no es título', '```'].join('\n'));

  assert.equal(html, '<pre><code>const **x** = 1;\n## no es título</code></pre>');
});

test('escapa el HTML de la consigna antes de interpretar el marcado', () => {
  const html = renderMarkdown('<script>alert(1)</script> y <b onclick="x">fuerte</b>');

  assert.ok(!html.includes('<script>'));
  assert.ok(!html.includes('<b onclick'));
  assert.match(html, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
  assert.match(html, /&lt;b onclick=&quot;x&quot;&gt;fuerte&lt;\/b&gt;/);
});

test('lista numerada, cita y separador', () => {
  const html = renderMarkdown(['1. primero', '2. segundo', '', '> notado', '', '---'].join('\n'));

  assert.match(html, /<ol><li>primero<\/li><li>segundo<\/li><\/ol>/);
  assert.match(html, /<blockquote><p>notado<\/p><\/blockquote>/);
  assert.match(html, /<hr>/);
});

test('enlaces solo con destino http y https', () => {
  assert.match(renderMarkdown('[documentación](https://ejemplo.edu/guia)'), /<a href="https:\/\/ejemplo\.edu\/guia">documentación<\/a>/);
  assert.doesNotMatch(renderMarkdown('[malo](javascript:alert(1))'), /<a href="javascript/);
});
