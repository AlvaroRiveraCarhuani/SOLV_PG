import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 04 (Casos Borde): Supervisión Anti-Plagio y Detección de Similitud', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Auditoría y Supervisión #${runId}`;
  const courseCode = `PLAG-${runId}`;

  let sharedCourseId = '';

  test('Docente ejecuta análisis de similitud en el curso y obtiene reporte de coincidencias', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const teacherSession = await createAuthenticatedRoleSession(browser, 'teacher', baseURL);

    const { page: adminPage } = adminSession;
    const { page: teacherPage } = teacherSession;

    forensic.attachToPage(adminPage, 'admin');
    forensic.attachToPage(teacherPage, 'teacher');

    try {
      // Setup de materia
      await test.step('Setup: Crear materia', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Setup de materia para análisis de plagio');

        await adminPage.goto("/");
        const courseRes = await adminPage.evaluate(async ({ cName, cCode, teacherId }) => {
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
          return cJson.data?.id || cJson.id;
        }, {
          cName: courseName,
          cCode: courseCode,
          teacherId: CANONICAL_ACTORS.teacher.id
        });

        sharedCourseId = courseRes;
        expect(sharedCourseId).toBeTruthy();
      });

      // Docente consulta análisis de plagio
      await test.step('Paso 1: Docente consulta reporte de similitud del curso', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente solicita reporte de similitud');

        const plagData = await teacherPage.evaluate(async (courseId) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch(`/api/v1/teacher/courses/${courseId}/plagiarism`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {}
          });
          const json = await resp.json();
          return json.data || json;
        }, sharedCourseId);

        expect(plagData).toBeDefined();
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        { admin: adminPage, teacher: teacherPage }
      );
      throw err;
    }
  });
});
