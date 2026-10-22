import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 04 (Casos Borde): Entradas Extremas, Caracteres UTF-8 y 10^18', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Estrés de Caracteres y BigInt #${runId}`;
  const courseCode = `UTF-${runId}`;
  const exerciseTitle = `Inversor UTF-8 y Enteros Grandes #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Valida que el Juez procesa sin corrupción emojis, acentos, saltos de línea y 10^18', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(studentPage, 'student');

    try {
      // Setup de ejercicio con datos UTF-8 y enteros masivos
      await test.step('Setup: Crear ejercicio con casos UTF-8 y saltos de línea', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de ejercicio con UTF-8');

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

          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title: exTitle,
              description: 'Invierte cada línea manteniendo la codificación UTF-8 intacta.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              language: 'python',
              difficulty: 'hard',
              time_limit_ms: 2000,
              reference_solution: 'import sys\nfor line in sys.stdin:\n    print(line.strip()[::-1])\n',
              test_cases: [
                {
                  input_data: 'Hola Mundo 👋🤖\n12345678901234567890\n',
                  expected_output: '🤖👋 odnuM aloH\n09876543210987654321',
                  is_hidden: false,
                  weight: 1.0,
                  order_index: 1
                }
              ]
            })
          });
          const exJson = await exResp.json();
          const exId = exJson.data?.id || exJson.id;
          await fetch(`/api/v1/exercises/${exId}/publish`, { method: 'POST', headers });

          return { courseId, exerciseId: exId };
        }, {
          cName: courseName,
          cCode: courseCode,
          teacherId: CANONICAL_ACTORS.teacher.id,
          studentId: CANONICAL_ACTORS.student.id,
          exTitle: exerciseTitle
        });

        sharedState.courseId = setupRes.courseId;
        sharedState.exerciseId = setupRes.exerciseId;
      });

      // Estudiante envía solución
      await test.step('Paso 1: Estudiante envía solución y obtiene veredicto AC', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante envía solución UTF-8');

        const code = 'import sys\nfor line in sys.stdin:\n    print(line.strip()[::-1])\n';
        const res = await studentPage.evaluate(async ({ exId, solution }) => {
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
              source_code_b64: btoa(unescape(encodeURIComponent(solution)))
            })
          });
          const json = await resp.json();
          return json.data || json;
        }, { exId: sharedState.exerciseId, solution: code });

        expect(res.verdict).toBe('AC');
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
