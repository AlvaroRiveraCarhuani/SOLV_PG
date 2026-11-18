import { describe, expect, it } from 'vitest';
import { isTreeBlock, parseTree, renderTreeBlock } from './markdown-tree.utils';

describe('markdown-tree.utils', () => {
  describe('isTreeBlock y parseTree', () => {
    it('detecta un bloque de árbol de directorios con conectores clásicos', () => {
      const tree = `
proyecto/
├── backend/
│   ├── prisma/
│   │   └── schema.prisma
│   └── src/
└── README.md
      `.trim();
      expect(isTreeBlock(tree)).toBe(true);
      const parsed = parseTree(tree.split('\n'));
      expect(parsed).not.toBeNull();
      expect(parsed![0].name).toBe('proyecto');
      expect(parsed![0].children.length).toBe(2);
    });

    it('detecta jerarquías de rutas indentadas sin conectores explícitos', () => {
      const indented = `
/backend
  /prisma/schema.prisma
  /src/index.ts
  /src/routes/tareas.ts
  .env
/frontend
  /src/App.tsx
  /src/components/TareaItem.tsx (o similar)
  vite.config.ts
README.md
      `.trim();
      expect(isTreeBlock(indented)).toBe(true);
      const parsed = parseTree(indented.split('\n'));
      expect(parsed).not.toBeNull();
      expect(parsed!.length).toBeGreaterThan(0);
    });

    it('rechaza código fuente común', () => {
      const code = `
function test() {
  const x = 10;
  return x + 5;
}
      `.trim();
      expect(isTreeBlock(code)).toBe(false);
    });

    it('rechaza bloques muy cortos', () => {
      expect(isTreeBlock('├── src/')).toBe(false);
      expect(isTreeBlock('/backend')).toBe(false);
    });
  });

  describe('renderTreeBlock', () => {
    it('genera una estructura semántica idéntica a solv-companion sin caracteres ASCII', () => {
      const tree = `proyecto/\n├── src/\n│   └── index.ts\n└── package.json`;
      const html = renderTreeBlock(tree);
      expect(html).toContain('class="tree"');
      expect(html).toContain('class="tree-list"');
      expect(html).toContain('class="tree-item tree-dir"');
      expect(html).toContain('class="tree-item tree-file"');
      expect(html).toContain('class="tree-icon"');
      expect(html).toContain('proyecto');
      expect(html).toContain('src');
      expect(html).toContain('index.ts');
      expect(html).toContain('package.json');
      // No debe contener los glifos de texto de las ramas
      expect(html).not.toContain('├──');
      expect(html).not.toContain('└──');
      expect(html).not.toContain('│');
    });

    it('renderiza jerarquías indentadas con la misma estructura ul/li', () => {
      const indented = `/backend\n  /src/index.ts\nREADME.md`;
      const html = renderTreeBlock(indented);
      expect(html).toContain('class="tree"');
      expect(html).toContain('class="tree-list"');
      expect(html).toContain('/backend');
      expect(html).toContain('/src/index.ts');
      expect(html).toContain('README.md');
    });
  });
});
