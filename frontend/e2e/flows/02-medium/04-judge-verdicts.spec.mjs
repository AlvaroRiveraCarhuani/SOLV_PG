import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 02 (Media): Matriz de Veredictos del Juez (AC, WA, RE, TLE)', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Algoritmos y Veredictos E2E #${runId}`;
  const courseCode = `VERD-${runId}`;
  const exerciseTitle = `Validador Multi-Veredicto #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Comprueba que el motor de evaluación clasifica exactamente cada veredicto', async ({ browser }, testInfo) => {
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
      // SETUP: CREAR MATERIA Y EJERCICIO
      // =========================================================================
      await test.step('Setup: Crear materia y publicar ejercicio con casos ocultos', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin crea materia para veredictos');

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

          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title: exTitle,
              description: 'Devuelve la suma de dos números.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              template_id: runner.id,
              language: 'python',
              difficulty: 'medium',
              // El límite incluye el arranque del contenedor del juez.
              time_limit_ms: 15000,
              reference_solution: 'a, b = map(int, input().split())\nprint(a + b)\n',
              test_cases: [
                { input: '10 20', expected_output: '30', is_hidden: false, weight: 1.0, order_index: 1 },
                { input: '50 50', expected_output: '100', is_hidden: true, weight: 1.0, order_index: 2 }
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
        expect(sharedState.exerciseId).toBeTruthy();
      });

      // =========================================================================
      // PRUEBAS DE VEREDICTOS
      // =========================================================================
      await test.step('Paso 1: Evaluar solución con WA (Wrong Answer)', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante envía solución incorrecta -> WA');

        const waCode = 'a, b = map(int, input().split())\nprint(a - b)\n';
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
        }, { exId: sharedState.exerciseId, code: waCode });

        expect(res.verdict).toBe('WA');
      });

      await test.step('Paso 2: Evaluar solución con RE (Runtime Error)', async () => {
        forensic.recordStep('Estudiante envía solución con fallo de ejecución -> RE');

        const reCode = 'a, b = map(int, input().split())\nprint([][100])\n';
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
        }, { exId: sharedState.exerciseId, code: reCode });

        expect(res.verdict).toBe('RE');
      });

      await test.step('Paso 3: Evaluar solución con TLE (Time Limit Exceeded)', async () => {
        forensic.recordStep('Estudiante envía solución con timeout -> TLE');

        const tleCode = 'import time\ntime.sleep(20)\n';
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
        }, { exId: sharedState.exerciseId, code: tleCode });

        expect(res.verdict).toBe('TLE');
      });

      await test.step('Paso 4: Evaluar solución con AC (Accepted)', async () => {
        forensic.recordStep('Estudiante envía solución óptima -> AC');

        const acCode = 'a, b = map(int, input().split())\nprint(a + b)\n';
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
        }, { exId: sharedState.exerciseId, code: acCode });

        expect(res.verdict).toBe('AC');
        expect(res.case_results?.length).toBe(2);
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
    }
  });
});
