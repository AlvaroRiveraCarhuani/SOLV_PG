import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 02 (Media): Módulos Curriculares y Cadena de Desbloqueo', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Programación Avanzada E2E #${runId}`;
  const courseCode = `ADV-${runId}`;

  let sharedState = {
    courseId: '',
    mod1Id: '',
    mod2Id: '',
    ex1Id: ''
  };

  test('Configura prerrequisitos entre módulos y valida el desbloqueo secuencial', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const studentSession = await createAuthenticatedRoleSession(browser, 'student', baseURL);

    const { page: adminPage } = adminSession;
    const { page: studentPage } = studentSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(studentPage, 'student');

    try {
      // Setup de materia y módulos con dependencia
      await test.step('Setup: Crear materia, Módulo 1, Módulo 2 y vincular prerrequisito', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de módulos con dependencia');

        await adminPage.goto("/");
        const setupRes = await adminPage.evaluate(async ({ cName, cCode, teacherId, studentId }) => {
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

          // Crear Módulo 1
          const m1Resp = await fetch(`/api/v1/teacher/courses/${courseId}/modules`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
              title: 'Módulo 1: Inicial',
              description: 'Requisito previo obligatorio',
              order_index: 1,
              pass_score: 51
            })
          });
          const m1Json = await m1Resp.json();
          const m1Id = m1Json.data?.id || m1Json.id;

          // Crear Módulo 2
          const m2Resp = await fetch(`/api/v1/teacher/courses/${courseId}/modules`, {
            method: 'POST',
            headers,
            body: JSON.stringify({
              title: 'Módulo 2: Avanzado',
              description: 'Requiere Módulo 1',
              order_index: 2,
              pass_score: 51
            })
          });
          const m2Json = await m2Resp.json();
          const m2Id = m2Json.data?.id || m2Json.id;

          // Set Prerrequisito
          await fetch(`/api/v1/teacher/modules/${m2Id}/prerequisites`, {
            method: 'POST',
            headers,
            body: JSON.stringify({ prerequisite_module_ids: [m1Id] })
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

          // Crear ejercicio en Módulo 1
          const exResp = await fetch('/api/v1/exercises', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              subject_id: courseId,
              title: 'Práctica de Desbloqueo',
              description: 'Resuelve para desbloquear el Módulo 2.',
              type: 'algorithm',
              environment_type: 'JUEZ_EFIMERO',
              template_id: runner.id,
              language: 'python',
              difficulty: 'easy',
              time_limit_ms: 5000,
              reference_solution: 'print("DESBLOQUEADO")\n',
              test_cases: [
                { input: '1', expected_output: 'DESBLOQUEADO', is_hidden: false, weight: 1.0, order_index: 1 }
              ]
            })
          });
          const exJson = await exResp.json();
          const exId = exJson.data?.id || exJson.id;
          await fetch(`/api/v1/exercises/${exId}/publish`, { method: 'POST', headers });
          await fetch(`/api/v1/teacher/exercises/${exId}/module`, {
            method: 'PUT',
            headers,
            body: JSON.stringify({ module_id: m1Id })
          });

          return { courseId, mod1Id: m1Id, mod2Id: m2Id, exId };
        }, {
          cName: courseName,
          cCode: courseCode,
          teacherId: CANONICAL_ACTORS.teacher.id,
          studentId: CANONICAL_ACTORS.student.id
        });

        expect(setupRes.error, JSON.stringify(setupRes)).toBeUndefined();
        sharedState = setupRes;
      });

      // Verificar estado del estudiante
      await test.step('Paso 1: Estudiante verifica Módulo 1 disponible y Módulo 2 bloqueado', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante consulta mapa curricular');

        const mapData = await studentPage.evaluate(async (courseId) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch(`/api/v1/student/courses/${courseId}/map`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {}
          });
          const json = await resp.json();
          return json.data || json;
        }, sharedState.courseId);

        const mod1 = mapData.modules?.find(m => m.id === sharedState.mod1Id);
        expect(mod1?.state).toBe('unlocked');

        const mod2 = mapData.modules?.find(m => m.id === sharedState.mod2Id);
        expect(mod2?.state).toBe('locked');
      });

      // Estudiante aprueba y desbloquea
      await test.step('Paso 2: Estudiante aprueba ejercicio de Módulo 1 y desbloquea Módulo 2', async () => {
        forensic.recordStep('Estudiante entrega solución para desbloqueo');

        await studentPage.evaluate(async ({ exId, studentId }) => {
          const token = window.sessionStorage.getItem('solv_token');
          const headers = {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {})
          };

          const evalResp = await fetch('/api/v1/evaluations', {
            method: 'POST',
            headers,
            body: JSON.stringify({
              exercise_id: exId,
              language: 'python',
              source_code_b64: btoa('print("DESBLOQUEADO")\n')
            })
          });
          const evalJson = await evalResp.json();
          const evalData = evalJson.data || evalJson;

          if (evalData.verdict === 'AC') {
            await fetch('/api/v1/submissions', {
              method: 'POST',
              headers,
              body: JSON.stringify({
                exercise_id: exId,
                student_id: studentId,
                code: 'print("DESBLOQUEADO")\n',
                verdict: evalData.verdict,
                evaluation_receipt: evalData.evaluation_receipt,
                execution_time_ms: evalData.execution_time_ms || 10,
                memory_used_mb: Math.round(evalData.memory_used_mb || 10)
              })
            });
          }
        }, { exId: sharedState.exId, studentId: CANONICAL_ACTORS.student.id });

        // Comprobar mapa actualizado
        const updatedMap = await studentPage.evaluate(async (courseId) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch(`/api/v1/student/courses/${courseId}/map`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {}
          });
          const json = await resp.json();
          return json.data || json;
        }, sharedState.courseId);

        const mod1 = updatedMap.modules?.find(m => m.id === sharedState.mod1Id);
        expect(mod1?.state).toBe('completed');

        const mod2 = updatedMap.modules?.find(m => m.id === sharedState.mod2Id);
        // Sin ejercicios propios, el módulo desbloqueado queda completado.
        expect(mod2?.state).not.toBe('locked');
      });

      // Verificación en el DOM real del mapa curricular del estudiante
      await test.step('Paso 3: Estudiante ve en /student/courses/:id el módulo desbloqueado', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre el mapa curricular en la UI');

        await studentPage.goto(`/student/courses/${sharedState.courseId}`);

        const completedCard = studentPage.locator('.module-card.is-completed .module-title', {
          hasText: 'Módulo 1: Inicial'
        });
        await expect(completedCard).toBeVisible({ timeout: 45000 });

        // Sin ejercicios propios, el Módulo 2 queda completado al desbloquearse:
        // lo que importa es que ya no aparezca bloqueado.
        await expect(
          studentPage.locator('.module-card.is-locked', { hasText: 'Módulo 2: Avanzado' })
        ).toHaveCount(0);
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
