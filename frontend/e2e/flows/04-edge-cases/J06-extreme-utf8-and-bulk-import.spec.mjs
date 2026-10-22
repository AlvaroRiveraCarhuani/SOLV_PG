import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';
import { typeInCodeMirror } from '../../fixtures/authoring.fixture.mjs';
import { apiJson, openCurricularMap } from '../../fixtures/api.fixture.mjs';

test.describe.configure({ mode: 'serial' });

/**
 * El mapa curricular de esta materia no declara módulos: la condición de
 * Readiness es que el ejercicio importado aparezca en la sección de
 * ejercicios generales.
 */
async function curricularMapReady(page, courseId, exerciseTitle) {
  const row = page.locator('curricular-map .exercise-row', { hasText: exerciseTitle });
  return await row.first().isVisible().catch(() => false);
}


/**
 * J06 — Importación masiva de ejercicios, analíticas del curso y evaluación con
 * UTF-8 extremo.
 *
 * El archivo importado usa el contrato jerárquico del importador
 * (metadata / statement / reference_solution / cases). El caso de prueba
 * mezcla emojis, tildes, CJK y un entero de 10^18: el juez debe comparar la
 * salida sin corrupción de codificación ni de precisión.
 */
test.describe('J06 — Importación masiva, analíticas y UTF-8 extremo', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Importación Masiva y UTF-8 #${runId}`;
  const courseCode = `IMP-${runId}`;
  const importedTitle = `Eco UTF-8 Extremo #${runId}`;

  const utf8Input = [
    'Hola Mundo 👋🤖',
    'Ñandú Álgebra y Cañón',
    'こんにちは世界',
    '1000000000000000000'
  ].join('\n');

  const echoSolution = [
    'import sys',
    '',
    'for line in sys.stdin:',
    "    print(line.rstrip('\\n'))",
    ''
  ].join('\n');

  const sharedState = {
    courseId: '',
    importedExerciseId: ''
  };

  test('El docente importa masivamente, consulta las analíticas y el estudiante resuelve el caso UTF-8 extremo', async ({ browser }, testInfo) => {
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
      await test.step('Setup: Crear la materia y matricular al estudiante', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente crea la materia para la importación masiva');

        const course = await apiJson(teacherPage.request, teacherSession.token, 'POST', '/api/v1/subjects', {
          name: courseName,
          code: courseCode,
          teacher_id: CANONICAL_ACTORS.teacher.id,
          period: '2026-II',
          max_students: 50,
          is_active: true
        });
        sharedState.courseId = course.id;

        await apiJson(teacherPage.request, teacherSession.token, 'POST', `/api/v1/subjects/${sharedState.courseId}/enroll`, {
          student_id: CANONICAL_ACTORS.student.id
        });

        expect(sharedState.courseId).toBeTruthy();
      });

      await test.step('Setup (Docente): Importar masivamente el ejercicio UTF-8 (dry-run y definitivo)', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente importa el ejercicio UTF-8 de forma masiva');

        const importPayload = JSON.stringify([
          {
            metadata: {
              title: importedTitle,
              difficulty: 'hard',
              modality: 'algorithm',
              purpose: 'class',
              time_limit_ms: 2000
            },
            statement: {
              description:
                'Repite en pantalla cada línea recibida, respetando emojis, tildes, ideogramas CJK y enteros de 10^18.'
            },
            reference_solution: {
              language: 'python',
              code: echoSolution
            },
            cases: [
              {
                input: utf8Input,
                expected_output: utf8Input,
                visibility: 'example',
                weight: 1.0
              }
            ]
          }
        ]);

        const uploadImport = async (url) => {
          const response = await teacherPage.request.fetch(url, {
            method: 'POST',
            headers: { Authorization: `Bearer ${teacherSession.token}` },
            multipart: {
              file: {
                name: 'exercises.json',
                mimeType: 'application/json',
                buffer: Buffer.from(importPayload, 'utf8')
              }
            }
          });
          expect([200, 201]).toContain(response.status());
          const body = await response.json();
          return body?.data ?? body;
        };

        const dryRunData = await uploadImport(
          `/api/v1/teacher/courses/${sharedState.courseId}/exercises/import?dry_run=true`
        );
        expect(dryRunData.mode).toBe('dry_run');
        expect(dryRunData.can_import).toBe(true);
        expect(dryRunData.total_valid).toBe(1);

        const importedData = await uploadImport(
          `/api/v1/teacher/courses/${sharedState.courseId}/exercises/import`
        );
        expect(importedData.imported_count).toBe(1);
        expect(importedData.imported_ids?.length).toBe(1);
        sharedState.importedExerciseId = importedData.imported_ids[0];
        expect(sharedState.importedExerciseId).toBeTruthy();
      });

      await test.step('Paso 1: El ejercicio importado aparece publicado en el curso del docente', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente publica el ejercicio importado');

        await apiJson(teacherPage.request, teacherSession.token, 'POST', `/api/v1/exercises/${sharedState.importedExerciseId}/publish`);

        const exercise = await apiJson(
          teacherPage.request,
          teacherSession.token,
          'GET',
          `/api/v1/exercises/${sharedState.importedExerciseId}`
        );
        expect(exercise.title).toBe(importedTitle);
        expect(exercise.status).toBe('published');
      });

      await test.step('Paso 2: El docente consulta las analíticas del curso', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente consulta el panel de analíticas del curso');

        await teacherPage.goto(`/teacher/courses/${sharedState.courseId}/analytics`, {
          waitUntil: 'domcontentloaded'
        });
        await expect(teacherPage.locator('h1.page-heading')).toContainText('Métricas y Analítica del Curso', {
          timeout: 45000
        });

        const analytics = await apiJson(
          teacherPage.request,
          teacherSession.token,
          'GET',
          `/api/v1/teacher/courses/${sharedState.courseId}/analytics`
        );
        expect(analytics).toBeDefined();
      });

      await test.step('Paso 3: El estudiante resuelve desde el editor el ejercicio UTF-8 extremo y obtiene AC', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante resuelve el ejercicio con UTF-8 extremo');

        const curricularMap = await openCurricularMap(studentPage, sharedState.courseId, {
          isReady: async () =>
            await curricularMapReady(studentPage, sharedState.courseId, importedTitle)
        });

        const exerciseRow = curricularMap.locator('.exercise-row', { hasText: importedTitle });
        await expect(exerciseRow).toBeVisible({ timeout: 45000 });
        await exerciseRow.locator('a.btn-solve').click();

        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(importedTitle, { timeout: 45000 });

        const editor = solver.locator('code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 45000 });
        await typeInCodeMirror(studentPage, editor, echoSolution);

        const evalResponse = studentPage.waitForResponse(
          (res) => res.url().includes('/api/v1/evaluations') && res.request().method() === 'POST',
          { timeout: 120000 }
        );
        await solver.locator('[data-testid="btn-submit-solution"]').click();

        const [evalRes] = await Promise.all([evalResponse]);
        expect(evalRes.ok()).toBeTruthy();

        const evalJson = await evalRes.json();
        const result = evalJson?.data ?? evalJson;
        expect(result.verdict).toBe('AC');

        const resultPanel = solver.locator('[data-testid="judge-result-panel"]');
        await expect(resultPanel).toBeVisible({ timeout: 45000 });
        await expect(resultPanel.locator('.verdict-acronym').first()).toHaveText('AC', { timeout: 45000 });
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
          student: studentPage
        }
      );
      throw err;
    } finally {
      await adminSession.context.close();
      await teacherSession.context.close();
      await studentSession.context.close();
    }
  });
});
