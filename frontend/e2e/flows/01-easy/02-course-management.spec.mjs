import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 01 (Fácil): Gestión de Materias y Matriculación de Estudiantes', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Programación I E2E #${runId}`;
  const courseCode = `PROG1-${runId}`;

  let sharedCourseId = '';

  test('Admin crea materia, asigna docente titular y matricula al estudiante', async ({ browser }, testInfo) => {
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
      // PASO 1: ADMIN CREA LA MATERIA MEDIANTE UI
      // =========================================================================
      await test.step('Paso 1: Admin abre /admin/cursos y crea la materia', async () => {
        forensic.recordStep('Admin crea materia desde interfaz');
        await adminPage.goto('/admin/cursos', { waitUntil: 'networkidle' });
        await expect(adminPage).toHaveURL(/\/admin\/cursos/);
        await expect(adminPage.locator('h1.page-title')).toContainText('Cursos y Periodos Académicos');

        // Abrir modal de creación
        const newCourseBtn = adminPage.locator('button.btn-primary', { hasText: 'Nuevo Curso' });
        await newCourseBtn.click();

        const modal = adminPage.locator('course-create-modal .modal-dialog');
        await expect(modal).toBeVisible();

        await modal.locator('#courseName').fill(courseName);
        await modal.locator('#courseCode').fill(courseCode);

        // Seleccionar Docente Titular (Alvaro Rivera)
        const teacherComboInput = modal
          .locator('form-field', { hasText: 'Docente Titular Asignado' })
          .locator('input[role="combobox"]');
        await teacherComboInput.click();
        await teacherComboInput.fill(CANONICAL_ACTORS.teacher.email);
        const teacherOption = modal
          .locator('form-field', { hasText: 'Docente Titular Asignado' })
          .locator('[role="option"]')
          .first();
        await expect(teacherOption).toBeVisible();
        await teacherOption.click();

        const submitCourseBtn = modal.locator('button.btn-primary', { hasText: 'Crear Curso' });
        const [createResp] = await Promise.all([
          adminPage.waitForResponse(r => r.url().includes('/api/v1/subjects') && r.request().method() === 'POST'),
          submitCourseBtn.click()
        ]);

        const createJson = await createResp.json();
        sharedCourseId = createJson.data?.id || createJson.id;
        expect(sharedCourseId).toBeTruthy();
      });

      // =========================================================================
      // PASO 2: MATRICULAR ESTUDIANTE Y COMPROBAR VISIBILIDAD MULTI-ROL
      // =========================================================================
      await test.step('Paso 2: Matricular estudiante y verificar visibilidad para Docente y Alumno', async () => {
        forensic.recordStep('Matriculación y comprobación multi-rol');

        // Matricular vía API
        await adminPage.evaluate(async ({ courseId, studentId }) => {
          const token = window.sessionStorage.getItem('solv_token');
          await fetch(`/api/v1/subjects/${courseId}/enroll`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            },
            body: JSON.stringify({ student_id: studentId })
          });
        }, { courseId: sharedCourseId, studentId: CANONICAL_ACTORS.student.id });

        // Docente ve la materia en su panel
        await teacherPage.bringToFront();
        await teacherPage.goto('/teacher/dashboard');
        await expect(teacherPage.getByText(courseName).first()).toBeVisible({ timeout: 15000 });

        // Estudiante ve la materia en su lista
        await studentPage.bringToFront();
        await studentPage.goto('/student', { waitUntil: 'networkidle' });
        await expect(studentPage).toHaveURL(/\/student/);
        await expect(studentPage.getByText(courseName).first()).toBeVisible({ timeout: 15000 });
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
    } finally {
      await adminSession.context.close();
      await teacherSession.context.close();
      await studentSession.context.close();
    }
  });
});
