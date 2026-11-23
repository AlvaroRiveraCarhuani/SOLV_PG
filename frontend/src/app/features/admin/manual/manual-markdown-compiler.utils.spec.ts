import { compileMarkdown, stripDocumentTitle } from './manual-markdown-compiler.utils';

describe('manual-markdown-compiler', () => {
  it('renderiza negrita y código inline dentro de un párrafo, sin literales crudos', () => {
    const { html } = compileMarkdown(
      'Texto con **negrita** y `codigo` en el mismo párrafo.\n',
      ''
    );

    expect(html).toContain('<strong class="manual-strong">negrita</strong>');
    expect(html).toContain('<code class="manual-inline-code">codigo</code>');
    expect(html).not.toContain('**negrita**');
    expect(html).not.toContain('`codigo`');
  });

  it('renderiza negrita y código inline dentro de un título y conserva el id del ancla', () => {
    const { html, toc } = compileMarkdown('## Sección con **negrita** y `codigo`\n', '');

    expect(html).toContain('<strong class="manual-strong">negrita</strong>');
    expect(html).toContain('<code class="manual-inline-code">codigo</code>');
    expect(html).not.toContain('**negrita**');
    expect(toc).toHaveLength(1);
    expect(toc[0].title).toBe('Sección con negrita y codigo');
    expect(toc[0].id).toBe('seccion-con-negrita-y-codigo');
  });

  it('renderiza negrita y código inline dentro de una cita', () => {
    const { html } = compileMarkdown('> Cita con **negrita** y `codigo`.\n', '');

    expect(html).toContain('<blockquote class="manual-blockquote">');
    expect(html).toContain('<strong class="manual-strong">negrita</strong>');
    expect(html).toContain('<code class="manual-inline-code">codigo</code>');
    expect(html).not.toContain('**negrita**');
  });

  it('renderiza negrita y código inline dentro de un callout', () => {
    const { html } = compileMarkdown(
      '> [!NOTE]\n> Aviso con **negrita** y `codigo`.\n',
      ''
    );

    expect(html).toContain('manual-callout callout-info');
    expect(html).toContain('<strong class="manual-strong">negrita</strong>');
    expect(html).toContain('<code class="manual-inline-code">codigo</code>');
    expect(html).not.toContain('**negrita**');
  });

  it('quita el título H1 del documento para no duplicar la cabecera de la página', () => {
    const markdown = '# Manual de Administración del Sistema — SOLV\n\nPárrafo inicial.';

    expect(stripDocumentTitle(markdown)).toBe('Párrafo inicial.');
  });

  it('conserva el contenido cuando no hay H1', () => {
    const markdown = '## Sección\n\nPárrafo.';

    expect(stripDocumentTitle(markdown)).toBe(markdown);
  });
});
