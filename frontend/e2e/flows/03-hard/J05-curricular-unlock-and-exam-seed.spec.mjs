import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedActorSession,
  createAuthenticatedRoleSession,
  createStudentViaAdminAPI
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';
import {
  createCourseViaAdminUI,
  enrollStudentViaAPI,
  typeInCodeMirror
} from '../../fixtures/authoring.fixture.mjs';
import { apiJson, openCurricularMap } from '../../fixtures/api.fixture.mjs';

test.describe.configure({ mode: 'serial' });

/**
 * J05 — Desbloqueo curricular encadenado y examen con semilla por estudiante.
 *
 * Comportamiento real del motor (verificado contra el stack vivo):
 *
 *  1. El estado de un módulo en el mapa curricular es `locked` mientras falta
 *     el prerrequisito y `unlocked` cuando se cumple (la UI rotula
 *     "Bloqueado" y "Disponible"). No existe el estado `available`.
 *  2. Un módulo se completa cuando todas sus ejercicios publicados tienen
 *     `best_score >= pass_score`. El veredicto AC sin nota explícita cuenta
 *     como 100 puntos, por eso resolver el ejercicio del Módulo 1 desde la UI
 *     desbloquea el Módulo 2.
 *  3. En los exámenes con `per_student_seed` la entrada de cada caso se genera
 *     con la semilla `sha256(ejercicio:estudiante)`, de modo que dos
 *     estudiantes reciben entradas distintas.
 *  4. El rol student nunca recibe la salida esperada de los casos generados:
 *     es la respuesta del ejercicio (GOB-SEGURO-04).
 */
test.describe('J05 — Desbloqueo curricular encadenado y examen con semilla por estudiante', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Cadena Curricular y Examen #${runId}`;
  const courseCode = `CURR-${runId}`;
  const moduleOneExerciseTitle = `Práctica del Módulo 1 #${runId}`;
  const moduleTwoExerciseTitle = `Práctica del Módulo 2 #${runId}`;
  const seededExamTitle = `Examen con Semilla #${runId}`;
  const secondStudentEmail = `estudiante.semilla.${runId}@uab.edu.bo`;

  const doubleSolution = 'n = int(input())\nprint(n * 2)\n';

  const sharedState = {
    courseId: '',
    moduleOneId: '',
    moduleTwoId: '',
    moduleOneExerciseId: '',
    seededExamId: '',
    secondStudentId: ''
  };

  test('El módulo con prerrequisito se desbloquea al aprobar y el examen entrega semillas distintas sin filtrar la respuesta', async ({ browser }, testInfo) => {
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
      await test.step('Setup (Admin UI): Crear la materia de la cadena curricular', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin crea la materia para la cadena curricular');

        sharedState.courseId = await createCourseViaAdminUI(adminPage, {
          courseName,
          courseCode,
          teacherEmail: CANONICAL_ACTORS.teacher.email
        });
        expect(sharedState.courseId).toBeTruthy();
      });

      await test.step('Setup: Registrar un segundo estudiante y matricular a ambos en la materia', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin registra el segundo estudiante del examen');

        sharedState.secondStudentId = await createStudentViaAdminAPI(adminPage, {
          email: secondStudentEmail,
          firstName: 'Estudiante',
          lastName: `Semilla ${runId}`
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

        // La plataforma no expone UI de matrícula: se registra por API.
        await enrollStudentViaAPI(studentPage, sharedState.courseId, CANONICAL_ACTORS.student.id);
        await enrollStudentViaAPI(secondStudentSession.page, sharedState.courseId, sharedState.secondStudentId);
      });

      await test.step('Setup (Docente): Crear Módulo 1, Módulo 2 y el prerrequisito', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente crea los dos módulos y el prerrequisito');

        const teacherRequest = teacherPage.request;
        const teacherToken = teacherSession.token;

        const moduleOne = await apiJson(teacherRequest, teacherToken, 'POST', `/api/v1/teacher/courses/${sharedState.courseId}/modules`, {
          title: 'Módulo 1: Fundamentos',
          description: 'Prerrequisito del módulo 2',
          order_index: 1,
          pass_score: 60
        });
        sharedState.moduleOneId = moduleOne.id;

        const moduleTwo = await apiJson(teacherRequest, teacherToken, 'POST', `/api/v1/teacher/courses/${sharedState.courseId}/modules`, {
          title: 'Módulo 2: Avanzado',
          description: 'Requiere aprobar el Módulo 1',
          order_index: 2,
          pass_score: 60
        });
        sharedState.moduleTwoId = moduleTwo.id;

        await apiJson(
          teacherRequest,
          teacherToken,
          'POST',
          `/api/v1/teacher/modules/${sharedState.moduleTwoId}/prerequisites`,
          { prerequisite_module_ids: [sharedState.moduleOneId] }
        );

        expect(sharedState.moduleOneId).toBeTruthy();
        expect(sharedState.moduleTwoId).toBeTruthy();
      });

      await test.step('Setup (Docente): Publicar el ejercicio del Módulo 1 y el examen con semilla', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente publica la práctica del Módulo 1 y el examen con semilla');

        const teacherRequest = teacherPage.request;
        const teacherToken = teacherSession.token;

        const createExercise = async (payload) => {
          const created = await apiJson(teacherRequest, teacherToken, 'POST', '/api/v1/exercises', payload);
          await apiJson(teacherRequest, teacherToken, 'POST', `/api/v1/exercises/${created.id}/publish`);
          return created.id;
        };

        const assignModule = async (exerciseId, moduleId) => {
          await apiJson(
            teacherRequest,
            teacherToken,
            'PUT',
            `/api/v1/teacher/exercises/${exerciseId}/module`,
            { module_id: moduleId }
          );
        };

        sharedState.moduleOneExerciseId = await createExercise({
          subject_id: sharedState.courseId,
          title: moduleOneExerciseTitle,
          description: 'Imprime el doble del entero recibido.',
          type: 'algorithm',
          environment_type: 'JUEZ_EFIMERO',
          language: 'python',
          difficulty: 'easy',
          purpose: 'class',
          time_limit_ms: 2000,
          reference_solution: doubleSolution,
          test_cases: [
            { input: '5', expected_output: '10', visibility: 'example', weight: 1.0, order_index: 1 },
            { input: '40', expected_output: '80', visibility: 'hidden', weight: 1.0, order_index: 2 }
          ]
        });
        await assignModule(sharedState.moduleOneExerciseId, sharedState.moduleOneId);

        const moduleTwoExerciseId = await createExercise({
          subject_id: sharedState.courseId,
          title: moduleTwoExerciseTitle,
          description: 'Ejercicio del módulo que se habilita al cumplir el prerrequisito.',
          type: 'algorithm',
          environment_type: 'JUEZ_EFIMERO',
          language: 'python',
          difficulty: 'medium',
          purpose: 'class',
          time_limit_ms: 2000,
          reference_solution: 'n = int(input())\nprint(n * 3)\n',
          test_cases: [
            { input: '7', expected_output: '21', visibility: 'example', weight: 1.0, order_index: 1 }
          ]
        });
        await assignModule(moduleTwoExerciseId, sharedState.moduleTwoId);

        sharedState.seededExamId = await createExercise({
          subject_id: sharedState.courseId,
          title: seededExamTitle,
          description: 'Examen con variante individual por estudiante.',
          type: 'algorithm',
          environment_type: 'JUEZ_EFIMERO',
          language: 'python',
          difficulty: 'hard',
          purpose: 'exam',
          per_student_seed: true,
          time_limit_ms: 2000,
          reference_solution: doubleSolution,
          config: {
            algorithm: {
              input_format: {
                version: 1,
                input: {
                  lines: [{ id: 'n', type: 'int', min: 100000, max: 999999 }]
                }
              }
            }
          },
          test_cases: [
            { input: '123456', expected_output: '246912', visibility: 'example', weight: 1.0, order_index: 1 }
          ]
        });

        expect(sharedState.moduleOneExerciseId).toBeTruthy();
        expect(sharedState.seededExamId).toBeTruthy();
      });

      await test.step('Paso 1: El Módulo 2 aparece bloqueado en el mapa curricular del estudiante', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre el mapa curricular con el Módulo 2 bloqueado');

        const curricularMap = await openCurricularMap(studentPage, sharedState.courseId);

        const moduleTwoCard = curricularMap.locator('.module-card', {
          hasText: 'Módulo 2: Avanzado'
        });
        await expect(moduleTwoCard).toBeVisible({ timeout: 45000 });
        await expect(moduleTwoCard).toHaveClass(/is-locked/);
        await expect(moduleTwoCard.locator('.module-state-badge')).toHaveText('Bloqueado');
      });

      await test.step('Paso 2: El estudiante resuelve el ejercicio del Módulo 1 desde la UI y obtiene AC', async () => {
        forensic.recordStep('Estudiante resuelve la práctica del Módulo 1 en el editor');

        const curricularMap = await openCurricularMap(studentPage, sharedState.courseId);

        const exerciseRow = curricularMap.locator('.exercise-row', {
          hasText: moduleOneExerciseTitle
        });
        await expect(exerciseRow).toBeVisible({ timeout: 45000 });
        await exerciseRow.locator('a.btn-solve').click();

        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(moduleOneExerciseTitle, {
          timeout: 45000
        });

        const editor = solver.locator('code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 45000 });
        await typeInCodeMirror(studentPage, editor, doubleSolution);

        const evalResponse = studentPage.waitForResponse(
          (res) => res.url().includes('/api/v1/evaluations') && res.request().method() === 'POST',
          { timeout: 120000 }
        );
        await solver.locator('[data-testid="btn-submit-solution"]').click();

        const [evalRes] = await Promise.all([evalResponse]);
        expect(evalRes.ok()).toBeTruthy();

        const evalBody = await evalRes.json();
        const result = evalBody?.data ?? evalBody;
        expect(result.verdict).toBe('AC');

        const resultPanel = solver.locator('[data-testid="judge-result-panel"]');
        await expect(resultPanel).toBeVisible({ timeout: 45000 });
        await expect(resultPanel.locator('.verdict-acronym').first()).toHaveText('AC', { timeout: 45000 });
      });

      await test.step('Paso 3: Al volver al mapa, el Módulo 2 queda disponible', async () => {
        forensic.recordStep('Estudiante vuelve al mapa curricular y el Módulo 2 está disponible');

        const curricularMap = await openCurricularMap(studentPage, sharedState.courseId);

        const moduleTwoCard = curricularMap.locator('.module-card', {
          hasText: 'Módulo 2: Avanzado'
        });
        await expect(moduleTwoCard).toBeVisible({ timeout: 45000 });
        await expect(moduleTwoCard).toHaveClass(/is-unlocked/);
        await expect(moduleTwoCard.locator('.module-state-badge')).toHaveText('Disponible');
      });

      const solveSeededExam = async (page) => {
        await page.bringToFront();
        await page.goto(`/student/exercises/${sharedState.seededExamId}`, { waitUntil: 'domcontentloaded' });

        const solver = page.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(seededExamTitle, { timeout: 45000 });

        const editor = solver.locator('code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 45000 });
        await typeInCodeMirror(page, editor, doubleSolution);

        const evalResponse = page.waitForResponse(
          (res) => res.url().includes('/api/v1/evaluations') && res.request().method() === 'POST',
          { timeout: 120000 }
        );
        await solver.locator('[data-testid="btn-submit-solution"]').click();

        const [evalRes] = await Promise.all([evalResponse]);
        expect(evalRes.ok()).toBeTruthy();

        const evalJson = await evalRes.json();
        const body = evalJson?.data ?? evalJson;
        return { body, raw: JSON.stringify(body) };
      };

      await test.step('Paso 4: Los dos estudiantes reciben entradas distintas generadas por su semilla', async () => {
        forensic.recordStep('Los dos estudiantes rinden el examen con semilla');

        const first = await solveSeededExam(studentPage);
        const second = await solveSeededExam(secondStudentSession.page);

        expect(first.body.verdict).toBe('AC');
        expect(second.body.verdict).toBe('AC');

        const firstInput = first.body.generated_cases?.[0]?.input;
        const secondInput = second.body.generated_cases?.[0]?.input;

        expect(firstInput).toBeTruthy();
        expect(secondInput).toBeTruthy();
        expect(firstInput).not.toBe(secondInput);
      });

      await test.step('Paso 5: El payload HTTP del estudiante nunca expone la salida esperada', async () => {
        forensic.recordStep('Se verifica que el payload de evaluación no expone expected_output');

        const second = await solveSeededExam(secondStudentSession.page);

        expect(second.raw).not.toContain('expected_output');
        expect(second.body.generated_cases ?? []).toHaveLength(1);

        for (const generatedCase of second.body.generated_cases ?? []) {
          expect(generatedCase).not.toHaveProperty('expected_output');
          expect(generatedCase.expected_output).toBeUndefined();
        }

        // La entrega persistida tampoco guarda la salida esperada declarada por
        // el cliente: el servidor conserva el snapshot autoritativo.
        const submissions = await apiJson(
          secondStudentSession.page.request,
          secondStudentSession.token,
          'GET',
          `/api/v1/exercises/${sharedState.seededExamId}/submissions`
        );

        const list = Array.isArray(submissions) ? submissions : [];
        expect(list.length).toBeGreaterThan(0);
        for (const submission of list) {
          expect(JSON.stringify(submission)).not.toContain('expected_output');
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
