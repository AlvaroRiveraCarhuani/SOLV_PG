import { execFileSync } from 'node:child_process';
import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import {
  createCourseViaAdminUI,
  enrollStudentViaAPI,
  typeInCodeMirror
} from '../../fixtures/authoring.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';
import { apiJson, openCurricularMap } from '../../fixtures/api.fixture.mjs';

test.describe.configure({ mode: 'serial' });

/**
 * J08 — Ciclo de vida completo de IDE Persistente (100% UI + Docker real):
 *
 * 1. Admin crea materia por UI, la asigna al docente titular, matricula al
 *    estudiante y homologa en Template Studio una plantilla con
 *    target_environment = 'IDE_PERSISTENTE' y base de datos PostgreSQL.
 * 2. Docente crea un ejercicio en modalidad IDE Persistente ('workspace' /
 *    'project') desde el editor en 3 pasos, selecciona la plantilla con BD,
 *    comprueba que <ide-db-init-script> es visible, escribe un db_init_script
 *    DDL/DML real, configura la rúbrica con suma de pesos = 100% y publica.
 * 3. Estudiante entra a /student/exercises/:id, verifica la vista de IDE
 *    Persistente (sin invocar /api/v1/evaluations), inicia el contenedor real
 *    (running + http://localhost:<port>), escribe los archivos del proyecto en
 *    el volumen del IDE y los entrega desde la UI
 *    (POST /api/v1/environments/{id}/submit -> PENDING).
 * 4. Estudiante pulsa Reconectar sobre el workspace activo y comprueba que la
 *    reactivación/reconexión es idempotente (_solv_exercise_migrations) y no
 *    falla ni duplica el db_init_script.
 * 5. Docente abre la entrega en SpeedGrader, verifica el árbol de archivos del
 *    snapshot, cambia de archivo, comprueba que no hay bloques del juez, evalúa
 *    con la rúbrica y un comentario general
 *    (POST /api/v1/submissions/{id}/rubric-evaluation) y comprueba que la nota
 *    ponderada y el comentario con author_id del docente quedan persistidos.
 */
test.describe('J08 — Ciclo de vida completo de IDE Persistente, BD satélite y Rúbrica en SpeedGrader', () => {
  const runId = Date.now().toString().slice(-6);
  const templateName = `IDE Python + Postgres #${runId}`;
  const courseName = `Taller de Proyectos IDE #${runId}`;
  const courseCode = `IDE-${runId}`;
  const exerciseTitle = `Microservicio con PostgreSQL #${runId}`;

  const dbInitScript = [
    'CREATE TABLE items (id INT PRIMARY KEY, name TEXT NOT NULL);',
    "INSERT INTO items (id, name) VALUES (1, 'ok');"
  ].join('\n');

  const mainPyContent = [
    '# Solución principal del proyecto IDE Persistente',
    'def fetch_items():',
    '    return [{"id": 1, "name": "ok"}]',
    '',
    'if __name__ == "__main__":',
    '    print(fetch_items())',
    ''
  ].join('\n');

  const helperPyContent = [
    '# Módulo auxiliar sincronizado en el volumen del workspace',
    'SCHEMA_VERSION = 1',
    ''
  ].join('\n');

  const generalRubricComment = `Excelente modelado SQL y estructura modular del proyecto (#${runId}).`;
  const blockComment = `El bloque de acceso a items debe validar la conexión antes de consultar (#${runId}).`;

  // El alumno edita dentro de OpenVSCode; la prueba deposita los archivos en el
  // mismo volumen del workspace porque manejar el editor web completo está
  // fuera del alcance del flujo. La entrega sí se dispara desde la UI.
  function writeWorkspaceFile(workspaceId, relativePath, content) {
    const container = `solv-environment-${workspaceId}`;
    const absolutePath = `/home/workspace/${relativePath}`;
    const directory = absolutePath.slice(0, absolutePath.lastIndexOf('/'));
    execFileSync('docker', ['exec', container, 'mkdir', '-p', directory]);
    execFileSync('docker', ['exec', '-i', container, 'sh', '-c', `cat > '${absolutePath}'`], {
      input: content
    });
  }

  const sharedState = {
    courseId: '',
    templateId: '',
    exerciseId: '',
    workspaceId: '',
    accessUrl: '',
    submissionId: ''
  };

  test('Admin homologa plantilla IDE+PG, Docente publica proyecto con db_init_script y rúbrica, Estudiante entrega desde contenedor real y Docente califica en SpeedGrader', async ({ browser }, testInfo) => {
    test.setTimeout(240000);
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const { page: adminPage } = adminSession;
    forensic.attachToPage(adminPage, 'admin');

    let teacherSession = null;
    let studentSession = null;
    let evaluationRequestsCount = 0;

    try {
      await test.step('Paso 1 (Admin UI): Homologa plantilla IDE_PERSISTENTE con PostgreSQL y crea la materia', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin abre Template Studio y configura plantilla IDE_PERSISTENTE con PostgreSQL');

        await adminPage.goto('/admin/plantillas/nueva', { waitUntil: 'domcontentloaded' });
        await expect(adminPage.locator('h1.page-title')).toContainText('Registrar Nueva Plantilla Oficial', {
          timeout: 45000
        });

        const idePurposeCard = adminPage.locator('.purpose-option-card', {
          hasText: 'Entorno de Desarrollo (Con Editor VS Code)'
        });
        await expect(idePurposeCard).toBeVisible({ timeout: 45000 });
        await idePurposeCard.click();

        const pythonCard = adminPage.locator('.tech-component-card', { hasText: /Python/i }).first();
        await expect(pythonCard).toBeVisible({ timeout: 45000 });
        await pythonCard.click();

        const postgresCard = adminPage.locator('.tech-component-card', { hasText: /PostgreSQL/i }).first();
        await expect(postgresCard).toBeVisible({ timeout: 45000 });
        await postgresCard.click();

        await adminPage.locator('#studio-tpl-name').fill(templateName);

        const sidebar = adminPage.locator('studio-summary-sidebar');
        await expect(sidebar).toContainText(templateName);

        const verifyBtn = sidebar.locator('button', { hasText: 'Comprobar funcionamiento' });
        await expect(verifyBtn).toBeEnabled({ timeout: 45000 });
        await verifyBtn.click();
        await expect(sidebar.locator('.verification-status-box')).toContainText('Entorno Verificado', {
          timeout: 90000
        });

        const publishBtn = sidebar.locator('button', { hasText: 'Publicar Plantilla Oficial' });
        await expect(publishBtn).toBeEnabled({ timeout: 45000 });
        const [createTplResp] = await Promise.all([
          adminPage.waitForResponse(
            (res) => res.url().includes('/api/v1/admin/templates') && res.request().method() === 'POST',
            { timeout: 60000 }
          ),
          publishBtn.click()
        ]);
        expect(createTplResp.ok()).toBeTruthy();
        const tplBody = await createTplResp.json().catch(() => ({}));
        sharedState.templateId = tplBody?.data?.id || tplBody?.id || '';
        await expect(adminPage).toHaveURL(/\/admin\/plantillas/, { timeout: 45000 });

        forensic.recordStep('Admin crea materia en UI, asigna al docente y matricula al estudiante');
        sharedState.courseId = await createCourseViaAdminUI(adminPage, {
          courseName,
          courseCode,
          teacherEmail: CANONICAL_ACTORS.teacher.email
        });
        expect(sharedState.courseId).toBeTruthy();

        await enrollStudentViaAPI(adminPage, sharedState.courseId, CANONICAL_ACTORS.student.id);
      });

      teacherSession = await createAuthenticatedRoleSession(browser, 'teacher', baseURL);
      const { page: teacherPage } = teacherSession;
      forensic.attachToPage(teacherPage, 'teacher');

      await test.step('Paso 2 (Docente UI): Crea ejercicio IDE_PERSISTENTE con plantilla+BD, db_init_script y rúbrica al 100%', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente abre el editor y selecciona la modalidad Entorno de Desarrollo (IDE_PERSISTENTE)');

        await teacherPage.goto(`/teacher/cursos/${sharedState.courseId}`, { waitUntil: 'domcontentloaded' });
        await expect(teacherPage.locator('h1.page-title')).toContainText(courseName, { timeout: 45000 });

        const newLabBtn = teacherPage.locator('button.btn-new-lab', { hasText: 'Nuevo Laboratorio' });
        await expect(newLabBtn).toBeVisible({ timeout: 45000 });
        await newLabBtn.click();
        await expect(teacherPage).toHaveURL(new RegExp(`/teacher/courses/${sharedState.courseId}/exercises/new`));

        // Elegir modalidad Entorno de Desarrollo (IDE_PERSISTENTE)
        const workspaceCard = teacherPage.locator('modality-purpose-cards button.segment-btn', {
          hasText: 'Entorno de Desarrollo'
        });
        await expect(workspaceCard).toBeVisible({ timeout: 45000 });
        await workspaceCard.click();
        await expect(workspaceCard).toHaveAttribute('aria-checked', 'true');

        await teacherPage.locator('#exercise-title').fill(exerciseTitle);

        await teacherPage
          .locator('#statement-description')
          .fill('Implementa el acceso a la tabla `items` aprovisionada en la base de datos satélite y entrega los módulos del proyecto.');

        // Entorno IDE Persistente
        const ideStep = teacherPage.locator('step-ide-environment');
        await expect(ideStep).toBeVisible({ timeout: 45000 });

        const templateComboInput = ideStep.locator('#template-section input[role="combobox"]');
        await expect(templateComboInput).toBeVisible({ timeout: 45000 });
        await templateComboInput.click();
        await templateComboInput.fill(templateName);

        const tplOption = ideStep.locator('#template-section [role="option"]', { hasText: templateName }).first();
        await expect(tplOption).toBeVisible({ timeout: 45000 });
        await tplOption.click();

        // Verificar que <ide-db-init-script> muestra el editor SQL al detectar PostgreSQL en la plantilla
        const dbScriptSection = ideStep.locator('ide-db-init-script section.card-section');
        await expect(dbScriptSection).toBeVisible({ timeout: 45000 });
        await expect(dbScriptSection.locator('h3.card-title')).toContainText(
          'Script inicial de base de datos'
        );

        const dbScriptTextarea = dbScriptSection.locator('textarea.form-textarea');
        await dbScriptTextarea.fill(dbInitScript);

        // Configurar la rúbrica con suma de pesos = 100%
        const rubricEditor = ideStep.locator('ide-rubric-editor');
        await rubricEditor.locator('button', { hasText: 'Agregar primer criterio' }).click();
        await rubricEditor
          .locator('.criterion-input-title')
          .fill('Acceso correcto a la tabla items');
        await rubricEditor.locator('.criterion-weight-input').fill('100');

        const weightBanner = rubricEditor.locator('.rubric-balance-badge');
        await expect(weightBanner).toHaveClass(/valid/, { timeout: 45000 });
        await expect(weightBanner).toContainText('100%');

        // Publicación (persiste ejercicio + PUT /rubric + POST /publish)
        const publishStepBtn = teacherPage.locator('button.btn-publish-step', { hasText: 'Publicar' });
        await expect(publishStepBtn).toBeEnabled({ timeout: 45000 });
        await publishStepBtn.click();

        const confirmPublishBtn = teacherPage.locator('confirm-modal button', { hasText: 'Publicar ahora' });
        await expect(confirmPublishBtn).toBeVisible({ timeout: 45000 });

        const rubricPutPromise = teacherPage.waitForResponse(
          (r) => r.url().includes('/rubric') && r.request().method() === 'PUT',
          { timeout: 45000 }
        );
        const publishPostPromise = teacherPage.waitForResponse(
          (r) => r.url().includes('/publish') && r.request().method() === 'POST',
          { timeout: 45000 }
        );

        await confirmPublishBtn.click();
        const [rubricPutResp, publishResp] = await Promise.all([rubricPutPromise, publishPostPromise]);
        expect(rubricPutResp.ok()).toBeTruthy();
        expect(publishResp.ok()).toBeTruthy();

        const matchExId = publishResp.url().match(/\/api\/v1\/exercises\/([^/]+)\/publish/);
        sharedState.exerciseId = matchExId ? matchExId[1] : '';
        expect(sharedState.exerciseId).toBeTruthy();
        await expect(teacherPage.locator('.published-pill')).toContainText('Publicada', { timeout: 45000 });
      });

      studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);
      const { page: studentPage } = studentSession;
      forensic.attachToPage(studentPage, 'student');
      studentPage.on('request', (req) => {
        if (req.url().includes('/api/v1/evaluations')) {
          evaluationRequestsCount += 1;
        }
      });

      await test.step('Paso 3 (Estudiante UI): Inicia contenedor real en /student/exercises/:id, edita archivos y entrega el proyecto', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre el ejercicio IDE Persistente desde el mapa curricular');

        await openCurricularMap(studentPage, sharedState.courseId, {
          isReady: async () =>
            await studentPage
              .locator('curricular-map .exercise-row', { hasText: exerciseTitle })
              .first()
              .isVisible()
              .catch(() => false)
        });

        await studentPage
          .locator('curricular-map .exercise-row', { hasText: exerciseTitle })
          .first()
          .locator('a.btn-solve')
          .click();

        await expect(studentPage).toHaveURL(new RegExp(`/student/exercises/${sharedState.exerciseId}`));
        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(exerciseTitle, { timeout: 45000 });

        // Verificar que renderiza la vista de IDE Persistente (isIdeExercise) y no el botón del Juez Efímero
        await expect(solver.locator('h2.section-title', { hasText: 'Proyecto en Entorno de Desarrollo' })).toBeVisible({
          timeout: 45000
        });
        await expect(solver.locator('[data-testid="btn-submit-solution"]')).toHaveCount(0);

        const startIdeBtn = solver.locator('[data-testid="btn-start-ide"]');
        await expect(startIdeBtn).toBeVisible({ timeout: 45000 });
        await expect(startIdeBtn).toContainText('Abrir entorno');

        const startWsResponse = studentPage.waitForResponse(
          (res) => res.url().includes('/api/v1/environments/start') && res.request().method() === 'POST',
          { timeout: 120000 }
        );
        await startIdeBtn.click();

        const startRes = await startWsResponse;
        expect(startRes.ok()).toBeTruthy();
        const startBody = await startRes.json();
        const wsData = startBody?.data ?? startBody;
        sharedState.workspaceId = wsData.environment_id || wsData.id || wsData.workspace_id || '';
        sharedState.accessUrl = wsData.access_url || '';

        expect(sharedState.workspaceId).toBeTruthy();
        expect(wsData.status).toBe('running');
        expect(sharedState.accessUrl).toMatch(/^http:\/\/localhost:\d+$/);

        await expect(solver.locator('.ide-ready')).toContainText('Entorno listo para trabajar.', {
          timeout: 45000
        });

        // El alumno trabaja en el IDE: los dos archivos quedan en el volumen real del workspace.
        writeWorkspaceFile(sharedState.workspaceId, 'main.py', mainPyContent);
        writeWorkspaceFile(sharedState.workspaceId, 'helper.py', helperPyContent);

        // Entregar Proyecto (POST /api/v1/environments/{id}/submit)
        const submitBtn = solver.locator('[data-testid="btn-submit-project"]');
        await expect(submitBtn).toBeEnabled({ timeout: 45000 });

        const submitResponsePromise = studentPage.waitForResponse(
          (res) =>
            res.url().includes(`/api/v1/environments/${sharedState.workspaceId}/submit`) &&
            res.request().method() === 'POST',
          { timeout: 120000 }
        );
        await submitBtn.click();

        const submitRes = await submitResponsePromise;
        expect(submitRes.status()).toBe(201);
        const submitBody = await submitRes.json();
        const subData = submitBody?.data ?? submitBody;
        sharedState.submissionId = subData.submission_id || '';
        expect(sharedState.submissionId).toBeTruthy();
        expect(subData.verdict).toBe('');

        await expect(solver.locator('.submission-saved-banner')).toContainText(sharedState.submissionId, {
          timeout: 45000
        });
        expect(evaluationRequestsCount).toBe(0);
      });

      await test.step('Paso 4 (Idempotencia BD satélite): Reconectar el workspace no duplica ni falla el db_init_script', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante reconecta el workspace activo para comprobar idempotencia de _solv_exercise_migrations');

        const solver = studentPage.locator('student-exercise-solver');
        const reconnectBtn = solver.locator('[data-testid="btn-start-ide"]');
        await expect(reconnectBtn).toContainText('Abrir entorno');

        const reconnectPromise = studentPage.waitForResponse(
          (res) => res.url().includes('/api/v1/environments/start') && res.request().method() === 'POST',
          { timeout: 60000 }
        );
        await reconnectBtn.click();

        const reconnectRes = await reconnectPromise;
        const reconnectRaw = await reconnectRes.text();
        expect(reconnectRes.ok(), reconnectRaw).toBeTruthy();
        const reconnectBody = JSON.parse(reconnectRaw);
        const reconnectData = reconnectBody?.data ?? reconnectBody;
        expect(reconnectData.environment_id || reconnectData.id || reconnectData.workspace_id).toBe(
          sharedState.workspaceId
        );
        expect(reconnectData.status).toBe('running');
        await expect(solver.locator('.ide-error')).toHaveCount(0);
      });

      await test.step('Paso 5 (Docente UI — SpeedGrader): Verifica el árbol de archivos del snapshot y califica con rúbrica', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente abre la entrega en SpeedGrader, revisa el árbol de archivos y evalúa con rúbrica');

        await teacherPage.goto(`/teacher/revision/${sharedState.submissionId}`, { waitUntil: 'domcontentloaded' });
        await expect(teacherPage.locator('speed-grader-header')).toBeVisible({ timeout: 45000 });
        await expect(teacherPage.locator('speed-grader-header')).toContainText('Entorno de Desarrollo');

        const codeViewer = teacherPage.locator('speed-grader-code-viewer');
        await expect(codeViewer).toBeVisible({ timeout: 45000 });

        // IDE: el snapshot se separa en un árbol; no se muestran los bloques del juez.
        const fileTree = teacherPage.locator('review-file-tree');
        await expect(fileTree).toBeVisible({ timeout: 45000 });
        await expect(fileTree.locator('.node-name', { hasText: 'helper.py' })).toBeVisible();
        await expect(fileTree.locator('.node-name', { hasText: 'main.py' })).toBeVisible();
        await expect(teacherPage.locator('speed-grader-eval-panel .verdict-card')).toHaveCount(0);
        await expect(teacherPage.locator('speed-grader-header button', { hasText: 'Re-ejecutar' })).toHaveCount(0);

        // El árbol abre el primer archivo (helper.py, orden alfabético) y permite cambiar a main.py.
        await expect(codeViewer.locator('code.file-path')).toHaveText('helper.py', { timeout: 45000 });
        await expect(codeViewer.locator('.cm-content')).toContainText('SCHEMA_VERSION = 1');

        await fileTree.locator('[role="treeitem"]', { hasText: 'main.py' }).click();
        await expect(codeViewer.locator('code.file-path')).toHaveText('main.py');
        await expect(codeViewer.locator('.cm-content')).toContainText('def fetch_items():');

        // El ejercicio se creó sin proyecto inicial, así que la entrega queda
        // sin base publicada (no_base) y el diff por archivo no se ofrece.
        const toolsButton = teacherPage.locator('speed-grader-header button', { hasText: 'Herramientas' });
        await toolsButton.click();
        await expect(
          teacherPage.locator('speed-grader-header .dropdown-item', { hasText: 'Comparar con Proyecto Base' })
        ).toHaveCount(0);
        await toolsButton.click();
        await expect(codeViewer.locator('.cm-content')).toContainText('def fetch_items():');

        // Selección por arrastre de las líneas locales 2 a 5 del archivo main.py.
        // La línea 4 está vacía: el bloque la incluye al terminar dentro de la 5.
        const codeLines = codeViewer.locator('.cm-line');
        const startBox = await codeLines.nth(1).boundingBox();
        const endBox = await codeLines.nth(4).boundingBox();
        expect(startBox).toBeTruthy();
        expect(endBox).toBeTruthy();
        await teacherPage.mouse.move(startBox.x + 4, startBox.y + startBox.height / 2);
        await teacherPage.mouse.down();
        await teacherPage.mouse.move(endBox.x + Math.min(60, endBox.width - 2), endBox.y + endBox.height / 2, {
          steps: 12
        });
        await teacherPage.mouse.up();

        await expect(codeViewer.locator('.comment-target-badge')).toHaveText('Comentario pedagógico en L2–L5', {
          timeout: 45000
        });

        const blockCommentRequest = teacherPage.waitForRequest(
          (req) =>
            req.url().includes(`/api/v1/teacher/submissions/${sharedState.submissionId}/comments`) &&
            req.method() === 'POST',
          { timeout: 45000 }
        );
        await codeViewer.locator('textarea.form-textarea').fill(blockComment);
        await codeViewer.locator('button.btn-submit-sm').click();

        const blockRequest = await blockCommentRequest;
        const blockPayload = JSON.parse(blockRequest.postData() || '{}');
        expect(blockPayload.line_number).toBeGreaterThanOrEqual(1);
        expect(blockPayload.end_line_number).toBe(blockPayload.line_number + 3);
        await expect(codeViewer.locator('.comment-line-badge', { hasText: 'L2–L5' })).toBeVisible({
          timeout: 45000
        });

        // Panel de Evaluación por Rúbrica
        const evalPanel = teacherPage.locator('speed-grader-eval-panel');
        const rubricBox = evalPanel.locator('.rubric-grader-box');
        await expect(rubricBox).toBeVisible({ timeout: 45000 });

        const selects = rubricBox.locator('select.level-select');
        const selectCount = await selects.count();
        expect(selectCount).toBeGreaterThan(0);

        const commentsTextarea = rubricBox.locator('textarea.form-textarea');
        await commentsTextarea.fill(generalRubricComment);

        const rubricEvalPromise = teacherPage.waitForResponse(
          (res) =>
            res.url().includes(`/api/v1/submissions/${sharedState.submissionId}/rubric-evaluation`) &&
            res.request().method() === 'POST',
          { timeout: 45000 }
        );

        await rubricBox.locator('button.btn-save-rubric').click();
        const rubricEvalRes = await rubricEvalPromise;
        expect(rubricEvalRes.ok()).toBeTruthy();

        const rubricEvalJson = await rubricEvalRes.json();
        const rubricEvalData = rubricEvalJson?.data ?? rubricEvalJson;
        expect(rubricEvalData.score).toBe(100);

        await expect(rubricBox.locator('.success-banner-sm')).toContainText('Calificación guardada', {
          timeout: 45000
        });

        // Comprobar en el backend que el score y el comentario con author_id del docente quedaron persistidos
        const review = await apiJson(
          teacherPage.request,
          teacherSession.token,
          'GET',
          `/api/v1/teacher/submissions/${sharedState.submissionId}/review`
        );
        expect(review.score).toBe(100);
        expect(review.verdict).toBe('');
        expect(review.manual_override).toBe(false);

        const commentsList = await apiJson(
          teacherPage.request,
          teacherSession.token,
          'GET',
          `/api/v1/teacher/submissions/${sharedState.submissionId}/comments`
        );
        expect(Array.isArray(commentsList)).toBe(true);
        const teacherComment = commentsList.find((c) => c.comment === generalRubricComment);
        expect(teacherComment).toBeTruthy();
        expect(teacherComment.author_id).toBe(CANONICAL_ACTORS.teacher.id);

        const blockSaved = commentsList.find((c) => c.comment === blockComment);
        expect(blockSaved).toBeTruthy();
        expect(blockSaved.end_line_number).toBe(blockSaved.line_number + 3);
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        {
          admin: adminPage,
          teacher: teacherSession?.page,
          student: studentSession?.page
        }
      );
      throw err;
    } finally {
      if (sharedState.workspaceId) {
        await apiJson(
          adminPage.request,
          adminSession.token,
          'DELETE',
          `/api/v1/environments/${sharedState.workspaceId}`
        ).catch(() => undefined);
      }
      await adminSession.context.close();
      if (teacherSession) {
        await teacherSession.context.close();
      }
      if (studentSession) {
        await studentSession.context.close();
      }
    }
  });
});
