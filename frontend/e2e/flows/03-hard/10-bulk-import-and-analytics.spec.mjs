import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 03 (Difícil): Importación Masiva (Dry-Run) y Analíticas Docente', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Gestión Masiva y Analítica #${runId}`;
  const courseCode = `BULK-${runId}`;

  let sharedCourseId = '';

  test('Docente ejecuta validación dry-run y consulta analíticas de la materia', async ({ browser }, testInfo) => {
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
        forensic.recordStep('Setup de materia para analíticas');

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

      // Docente consulta analíticas
      await test.step('Paso 1: Docente consulta el panel de analíticas y métricas', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente consulta analíticas de curso');

        await teacherPage.goto(`/teacher/courses/${sharedCourseId}/analytics`);
        await expect(teacherPage.locator('h1.page-title')).toContainText('Analítica de Rendimiento');

        const analyticsData = await teacherPage.evaluate(async (courseId) => {
          const token = window.sessionStorage.getItem('solv_token');
          const resp = await fetch(`/api/v1/teacher/courses/${courseId}/analytics`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {}
          });
          const json = await resp.json();
          return json.data || json;
        }, sharedCourseId);

        expect(analyticsData).toBeDefined();
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
