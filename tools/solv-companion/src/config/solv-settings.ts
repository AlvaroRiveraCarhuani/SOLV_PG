import * as vscode from 'vscode';

const TOKEN_SECRET_KEY = 'solv.authToken';

/**
 * Lee primero las variables de entorno del sistema, luego la configuración del workspace.
 * Esto permite que la extensión funcione sin configuración manual cuando se ejecuta
 * dentro de OpenVSCode Server con variables de entorno inyectadas.
 */
export class SolvSettings {
  constructor(private readonly context: vscode.ExtensionContext) {
    // Cargar desde variables de entorno si existen (para OpenVSCode Server)
    this._envApiBaseUrl = process.env.SOLV_API_URL;
    this._envSubmissionId = process.env.SOLV_SUBMISSION_ID;
    this._envToken = process.env.SOLV_STUDENT_TOKEN;
  }

  private readonly _envApiBaseUrl: string | undefined;
  private readonly _envSubmissionId: string | undefined;
  private readonly _envToken: string | undefined;

  get apiBaseUrl(): string {
    return this._envApiBaseUrl?.replace(/\/+$/, '') || this.readString('apiBaseUrl').replace(/\/+$/, '');
  }

  get submissionId(): string {
    return this._envSubmissionId || this.readString('submissionId');
  }

  get refreshIntervalMs(): number {
    const seconds = vscode.workspace
      .getConfiguration('solv')
      .get<number>('comments.refreshSeconds', 60);
    return Math.max(15, Number.isFinite(seconds) ? seconds : 60) * 1000;
  }

  get telemetryEnabled(): boolean {
    return vscode.workspace.getConfiguration('solv').get<boolean>('telemetry.enabled', true);
  }

  get tokenFromEnv(): string | undefined {
    return this._envToken;
  }

  /**
   * Lee el token: primero desde variables de entorno (para OpenVSCode Server),
   * luego desde SecretStorage.
   */
  async getTokenEnv(): Promise<string> {
    if (this._envToken) {
      return this._envToken;
    }
    return (await this.context.secrets.get(TOKEN_SECRET_KEY)) ?? '';
  }

  private readString(key: string | 'apiBaseUrl' | 'submissionId'): string {
    return vscode.workspace.getConfiguration('solv').get<string>(key, '').trim();
  }

  get pasteThreshold(): number {
    const threshold = vscode.workspace
      .getConfiguration('solv')
      .get<number>('telemetry.pasteThreshold', 50);
    return Math.max(2, Number.isFinite(threshold) ? threshold : 50);
  }

  get isConfigured(): boolean {
    return this.apiBaseUrl !== '' && this.submissionId !== '';
  }

  async getToken(): Promise<string> {
    return (await this.context.secrets.get(TOKEN_SECRET_KEY)) ?? '';
  }

  /**
   * Pide URL, entrega y token. Dejar el token vacío conserva el guardado.
   * Devuelve false si la persona cancela o si falta un dato obligatorio.
   */
  async configure(): Promise<boolean> {
    const baseUrl = await vscode.window.showInputBox({
      title: 'SOLV Companion: URL de la API',
      prompt: 'URL base sin barra final, por ejemplo https://solv.example.edu',
      value: this.apiBaseUrl,
      ignoreFocusOut: true,
      validateInput: candidate => this.validateBaseUrl(candidate)
    });
    if (baseUrl === undefined) {
      return false;
    }

    const submissionId = await vscode.window.showInputBox({
      title: 'SOLV Companion: entrega',
      prompt: 'Identificador de la entrega que se está revisando',
      value: this.submissionId,
      ignoreFocusOut: true,
      validateInput: candidate => (candidate.trim() === '' ? 'La entrega es obligatoria' : undefined)
    });
    if (submissionId === undefined) {
      return false;
    }

    const currentToken = await this.getToken();
    const token = await vscode.window.showInputBox({
      title: 'SOLV Companion: token de sesión',
      prompt: currentToken
        ? 'Pegá el token nuevo o dejalo vacío para conservar el actual'
        : 'Pegá el token de sesión de SOLV',
      password: true,
      ignoreFocusOut: true,
      validateInput: candidate =>
        candidate.trim() === '' && currentToken === '' ? 'El token es obligatorio' : undefined
    });
    if (token === undefined) {
      return false;
    }

    await vscode.workspace
      .getConfiguration('solv')
      .update('apiBaseUrl', baseUrl.trim().replace(/\/+$/, ''), vscode.ConfigurationTarget.Workspace);
    await vscode.workspace
      .getConfiguration('solv')
      .update('submissionId', submissionId.trim(), vscode.ConfigurationTarget.Workspace);
    if (token.trim() !== '') {
      await this.context.secrets.store(TOKEN_SECRET_KEY, token.trim());
    }

    void vscode.window.showInformationMessage('SOLV Companion: conexión guardada.');
    return true;
  }

  private validateBaseUrl(candidate: string): string | undefined {
    const value = candidate.trim();
    if (value === '') {
      return 'La URL de la API es obligatoria';
    }
    try {
      const parsed = new URL(value);
      if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
        return 'Usá http o https';
      }
    } catch {
      return 'La URL no es válida';
    }
    return undefined;
  }
}
