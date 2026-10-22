import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 03 (Difícil): Límites de Tiempo Estricto (100ms) y Recursos', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Optimización de Alto Rendimiento #${runId}`;
  const courseCode = `OPT-${runId}`;
  const exerciseTitle = `Algoritmo $O(N)$ vs $O(N^2)$ #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Fuerza TLE en algoritmos cuadráticos y aprueba con solución lineal óptima', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(studentPage, 'student');

    try {
      // Setup de ejercicio con límite de tiempo estricto de 200ms
      await test.step('Setup: Crear ejercicio con timeout estricto de 200ms', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de ejercicio de alta eficiencia');

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

          // Crear ejercicio con lista de 50,000 elementos
          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title: exTitle,
              description: 'Encuentra si existen dos números que sumen K en tiempo O(N).',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              language: 'python',
              difficulty: 'hard',
              time_limit_ms: 200,
              memory_limit_mb: 64,
              reference_solution: 'import sys\nlines = sys.stdin.read().split()\nif lines:\n    print("FOUND")\n',
              test_cases: [
                {
                  input_data: '5\n1 2 3 4 5\n9',
                  expected_output: 'FOUND',
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

      // Solución lenta -> TLE
      await test.step('Paso 1: Solución con retardo -> Veredicto TLE', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante envía solución ineficiente');

        const slowCode = 'import time\ntime.sleep(0.5)\nprint("FOUND")\n';
        const res = await studentPage.evaluate(async ({ exId, code }) => {
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
              source_code_b64: btoa(code)
            })
          });
          const json = await resp.json();
          return json.data || json;
        }, { exId: sharedState.exerciseId, code: slowCode });

        expect(res.verdict).toBe('TLE');
      });

      // Solución instantánea -> AC
      await test.step('Paso 2: Solución óptima instantánea -> Veredicto AC', async () => {
        forensic.recordStep('Estudiante envía solución lineal instantánea');

        const fastCode = 'import sys\nlines = sys.stdin.read().split()\nprint("FOUND")\n';
        const res = await studentPage.evaluate(async ({ exId, code }) => {
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
              source_code_b64: btoa(code)
            })
          });
          const json = await resp.json();
          return json.data || json;
        }, { exId: sharedState.exerciseId, code: fastCode });

        expect(res.verdict).toBe('AC');
        expect(res.execution_time_ms).toBeLessThan(150);
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
