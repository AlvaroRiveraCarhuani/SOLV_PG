import * as vscode from 'vscode';
import { PanelState } from './panel-state';
import { renderPanelHtml } from './panel-html';
import { PanelMessage } from './solv-panel-view';

/**
 * Pestaña completa del panel. Permite leer consignas largas o trabajar en
 * pantalla dividida (Split Editor) con el código a la derecha.
 */
export class AssignmentTab implements vscode.Disposable {
  private panel?: vscode.WebviewPanel;
  private readonly disposables: vscode.Disposable[] = [];

  constructor(
    private readonly provider: () => Promise<PanelState>,
    private readonly handler: (message: PanelMessage) => Promise<void>
  ) {}

  async open(): Promise<void> {
    const targetColumn = vscode.ViewColumn.Active;
    if (this.panel) {
      this.panel.reveal(targetColumn);
      await this.render();
      return;
    }

    this.panel = vscode.window.createWebviewPanel(
      'solv.assignment',
      'SOLV — Consigna',
      { viewColumn: targetColumn, preserveFocus: false },
      { enableScripts: true, retainContextWhenHidden: true }
    );

    this.disposables.push(
      this.panel.webview.onDidReceiveMessage(async (message: unknown) => {
        if (typeof message === 'object' && message !== null) {
          const type = (message as { type?: unknown }).type;
          if (typeof type === 'string') {
            await this.handler(message as PanelMessage);
          }
        }
      }),
      this.panel.onDidDispose(() => {
        this.panel = undefined;
      })
    );

    await this.render();
  }

  async render(): Promise<void> {
    if (!this.panel) {
      return;
    }
    const state = await this.provider();
    this.panel.webview.html = renderPanelHtml(this.panel.webview, state);
  }

  dispose(): void {
    this.panel?.dispose();
    for (const disposable of this.disposables) {
      disposable.dispose();
    }
  }
}
