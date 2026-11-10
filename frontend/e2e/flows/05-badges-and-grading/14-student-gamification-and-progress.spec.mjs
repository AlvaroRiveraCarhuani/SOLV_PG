import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 05 (Badges & Calificación): Progreso del Estudiante y Gamificación', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Gamificación y Progreso #${runId}`;
  const courseCode = `GAME-${runId}`;
  const exerciseTitle = `Reto de Nivel #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Comprueba el registro de progreso, historial de entregas y cálculo de estadísticas', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(studentPage, 'student');

    try {
      // Setup de materia y entrega
      await test.step('Setup: Crear materia, publicar ejercicio y registrar entrega con veredicto AC', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de progreso del estudiante');

        await adminPage.goto("/");
        const setupRes = await adminPage.evaluate(async ({ cName, cCode, teacherId, studentId, exTitle }) => {
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
              teacher_id: teacherId,
              period: '2026-II',
              max_students: 50,
              is_active: true
            })
          });
          const cJson = await cResp.json();
          const courseId = cJson.data?.id || cJson.id;

          await fetch(`/api/v1/subjects/${courseId}/enroll`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ student_id: studentId })
          });

          // El juez exige una plantilla runner aprobada para crear el ejercicio.
          const tplResp = await fetch('/api/v1/templates?limit=50', { headers });
          const tplJson = await tplResp.json();
          const tplItems = tplJson.data?.data ?? tplJson.data ?? tplJson;
          const runner = (Array.isArray(tplItems) ? tplItems : []).find(
            (t) => /Python Runner Base/i.test(t.name || '')
          );
          if (!runner?.id) {
            return { error: 'No hay plantilla Python Runner Base disponible' };
          }

          const sourceCode = 'print("LEVEL_UP")\n';
          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title: exTitle,
              description: 'Reto para evaluar progreso.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              template_id: runner.id,
              language: 'python',
              difficulty: 'easy',
              time_limit_ms: 5000,
              reference_solution: sourceCode,
              test_cases: [
                { input: '1', expected_output: 'LEVEL_UP', is_hidden: false, weight: 1.0, order_index: 1 }
              ]
            })
          });
          const exJson = await exResp.json();
          const exerciseId = exJson.data?.id || exJson.id;
          await fetch(`/api/v1/exercises/${exerciseId}/publish`, { method: 'POST', headers });

          // El veredicto sale del recibo firmado por el servidor, no del payload.
          const evalResp = await fetch('/api/v1/evaluations', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              exercise_id: exerciseId,
              language: 'python',
              source_code_b64: btoa(sourceCode)
            })
          });
          const evalJson = await evalResp.json();
          const evalData = evalJson.data || evalJson;
          if (!evalData?.evaluation_receipt) {
            return { error: `Evaluación sin recibo: ${JSON.stringify(evalData)}` };
          }

          // Entrega del estudiante
          await fetch('/api/v1/submissions', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              exercise_id: exerciseId,
              student_id: studentId,
              code: sourceCode,
              verdict: evalData.verdict,
              evaluation_receipt: evalData.evaluation_receipt,
              score: 100,
              execution_time_ms: evalData.execution_time_ms || 12,
              memory_used_mb: Math.round(evalData.memory_used_mb || 8)
            })
          });

          return { courseId, exerciseId };
        }, {
          cName: courseName,
          cCode: courseCode,
          teacherId: CANONICAL_ACTORS.teacher.id,
          studentId: CANONICAL_ACTORS.student.id,
          exTitle: exerciseTitle
        });

        expect(setupRes.error, JSON.stringify(setupRes)).toBeUndefined();
        sharedState.courseId = setupRes.courseId;
        sharedState.exerciseId = setupRes.exerciseId;
      });

      // Estudiante consulta su panel de materias y su mapa de progreso
      await test.step('Paso 1: Estudiante verifica el curso y el estado completado de su entrega', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante consulta avance académico');

        await studentPage.goto(`/student/courses/${sharedState.courseId}`);
        await expect(studentPage.locator('h1.course-title, h1.page-title')).toContainText(courseName);

        // Mapa de progreso del estudiante
        const mapData = await studentPage.evaluate(async (courseId) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch(`/api/v1/student/courses/${courseId}/map`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {}
          });
          const json = await resp.json();
          return json.data || json;
        }, sharedState.courseId);

        expect(mapData).toBeDefined();
      });

      // Dashboard del estudiante: widgets de progreso y accesos recientes con datos reales
      await test.step('Paso 2: Dashboard muestra el curso al 100% y la entrega AC reciente', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante verifica widgets del dashboard');

        await studentPage.goto('/student');

        // "Tus materias": la materia matriculada aparece con su avance
        const courseRow = studentPage.locator('a.course-row', { hasText: courseCode });
        await expect(courseRow).toBeVisible();
        await expect(courseRow.locator('.course-progress')).toContainText('1/1');

        // "Últimas entregas": título del ejercicio, veredicto y tiempo semántico
        const recentItem = studentPage.locator('.recent-item', { hasText: exerciseTitle });
        await expect(recentItem).toBeVisible();
        await expect(recentItem.locator('verdict-badge')).toContainText('Cumple');
        await expect(recentItem.locator('time')).toBeVisible();
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        { admin: adminPage, student: studentPage }
      );
      throw err;
    }
  });
});
