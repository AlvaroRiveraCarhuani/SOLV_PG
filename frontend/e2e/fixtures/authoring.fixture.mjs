/**
 * Helpers compartidos por los specs J0x de autoría del juez virtual.
 *
 * Nota de alcance: la plataforma NO expone UI de matrícula de estudiantes
 * (no existe ninguna llamada a `/api/v1/subjects/{id}/enroll` en el frontend).
 * Por eso la matriculación del estudiante canónico se hace contra la API
 * desde el contexto de la página del estudiante, igual que en los specs
 * heredados (`01-core-lifecycle`, `04-judge-verdicts`, `J01`).
 */

import { expect } from '@playwright/test';

export async function createCourseViaAdminUI(adminPage, { courseName, courseCode, teacherEmail }) {
  await adminPage.goto('/admin/cursos');
  await adminPage.getByRole('button', { name: /Nuevo Curso/i }).click();

  const modal = adminPage.locator('course-create-modal .modal-dialog');
  await modal.waitFor({ state: 'visible' });
  await modal.locator('#courseName').fill(courseName);
  await modal.locator('#courseCode').fill(courseCode);

  const teacherField = modal.locator('form-field', { hasText: 'Docente Titular Asignado' });
  const teacherComboInput = teacherField.locator('input[role="combobox"]');
  await teacherComboInput.click();
  await teacherComboInput.fill(teacherEmail);
  const teacherOption = teacherField.locator('[role="option"]').first();
  await teacherOption.waitFor({ state: 'visible' });
  await teacherOption.click();

  const submitCourseBtn = modal.locator('button.btn-primary', { hasText: 'Crear Curso' });
  const [createCourseRes] = await Promise.all([
    adminPage.waitForResponse(
      (res) => res.url().includes('/api/v1/subjects') && res.request().method() === 'POST'
    ),
    submitCourseBtn.click()
  ]);

  const courseBody = await createCourseRes.json();
  const courseId = courseBody?.id || courseBody?.data?.id || '';
  if (!courseId) {
    throw new Error(`No se pudo obtener el courseId: ${JSON.stringify(courseBody)}`);
  }
  return courseId;
}

export async function enrollStudentViaAPI(studentPage, courseId, studentId) {
  // sessionStorage solo es accesible en documentos con origen http(s).
  if (!/^https?:\/\//.test(studentPage.url())) {
    await studentPage.goto('/');
  }

  const status = await studentPage.evaluate(
    async ({ courseId, studentId }) => {
      const token = window.sessionStorage.getItem('solv_token');
      const resp = await fetch(`/api/v1/subjects/${courseId}/enroll`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        },
        body: JSON.stringify({ student_id: studentId })
      });
      return resp.status;
    },
    { courseId, studentId }
  );

  if (![200, 201].includes(status)) {
    throw new Error(`La matriculación del estudiante falló con status ${status}`);
  }
  return status;
}

/** Escribe código en el editor CodeMirror del panel indicado. */
export async function typeInCodeMirror(page, editorLocator, code) {
  await editorLocator.click();
  await page.keyboard.press('Control+A');
  await page.keyboard.insertText(code);
}

/**
 * Autoría completa de un ejercicio de Juez Efímero por la UI del docente
 * (3 pasos) y su publicación. Devuelve el id del ejercicio publicado.
 *
 * @param {object} opts
 * @param {string} opts.courseId        Materia donde se crea el ejercicio.
 * @param {string} opts.courseName      Nombre de la materia (título de la página del docente).
 * @param {string} opts.title           Título del ejercicio.
 * @param {string} opts.statement       Enunciado (markdown).
 * @param {string} opts.referenceSolution Código de la solución de referencia.
 * @param {{input: string, expectedOutput: string}} opts.exampleCase Caso visible (Ejemplo).
 * @param {Array<{input: string, expectedOutput: string}>} opts.hiddenCases Casos ocultos (Oculto).
 * @param {object} [opts.astRules]      Configuración AST del Paso 3:
 *   - `prohibitedPatterns`: string con patrones separados por comas para el formulario "Prohibir".
 *   - `blockSystemModules`: boolean para la política "Restringir Módulos del Sistema Operativo".
 */
export async function authorJudgeExercise(teacherPage, opts) {
  const {
    courseId,
    courseName,
    title,
    statement,
    referenceSolution,
    exampleCase,
    hiddenCases = [],
    astRules = null
  } = opts;

  await teacherPage.goto(`/teacher/cursos/${courseId}`, { waitUntil: 'domcontentloaded' });
  await expect(teacherPage.locator('h1.page-title')).toContainText(courseName, { timeout: 45000 });

  const newLabBtn = teacherPage.locator('button.btn-new-lab', { hasText: 'Nuevo Laboratorio' });
  await expect(newLabBtn).toBeVisible({ timeout: 45000 });
  await newLabBtn.click();
  await expect(teacherPage).toHaveURL(new RegExp(`/teacher/courses/${courseId}/exercises/new`));

  await teacherPage.locator('#exercise-title').fill(title);

  const diffInput = teacherPage.locator('#exercise-difficulty input[role="combobox"]');
  await diffInput.click();
  await teacherPage.locator('#exercise-difficulty [role="option"]', { hasText: 'Fácil' }).first().click();

  const langInput = teacherPage.locator('#exercise-language input[role="combobox"]');
  await langInput.click();
  await teacherPage.locator('#exercise-language [role="option"]', { hasText: /python/i }).first().click();

  await teacherPage.locator('#statement-description').fill(statement);

  // El autoguardado crea el borrador y redirige al editor del ejercicio persistido.
  await teacherPage
    .waitForURL(/\/exercises\/[0-9a-f-]{36}\/edit/, { timeout: 20000 })
    .catch(() => undefined);

  await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
  await expect(teacherPage.locator('#reference-solution-section')).toBeVisible({ timeout: 45000 });

  await typeInCodeMirror(
    teacherPage,
    teacherPage.locator('#reference-solution-section code-editor .cm-content').first(),
    referenceSolution
  );

  const addCaseBtn = teacherPage.locator('cases-table .toolbar-actions button.btn-primary', {
    hasText: 'Agregar Caso'
  });
  const caseRows = teacherPage.locator('cases-table tbody tr.case-row');

  // Caso de ejemplo (visible para el estudiante).
  await addCaseBtn.click();
  await expect(caseRows).toHaveCount(1);
  await caseRows.nth(0).locator('td.col-input textarea').fill(exampleCase.input);
  await caseRows.nth(0).locator('td.col-output textarea').fill(exampleCase.expectedOutput);

  // Casos ocultos (visibilidad "Oculto").
  for (let i = 0; i < hiddenCases.length; i++) {
    await addCaseBtn.click();
    await expect(caseRows).toHaveCount(i + 2);
    const row = caseRows.nth(i + 1);
    await row.locator('td.col-input textarea').fill(hiddenCases[i].input);
    await row.locator('td.col-output textarea').fill(hiddenCases[i].expectedOutput);
    const visInput = row.locator('td.col-vis input[role="combobox"]');
    await visInput.click();
    await row.locator('td.col-vis [role="option"]', { hasText: 'Oculto' }).first().click();
  }

  await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
  await expect(teacherPage.locator('publication-checklist')).toBeVisible({ timeout: 45000 });

  if (astRules) {
    const astBuilder = teacherPage.locator('ast-rules-builder');
    await expect(astBuilder).toBeVisible({ timeout: 45000 });

    if (astRules.blockSystemModules) {
      const systemPolicy = astBuilder.locator('.policy-card', {
        hasText: 'Restringir Módulos del Sistema Operativo'
      });
      await expect(systemPolicy).toBeVisible({ timeout: 45000 });
      const policyCheckbox = systemPolicy.locator('input.policy-checkbox');
      await expect(policyCheckbox).not.toBeChecked();
      await policyCheckbox.check();
      await expect(policyCheckbox).toBeChecked();
    }

    if (astRules.prohibitedPatterns) {
      await astBuilder.locator('#rule-pattern').fill(astRules.prohibitedPatterns);
      await astBuilder
        .locator('#rule-msg')
        .fill('El uso de estos módulos o funciones no está permitido en este ejercicio.');
      await astBuilder.locator('button.btn-primary', { hasText: 'Prohibir' }).click();

      const expectedPatterns = astRules.prohibitedPatterns
        .split(',')
        .map((p) => p.trim())
        .filter(Boolean);
      const patternCodes = astBuilder.locator('.rules-list .rule-item .pattern-code');
      await expect(patternCodes).toHaveCount(expectedPatterns.length, { timeout: 45000 });
      await expect(patternCodes).toHaveText(expectedPatterns, { timeout: 45000 });
    }
  }

  const checklist = teacherPage.locator('publication-checklist');
  await expect(checklist.locator('.section-blockers .section-name')).toHaveText('Sin Bloqueantes', {
    timeout: 45000
  });

  const ackCheckbox = checklist.locator('input.ack-checkbox');
  if (await ackCheckbox.isVisible().catch(() => false)) {
    await ackCheckbox.check();
  }

  const publishStepBtn = teacherPage.locator('button.btn-publish-step', { hasText: 'Publicar' });
  await expect(publishStepBtn).toBeEnabled({ timeout: 45000 });
  await publishStepBtn.click();

  const confirmPublishBtn = teacherPage.locator('confirm-modal button', { hasText: 'Publicar ahora' });
  await expect(confirmPublishBtn).toBeVisible();

  const [publishResp] = await Promise.all([
    teacherPage.waitForResponse(
      (r) => r.url().includes('/publish') && r.request().method() === 'POST'
    ),
    confirmPublishBtn.click()
  ]);
  if (!publishResp.ok()) {
    throw new Error(`La publicación falló con status ${publishResp.status()}`);
  }

  const matchExId = publishResp.url().match(/\/api\/v1\/exercises\/([^/]+)\/publish/);
  const exerciseId = matchExId ? matchExId[1] : '';
  if (!exerciseId) {
    throw new Error(`No se pudo obtener el exerciseId de ${publishResp.url()}`);
  }
  await expect(teacherPage.locator('.published-pill')).toContainText('Publicado', { timeout: 45000 });

  return exerciseId;
}

