import * as vscode from 'vscode';
import { SolvClient } from './client/solv-client';
import { AdvanceClient } from './advances/advance-client';
import { AdvanceContentProvider } from './advances/advance-content-provider';
import { AdvanceController, describeAdvanceError } from './advances/advance-controller';
import { CommentDecorator } from './comments/comment-decorator';
import { CommentsController } from './comments/comments-controller';
import { SolvSettings } from './config/solv-settings';
import {
  CommentView,
  ConnectionState,
  DeliveryState,
  PanelState,
  RubricCriterionView,
  deriveDeliveryState
} from './views/panel-state';
import { AssignmentTab } from './views/assignment-tab';
import { PanelMessage, SolvPanelView } from './views/solv-panel-view';
import { StatusBarController } from './status/status-bar-controller';
import { KeystrokeCapturer } from './telemetry/keystroke-capturer';

/** Sin respuesta durante este lapso, la barra de estado deja de decir "conectando". */
const OFFLINE_AFTER_MS = 120000;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const settings = new SolvSettings(context);
  const client = new SolvClient(settings);
  const decorator = new CommentDecorator();
  const comments = new CommentsController(client, decorator, settings);
  const telemetry = new KeystrokeCapturer(client, settings);
  const contentProvider = new AdvanceContentProvider();
  const advanceClient = new AdvanceClient(client);
  const advances = new AdvanceController(advanceClient, contentProvider, settings, context.workspaceState);

  // Roles aislados: un solo paquete sirve al alumno y al docente. Las context
  // keys deciden qué comandos y vistas quedan disponibles en el editor.
  await vscode.commands.executeCommand('setContext', 'solv:isStudent', settings.role === 'student');
  await vscode.commands.executeCommand(
    'setContext',
    'solv:isTeacher',
    settings.role === 'teacher' || settings.role === 'admin'
  );

  await configureWorkspaceExcludes();

  let connection: ConnectionState = 'connecting';
  let lastOkAt = Date.now();
  let notice: PanelState['message'];
  let progress:
    | {
        delivery: DeliveryState;
        score?: number;
        code?: string;
        courseTitle?: string;
        title?: string;
        exerciseId?: string;
        purpose?: string;
      }
    | undefined;

  let rubricCriteria: RubricCriterionView[] = [];

  const panel = new SolvPanelView(buildState, handleMessage);
  const tab = new AssignmentTab(buildState, handleMessage);
  const status = new StatusBarController({
    focus: 'solv.panel.focus',
    submit: 'solv.submit',
    broadcast: 'solv.broadcastAdvance',
    presence: 'solv.showPresence'
  });

  context.subscriptions.push(
    decorator,
    comments,
    telemetry,
    contentProvider,
    advances,
    panel,
    tab,
    status,
    vscode.window.registerWebviewViewProvider('solv.panel', panel, {
      webviewOptions: { retainContextWhenHidden: true }
    }),
    vscode.commands.registerCommand('solv.openTab', () => tab.open()),
    vscode.commands.registerCommand('solv.openAula', () => startClassSession()),
    vscode.commands.registerCommand('solv.closeAula', () => closeClassSession()),
    vscode.commands.registerCommand('solv.startSession', () => startClassSession()),
    vscode.commands.registerCommand('solv.closeSession', () => closeClassSession()),
    vscode.commands.registerCommand('solv.refreshPanel', () => refreshAll(true)),
    vscode.commands.registerCommand('solv.submit', () => submitPractice()),
    vscode.commands.registerCommand('solv.broadcastAdvance', () => broadcastAdvance()),
    vscode.commands.registerCommand('solv.toggleFreeze', () => toggleFreeze()),
    vscode.commands.registerCommand('solv.showPresence', () => showPresence()),
    vscode.commands.registerCommand('solv.openSpeedGrader', () => openSpeedGrader()),
    vscode.commands.registerCommand('solv.configure', async () => {
      if (await settings.configure()) {
        await refreshAll(true);
      }
    }),
    vscode.commands.registerCommand('solv.refreshComments', async () => {
      await comments.refresh(true);
      await renderAll();
    }),
    vscode.commands.registerCommand('solv.showTelemetryStatus', () => telemetry.showStatus()),
    vscode.workspace.onDidChangeConfiguration(event => {
      if (event.affectsConfiguration('solv')) {
        void refreshAll(false);
      }
    })
  );

  const timer = setInterval(() => void refreshAll(false), settings.advanceIntervalMs);
  context.subscriptions.push({ dispose: () => clearInterval(timer) });

  telemetry.start();
  await refreshAll(false);

  async function buildState(): Promise<PanelState> {
    const snapshot = advances.snapshot();
    const assignmentTitle = settings.assignmentTitle || progress?.title || '';
    const purpose =
      settings.exercisePurpose ||
      progress?.purpose ||
      snapshot.board?.session.purpose ||
      'class';
    const state: PanelState = {
      role: settings.isTeacher ? 'teacher' : 'student',
      connection,
      course: {
        code: settings.subjectCode || progress?.code || '',
        title: settings.subjectTitle || progress?.courseTitle || '',
        teacher: settings.teacherName,
        delivery: progress?.delivery ?? 'draft',
        score: progress?.score,
        maxScore: progress?.score === undefined ? undefined : 100
      },
      assignment: {
        title: assignmentTitle,
        summary: settings.assignmentSummary,
        criteria: settings.assignmentCriteria,
        rubricCriteria: rubricCriteria.length > 0 ? rubricCriteria : undefined,
        totalPoints: rubricCriteria.reduce((sum, item) => sum + item.weight, 0) || undefined,
        purpose
      },
      comments: commentViews(),
      canSubmit: settings.role === 'student'
    };

    if (snapshot.advance) {
      state.advance = {
        id: snapshot.advance.delivery.id,
        title: assignmentTitle ? `Avance de ${assignmentTitle}` : 'Avance de clase',
        createdAt: snapshot.advance.delivery.created_at,
        files: snapshot.advance.files,
        applied: false,
        canRollback: false
      };
    } else if (snapshot.applied) {
      state.advance = {
        id: snapshot.applied.id,
        title: 'Avance aplicado',
        createdAt: snapshot.applied.createdAt,
        files: [],
        applied: true,
        canRollback: true
      };
    }

    if (settings.isTeacher) {
      const board = snapshot.board;
      state.teacher = {
        sessionOpen: board?.session.status === 'open',
        frozen: Boolean(board?.session.frozen_at),
        sessionId: board?.session.id || settings.sessionId,
        endsAt: board?.session.ends_at ?? null,
        present: board?.summary.present ?? 0,
        enrolled: board?.summary.enrolled ?? 0,
        submitted: board?.summary.submitted ?? 0
      };
    }

    if (notice) {
      state.message = notice;
    }
    return state;
  }

  function commentViews(): CommentView[] {
    return comments.currentComments
      .filter(
        comment =>
          comment.file_path !== undefined &&
          (comment.local_line_number ?? comment.line_number) > 0
      )
      .map(comment => ({
        id: comment.id,
        file: comment.file_path ?? '',
        line: comment.local_line_number ?? comment.line_number,
        author: comment.author_name?.trim() || 'Docente',
        text: comment.comment
      }));
  }

  async function handleMessage(message: PanelMessage): Promise<void> {
    switch (message.type) {
      case 'refresh':
        await refreshAll(true);
        return;
      case 'diff':
        try {
          await advances.diff(message.path);
        } catch (error) {
          warn(describeAdvanceError(error));
        }
        return;
      case 'apply':
        await applySelection(message.paths);
        return;
      case 'dismiss':
        await dismissAdvance();
        return;
      case 'rollback':
        await rollbackAdvance();
        return;
      case 'broadcast':
        await broadcastAdvance();
        return;
      case 'freeze':
        await toggleFreeze();
        return;
      case 'submit':
        await submitPractice();
        return;
      case 'speedgrader':
        await openSpeedGrader();
        return;
      case 'open-tab':
        await tab.open();
        return;
      case 'open-aula':
        await startClassSession();
        return;
      case 'close-aula':
        await closeClassSession();
        return;
      case 'open-comment':
        await openComment(message.file, message.line);
        return;
      default:
        return;
    }
  }

  async function applySelection(paths: string[]): Promise<void> {
    if (!paths || paths.length === 0) {
      warn('Marcá al menos un archivo del checklist para aplicar el avance.');
      return;
    }
    try {
      advances.select(paths);
      const result = await advances.applySelection(paths);
      info(`Se aplicaron ${result.filesWritten} archivos del avance.`);
      await refreshAll(false);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function dismissAdvance(): Promise<void> {
    const confirm = await vscode.window.showWarningMessage(
      '¿Descartar el avance del docente? Tu proyecto queda como está.',
      { modal: true },
      'Descartar avance'
    );
    if (confirm !== 'Descartar avance') {
      return;
    }
    try {
      await advances.dismiss();
      info('Avance descartado.');
      await refreshAll(false);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function rollbackAdvance(): Promise<void> {
    const confirm = await vscode.window.showWarningMessage(
      'Se restaurarán los archivos que reemplazó el último avance. Los cambios posteriores sobre esos archivos se pierden.',
      { modal: true },
      'Deshacer avance'
    );
    if (confirm !== 'Deshacer avance') {
      return;
    }
    try {
      const result = await advances.rollback();
      info(`Se restauraron ${result.filesRestored} archivos a la versión anterior.`);
      await refreshAll(false);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function toggleFreeze(): Promise<void> {
    try {
      const frozen = await advances.toggleFreeze();
      info(frozen ? 'Clase congelada: no se emiten avances nuevos.' : 'Clase habilitada para emitir avances.');
      await renderAll();
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function showPresence(): Promise<void> {
    const board = advances.snapshot().board;
    if (!board || board.students.length === 0) {
      warn('No hay un aula abierta con alumnos para mostrar.');
      return;
    }
    const items = board.students.map(student => ({
      label: student.name || student.student_id,
      description: student.present ? 'Presente' : 'Ausente',
      detail: student.student_id
    }));
    await vscode.window.showQuickPick(items, {
      title: `Presencia: ${board.summary.present} de ${board.summary.enrolled}`,
      placeHolder: 'Alumnos matriculados en el aula'
    });
  }

  async function openComment(file: string, line: number): Promise<void> {
    const root = vscode.workspace.workspaceFolders?.[0]?.uri;
    if (!root || !file) {
      warn('No se encontró el archivo del comentario.');
      return;
    }
    const uri = vscode.Uri.joinPath(root, file);
    try {
      const document = await vscode.workspace.openTextDocument(uri);
      const editor = await vscode.window.showTextDocument(document, { preview: false });
      const target = new vscode.Position(Math.max(0, line - 1), 0);
      editor.selection = new vscode.Selection(target, target);
      editor.revealRange(new vscode.Range(target, target), vscode.TextEditorRevealType.InCenter);
    } catch {
      warn(`No se pudo abrir ${file}.`);
    }
  }

  async function openSpeedGrader(): Promise<void> {
    const target = settings.speedGraderUrl || settings.apiBaseUrl;
    if (!target) {
      warn('Falta configurar la URL de la plataforma (solv.apiBaseUrl).');
      return;
    }
    await vscode.env.openExternal(vscode.Uri.parse(target));
  }

  async function broadcastAdvance(): Promise<void> {
    const sessionId = settings.sessionId;
    if (!sessionId) {
      warn('No hay un aula abierta. Abrí la clase desde la plataforma antes de emitir.');
      return;
    }
    const exerciseId = settings.exerciseId;
    if (!exerciseId) {
      warn('Falta el ejercicio de la práctica para emitir el avance.');
      return;
    }

    const title = await vscode.window.showInputBox({
      title: 'Emitir avance a la clase',
      prompt: 'Título del avance que verán los alumnos',
      placeHolder: 'Paso 2: rutas del servicio',
      ignoreFocusOut: true,
      validateInput: value => (value.trim() === '' ? 'El título es obligatorio' : undefined)
    });
    if (title === undefined) {
      return;
    }

    const scope = await vscode.window.showQuickPick(
      [
        { label: 'Toda la clase', description: 'Todos los alumnos del aula', value: 'class' as const },
        { label: 'Seleccionados', description: 'Elegir alumnos del aula', value: 'selected' as const }
      ],
      { title: 'Alcance del avance', placeHolder: 'A quién se emite el avance' }
    );
    if (!scope) {
      return;
    }

    let studentIds: string[] = [];
    if (scope.value === 'selected') {
      const board = advances.snapshot().board ?? (await advanceClient.board(sessionId));
      const picked = await vscode.window.showQuickPick(
        (board?.students ?? []).map(student => ({
          label: student.name || student.student_id,
          description: student.present ? 'Presente' : 'Ausente',
          studentId: student.student_id
        })),
        { title: 'Alumnos destinatarios', canPickMany: true }
      );
      if (!picked || picked.length === 0) {
        warn('Elegí al menos un alumno para emitir el avance.');
        return;
      }
      studentIds = picked.map(item => item.studentId);
    }

    try {
      const emitted = await advanceClient.emit({
        sessionId,
        exerciseId,
        scope: scope.value,
        studentIds
      });
      info(`Avance «${title.trim()}» emitido a ${emitted.recipients.length} alumnos.`);
      await refreshAll(false);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function submitPractice(): Promise<void> {
    const errors = vscode.languages
      .getDiagnostics()
      .filter(([uri]) => uri.scheme === 'file')
      .flatMap(([uri, diagnostics]) =>
        diagnostics
          .filter(item => item.severity === vscode.DiagnosticSeverity.Error)
          .map(item => ({ uri, diagnostic: item }))
      );

    if (errors.length > 0) {
      const choice = await vscode.window.showWarningMessage(
        `Se detectaron ${errors.length} errores de compilación en tu código. ¿Entregar igualmente o preferís revisarlos?`,
        { modal: true },
        'Revisar errores',
        'Entregar igualmente'
      );
      if (choice !== 'Entregar igualmente') {
        if (choice === 'Revisar errores') {
          await revealDiagnostic(errors[0]);
        }
        return;
      }
    }

    const isExam = settings.exercisePurpose === 'exam' || progress?.purpose === 'exam';
    const confirm = await vscode.window.showWarningMessage(
      isExam
        ? '¿Entregar el examen? Se guarda el estado actual del proyecto.'
        : '¿Entregar la práctica? Se guarda el estado actual del proyecto.',
      { modal: true },
      'Entregar'
    );
    if (confirm !== 'Entregar') {
      return;
    }

    const exerciseId = settings.exerciseId;
    if (!exerciseId) {
      warn(isExam ? 'Falta el ejercicio del examen para entregar.' : 'Falta el ejercicio de la práctica para entregar.');
      return;
    }

    try {
      const workspaceId = await resolveWorkspaceId();
      if (!workspaceId) {
        warn(
          isExam
            ? 'No se encontró el laboratorio de este examen. Abrilo desde la plataforma.'
            : 'No se encontró el laboratorio de esta práctica. Abrila desde la plataforma.'
        );
        return;
      }
      await vscode.workspace.saveAll(false);
      await client.submitWorkspace(workspaceId, exerciseId);
      info(
        isExam
          ? 'Examen entregado. La revisión queda disponible en la plataforma.'
          : 'Práctica entregada. La revisión queda disponible en la plataforma.'
      );
      await refreshAll(false);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function resolveWorkspaceId(): Promise<string> {
    if (settings.workspaceId) {
      return settings.workspaceId;
    }
    const resolved = await client.resolveWorkspace(settings.subjectId);
    return resolved?.id ?? '';
  }

  /**
   * Toma materia, práctica y estado de la entrega de la API del alumno. Si la
   * consulta falla, el panel conserva lo configurado por entorno.
   */
  async function loadSubmissionContext(): Promise<void> {
    if (settings.isTeacher) {
      return;
    }
    try {
      const submissions = await client.studentSubmissions();
      const current = settings.submissionId
        ? submissions.find(item => item.id === settings.submissionId)
        : undefined;
      const latest = current ?? submissions[0];
      if (!latest) {
        return;
      }
      progress = {
        delivery: deriveDeliveryState(latest.score),
        score: latest.score ?? undefined,
        code: latest.subject_code,
        courseTitle: latest.subject_name,
        title: latest.exercise_title,
        exerciseId: latest.exercise_id,
        purpose: latest.exercise_purpose
      };
    } catch {
      // El aviso de conexión ya lo informa refreshAll; el panel no se vacía.
    }
  }

  async function loadRubricCriteria(): Promise<void> {
    const exerciseId = settings.exerciseId || progress?.exerciseId;
    if (!exerciseId) {
      return;
    }
    try {
      const rubric = await client.fetchRubric(exerciseId);
      if (rubric && rubric.criteria && rubric.criteria.length > 0) {
        rubricCriteria = rubric.criteria
          .map(c => {
            const name = (c.name || c.title || '').trim();
            const weight = c.weight || c.max_score || 0;
            const description = (c.description || '').trim();
            const levels = (c.levels || []).map(l => ({
              name: l.name,
              score: l.score,
              description: l.description
            }));
            return {
              id: c.id,
              name,
              weight,
              description,
              levels
            };
          })
          .filter(c => c.name.length > 0);
      }
    } catch {
      // Si la API falla, queda vacío y se usa la lista configurada por defecto
    }
  }

  async function sendStudentPresenceHeartbeat(): Promise<void> {
    if (settings.isTeacher) {
      return;
    }
    try {
      const activeSessions = await client.activeStudentSessions();
      if (activeSessions && activeSessions.length > 0) {
        const session = activeSessions[0];
        await client.sendStudentHeartbeat(session.id);
      }
    } catch {
      // El latido en segundo plano no bloquea la interfaz
    }
  }

  async function startClassSession(): Promise<void> {
    const subjectId = settings.subjectId;
    if (!subjectId) {
      warn('No hay una materia configurada para abrir el aula.');
      return;
    }

    const options = [
      { label: '60 minutos', minutes: 60, description: '1 hora de práctica' },
      { label: '90 minutos (Recomendado)', minutes: 90, description: '1 hora y media' },
      { label: '120 minutos', minutes: 120, description: '2 horas de examen o práctica' },
      { label: '180 minutos', minutes: 180, description: '3 horas de laboratorio' },
      { label: 'Personalizado...', minutes: -1, description: 'Ingresar tiempo en minutos' }
    ];

    const pick = await vscode.window.showQuickPick(options, {
      placeHolder: 'Duración de la sesión de clase'
    });
    if (!pick) {
      return;
    }

    let minutes = pick.minutes;
    if (minutes === -1) {
      const input = await vscode.window.showInputBox({
        prompt: 'Duración en minutos (15 a 480)',
        value: '90',
        validateInput: value => {
          const num = Number(value);
          if (isNaN(num) || num < 15 || num > 480) {
            return 'Ingresá un número entre 15 y 480 minutos.';
          }
          return null;
        }
      });
      if (!input) {
        return;
      }
      minutes = Number(input);
    }

    try {
      await advances.openSession(subjectId, minutes);
      info('Clase iniciada. La presencia de los alumnos se actualizará en vivo.');
      await refreshAll(true);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  async function closeClassSession(): Promise<void> {
    const sessionId = settings.sessionId || advances.snapshot().board?.session.id;
    if (!sessionId) {
      warn('No hay una sesión activa para cerrar.');
      return;
    }
    const confirm = await vscode.window.showWarningMessage(
      '¿Cerrar la sesión de clase actual? Los alumnos ya no podrán emitir avances ni asistencia.',
      { modal: true },
      'Cerrar clase'
    );
    if (confirm !== 'Cerrar clase') {
      return;
    }
    try {
      await advances.closeSession(sessionId);
      info('Sesión de clase cerrada.');
      await refreshAll(true);
    } catch (error) {
      warn(describeAdvanceError(error));
    }
  }

  /**
   * Deja fuera del explorador las carpetas que no son código del alumno. Solo
   * agrega claves ausentes: una preferencia explícita del workspace se respeta.
   */
  async function configureWorkspaceExcludes(): Promise<void> {
    if (!vscode.workspace.workspaceFolders || vscode.workspace.workspaceFolders.length === 0) {
      return;
    }
    const configuration = vscode.workspace.getConfiguration('files');
    const current =
      configuration.inspect<Record<string, boolean>>('exclude')?.workspaceValue ?? {};
    const merged: Record<string, boolean> = { ...current };
    let changed = false;
    for (const key of ['**/.git', '**/.cache', '**/.openvscode-server', '**/.solv_backups']) {
      if (!(key in merged)) {
        merged[key] = true;
        changed = true;
      }
    }
    if (!changed) {
      return;
    }
    try {
      await configuration.update('exclude', merged, vscode.ConfigurationTarget.Workspace);
    } catch {
      // Sin permiso de escritura sobre .vscode/settings.json el editor sigue
      // funcionando; solo se ven las carpetas internas.
    }
  }

  async function revealDiagnostic(entry: { uri: vscode.Uri; diagnostic: vscode.Diagnostic }): Promise<void> {
    try {
      const document = await vscode.workspace.openTextDocument(entry.uri);
      const editor = await vscode.window.showTextDocument(document, { preview: false });
      editor.selection = new vscode.Selection(
        entry.diagnostic.range.start,
        entry.diagnostic.range.end
      );
      editor.revealRange(entry.diagnostic.range, vscode.TextEditorRevealType.InCenter);
    } catch {
      warn('No se pudo abrir el archivo con el error.');
    }
  }

  async function refreshAll(showErrors: boolean): Promise<void> {
    try {
      await advances.refresh();
      connection = 'connected';
      lastOkAt = Date.now();
      notice = undefined;
    } catch (error) {
      const persisted = Date.now() - lastOkAt > OFFLINE_AFTER_MS;
      connection = persisted ? 'offline' : 'connecting';
      notice =
        persisted || showErrors
          ? { kind: 'warning', text: describeAdvanceError(error) }
          : undefined;
    }
    await loadSubmissionContext();
    await loadRubricCriteria();
    await sendStudentPresenceHeartbeat();
    await comments.refresh(showErrors);
    await renderAll();
  }

  async function renderAll(): Promise<void> {
    await Promise.all([panel.render(), tab.render()]);
    status.update(await buildState());
  }

  function info(text: string): void {
    void vscode.window.showInformationMessage(`SOLV Companion: ${text}`);
  }

  function warn(text: string): void {
    void vscode.window.showWarningMessage(`SOLV Companion: ${text}`);
  }
}

export function deactivate(): void {
  // Los recursos se liberan por context.subscriptions.
}
