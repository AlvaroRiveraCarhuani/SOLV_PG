import * as vscode from 'vscode';
import { AdvanceClient, SessionBoard } from './advance-client';
import { AdvanceContentProvider } from './advance-content-provider';
import { AdvanceDelivery, AdvanceFileEntry, AdvancePreview } from '../domain/advance';
import { SolvApiError } from '../client/solv-client';
import { SolvSettings } from '../config/solv-settings';
import { applySelection, buildChecklist, diffTitleText, selectedPaths } from '../views/panel-state';

/** Avance recordado para ofrecer deshacer aun cuando ya no está pendiente. */
export interface AppliedAdvance {
  id: string;
  createdAt: string;
}

/** Lo que el panel necesita del dominio de avances. */
export interface AdvanceSnapshot {
  advance?: {
    delivery: AdvanceDelivery;
    preview: AdvancePreview;
    files: AdvanceFileEntry[];
  };
  applied?: AppliedAdvance;
  /** Archivos que el respaldo todavía puede restaurar. */
  appliedFiles?: string[];
  board?: SessionBoard;
}

const APPLIED_KEY = 'solv.appliedAdvance';

/**
 * Mantiene el estado del avance del alumno y los datos del aula del docente.
 * Las operaciones que escriben en el entorno guardan antes las pestañas
 * abiertas, para que el diff y la copia no compitan con cambios en memoria.
 */
export class AdvanceController implements vscode.Disposable {
  private readonly emitter = new vscode.EventEmitter<void>();
  private readonly disposables: vscode.Disposable[] = [];
  private current?: SnapshotCache;
  private board?: SessionBoard;
  private appliedFiles?: string[];

  readonly onDidChange = this.emitter.event;

  constructor(
    private readonly client: AdvanceClient,
    private readonly provider: AdvanceContentProvider,
    private readonly settings: SolvSettings,
    private readonly state: vscode.Memento
  ) {}

  get role(): string {
    return this.settings.role;
  }

  get lastApplied(): AppliedAdvance | undefined {
    return this.state.get<AppliedAdvance>(APPLIED_KEY);
  }

  /** Vuelve a leer los avisos pendientes y el aula, sin escribir nada. */
  async refresh(): Promise<AdvanceSnapshot> {
    const snapshot: AdvanceSnapshot = {};
    try {
      if (this.role === 'teacher' || this.role === 'admin') {
        snapshot.board = await this.loadBoard();
      } else {
        const pending = await this.client.pending();
        const delivery = pending[0];
        if (delivery) {
          const preview = await this.client.preview(delivery.id);
          snapshot.advance = { delivery, preview, files: await this.checklist(preview) };
        }
        snapshot.applied = this.lastApplied;
        if (snapshot.applied) {
          snapshot.appliedFiles = await this.loadRollbackFiles(snapshot.applied.id);
        }
      }
    } catch (error) {
      this.current = undefined;
      this.board = undefined;
      throw error;
    }

    this.board = snapshot.board;
    this.appliedFiles = snapshot.appliedFiles;
    this.current = snapshot.advance
      ? {
          delivery: snapshot.advance.delivery,
          preview: snapshot.advance.preview,
          files: snapshot.advance.files
        }
      : undefined;
    return snapshot;
  }

  snapshot(): AdvanceSnapshot {
    return {
      advance: this.current
        ? {
            delivery: this.current.delivery,
            preview: this.current.preview,
            files: this.current.files
          }
        : undefined,
      applied: this.lastApplied,
      appliedFiles: this.appliedFiles,
      board: this.board
    };
  }

  /** Marca el checklist según lo que llega de la webview. */
  select(paths: string[]): AdvanceFileEntry[] {
    if (!this.current) {
      return [];
    }
    this.current.files = applySelection(this.current.files, paths);
    return this.current.files;
  }

  /**
   * Abre el diff nativo entre el archivo local y el contenido emitido. Es
   * puramente en memoria: el proveedor virtual sirve el texto del avance.
   */
  async diff(relativePath: string): Promise<void> {
    const current = this.current;
    if (!current) {
      throw new Error('No hay un avance pendiente para comparar.');
    }
    const content = current.preview.files[relativePath];
    if (content === undefined) {
      throw new Error(`El archivo ${relativePath} no forma parte del avance.`);
    }

    await vscode.workspace.saveAll(false);

    const virtual = AdvanceContentProvider.uriFor(current.preview.advance_id, relativePath);
    this.provider.publish(virtual, content);

    const root = vscode.workspace.workspaceFolders?.[0]?.uri;
    const local = root ? vscode.Uri.joinPath(root, relativePath) : vscode.Uri.file(relativePath);
    // Un archivo que el alumno todavía no tiene no se puede comparar contra un
    // `file:` inexistente: el editor responde "file was not found" y la pestaña
    // queda vacía. Se compara contra un documento virtual vacío y el avance se
    // lee completo como agregado.
    const isNew = !(await fileExists(local));
    await vscode.commands.executeCommand(
      'vscode.diff',
      isNew ? AdvanceContentProvider.emptyBaseUri(current.preview.advance_id, relativePath) : local,
      virtual,
      diffTitleText(relativePath, isNew)
    );
  }

  /** Aplica solo las rutas marcadas en el checklist. */
  async applySelection(paths?: string[]): Promise<{ filesWritten: number }> {
    const current = this.current;
    if (!current) {
      throw new Error('No hay un avance pendiente para aplicar.');
    }
    const chosen = paths && paths.length > 0 ? paths : selectedPaths(current.files);
    if (chosen.length === 0) {
      throw new Error('Marcá al menos un archivo del checklist para aplicar.');
    }

    await vscode.workspace.saveAll(false);
    const result = await this.client.apply(current.delivery.id, chosen);
    await this.state.update(APPLIED_KEY, {
      id: result.advance_id,
      createdAt: current.delivery.created_at
    });
    this.current = undefined;
    this.emitter.fire();
    return { filesWritten: result.files_written };
  }

  /** Descarta el aviso sin tocar el proyecto del alumno. */
  async dismiss(): Promise<void> {
    const current = this.current;
    if (!current) {
      return;
    }
    await this.client.dismiss(current.delivery.id);
    this.current = undefined;
    this.emitter.fire();
  }

  /**
   * Restaura los archivos elegidos desde el respaldo; sin lista restaura todo.
   * Mientras quede respaldo el aviso sigue aplicado, para deshacer el resto
   * después; recién al restaurar el último archivo vuelve a pendiente.
   */
  async rollback(
    paths?: string[]
  ): Promise<{ filesRestored: number; remaining: number; fullyRestored: boolean }> {
    const applied = this.lastApplied;
    if (!applied) {
      throw new Error('No hay un avance aplicado para deshacer.');
    }

    await vscode.workspace.saveAll(false);
    const result = await this.client.rollback(applied.id, paths ?? []);
    if (result.fully_restored) {
      await this.state.update(APPLIED_KEY, undefined);
      this.appliedFiles = undefined;
    } else {
      this.appliedFiles = result.remaining;
    }
    this.emitter.fire();
    return {
      filesRestored: result.files_restored,
      remaining: result.remaining?.length ?? 0,
      fullyRestored: result.fully_restored
    };
  }

  /**
   * Archivos todavía restaurables del avance aplicado. Un fallo no rompe el
   * panel: devuelve `undefined` y la vista ofrece el deshacer completo.
   */
  private async loadRollbackFiles(advanceId: string): Promise<string[] | undefined> {
    try {
      const inventory = await this.client.rollbackFiles(advanceId);
      return inventory.files;
    } catch {
      return undefined;
    }
  }

  /** Congela o descongela la emisión del aula. */
  async toggleFreeze(): Promise<boolean> {
    const sessionId = this.settings.sessionId || this.board?.session.id;
    if (!sessionId) {
      throw new Error('Falta el aula abierta para pausar la clase.');
    }
    const frozen = Boolean(this.board?.session.frozen_at);
    await this.client.setFrozen(sessionId, !frozen);
    this.board = await this.loadBoard();
    this.emitter.fire();
    return !frozen;
  }

  async openSession(subjectId: string, durationMinutes = 90, exerciseId?: string): Promise<{ id: string }> {
    const res = await this.client.openSession(subjectId, durationMinutes, exerciseId);
    // El aula recién abierta se recuerda en la sesión del editor: emitir y
    // congelar leen el identificador de ahí.
    this.settings.setSessionId(res.id);
    this.board = await this.loadBoard();
    this.emitter.fire();
    return res;
  }

  async closeSession(sessionId: string): Promise<unknown> {
    const res = await this.client.closeSession(sessionId);
    this.board = undefined;
    // El aula cerrada ya no aplica: el panel vuelve a ofrecer iniciar clase y
    // emitir o congelar dejan de tener un identificador vigente.
    this.settings.clearSessionId();
    this.emitter.fire();
    return res;
  }

  /** Presencia y entregas del aula; si no hay sesión configurada intenta autodescubrir una abierta. */
  private async loadBoard(): Promise<SessionBoard | undefined> {
    let sessionId = this.settings.sessionId;
    if (!sessionId) {
      const open = await this.client.currentOpenSession();
      if (open?.id) {
        sessionId = open.id;
        // El aula abierta desde la web no llega por entorno: se guarda igual que
        // la que abre el panel, para que las dos acciones del aula la usen.
        this.settings.setSessionId(sessionId);
      }
    }
    if (!sessionId) {
      return undefined;
    }
    const board = await this.client.board(sessionId);
    return board ?? undefined;
  }

  /** Lee cada archivo del avance desde el disco para clasificarlo. */
  private async checklist(preview: AdvancePreview): Promise<AdvanceFileEntry[]> {
    const root = vscode.workspace.workspaceFolders?.[0]?.uri;
    const local: Record<string, string | undefined> = {};
    for (const path of Object.keys(preview.files)) {
      local[path] = await this.readLocal(root, path);
    }
    return buildChecklist(preview, local);
  }

  private async readLocal(
    root: vscode.Uri | undefined,
    relativePath: string
  ): Promise<string | undefined> {
    if (!root) {
      return undefined;
    }
    try {
      const bytes = await vscode.workspace.fs.readFile(vscode.Uri.joinPath(root, relativePath));
      return Buffer.from(bytes).toString('utf8');
    } catch {
      return undefined;
    }
  }

  dispose(): void {
    this.emitter.dispose();
    for (const disposable of this.disposables) {
      disposable.dispose();
    }
  }
}

interface SnapshotCache {
  delivery: AdvanceDelivery;
  preview: AdvancePreview;
  files: AdvanceFileEntry[];
}

async function fileExists(uri: vscode.Uri): Promise<boolean> {
  try {
    await vscode.workspace.fs.stat(uri);
    return true;
  } catch {
    return false;
  }
}

export function describeAdvanceError(error: unknown): string {
  if (error instanceof SolvApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return 'No se pudo completar la operación con el avance.';
}
