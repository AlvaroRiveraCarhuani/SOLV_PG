import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 05 (Badges & Calificación): SpeedGrader, Comentarios y Override de Nota', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Evaluación Continua y Calificación #${runId}`;
  const courseCode = `CALIF-${runId}`;
  const exerciseTitle = `Práctica Calificada #${runId}`;

  let sharedState = {
    courseId: '',
    exerciseId: '',
    submissionId: ''
  };

  test('Docente califica, agrega comentarios por línea y ajusta nota en SpeedGrader', async ({ browser }, testInfo) => {
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
      // Setup de entrega
      await test.step('Setup: Crear curso, publicar ejercicio y registrar entrega', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de entrega para SpeedGrader');

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
          // Solo sirve un runner de juez: las plantillas de IDE no pueden
          // calificar entregas. Se prefiere el juez puro sobre el dual.
          const judgeRunners = (Array.isArray(tplItems) ? tplItems : []).filter(
            (t) =>
              /python/i.test(t.name || '') &&
              ['JUEZ_EFIMERO', 'AMBOS'].includes(t.target_environment || t.environment_type)
          );
          const runner =
            judgeRunners.find((t) => (t.target_environment || t.environment_type) === 'JUEZ_EFIMERO') ||
            judgeRunners[0];
          if (!runner?.id) {
            return { error: 'No hay plantilla de juez Python disponible' };
          }

          const sourceCode = '# Codigo del estudiante\nprint("OK")\n# Fin\n';
          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title: exTitle,
              description: 'Práctica para SpeedGrader.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              template_id: runner.id,
              language: 'python',
              difficulty: 'medium',
              time_limit_ms: 5000,
              reference_solution: 'print("OK")\n',
              test_cases: [
                { input: '1', expected_output: 'OK', is_hidden: false, weight: 1.0, order_index: 1 }
              ]
            })
          });
          const exJson = await exResp.json();
          const exerciseId = exJson.data?.id || exJson.id;
          await fetch(`/api/v1/exercises/${exerciseId}/publish`, { method: 'POST', headers });

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

        // La evaluación y la entrega se hacen con la sesión del estudiante: el
        // recibo del juez queda atado a quien evalúa y a quien entrega.
        await studentPage.goto('/');
        const submitRes = await studentPage.evaluate(
          async ({ exerciseId, sourceCode }) => {
            const token = window.sessionStorage.getItem('solv_token');
            const headers = {
              'Content-Type': 'application/json',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            };
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
            const subResp = await fetch('/api/v1/submissions', {
              method: 'POST',
              headers,
              body: JSON.stringify({
                exercise_id: exerciseId,
                code: sourceCode,
                verdict: evalData.verdict,
                evaluation_receipt: evalData.evaluation_receipt,
                score: 100,
                execution_time_ms: evalData.execution_time_ms || 20,
                memory_used_mb: Math.round(evalData.memory_used_mb || 12)
              })
            });
            const subJson = await subResp.json();
            return { status: subResp.status, id: subJson.id || subJson.data?.id, body: subJson };
          },
          { exerciseId: sharedState.exerciseId, sourceCode: '# Codigo del estudiante\nprint("OK")\n# Fin\n' }
        );
        expect(submitRes.error, JSON.stringify(submitRes)).toBeUndefined();
        expect(submitRes.id, JSON.stringify(submitRes.body)).toBeTruthy();
        sharedState.submissionId = submitRes.id;
      });

      // Docente abre SpeedGrader
      await test.step('Paso 1: Docente abre SpeedGrader y verifica visor de código', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente entra a SpeedGrader', { submissionId: sharedState.submissionId });

        await teacherPage.goto(`/teacher/revision/${sharedState.submissionId}`);
        await expect(teacherPage.locator('speed-grader-header')).toBeVisible();
        await expect(teacherPage.locator('speed-grader-code-viewer')).toBeVisible();
      });

      // Docente agrega comentario
      await test.step('Paso 2: Docente agrega comentario por línea en el código', async () => {
        forensic.recordStep('Docente registra comentario pedagógico');

        await teacherPage.evaluate(async ({ subId }) => {
          const token = window.sessionStorage.getItem('solv_token');
          await fetch(`/api/v1/submissions/${subId}/comments`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            },
            body: JSON.stringify({
              line_number: 2,
              comment: 'Excelente solución y salida limpia.'
            })
          });
        }, { subId: sharedState.submissionId });
      });

      // Override de nota
      await test.step('Paso 3: Docente realiza override manual de nota', async () => {
        forensic.recordStep('Docente ajusta nota manual');

        const overrideOk = await teacherPage.evaluate(async ({ subId }) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch(`/api/v1/submissions/${subId}/override`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            },
            body: JSON.stringify({
              verdict: 'AC',
              score: 98,
              override_reason: 'Reconocimiento por formato y claridad.'
            })
          });
          return resp.status === 200 || resp.status === 201;
        }, { subId: sharedState.submissionId });

        expect(overrideOk).toBe(true);
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
