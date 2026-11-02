import * as vscode from 'vscode';
import { SolvApiError, SolvClient } from '../client/solv-client';
import { SolvSettings } from '../config/solv-settings';
import { KeystrokeEvent } from '../domain/keystroke';
import { buildKeystrokeEvent } from './keystroke-classifier';

const FLUSH_INTERVAL_MS = 5000;
const MAX_BUFFERED_EVENTS = 5000;
const MAX_BATCH_SIZE = 200;
const WARNING_INTERVAL_MS = 60000;

/**
 * Captura inserciones, borrados y pegados masivos de archivos del workspace y
 * los envía en lotes. No registra contenido de archivos fuera del proyecto ni
 * documentos virtuales.
 */
export class KeystrokeCapturer implements vscode.Disposable {
  private readonly disposables: vscode.Disposable[] = [];
  private readonly buffer: KeystrokeEvent[] = [];
  private timer: NodeJS.Timeout | undefined;
  private startedAt = Date.now();
  private flushing = false;
  private started = false;
  private lastWarningAt = 0;

  constructor(
    private readonly client: SolvClient,
    private readonly settings: SolvSettings
  ) {}

  start(): void {
    if (this.started) {
      return;
    }
    this.started = true;
    this.startedAt = Date.now();

    this.disposables.push(
      vscode.workspace.onDidChangeTextDocument(event => this.capture(event)),
      vscode.workspace.onDidSaveTextDocument(() => void this.flush())
    );
    this.timer = setInterval(() => void this.flush(), FLUSH_INTERVAL_MS);
  }

  showStatus(): void {
    const state = this.settings.telemetryEnabled ? 'activa' : 'desactivada';
    void vscode.window.showInformationMessage(
      `SOLV Companion: telemetría ${state}, ${this.buffer.length} eventos pendientes.`
    );
  }

  async flush(): Promise<void> {
    if (this.flushing || this.buffer.length === 0) {
      return;
    }

    if (!this.settings.telemetryEnabled || !this.settings.submissionId) {
      this.buffer.length = 0;
      return;
    }

    this.flushing = true;
    const batch = this.buffer.slice(0, MAX_BATCH_SIZE);
    try {
      await this.client.sendKeystrokeEvents(batch);
      this.buffer.splice(0, batch.length);
    } catch (error) {
      this.warnOnce(error);
    } finally {
      this.flushing = false;
    }
  }

  dispose(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = undefined;
    }
    for (const disposable of this.disposables) {
      disposable.dispose();
    }
    void this.flush();
  }

  private capture(event: vscode.TextDocumentChangeEvent): void {
    if (!this.settings.telemetryEnabled || !this.settings.submissionId) {
      return;
    }
    if (event.document.uri.scheme !== 'file') {
      return;
    }
    if (!vscode.workspace.getWorkspaceFolder(event.document.uri)) {
      return;
    }
    if (event.contentChanges.length === 0) {
      return;
    }

    const elapsed = Date.now() - this.startedAt;
    const threshold = this.settings.pasteThreshold;
    for (const change of event.contentChanges) {
      this.buffer.push(
        buildKeystrokeEvent(
          change.text,
          change.rangeLength,
          event.document.offsetAt(change.range.start),
          elapsed,
          threshold
        )
      );
    }

    if (this.buffer.length > MAX_BUFFERED_EVENTS) {
      this.buffer.splice(0, this.buffer.length - MAX_BUFFERED_EVENTS);
    }
  }

  private warnOnce(error: unknown): void {
    const now = Date.now();
    if (now - this.lastWarningAt < WARNING_INTERVAL_MS) {
      return;
    }
    this.lastWarningAt = now;
    const message =
      error instanceof SolvApiError ? error.message : 'No se pudo enviar la telemetría.';
    void vscode.window.showWarningMessage(`SOLV Companion: ${message}`);
  }
}
