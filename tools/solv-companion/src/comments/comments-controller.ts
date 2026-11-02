import * as vscode from 'vscode';
import { SolvClient, SolvApiError } from '../client/solv-client';
import { SolvSettings } from '../config/solv-settings';
import { SubmissionComment } from '../domain/comment';
import { CommentDecorator } from './comment-decorator';

/** Orquesta la consulta de comentarios y su render en los editores visibles. */
export class CommentsController implements vscode.Disposable {
  private comments: SubmissionComment[] = [];
  private refreshing = false;
  private timer: NodeJS.Timeout | undefined;
  private readonly disposables: vscode.Disposable[] = [];

  constructor(
    private readonly client: SolvClient,
    private readonly decorator: CommentDecorator,
    private readonly settings: SolvSettings
  ) {
    this.disposables.push(
      vscode.window.onDidChangeVisibleTextEditors(() =>
        this.decorator.render(this.comments)
      ),
      vscode.window.onDidChangeActiveTextEditor(() =>
        this.decorator.render(this.comments)
      ),
      vscode.workspace.onDidSaveTextDocument(() => void this.refresh()),
      vscode.workspace.onDidChangeConfiguration(event => {
        if (!event.affectsConfiguration('solv')) {
          return;
        }
        this.restartTimer();
        void this.refresh();
      })
    );
    this.restartTimer();
  }

  async refresh(showErrors = false): Promise<void> {
    if (!this.settings.isConfigured) {
      this.comments = [];
      this.decorator.clearAll();
      if (showErrors) {
        void vscode.window.showWarningMessage(
          'SOLV Companion: configurá la URL y la entrega con el comando "SOLV: Configurar conexión".'
        );
      }
      return;
    }

    if (this.refreshing) {
      return;
    }
    this.refreshing = true;
    try {
      this.comments = await this.client.fetchComments();
      this.decorator.render(this.comments);
    } catch (error) {
      if (showErrors) {
        const message =
          error instanceof SolvApiError ? error.message : 'No se pudieron cargar los comentarios.';
        void vscode.window.showWarningMessage(`SOLV Companion: ${message}`);
      }
    } finally {
      this.refreshing = false;
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
  }

  private restartTimer(): void {
    if (this.timer) {
      clearInterval(this.timer);
    }
    this.timer = setInterval(() => void this.refresh(), this.settings.refreshIntervalMs);
  }
}
