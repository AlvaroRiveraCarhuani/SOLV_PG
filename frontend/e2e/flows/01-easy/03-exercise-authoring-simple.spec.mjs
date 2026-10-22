import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 01 (Fácil): Editor de Ejercicios en 3 Pasos y Resolución Básica', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Matemática Discreta E2E #${runId}`;
  const courseCode = `DISC-${runId}`;
  const exerciseTitle = `Multiplicación Simple #${runId}`;
  const canonicalSolution = 'a, b = map(int, input().split())\nprint(a * b)\n';

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Docente crea ejercicio en 3 pasos, publica y estudiante resuelve con AC', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const teacherSession = await createAuthenticatedRoleSession(browser, 'teacher', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: teacherPage } = teacherSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(teacherPage, 'teacher');
    forensic.attachToPage(studentPage, 'student');

    try {
      // =========================================================================
      // SETUP: ADMIN CREA MATERIA Y MATRICULA
      // =========================================================================
      await test.step('Setup: Admin crea materia y matricula alumno', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de materia');

        await adminPage.goto("/");
        const courseRes = await adminPage.evaluate(async ({ cName, cCode, teacherId, studentId }) => {
          const token = window.sessionStorage.getItem('solv_token');
          const headers = {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          };

          const cResp = await fetch('/api/v1/subjects', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              name: cName,
              code: cCode,
              teacher_id: teacherId
            })
          });
          const cJson = await cResp.json();
          const courseId = cJson.data?.id || cJson.id;

          if (courseId) {
            await fetch(`/api/v1/subjects/${courseId}/enroll`, {
              method: 'POST',
              headers,
              body: JSON.stringify({ student_id: studentId })
            });
          }
          return { ok: cResp.status === 201 || cResp.status === 200, courseId };
        }, {
          cName: courseName,
          cCode: courseCode,
          teacherId: CANONICAL_ACTORS.teacher.id,
          studentId: CANONICAL_ACTORS.student.id
        });

        expect(courseRes.ok).toBe(true);
        sharedState.courseId = courseRes.courseId;
      });

      // =========================================================================
      // PASO 1: DOCENTE ENTRA AL EDITOR Y CREA EL EJERCICIO
      // =========================================================================
      await test.step('Paso 1: Docente completa Paso 1 (Metadatos y Enunciado)', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente entra a la materia y crea ejercicio');
        await teacherPage.goto(`/teacher/cursos/${sharedState.courseId}`, { waitUntil: 'domcontentloaded' });
        await expect(teacherPage).toHaveURL(new RegExp(`/teacher/cursos/${sharedState.courseId}`));
        await expect(teacherPage.locator('h1.page-title')).toContainText(courseName, { timeout: 15000 });

        const newLabBtn = teacherPage.locator('button.btn-new-lab', { hasText: 'Nuevo Laboratorio' });
        await expect(newLabBtn).toBeVisible({ timeout: 15000 });
        await newLabBtn.click();
        await expect(teacherPage).toHaveURL(new RegExp(`/teacher/courses/${sharedState.courseId}/exercises/new`));

        // Metadatos
        await teacherPage.locator('#exercise-title').fill(exerciseTitle);

        // Dificultad
        const diffInput = teacherPage.locator('#exercise-difficulty input[role="combobox"]');
        await diffInput.click();
        await teacherPage.locator('#exercise-difficulty [role="option"]', { hasText: 'Fácil' }).first().click();

        // Lenguaje
        const langInput = teacherPage.locator('#exercise-language input[role="combobox"]');
        await langInput.click();
        await teacherPage.locator('#exercise-language [role="option"]', { hasText: 'Python' }).first().click();

        // Enunciado
        await teacherPage.locator('#statement-description').fill('Calcula el producto de dos números enteros.');

        // Avanzar a Paso 2
        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
      });

      await test.step('Paso 2: Docente completa Paso 2 (Solución y Casos) y Paso 3 (Publicar)', async () => {
        forensic.recordStep('Docente completa solución, casos y publica');

        // Solución de referencia
        await expect(teacherPage.locator('#reference-solution-section')).toBeVisible();
        const cmContent = teacherPage.locator('#reference-solution-section .cm-content').first();
        await cmContent.click();
        await teacherPage.keyboard.press('Control+A');
        await teacherPage.keyboard.type(canonicalSolution);

        // Agregar caso
        const addCaseBtn = teacherPage.locator('cases-table button', { hasText: 'Agregar Caso' });
        await addCaseBtn.click();

        const caseRows = teacherPage.locator('cases-table tbody tr.case-row');
        await expect(caseRows).toHaveCount(1);
        await caseRows.first().locator('td.col-input textarea').fill('4 5');
        await caseRows.first().locator('td.col-output textarea').fill('20');

        // Avanzar a Paso 3
        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
        await expect(teacherPage.locator('publication-checklist')).toBeVisible();

        // Aceptar advertencias si hay
        const ackCheckbox = teacherPage.locator('publication-checklist input.ack-checkbox');
        if (await ackCheckbox.isVisible().catch(() => false)) {
          await ackCheckbox.check();
        }

        // Publicar
        const publishStepBtn = teacherPage.locator('button.btn-publish-step', { hasText: 'Publicar' });
        await expect(publishStepBtn).toBeEnabled();
        await publishStepBtn.click();

        const confirmPublishBtn = teacherPage.locator('confirm-modal button', { hasText: 'Publicar ahora' });
        await expect(confirmPublishBtn).toBeVisible();

        const [publishResp] = await Promise.all([
          teacherPage.waitForResponse(r => r.url().includes('/publish') && r.request().method() === 'POST'),
          confirmPublishBtn.click()
        ]);

        const matchExId = publishResp.url().match(/\/api\/v1\/exercises\/([^/]+)\/publish/);
        const publishJson = await publishResp.json().catch(() => ({}));
        sharedState.exerciseId = matchExId ? matchExId[1] : (publishJson.data?.id || publishJson.id);
        await expect(teacherPage.locator('.published-pill')).toContainText('Publicado');
      });

      // =========================================================================
      // PASO 3: ESTUDIANTE RESUELVE Y OBTIENE AC
      // =========================================================================
      await test.step('Paso 3: Estudiante entra y resuelve el ejercicio con veredicto AC', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante envía solución y obtiene AC');
        await studentPage.goto(`/student/courses/${sharedState.courseId}`);
        await expect(studentPage.locator('h1.course-title, h1.page-title')).toContainText(courseName);

        const evalRes = await studentPage.evaluate(async ({ exId, solution }) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch('/api/v1/evaluations', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            },
            body: JSON.stringify({
              exercise_id: exId,
              language: 'python',
              source_code_b64: btoa(solution)
            })
          });
          const json = await resp.json();
          return json.data || json;
        }, { exId: sharedState.exerciseId, solution: canonicalSolution });

        expect(evalRes.verdict).toBe('AC');
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        { admin: adminPage, teacher: teacherPage, student: studentPage }
      );
      throw err;
    } finally {
      await adminSession.context.close();
      await teacherSession.context.close();
      await studentSession.context.close();
    }
  });
});
