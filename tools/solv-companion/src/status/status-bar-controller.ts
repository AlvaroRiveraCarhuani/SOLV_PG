import * as vscode from 'vscode';
import { PanelState } from '../views/panel-state';

/**
 * Barra de estado inferior. El alumno ve la materia y la entrega; el docente ve
 * la cátedra, la emisión y la presencia del aula. Los ítems se ocultan en el
 * rol que no corresponde.
 */
export class StatusBarController implements vscode.Disposable {
  private readonly course: vscode.StatusBarItem;
  private readonly action: vscode.StatusBarItem;
  private readonly presence: vscode.StatusBarItem;
  private readonly connection: vscode.StatusBarItem;

  constructor(
    private readonly commands: {
      focus: string;
      submit: string;
      broadcast: string;
      presence: string;
    }
  ) {
    this.course = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
    this.course.command = this.commands.focus;

    this.action = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 99);
    this.presence = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 98);
    this.presence.command = this.commands.presence;

    this.connection = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 97);
    this.connection.command = this.commands.focus;
  }

  update(state: PanelState): void {
    const code = state.course.code || 'SOLV';
    const teacher = state.role === 'teacher';

    this.course.text = teacher
      ? `$(mortar-board) ${code} · Docente`
      : `$(mortar-board) ${code}`;
    this.course.tooltip = teacher
      ? 'SOLV: panel de cátedra'
      : `SOLV: ${state.course.title || 'laboratorio'}`;
    this.course.show();

    if (teacher) {
      this.action.text = '$(broadcast) Emitir avance';
      this.action.tooltip = 'Emitir un avance de código a la clase';
      this.action.command = this.commands.broadcast;
      this.action.show();

      const board = state.teacher;
      this.presence.text = `$(organization) ${board?.present ?? 0} alumnos`;
      this.presence.tooltip = board
        ? `${board.present} presentes de ${board.enrolled} matriculados`
        : 'Sin aula abierta';
      this.presence.show();
    } else {
      this.action.text = '$(cloud-upload) Entregar práctica';
      this.action.tooltip = 'Revisar el código y entregar la práctica';
      this.action.command = this.commands.submit;
      this.action.show();
      this.presence.hide();
    }

    if (state.connection === 'connected') {
      this.connection.text = '$(check) SOLV';
      this.connection.tooltip = 'Conexión con SOLV activa';
    } else if (state.connection === 'connecting') {
      this.connection.text = '$(sync~spin) Conectando…';
      this.connection.tooltip = 'Restableciendo la conexión con SOLV';
    } else {
      this.connection.text = '$(warning) SOLV';
      this.connection.tooltip = 'Sin conexión con SOLV';
    }
    this.connection.show();
  }

  dispose(): void {
    this.course.dispose();
    this.action.dispose();
    this.presence.dispose();
    this.connection.dispose();
  }
}
