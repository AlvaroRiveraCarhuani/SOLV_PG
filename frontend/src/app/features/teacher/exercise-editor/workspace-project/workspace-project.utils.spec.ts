import {
  WORKSPACE_PROJECT_MAX_FILE_BYTES,
  WORKSPACE_PROJECT_MAX_FILES,
  canonicalizeProjectContent,
  encodeWorkspaceSnapshot,
  normalizeWorkspacePath,
  parseWorkspaceSnapshot,
  summarizeProjectChanges,
  validateProjectFiles
} from './workspace-project.utils';

describe('workspace project snapshot', () => {
  it('conserva rutas anidadas al codificar y decodificar', () => {
    const result = encodeWorkspaceSnapshot([
      { path: 'proyecto-go/main.go', content: 'package main\n' },
      { path: 'proyecto-go/internal/calculator/calculator.go', content: 'package calculator\n' },
      { path: 'proyecto-go/README.md', content: '# Proyecto\n' }
    ]);

    expect(result.error).toBeUndefined();
    const parsed = parseWorkspaceSnapshot(result.text as string);
    expect(parsed.error).toBeUndefined();
    expect(parsed.files?.map(file => file.path)).toEqual([
      'proyecto-go/README.md',
      'proyecto-go/internal/calculator/calculator.go',
      'proyecto-go/main.go'
    ]);
  });

  it('canoniza CRLF y un único salto final', () => {
    expect(canonicalizeProjectContent('uno\r\ndos\r\n\r\n')).toBe('uno\ndos\n');
    expect(canonicalizeProjectContent('   \n')).toBe('');
  });

  it('rechaza traversal, duplicados y binarios', () => {
    expect(normalizeWorkspacePath('../secreto.txt').error).toBeTruthy();
    expect(normalizeWorkspacePath('/etc/passwd').error).toBeTruthy();
    expect(normalizeWorkspacePath('src\\main.go').error).toBeTruthy();
    expect(validateProjectFiles([
      { path: 'main.go', content: 'a\n' },
      { path: 'main.go', content: 'b\n' }
    ])).toContain('duplicada');
    expect(validateProjectFiles([{ path: 'main.go', content: 'a\u0000b' }])).toContain('control');
  });

  it('aplica los límites funcionales antes de enviar', () => {
    const files = Array.from({ length: WORKSPACE_PROJECT_MAX_FILES + 1 }, (_, index) => ({
      path: `archivo-${index}.txt`,
      content: 'x\n'
    }));
    expect(validateProjectFiles(files)).toContain(String(WORKSPACE_PROJECT_MAX_FILES));
    expect(validateProjectFiles([{ path: 'grande.txt', content: 'x'.repeat(WORKSPACE_PROJECT_MAX_FILE_BYTES + 1) }])).toContain('supera');
  });

  it('rechaza una cabecera de snapshot dentro del contenido', () => {
    const result = encodeWorkspaceSnapshot([{ path: 'main.py', content: '# === FILE: otro.py ===\n' }]);
    expect(result.error).toContain('cabecera');
  });

  it('resume agregados, modificados y eliminados contra la versión publicada', () => {
    const summary = summarizeProjectChanges(
      [{ path: 'a.py', content: 'a\n' }, { path: 'b.py', content: 'b\n' }],
      [{ path: 'a.py', content: 'a2\n' }, { path: 'c.py', content: 'c\n' }]
    );
    expect(summary.added).toEqual(['c.py']);
    expect(summary.modified).toEqual(['a.py']);
    expect(summary.removed).toEqual(['b.py']);
  });
});