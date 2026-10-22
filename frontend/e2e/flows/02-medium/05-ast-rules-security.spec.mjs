import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 02 (Media): Reglas AST y Barrera de Seguridad Estática', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Seguridad y Sandbox E2E #${runId}`;
  const courseCode = `AST-${runId}`;
  const exerciseTitle = `Ejercicio con AST Seguro #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Bloquea código malicioso o no autorizado antes de la ejecución en sandbox', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(studentPage, 'student');

    try {
      // Setup de ejercicio con AST estricto
      await test.step('Setup: Crear ejercicio con reglas AST restrictivas', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de ejercicio con reglas AST');

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
              description: 'Algoritmo puro sin librerías de sistema ni introspección.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              language: 'python',
              difficulty: 'hard',
              time_limit_ms: 1000,
              reference_solution: 'print("SAFE")\n',
              ast_rules: {
                block_native_sort: false,
                block_system_modules: true,
                forbidden_imports: ['os', 'sys', 'subprocess', 'socket', 'shutil'],
                forbidden_functions: ['eval', 'exec', '__import__', 'open']
              },
              test_cases: [
                { input_data: '1', expected_output: 'SAFE', is_hidden: false, weight: 1.0, order_index: 1 }
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

      // Intento 1: Import de OS
      await test.step('Paso 1: Intento de importar módulo OS -> AST_BLOCKED', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante envía import os -> AST_BLOCKED');

        const badCode = 'import os\nprint("SAFE")\n';
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
        }, { exId: sharedState.exerciseId, code: badCode });

        expect(res.verdict).toBe('AST_BLOCKED');
      });

      // Intento 2: Uso de eval
      await test.step('Paso 2: Intento de usar eval() dinámico -> AST_BLOCKED', async () => {
        forensic.recordStep('Estudiante envía eval() -> AST_BLOCKED');

        const badCode = 'eval("print(\'SAFE\')")\n';
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
        }, { exId: sharedState.exerciseId, code: badCode });

        expect(res.verdict).toBe('AST_BLOCKED');
      });

      // Solución limpia
      await test.step('Paso 3: Envío de solución limpia sin violaciones -> AC', async () => {
        forensic.recordStep('Estudiante envía código limpio -> AC');

        const safeCode = 'print("SAFE")\n';
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
        }, { exId: sharedState.exerciseId, code: safeCode });

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
