import * as vscode from 'vscode';

/** Esquema virtual del avance: el diff de VS Code lo lee sin tocar el disco. */
export const ADVANCE_SCHEME = 'solv-advance';

/**
 * Sirve el contenido emitido por el docente a `vscode.diff` como documento de
 * solo lectura. No se crean archivos temporales ni se escribe en el workspace.
 */
export class AdvanceContentProvider
  implements vscode.TextDocumentContentProvider, vscode.Disposable
{
  private readonly contents = new Map<string, string>();
  private readonly emitter = new vscode.EventEmitter<vscode.Uri>();
  private readonly registration: vscode.Disposable;

  readonly onDidChange = this.emitter.event;

  constructor() {
    this.registration = vscode.workspace.registerTextDocumentContentProvider(
      ADVANCE_SCHEME,
      this
    );
  }

  /** URI virtual del archivo del avance, con su ruta relativa. */
  static uriFor(advanceId: string, relativePath: string): vscode.Uri {
    const path = relativePath.replace(/\\/g, '/').replace(/^\.?\//, '');
    return vscode.Uri.from({
      scheme: ADVANCE_SCHEME,
      path: `/advances/${advanceId}/${path}`
    });
  }

  /** Publica el contenido y avisa a los editores que ya lo están mostrando. */
  publish(uri: vscode.Uri, content: string): void {
    const key = uri.toString();
    if (this.contents.get(key) === content) {
      return;
    }
    this.contents.set(key, content);
    this.emitter.fire(uri);
  }

  has(uri: vscode.Uri): boolean {
    return this.contents.has(uri.toString());
  }

  provideTextDocumentContent(uri: vscode.Uri): string {
    return this.contents.get(uri.toString()) ?? '';
  }

  clear(): void {
    for (const key of this.contents.keys()) {
      this.contents.delete(key);
      this.emitter.fire(vscode.Uri.parse(key));
    }
  }

  dispose(): void {
    this.registration.dispose();
    this.emitter.dispose();
    this.contents.clear();
  }
}
