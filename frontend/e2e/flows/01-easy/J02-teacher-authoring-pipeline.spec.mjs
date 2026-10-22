import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('J02 — Pipeline de Autoría del Docente (UI) y Validación de Stack', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Autoría Pipeline #${runId}`;
  const courseCode = `AUT-${runId}`;
  const exerciseTitle = `Duplicar Entero #${runId}`;

  const pythonSolution = 'n = int(input())\nprint(n * 2)\n';
  const exampleInput = '5';
  const exampleOutput = '10';

  const sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Docente crea la materia, autora el ejercicio en 3 pasos por UI, valida el stack en Docker y publica', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const teacherSession = await createAuthenticatedRoleSession(browser, 'teacher', baseURL);

    const { page: adminPage } = adminSession;
    const { page: teacherPage } = teacherSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(teacherPage, 'teacher');

    try {
      await test.step('Setup (Admin UI): Crear materia y asignar al docente titular', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin crea materia en UI y asigna al docente titular');

        await adminPage.goto('/admin/cursos');
        await adminPage.getByRole('button', { name: /Nuevo Curso/i }).click();

        const modal = adminPage.locator('course-create-modal .modal-dialog');
        await expect(modal).toBeVisible();
        await modal.locator('#courseName').fill(courseName);
        await modal.locator('#courseCode').fill(courseCode);

        const teacherField = modal.locator('form-field', { hasText: 'Docente Titular Asignado' });
        const teacherComboInput = teacherField.locator('input[role="combobox"]');
        await teacherComboInput.click();
        await teacherComboInput.fill(CANONICAL_ACTORS.teacher.email);
        const teacherOption = teacherField.locator('[role="option"]').first();
        await expect(teacherOption).toBeVisible();
        await teacherOption.click();

        const submitCourseBtn = modal.locator('button.btn-primary', { hasText: 'Crear Curso' });
        const [createCourseRes] = await Promise.all([
          adminPage.waitForResponse(
            (res) => res.url().includes('/api/v1/subjects') && res.request().method() === 'POST'
          ),
          submitCourseBtn.click()
        ]);
        const courseBody = await createCourseRes.json();
        sharedState.courseId = courseBody?.id || courseBody?.data?.id || '';
        expect(sharedState.courseId).toBeTruthy();
      });

      await test.step('Paso 1 (Docente UI): Identidad, dificultad, lenguaje y enunciado', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente abre el editor en 3 pasos y completa la identidad del ejercicio');

        await teacherPage.goto(`/teacher/cursos/${sharedState.courseId}`, { waitUntil: 'domcontentloaded' });
        await expect(teacherPage.locator('h1.page-title')).toContainText(courseName, { timeout: 45000 });

        const newLabBtn = teacherPage.locator('button.btn-new-lab', { hasText: 'Nuevo Laboratorio' });
        await expect(newLabBtn).toBeVisible({ timeout: 15000 });
        await newLabBtn.click();
        await expect(teacherPage).toHaveURL(
          new RegExp(`/teacher/courses/${sharedState.courseId}/exercises/new`)
        );

        await teacherPage.locator('#exercise-title').fill(exerciseTitle);

        const diffInput = teacherPage.locator('#exercise-difficulty input[role="combobox"]');
        await diffInput.click();
        await teacherPage.locator('#exercise-difficulty [role="option"]', { hasText: 'Medio' }).first().click();

        const langInput = teacherPage.locator('#exercise-language input[role="combobox"]');
        await langInput.click();
        await teacherPage.locator('#exercise-language [role="option"]', { hasText: /python/i }).first().click();

        await teacherPage
          .locator('#statement-description')
          .fill(
            '### Problema\n\n' +
              'Dado un entero positivo `n` por la entrada estándar, imprime su doble.\n\n' +
              '### Formato de Entrada\n\n' +
              'Un entero `n` (1 <= n <= 1000000).\n\n' +
              '### Formato de Salida\n\n' +
              'El entero `2 * n`.\n\n' +
              '### Restricciones\n\n' +
              'La solución debe leer de la entrada estandar y devolver el resultado por la salida estandar.'
          );

        // El autoguardado crea el borrador y redirige al editor del ejercicio ya persistido.
        await teacherPage
          .waitForURL(/\/exercises\/[0-9a-f-]{36}\/edit/, { timeout: 20000 })
          .catch(() => undefined);

        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
        await expect(teacherPage.locator('#reference-solution-section')).toBeVisible({ timeout: 15000 });
      });

      await test.step('Paso 2 (Docente UI): Solución, generación de casos, cálculo de salidas y validación en Docker', async () => {
        forensic.recordStep('Docente escribe la solución, genera casos y valida salidas en el contenedor');

        const editor = teacherPage.locator('#reference-solution-section code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 15000 });
        await editor.click();
        await teacherPage.keyboard.press('Control+A');
        await teacherPage.keyboard.insertText(pythonSolution);

        // Definición de restricciones de formato: el builder visual si está montado,
        // o el generador de casos por plantillas estructuradas (ruta real expuesta hoy).
        const formatBuilder = teacherPage.locator('format-builder');
        if (await formatBuilder.isVisible().catch(() => false)) {
          await formatBuilder.locator('button', { hasText: 'Activar Esquema' }).click();
          await formatBuilder.locator('preset-button button').first().click();
          await formatBuilder.locator('button.add-line-btn', { hasText: 'Agregar Línea' }).click();
          await expect(formatBuilder.locator('.line-card')).toHaveCount(1);
        }

        await teacherPage.locator('cases-table .toolbar-actions button', { hasText: 'Generar Casos' }).click();
        const generateModal = teacherPage.locator('generate-cases-modal .modal-container');
        await expect(generateModal).toBeVisible();

        await generateModal.locator('#gen-count').fill('3');
        const [generateRes] = await Promise.all([
          teacherPage.waitForResponse(
            (res) => res.url().includes('/api/v1/exercises/generate-cases') && res.request().method() === 'POST'
          ),
          generateModal.locator('.modal-footer button.btn-primary').click()
        ]);
        expect(generateRes.ok()).toBeTruthy();
        await expect(generateModal).toBeHidden();

        const caseRows = teacherPage.locator('cases-table tbody tr.case-row');
        await expect(caseRows).toHaveCount(3, { timeout: 15000 });

        // Caso de ejemplo público: el estudiante lo ve en el enunciado.
        await teacherPage.locator('cases-table .toolbar-actions button.btn-primary', { hasText: 'Agregar Caso' }).click();
        await expect(caseRows).toHaveCount(4);

        const exampleRow = caseRows.nth(3);
        await exampleRow.locator('td.col-input textarea').fill(exampleInput);

        const visInput = exampleRow.locator('td.col-vis input[role="combobox"]');
        await visInput.click();
        await exampleRow.locator('td.col-vis [role="option"]', { hasText: 'Ejemplo' }).first().click();

        // Cálculo de salidas: ejecuta la solución de referencia en el contenedor Docker.
        const [calcRes] = await Promise.all([
          teacherPage.waitForResponse(
            (res) => res.url().includes('/api/v1/exercises/calculate-outputs') && res.request().method() === 'POST'
          ),
          teacherPage.locator('cases-table .toolbar-actions button.btn-calc-toolbar').click()
        ]);
        expect(calcRes.ok()).toBeTruthy();

        await expect(caseRows.first().locator('td.col-output textarea')).not.toHaveValue('', { timeout: 30000 });
        await expect(exampleRow.locator('td.col-output textarea')).toHaveValue(exampleOutput, { timeout: 30000 });
        await expect(teacherPage.locator('cases-table .table-error-banner')).toHaveCount(0);

        // Vista previa: los casos de ejemplo se visualizan correctamente.
        await teacherPage.locator('button.preview-btn', { hasText: 'Vista previa' }).click();
        const previewModal = teacherPage.locator('student-preview-modal .student-preview-window');
        await expect(previewModal).toBeVisible({ timeout: 15000 });
        await expect(previewModal.locator('.example-cases-section .example-card')).toHaveCount(1, { timeout: 15000 });
        await expect(previewModal.locator('.example-cases-section .example-card')).toContainText(exampleInput);
        await previewModal.locator('button.btn-close').click();
        await expect(previewModal).toBeHidden();
      });

      await test.step('Paso 3 (Docente UI): Checklist sin bloqueantes, dry-run en Docker y publicación', async () => {
        forensic.recordStep('Docente valida el checklist, ejecuta el dry-run y publica el ejercicio');

        const checklistResponse = teacherPage
          .waitForResponse(
            (res) => res.url().includes('/checklist') && res.request().method() === 'POST',
            { timeout: 20000 }
          )
          .catch(() => null);

        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
        await checklistResponse;

        const checklist = teacherPage.locator('publication-checklist');
        await expect(checklist).toBeVisible({ timeout: 20000 });

        const blockersName = checklist.locator('.section-blockers .section-name');
        await expect(blockersName).toHaveText('Sin Bloqueantes', { timeout: 30000 });

        const ackCheckbox = checklist.locator('input.ack-checkbox');
        if (await ackCheckbox.isVisible().catch(() => false)) {
          await ackCheckbox.check();
        }

        // Simulador de estudiante: ejecuta la solución contra los casos en el contenedor.
        await checklist.locator('button.btn-simulate', { hasText: 'Simular como Estudiante' }).click();
        const simulator = teacherPage.locator('student-preview-modal .student-preview-window');
        await expect(simulator).toBeVisible({ timeout: 15000 });
        await expect(simulator.locator('#preview-modal-title')).toHaveText('Simulador de Estudiante');

        await simulator.locator('button.btn-editor-helper', { hasText: 'Solución Canónica' }).click();
        const [dryRunRes] = await Promise.all([
          teacherPage.waitForResponse(
            (res) => res.url().includes('/api/v1/exercises/calculate-outputs') && res.request().method() === 'POST',
            { timeout: 60000 }
          ),
          simulator.locator('button.btn-run-simulation', { hasText: 'Correr Pruebas' }).click()
        ]);
        expect(dryRunRes.ok()).toBeTruthy();

        await expect(simulator.locator('.sim-verdict-summary-wrap verdict-badge')).toContainText('AC', { timeout: 60000 });
        await expect(simulator.locator('.sim-cases-tabs .sim-case-tab')).toHaveCount(1, { timeout: 60000 });
        await simulator.locator('button.btn-close').click();
        await expect(simulator).toBeHidden();

        const publishStepBtn = teacherPage.locator('button.btn-publish-step', { hasText: 'Publicar' });
        await expect(publishStepBtn).toBeEnabled({ timeout: 15000 });
        await publishStepBtn.click();

        const confirmPublishBtn = teacherPage.locator('confirm-modal button', { hasText: 'Publicar ahora' });
        await expect(confirmPublishBtn).toBeVisible();

        const [publishResp] = await Promise.all([
          teacherPage.waitForResponse(
            (r) => r.url().includes('/publish') && r.request().method() === 'POST'
          ),
          confirmPublishBtn.click()
        ]);
        expect(publishResp.ok()).toBeTruthy();

        const matchExId = publishResp.url().match(/\/api\/v1\/exercises\/([^/]+)\/publish/);
        const publishJson = await publishResp.json().catch(() => ({}));
        sharedState.exerciseId = matchExId ? matchExId[1] : (publishJson?.data?.id || publishJson?.id);
        expect(sharedState.exerciseId).toBeTruthy();

        await expect(teacherPage.locator('.published-pill')).toContainText('Publicado', { timeout: 30000 });
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        { admin: adminPage, teacher: teacherPage }
      );
      throw err;
    } finally {
      await adminSession.context.close();
      await teacherSession.context.close();
    }
  });
});
