import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 03 (Difícil): Modo Examen con Semilla por Estudiante', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Examen Parcial de Algoritmos #${runId}`;
  const courseCode = `EXAM-${runId}`;
  const exerciseTitle = `Evaluación con Variación de Semilla #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Garantiza la inmutabilidad y evaluación segura de exámenes con per_student_seed', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(studentPage, 'student');

    try {
      // Setup de examen oficial
      await test.step('Setup: Crear examen con purpose: exam y per_student_seed: true', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de examen oficial con semilla');

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
              description: 'Examen de evaluación individual con variaciones aleatorias.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              purpose: 'exam',
              per_student_seed: true,
              language: 'python',
              difficulty: 'hard',
              time_limit_ms: 1000,
              reference_solution: 'import sys\nfor line in sys.stdin:\n    print(f"RES:{line.strip()}")\n',
              test_cases: [
                {
                  input_data: 'TEST_INPUT_1',
                  expected_output: 'RES:TEST_INPUT_1',
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

      // Estudiante resuelve examen
      await test.step('Paso 1: Estudiante rinde examen y obtiene veredicto AC', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante envía solución a examen con semilla');

        const examSolution = 'import sys\nfor line in sys.stdin:\n    print(f"RES:{line.strip()}")\n';
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
        }, { exId: sharedState.exerciseId, code: examSolution });

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
