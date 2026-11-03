import * as vscode from 'vscode';
import { PanelState } from './panel-state';
import { renderPanelHtml } from './panel-html';

/** Mensajes que la webview envía a la extensión. */
export type PanelMessage =
  | { type: 'apply'; paths: string[] }
  | { type: 'diff'; path: string }
  | { type: 'open-comment'; file: string; line: number; commentId: string }
  | {
      type:
        | 'dismiss'
        | 'rollback'
        | 'refresh'
        | 'broadcast'
        | 'freeze'
        | 'submit'
        | 'speedgrader'
        | 'open-tab'
        | 'open-aula'
        | 'close-aula';
    };

export function isPanelMessage(value: unknown): value is PanelMessage {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const type = (value as { type?: unknown }).type;
  return typeof type === 'string';
}

/** Panel webview de la barra lateral: consigna, avances, comentarios y aula. */
export class SolvPanelView implements vscode.WebviewViewProvider, vscode.Disposable {
  private view?: vscode.WebviewView;
  private readonly disposables: vscode.Disposable[] = [];

  constructor(
    private readonly provider: () => Promise<PanelState>,
    private readonly handler: (message: PanelMessage) => Promise<void>
  ) {}

  resolveWebviewView(view: vscode.WebviewView): void {
    this.view = view;
    view.webview.options = { enableScripts: true };
    this.disposables.push(
      view.webview.onDidReceiveMessage(async (message: unknown) => {
        if (isPanelMessage(message)) {
          await this.handler(message);
        }
      }),
      view.onDidDispose(() => {
        this.view = undefined;
      })
    );
    void this.render();
  }

  async render(): Promise<void> {
    if (!this.view) {
      return;
    }
    const state = await this.provider();
    this.view.webview.html = renderPanelHtml(this.view.webview, state, { isTab: false });
  }

  async reveal(): Promise<void> {
    await vscode.commands.executeCommand('solv.panel.focus');
  }

  dispose(): void {
    for (const disposable of this.disposables) {
      disposable.dispose();
    }
  }
}
