import * as vscode from 'vscode';

const TOKEN_SECRET_KEY = 'solv.authToken';

export type SolvRole = 'student' | 'teacher' | 'admin';

function extractRoleFromJwt(token?: string): SolvRole | undefined {
  if (!token) return undefined;
  try {
    const parts = token.split('.');
    if (parts.length < 2) return undefined;
    const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString('utf-8'));
    if (payload.role === 'teacher' || payload.role === 'admin' || payload.role === 'student') {
      return payload.role;
    }
  } catch {
    // ignorar token con formato inválido
  }
  return undefined;
}

/**
 * Lee primero las variables de entorno del sistema, luego la configuración del
 * workspace. Esto permite que la extensión funcione sin configuración manual
 * cuando se ejecuta dentro de OpenVSCode Server con variables inyectadas.
 */
export class SolvSettings {
  constructor(private readonly context: vscode.ExtensionContext) {
    this._envApiBaseUrl = process.env.SOLV_API_URL;
    this._envSubmissionId = process.env.SOLV_SUBMISSION_ID;
    this._envToken = process.env.SOLV_STUDENT_TOKEN;
    this._envRole = process.env.SOLV_USER_ROLE;
    this._envSessionId = process.env.SOLV_SESSION_ID;
    this._envExerciseId = process.env.SOLV_EXERCISE_ID;
    this._envSubjectId = process.env.SOLV_SUBJECT_ID;
    this._envWorkspaceId = process.env.SOLV_WORKSPACE_ID;
    this._envSubjectCode = process.env.SOLV_SUBJECT_CODE;
    this._envSubjectTitle = process.env.SOLV_SUBJECT_TITLE;
    this._envTeacherName = process.env.SOLV_TEACHER_NAME;
    this._envAssignmentTitle = process.env.SOLV_ASSIGNMENT_TITLE;
    this._envAssignmentSummary = process.env.SOLV_ASSIGNMENT_SUMMARY;
    this._envExercisePurpose = process.env.SOLV_EXERCISE_PURPOSE;
    this._envSpeedGraderUrl = process.env.SOLV_SPEEDGRADER_URL;
  }

  private readonly _envApiBaseUrl: string | undefined;
  private readonly _envSubmissionId: string | undefined;
  private readonly _envToken: string | undefined;
  private readonly _envRole: string | undefined;
  private _envSessionId: string | undefined;
  private readonly _envExerciseId: string | undefined;
  private readonly _envSubjectId: string | undefined;
  private readonly _envWorkspaceId: string | undefined;
  private readonly _envSubjectCode: string | undefined;
  private readonly _envSubjectTitle: string | undefined;
  private readonly _envTeacherName: string | undefined;
  private readonly _envAssignmentTitle: string | undefined;
  private readonly _envAssignmentSummary: string | undefined;
  private readonly _envExercisePurpose: string | undefined;
  private readonly _envSpeedGraderUrl: string | undefined;

  get apiBaseUrl(): string {
    return this._envApiBaseUrl?.replace(/\/+$/, '') || this.readString('apiBaseUrl').replace(/\/+$/, '');
  }

  get submissionId(): string {
    return this._envSubmissionId || this.readString('submissionId');
  }

  /** Rol del entorno: decide las context keys y qué controles se muestran. */
  get role(): SolvRole {
    const value = (
      this._envRole ||
      extractRoleFromJwt(this._envToken) ||
      this.readString('role') ||
      'student'
    ).toLowerCase();
    if (value === 'teacher' || value === 'admin') {
      return value;
    }
    return 'student';
  }

  get isTeacher(): boolean {
    return this.role === 'teacher' || this.role === 'admin';
  }

  get sessionId(): string {
    return this._envSessionId || this.readString('sessionId');
  }

  setSessionId(sessionId: string): void {
    this._envSessionId = sessionId;
  }

  get exerciseId(): string {
    return this._envExerciseId || this.readString('exerciseId');
  }

  get subjectId(): string {
    return this._envSubjectId || this.readString('subjectId');
  }

  get workspaceId(): string {
    return this._envWorkspaceId || this.readString('workspaceId');
  }

  get subjectCode(): string {
    return this._envSubjectCode || this.readString('subjectCode');
  }

  get subjectTitle(): string {
    return this._envSubjectTitle || this.readString('subjectTitle');
  }

  get teacherName(): string {
    return this._envTeacherName || this.readString('teacherName');
  }

  get assignmentTitle(): string {
    return this._envAssignmentTitle || this.readString('assignmentTitle');
  }

  get assignmentSummary(): string {
    return this._envAssignmentSummary || this.readString('assignmentSummary');
  }

  get exercisePurpose(): 'class' | 'exam' | undefined {
    const value = (this._envExercisePurpose || this.readString('exercisePurpose')).toLowerCase().trim();
    if (value === 'exam' || value === 'class') {
      return value;
    }
    return undefined;
  }

  get assignmentCriteria(): string[] {
    const fromConfig = vscode.workspace.getConfiguration('solv').get<string[]>('assignmentCriteria', []);
    return fromConfig.map(item => item.trim()).filter(item => item !== '');
  }

  get speedGraderUrl(): string {
    return this._envSpeedGraderUrl || this.readString('speedGraderUrl');
  }

  get refreshIntervalMs(): number {
    const seconds = vscode.workspace
      .getConfiguration('solv')
      .get<number>('comments.refreshSeconds', 60);
    return Math.max(15, Number.isFinite(seconds) ? seconds : 60) * 1000;
  }

  get advanceIntervalMs(): number {
    const seconds = vscode.workspace
      .getConfiguration('solv')
      .get<number>('advances.refreshSeconds', 30);
    return Math.max(15, Number.isFinite(seconds) ? seconds : 30) * 1000;
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

  private readString(key: string): string {
    return vscode.workspace.getConfiguration('solv').get<string>(key, '').trim();
  }

  get pasteThreshold(): number {
    const threshold = vscode.workspace
      .getConfiguration('solv')
      .get<number>('telemetry.pasteThreshold', 50);
    return Math.max(2, Number.isFinite(threshold) ? threshold : 50);
  }

  get isConfigured(): boolean {
    return this.apiBaseUrl !== '' && (this.submissionId !== '' || this.sessionId !== '');
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
