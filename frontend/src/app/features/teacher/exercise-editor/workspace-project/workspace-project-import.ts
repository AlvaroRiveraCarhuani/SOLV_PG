import { WorkspaceProjectFile } from './workspace-project.utils';

export interface WorkspaceFileHandleLike {
  kind: 'file' | 'directory';
  name: string;
  getFile(): Promise<File>;
  entries?(): AsyncIterableIterator<[string, WorkspaceFileHandleLike]>;
}

export interface WorkspaceDropEntryLike {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
  file?(success: (file: File) => void, error?: (error: DOMException) => void): void;
  createReader?(): { readEntries(success: (entries: WorkspaceDropEntryLike[]) => void, error?: (error: DOMException) => void): void };
}

export type WorkspaceImportSelection =
  | { kind: 'folder'; files: WorkspaceProjectFile[] }
  | { kind: 'zip'; file: File };

interface DirectoryPickerWindow {
  showDirectoryPicker?: () => Promise<WorkspaceFileHandleLike>;
}

export function canPickDirectory(): boolean {
  return typeof (globalThis as unknown as DirectoryPickerWindow).showDirectoryPicker === 'function';
}

export async function pickDirectory(): Promise<WorkspaceProjectFile[]> {
  const picker = (globalThis as unknown as DirectoryPickerWindow).showDirectoryPicker;
  if (!picker) throw new Error('El navegador no permite seleccionar carpetas directamente.');
  const handle = await picker();
  return readDirectoryHandle(handle);
}

export async function readDirectoryHandle(handle: WorkspaceFileHandleLike, prefix = ''): Promise<WorkspaceProjectFile[]> {
  if (handle.kind !== 'directory' || !handle.entries) return [];
  const files: WorkspaceProjectFile[] = [];
  for await (const [name, entry] of handle.entries()) {
    const path = `${prefix}${name}`;
    if (entry.kind === 'directory') {
      files.push(...await readDirectoryHandle(entry, `${path}/`));
      continue;
    }
    const file = await entry.getFile();
    files.push({ path, content: await readTextFile(file) });
  }
  return files;
}

/** `webkitdirectory` entrega la ruta relativa en cada File; se conserva tal cual. */
export async function readSelectedFiles(files: ArrayLike<File>): Promise<WorkspaceProjectFile[]> {
  const result: WorkspaceProjectFile[] = [];
  for (let index = 0; index < files.length; index += 1) {
    const file = files[index];
    const relative = (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name;
    const content = await readTextFile(file);
    result.push({ path: relative, content });
  }
  return result;
}

export async function readDroppedEntries(entries: WorkspaceDropEntryLike[]): Promise<WorkspaceImportSelection> {
  if (entries.length === 1 && entries[0].isFile && entries[0].name.toLowerCase().endsWith('.zip')) {
    const file = await fileFromEntry(entries[0]);
    return { kind: 'zip', file };
  }
  const files: WorkspaceProjectFile[] = [];
  for (const entry of entries) {
    if (!entry.isDirectory) throw new Error('Arrastra una carpeta o un único archivo ZIP.');
    files.push(...await readDroppedDirectory(entry));
  }
  return { kind: 'folder', files };
}

async function readDroppedDirectory(entry: WorkspaceDropEntryLike, prefix = ''): Promise<WorkspaceProjectFile[]> {
  if (!entry.createReader) return [];
  const reader = entry.createReader();
  const files: WorkspaceProjectFile[] = [];
  while (true) {
    const batch = await new Promise<WorkspaceDropEntryLike[]>((resolve, reject) => {
      reader.readEntries(resolve, reject);
    });
    if (batch.length === 0) break;
    for (const child of batch) {
      const path = `${prefix}${child.name}`;
      if (child.isDirectory) {
        files.push(...await readDroppedDirectory(child, `${path}/`));
      } else if (child.isFile) {
        const file = await fileFromEntry(child);
        files.push({ path, content: await readTextFile(file) });
      }
    }
  }
  return files;
}

// `.text()` reemplaza bytes inválidos por U+FFFD y convertiría un binario en
// texto válido. La lectura fatal conserva la garantía de solo texto UTF-8.
async function readTextFile(file: File): Promise<string> {
  const bytes = await file.arrayBuffer();
  return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
}

function fileFromEntry(entry: WorkspaceDropEntryLike): Promise<File> {
  return new Promise((resolve, reject) => {
    if (!entry.file) {
      reject(new Error(`No se pudo leer ${entry.name}.`));
      return;
    }
    entry.file(resolve, reject);
  });
}