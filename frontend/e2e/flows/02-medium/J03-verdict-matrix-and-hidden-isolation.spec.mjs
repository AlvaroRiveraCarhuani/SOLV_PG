import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';
import {
  authorJudgeExercise,
  createCourseViaAdminUI,
  enrollStudentViaAPI,
  typeInCodeMirror
} from '../../fixtures/authoring.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('J03 — Matriz de Veredictos del Juez y Aislamiento de Casos Ocultos', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Matriz de Veredictos #${runId}`;
  const courseCode = `VMX-${runId}`;
  const exerciseTitle = `Doble de un Entero #${runId}`;

  const referenceSolution = 'n = int(input())\nprint(n * 2)\n';
  const exampleCase = { input: '5', expectedOutput: '10' };

  // Tres casos ocultos con valores distintivos: ninguno puede llegar al estudiante.
  const hiddenCases = [
    { input: '424242', expectedOutput: '848484' },
    { input: '737373', expectedOutput: '1474746' },
    { input: '919191', expectedOutput: '1838382' }
  ];
  const hiddenValues = hiddenCases.flatMap((c) => [c.input, c.expectedOutput]);

  const sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Docente publica con 3 casos ocultos y el estudiante obtiene WA, RE, TLE y AC desde el editor sin exposición de datos ocultos', async ({ browser }, testInfo) => {
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
      await test.step('Setup (Admin UI): Crear materia y asignar al docente titular', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin crea la materia para la matriz de veredictos');

        sharedState.courseId = await createCourseViaAdminUI(adminPage, {
          courseName,
          courseCode,
          teacherEmail: CANONICAL_ACTORS.teacher.email
        });
        expect(sharedState.courseId).toBeTruthy();
      });

      await test.step('Setup: Matricular al estudiante canónico (no existe UI de matrícula)', async () => {
        await enrollStudentViaAPI(studentPage, sharedState.courseId, CANONICAL_ACTORS.student.id);
      });

      await test.step('Docente UI: Publicar el ejercicio con 1 caso ejemplo y 3 casos ocultos', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente crea y publica el ejercicio con caso de ejemplo y tres casos ocultos');

        sharedState.exerciseId = await authorJudgeExercise(teacherPage, {
          courseId: sharedState.courseId,
          courseName,
          title: exerciseTitle,
          statement:
            '### Problema\n\n' +
            'Dado un entero `n`, imprime su doble.\n\n' +
            '### Formato de Entrada\n\n' +
            'Un entero `n`.\n\n' +
            '### Formato de Salida\n\n' +
            'El entero `2 * n`.\n\n' +
            '### Restricciones\n\n' +
            'La solucion debe leer de la entrada estandar y escribir el resultado en la salida estandar.',
          referenceSolution,
          exampleCase,
          hiddenCases
        });
        expect(sharedState.exerciseId).toBeTruthy();
      });

      await test.step('Aislamiento: el payload de GET /api/v1/exercises/:id no expone casos ocultos ni reference_solution', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre el ejercicio y se intercepta el payload público');

        await studentPage.goto(`/student/courses/${sharedState.courseId}`);
        const curricularMap = studentPage.locator('curricular-map');
        await expect(curricularMap).toBeVisible({ timeout: 45000 });

        const publicPayload = studentPage.waitForResponse(
          (res) =>
            res.url().includes(`/api/v1/exercises/${sharedState.exerciseId}`) &&
            res.request().method() === 'GET'
        );

        const exerciseRow = curricularMap.locator('.exercise-row', { hasText: exerciseTitle });
        await expect(exerciseRow).toBeVisible({ timeout: 45000 });
        await exerciseRow.locator('a.btn-solve').click();

        const [publicRes] = await Promise.all([publicPayload]);
        await expect(studentPage).toHaveURL(
          new RegExp(`/student/exercises/${sharedState.exerciseId}`),
          { timeout: 45000 }
        );

        const publicJson = await publicRes.json();
        const publicBody = publicJson?.data ?? publicJson;
        const publicRaw = JSON.stringify(publicBody);

        // Sin solución de referencia y sin bloque de casos de prueba.
        expect(publicBody).not.toHaveProperty('reference_solution');
        expect(publicBody).not.toHaveProperty('test_cases');
        expect(publicRaw).not.toContain('print(n * 2)');

        // Ningún caso oculto viaja al navegador del estudiante.
        expect(publicRaw).not.toContain('"visibility":"hidden"');
        for (const value of hiddenValues) {
          expect(publicRaw).not.toContain(value);
        }

        // Solo el caso de ejemplo es visible.
        const publicCases = publicBody.public_test_cases || [];
        expect(publicCases).toHaveLength(1);
        expect(publicCases[0].input).toBe(exampleCase.input);
        expect(publicCases[0].expected_output).toBe(exampleCase.expectedOutput);
      });

      await test.step('Aislamiento: el DOM del estudiante no contiene datos ocultos ni la solución de referencia', async () => {
        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(exerciseTitle, { timeout: 45000 });

        // El enunciado muestra el caso de ejemplo y nada más.
        await expect(solver.locator('.example-box')).toHaveCount(1, { timeout: 45000 });
        await expect(solver.locator('.example-box')).toContainText(exampleCase.input);
        await expect(solver.locator('.example-box')).toContainText(exampleCase.expectedOutput);

        const body = studentPage.locator('body');
        await expect(body).not.toContainText('print(n * 2)');
        for (const value of hiddenValues) {
          await expect(body).not.toContainText(value);
        }
      });

      const submitAndExpect = async (code, expectedVerdict) => {
        const solver = studentPage.locator('student-exercise-solver');
        const editor = solver.locator('code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 45000 });

        await typeInCodeMirror(studentPage, editor, code);

        const evalResponse = studentPage.waitForResponse(
          (res) => res.url().includes('/api/v1/evaluations') && res.request().method() === 'POST',
          { timeout: 120000 }
        );

        await solver.locator('[data-testid="btn-submit-solution"]').click();

        const [evalRes] = await Promise.all([evalResponse]);
        expect(evalRes.ok()).toBeTruthy();

        const evalJson = await evalRes.json();
        const evalBody = evalJson?.data ?? evalJson;
        const evalRaw = JSON.stringify(evalBody);

        // Veredicto exacto en el payload del juez.
        expect(evalBody.verdict).toBe(expectedVerdict);

        // Ningún dato de caso oculto viaja en la respuesta de evaluación.
        expect(evalRaw).not.toContain('print(n * 2)');
        for (const value of hiddenValues) {
          expect(evalRaw).not.toContain(value);
        }

        const resultPanel = solver.locator('[data-testid="judge-result-panel"]');
        await expect(resultPanel).toBeVisible({ timeout: 45000 });
        await expect(resultPanel.locator('.verdict-acronym').first()).toHaveText(expectedVerdict, {
          timeout: 45000
        });

        // El panel de resultados nunca muestra el input/output de los casos ocultos.
        const panelText = (await resultPanel.innerText()) || '';
        for (const value of hiddenValues) {
          expect(panelText).not.toContain(value);
        }

        // Los datos del caso de ejemplo siguen visibles para el estudiante.
        await expect(solver.locator('.example-box')).toHaveCount(1);
        await expect(solver.locator('.example-box')).toContainText(exampleCase.input);
        await expect(solver.locator('.example-box')).toContainText(exampleCase.expectedOutput);

        return evalBody;
      };

      await test.step('Veredicto WA: solución con resultado incorrecto', async () => {
        forensic.recordStep('Estudiante envía solución incorrecta -> WA');
        const result = await submitAndExpect('n = int(input())\nprint(n + 1)\n', 'WA');
        expect(result.case_results?.length).toBeGreaterThan(0);
        await expect(
          studentPage.locator('student-exercise-solver .submission-saved-banner')
        ).toContainText('Entrega registrada', { timeout: 45000 });
      });

      await test.step('Veredicto RE: solución con error en tiempo de ejecución', async () => {
        forensic.recordStep('Estudiante envía solución con división por cero -> RE');
        await submitAndExpect('n = int(input())\nprint(1 / 0)\n', 'RE');
      });

      await test.step('Veredicto TLE: solución con bucle infinito', async () => {
        forensic.recordStep('Estudiante envía solución con bucle infinito -> TLE');
        await submitAndExpect('n = int(input())\nwhile True:\n    pass\n', 'TLE');
      });

      await test.step('Veredicto AC: solución correcta', async () => {
        forensic.recordStep('Estudiante envía la solución correcta -> AC');
        const result = await submitAndExpect(referenceSolution, 'AC');
        expect(result.case_results?.length).toBe(4);
        for (const caseResult of result.case_results) {
          expect(caseResult.verdict).toBe('AC');
        }
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
