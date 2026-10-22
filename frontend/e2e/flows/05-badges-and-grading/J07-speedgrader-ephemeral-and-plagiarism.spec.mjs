import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedActorSession,
  createAuthenticatedRoleSession,
  createStudentViaAdminAPI
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';
import { typeInCodeMirror } from '../../fixtures/authoring.fixture.mjs';
import { apiJson, openCurricularMap } from '../../fixtures/api.fixture.mjs';

test.describe.configure({ mode: 'serial' });

/**
 * J07 — SpeedGrader: prueba efímera, comentario por línea, override auditado y
 * detección de similitud.
 *
 * Los dos estudiantes resuelven el mismo ejercicio desde el editor con código
 * estructuralmente equivalente (el motor anti-plagio normaliza los
 * identificadores a `$ID`, por eso el renombrado de variables no oculta la
 * copia). El docente revisa la entrega en SpeedGrader, re-ejecuta la prueba
 * efímera, comenta una línea, ajusta la nota con motivo auditado y comprueba
 * que el panel anti-plagio marca el par.
 */
test.describe('J07 — SpeedGrader, efímera, override auditado y anti-plagio', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `SpeedGrader y Antiplagio #${runId}`;
  const courseCode = `SPD-${runId}`;
  const exerciseTitle = `Práctica Supervisada #${runId}`;
  const secondStudentEmail = `estudiante.plagio.${runId}@uab.edu.bo`;

  const solutionA = [
    'n = int(input())',
    'total = 0',
    'for i in range(n):',
    '    total += i',
    'print(total)',
    ''
  ].join('\n');

  const solutionB = [
    'n = int(input())',
    'accumulator = 0',
    'for i in range(n):',
    '    accumulator += i',
    'print(accumulator)',
    ''
  ].join('\n');

  const sharedState = {
    courseId: '',
    exerciseId: '',
    secondStudentId: '',
    submissionA: '',
    submissionB: ''
  };

  const solveFromEditor = async (page) => {
    const evalResponse = page.waitForResponse(
      (res) => res.url().includes('/api/v1/evaluations') && res.request().method() === 'POST',
      { timeout: 120000 }
    );
    const submissionResponse = page.waitForResponse(
      (res) => res.url().includes('/api/v1/submissions') && res.request().method() === 'POST',
      { timeout: 120000 }
    );

    await page.locator('student-exercise-solver [data-testid="btn-submit-solution"]').click();

    const [evalRes, submissionRes] = await Promise.all([evalResponse, submissionResponse]);
    expect(evalRes.ok()).toBeTruthy();

    const evalJson = await evalRes.json();
    const result = evalJson?.data ?? evalJson;
    expect(result.verdict).toBe('AC');

    const submissionJson = await submissionRes.json();
    return submissionJson?.data ?? submissionJson;
  };

  test('El docente revisa la entrega, re-ejecuta la efímera, comenta, ajusta la nota y detecta la similitud', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const teacherSession = await createAuthenticatedRoleSession(browser, 'teacher', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: teacherPage } = teacherSession;
    const { page: studentPage } = studentSession;

    let secondStudentSession = null;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(teacherPage, 'teacher');
    forensic.attachToPage(studentPage, 'student');

    try {
      await test.step('Setup: Crear la materia, el ejercicio y matricular a los dos estudiantes', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente prepara la materia vigilada');

        const course = await apiJson(teacherPage.request, teacherSession.token, 'POST', '/api/v1/subjects', {
          name: courseName,
          code: courseCode,
          teacher_id: CANONICAL_ACTORS.teacher.id,
          period: '2026-II',
          max_students: 50,
          is_active: true
        });
        sharedState.courseId = course.id;

        const exercise = await apiJson(teacherPage.request, teacherSession.token, 'POST', '/api/v1/exercises', {
          subject_id: sharedState.courseId,
          title: exerciseTitle,
          description: 'Suma los números del 0 al n-1.',
          type: 'algorithm',
          environment_type: 'JUEZ_EFIMERO',
          language: 'python',
          difficulty: 'medium',
          purpose: 'class',
          time_limit_ms: 2000,
          reference_solution: solutionA,
          test_cases: [
            { input: '5', expected_output: '10', visibility: 'example', weight: 1.0, order_index: 1 },
            { input: '4', expected_output: '6', visibility: 'hidden', weight: 1.0, order_index: 2 }
          ]
        });
        sharedState.exerciseId = exercise.id;
        await apiJson(teacherPage.request, teacherSession.token, 'POST', `/api/v1/exercises/${sharedState.exerciseId}/publish`);

        sharedState.secondStudentId = await createStudentViaAdminAPI(adminPage, {
          email: secondStudentEmail,
          firstName: 'Estudiante',
          lastName: `Plagio ${runId}`
        });

        secondStudentSession = await createAuthenticatedActorSession(
          browser,
          {
            id: sharedState.secondStudentId,
            email: secondStudentEmail,
            role: 'student',
            tenantId: CANONICAL_ACTORS.student.tenantId
          },
          baseURL
        );
        forensic.attachToPage(secondStudentSession.page, 'student2');

        await apiJson(teacherPage.request, teacherSession.token, 'POST', `/api/v1/subjects/${sharedState.courseId}/enroll`, {
          student_id: CANONICAL_ACTORS.student.id
        });
        await apiJson(teacherPage.request, teacherSession.token, 'POST', `/api/v1/subjects/${sharedState.courseId}/enroll`, {
          student_id: sharedState.secondStudentId
        });

        expect(sharedState.courseId).toBeTruthy();
        expect(sharedState.exerciseId).toBeTruthy();
      });

      await test.step('Paso 1: Los dos estudiantes envían soluciones casi idénticas desde el editor', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Primer estudiante entrega solución casi idéntica');

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

        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(exerciseTitle, { timeout: 45000 });

        const editor = solver.locator('code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 45000 });
        await typeInCodeMirror(studentPage, editor, solutionA);

        const firstSubmission = await solveFromEditor(studentPage);
        sharedState.submissionA = firstSubmission.id;
        expect(sharedState.submissionA).toBeTruthy();

        await secondStudentSession.page.goto(`/student/exercises/${sharedState.exerciseId}`, {
          waitUntil: 'domcontentloaded'
        });
        const secondSolver = secondStudentSession.page.locator('student-exercise-solver');
        await expect(secondSolver.locator('h1.page-title')).toContainText(exerciseTitle, { timeout: 45000 });

        const secondEditor = secondSolver.locator('code-editor .cm-content').first();
        await expect(secondEditor).toBeVisible({ timeout: 45000 });
        await typeInCodeMirror(secondStudentSession.page, secondEditor, solutionB);

        const secondSubmission = await solveFromEditor(secondStudentSession.page);
        sharedState.submissionB = secondSubmission.id;
        expect(sharedState.submissionB).toBeTruthy();
      });

      await test.step('Paso 2: El docente abre la entrega en SpeedGrader', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente entra al SpeedGrader');

        await teacherPage.goto(`/teacher/revision/${sharedState.submissionA}`, { waitUntil: 'domcontentloaded' });
        await expect(teacherPage.locator('speed-grader-header')).toBeVisible({ timeout: 45000 });
        await expect(teacherPage.locator('speed-grader-code-viewer')).toBeVisible({ timeout: 45000 });
        await expect(teacherPage.locator('.code-table .code-row').first()).toBeVisible({ timeout: 45000 });
      });

      await test.step('Paso 3: El docente ejecuta la prueba efímera', async () => {
        forensic.recordStep('Docente re-ejecuta la prueba efímera del envío');

        const ephemeralResponse = teacherPage.waitForResponse(
          (res) => res.url().includes('/api/v1/teacher/submissions/') && res.url().includes('/run-ephemeral'),
          { timeout: 120000 }
        );

        await teacherPage.locator('speed-grader-header button', { hasText: 'Re-ejecutar Prueba' }).click();

        const [ephemeralRes] = await Promise.all([ephemeralResponse]);
        expect(ephemeralRes.ok()).toBeTruthy();

        const ephemeralJson = await ephemeralRes.json();
        const ephemeralData = ephemeralJson?.data ?? ephemeralJson;
        expect(ephemeralData).toBeDefined();

        await expect(teacherPage.locator('.ephemeral-result-box')).toBeVisible({ timeout: 45000 });
      });

      await test.step('Paso 4: El docente registra un comentario por línea', async () => {
        forensic.recordStep('Docente comenta la línea 2 del código entregado');

        const commentResponse = teacherPage.waitForResponse(
          (res) =>
            res.url().includes('/api/v1/teacher/submissions/') &&
            res.url().includes('/comments') &&
            res.request().method() === 'POST',
          { timeout: 45000 }
        );

        await teacherPage.locator('.code-table .code-row .line-number').nth(1).click();
        const textarea = teacherPage.locator('.inline-comment-form-row textarea.form-textarea');
        await expect(textarea).toBeVisible({ timeout: 45000 });
        await textarea.fill('Buen uso de la iteración; explicá mejor el acumulado en la entrega escrita.');
        await teacherPage.locator('.inline-comment-form-row button.btn-submit-sm').click();

        const [commentRes] = await Promise.all([commentResponse]);
        expect(commentRes.ok()).toBeTruthy();

        await expect(teacherPage.locator('.comment-bubble .comment-text')).toContainText('Buen uso de la iteración', {
          timeout: 45000
        });
      });

      await test.step('Paso 5: El docente aplica un override de nota con motivo auditado', async () => {
        forensic.recordStep('Docente ajusta la nota con motivo auditable');

        const overrideResponse = teacherPage.waitForResponse(
          (res) => res.url().includes('/api/v1/submissions/') && res.url().includes('/override'),
          { timeout: 45000 }
        );

        await teacherPage.locator('speed-grader-header button', { hasText: 'Calificar' }).click();

        // El host del componente no tiene caja propia: el diálogo vive en el
        // overlay de modal-shell.
        const modal = teacherPage.locator('.modal-backdrop .modal-dialog');
        await expect(modal).toBeVisible({ timeout: 45000 });

        await modal.locator('input.form-input').fill('92');
        await modal.locator('textarea.form-textarea').fill('Convalidación por claridad y buena estructura del algoritmo.');
        await modal.locator('button.btn-primary', { hasText: 'Aplicar Modificación' }).click();

        const [overrideRes] = await Promise.all([overrideResponse]);
        expect(overrideRes.ok()).toBeTruthy();

        const review = await apiJson(
          teacherPage.request,
          teacherSession.token,
          'GET',
          `/api/v1/teacher/submissions/${sharedState.submissionA}/review`
        );
        expect(review.score).toBe(92);
        expect(review.override_reason).toContain('Convalidación por claridad');
      });

      await test.step('Paso 6: El panel anti-plagio marca el par de entregas similares', async () => {
        forensic.recordStep('Docente revisa la detección de similitud');

        const report = await apiJson(
          teacherPage.request,
          teacherSession.token,
          'GET',
          `/api/v1/teacher/courses/${sharedState.courseId}/plagiarism`
        );

        expect(report.total_submissions).toBeGreaterThanOrEqual(2);
        expect(report.matches.length).toBeGreaterThanOrEqual(1);

        const flagged = report.matches.find(
          (match) =>
            (match.student_id_a === CANONICAL_ACTORS.student.id &&
              match.student_id_b === sharedState.secondStudentId) ||
            (match.student_id_b === CANONICAL_ACTORS.student.id &&
              match.student_id_a === sharedState.secondStudentId)
        );
        expect(flagged).toBeTruthy();
        expect(flagged.similarity).toBeGreaterThanOrEqual(50);

        await teacherPage.goto('/teacher/anti-plagio', { waitUntil: 'domcontentloaded' });
        await expect(teacherPage.locator('h1.page-title')).toContainText('Control de Plagio', { timeout: 45000 });
        await expect(teacherPage.locator('.kpi-summary-grid .kpi-box').first()).toBeVisible({ timeout: 45000 });
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
          teacher: teacherPage,
          student: studentPage,
          student2: secondStudentSession?.page
        }
      );
      throw err;
    } finally {
      await adminSession.context.close();
      await teacherSession.context.close();
      await studentSession.context.close();
      if (secondStudentSession) {
        await secondStudentSession.context.close();
      }
    }
  });
});
