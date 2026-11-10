import { readDroppedEntries, readSelectedFiles, WorkspaceDropEntryLike } from './workspace-project-import';

describe('workspace project import', () => {
  it('conserva la ruta relativa de una selección webkitdirectory', async () => {
    const first = new File(['package main\n'], 'main.go', { type: 'text/plain' });
    const second = new File(['package calc\n'], 'calculator.go', { type: 'text/plain' });
    Object.defineProperty(first, 'webkitRelativePath', { value: 'proyecto-go/main.go' });
    Object.defineProperty(second, 'webkitRelativePath', { value: 'proyecto-go/internal/calculator.go' });

    const files = await readSelectedFiles([first, second]);

    expect(files).toEqual([
      { path: 'proyecto-go/main.go', content: 'package main\n' },
      { path: 'proyecto-go/internal/calculator.go', content: 'package calc\n' }
    ]);
  });

  it('rechaza un archivo que no es UTF-8 válido en vez de reemplazarlo', async () => {
    const binary = new File([new Uint8Array([0xff, 0xfe, 0x00])], 'binario.dat');
    await expect(readSelectedFiles([binary])).rejects.toThrow();
  });

  it('reconoce un ZIP soltado sobre la zona de importación', async () => {
    const archive = new File(['zip'], 'proyecto.zip', { type: 'application/zip' });
    const entry: WorkspaceDropEntryLike = {
      isFile: true,
      isDirectory: false,
      name: 'proyecto.zip',
      file: success => success(archive)
    };

    const selection = await readDroppedEntries([entry]);

    expect(selection.kind).toBe('zip');
    if (selection.kind === 'zip') expect(selection.file.name).toBe('proyecto.zip');
  });

  it('rechaza varios archivos sueltos que no son una carpeta', async () => {
    const entry: WorkspaceDropEntryLike = { isFile: true, isDirectory: false, name: 'main.go' };
    await expect(readDroppedEntries([entry])).rejects.toThrow('carpeta');
  });
});