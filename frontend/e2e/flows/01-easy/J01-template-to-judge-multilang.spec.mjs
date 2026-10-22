import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('J01 — Plantilla Homologada Multi-Lenguaje -> Juez Real (Python + C++)', () => {
  const runId = Date.now().toString().slice(-6);
  const templateName = `Juez Oficial MultiLang #${runId}`;
  const courseName = `Algoritmos Multi-Lenguaje #${runId}`;
  const courseCode = `ALG-ML-${runId}`;
  const exerciseTitle = `Suma Multi-Lenguaje #${runId}`;

  const pythonSolution = 'import sys\ndata = sys.stdin.read().strip().split()\nif len(data) >= 2:\n    print(int(data[0]) + int(data[1]))\n';
  const cppSolution = '#include <iostream>\nusing namespace std;\nint main() {\n    long long a, b;\n    if (cin >> a >> b) {\n        cout << (a + b) << "\\n";\n    }\n    return 0;\n}\n';

  const sharedState = {
    courseId: '',
    exerciseId: ''
  };

  test('Admin homologa plantilla en Docker, Docente publica por UI y Estudiante resuelve desde UI en C++ y Python', async ({ browser }, testInfo) => {
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
      await test.step('Paso 1 (Admin UI): Homologa plantilla de Juez en Template Studio y crea materia', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin abre Template Studio y verifica plantilla de Juez en Docker');

        await adminPage.goto('/admin/plantillas/nueva');
        await expect(adminPage.locator('h1.page-title')).toContainText('Registrar Nueva Plantilla Oficial');

        const judgeCard = adminPage.locator('.purpose-option-card', { hasText: 'Evaluador Automático de Ejercicios' });
        await expect(judgeCard).toBeVisible();
        await judgeCard.click();

        const pythonCard = adminPage.locator('.tech-component-card', { hasText: 'Python 3.12' }).first();
        await expect(pythonCard).toBeVisible();
        await pythonCard.click();

        await adminPage.locator('#studio-tpl-name').fill(templateName);

        const sidebar = adminPage.locator('studio-summary-sidebar');
        await expect(sidebar).toContainText(templateName);

        const verifyBtn = sidebar.locator('button', { hasText: 'Comprobar funcionamiento' });
        await expect(verifyBtn).toBeEnabled();
        await verifyBtn.click();
        await expect(sidebar.locator('.verification-status-box')).toContainText('Entorno Verificado', { timeout: 60000 });

        const publishBtn = sidebar.locator('button', { hasText: 'Publicar Plantilla Oficial' });
        await expect(publishBtn).toBeEnabled();
        await publishBtn.click();
        await expect(adminPage).toHaveURL(/\/admin\/plantillas/);

        forensic.recordStep('Admin crea materia en UI y matricula al estudiante');
        await adminPage.goto('/admin/cursos');
        await adminPage.getByRole('button', { name: /Nuevo Curso/i }).click();

        const modal = adminPage.locator('course-create-modal .modal-dialog');
        await expect(modal).toBeVisible();
        await modal.locator('#courseName').fill(courseName);
        await modal.locator('#courseCode').fill(courseCode);

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
        const [createCourseRes] = await Promise.all([
          adminPage.waitForResponse((res) => res.url().includes('/api/v1/subjects') && res.request().method() === 'POST'),
          submitCourseBtn.click()
        ]);
        const courseBody = await createCourseRes.json();
        sharedState.courseId = courseBody?.id || courseBody?.data?.id || '';
        expect(sharedState.courseId).toBeTruthy();

        const enrollStatus = await adminPage.evaluate(
          async ({ courseId, studentId }) => {
            const token = window.sessionStorage.getItem('solv_token');
            const resp = await fetch(`/api/v1/subjects/${courseId}/enroll`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
                ...(token ? { Authorization: `Bearer ${token}` } : {})
              },
              body: JSON.stringify({ student_id: studentId })
            });
            return resp.status;
          },
          { courseId: sharedState.courseId, studentId: CANONICAL_ACTORS.student.id }
        );
        expect([200, 201]).toContain(enrollStatus);
      });

      await test.step('Paso 2 (Docente UI): Crea ejercicio en 3 pasos vinculado a plantilla y publica', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente abre editor de ejercicios en 3 pasos');

        await teacherPage.goto(`/teacher/cursos/${sharedState.courseId}`, { waitUntil: 'domcontentloaded' });
        await expect(teacherPage.locator('h1.page-title')).toContainText(courseName, { timeout: 45000 });

        const newLabBtn = teacherPage.locator('button.btn-new-lab', { hasText: 'Nuevo Laboratorio' });
        await expect(newLabBtn).toBeVisible({ timeout: 15000 });
        await newLabBtn.click();
        await expect(teacherPage).toHaveURL(new RegExp(`/teacher/courses/${sharedState.courseId}/exercises/new`));

        await teacherPage.locator('#exercise-title').fill(exerciseTitle);

        const diffInput = teacherPage.locator('#exercise-difficulty input[role="combobox"]');
        await diffInput.click();
        await teacherPage.locator('#exercise-difficulty [role="option"]', { hasText: 'Fácil' }).first().click();

        const langInput = teacherPage.locator('#exercise-language input[role="combobox"]');
        await langInput.click();
        await teacherPage.locator('#exercise-language [role="option"]', { hasText: 'Python' }).first().click();

        await teacherPage.locator('#statement-description').fill('Dados dos enteros separados por espacio, imprime su suma exacta.');

        // El límite de tiempo del juez se aplica al ciclo completo del contenedor
        // (arranque + ejecución). Con 1000 ms la compilación de C++ compite con el
        // arranque de Docker y el caso muere por TLE de forma intermitente.
        await teacherPage.locator('#time-limit').fill('5000');

        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();

        // Paso 2: Solución de referencia y casos
        await expect(teacherPage.locator('#reference-solution-section')).toBeVisible();
        const cmContent = teacherPage.locator('#reference-solution-section .cm-content').first();
        await cmContent.click();
        await teacherPage.keyboard.press('Control+A');
        await teacherPage.keyboard.insertText(pythonSolution);

        const addCaseBtn = teacherPage.locator('cases-table button', { hasText: 'Agregar Caso' });
        await addCaseBtn.click();

        const caseRows = teacherPage.locator('cases-table tbody tr.case-row');
        await expect(caseRows).toHaveCount(1);
        await caseRows.first().locator('td.col-input textarea').fill('10 25');
        await caseRows.first().locator('td.col-output textarea').fill('35');

        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();
        await expect(teacherPage.locator('publication-checklist')).toBeVisible();

        const ackCheckbox = teacherPage.locator('publication-checklist input.ack-checkbox');
        if (await ackCheckbox.isVisible().catch(() => false)) {
          await ackCheckbox.check();
        }

        const publishStepBtn = teacherPage.locator('button.btn-publish-step', { hasText: 'Publicar' });
        await expect(publishStepBtn).toBeEnabled();
        await publishStepBtn.click();

        const confirmPublishBtn = teacherPage.locator('confirm-modal button', { hasText: 'Publicar ahora' });
        await expect(confirmPublishBtn).toBeVisible();

        const [publishResp] = await Promise.all([
          teacherPage.waitForResponse((r) => r.url().includes('/publish') && r.request().method() === 'POST'),
          confirmPublishBtn.click()
        ]);

        const matchExId = publishResp.url().match(/\/api\/v1\/exercises\/([^/]+)\/publish/);
        const publishJson = await publishResp.json().catch(() => ({}));
        sharedState.exerciseId = matchExId ? matchExId[1] : (publishJson.data?.id || publishJson.id);
        expect(sharedState.exerciseId).toBeTruthy();
        await expect(teacherPage.locator('.published-pill')).toContainText('Publicado');
      });

      await test.step('Paso 3 (Estudiante UI): Abre Resolver desde el Mapa Curricular y obtiene AC en C++ y en Python', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre curso, hace clic en Resolver y evalúa en C++ y Python');

        await studentPage.goto(`/student/courses/${sharedState.courseId}`);
        const curricularMap = studentPage.locator('curricular-map');
        await expect(curricularMap).toBeVisible();

        const exerciseRow = curricularMap.locator('.exercise-row', { hasText: exerciseTitle });
        await expect(exerciseRow).toBeVisible();
        await exerciseRow.locator('a.btn-solve').click();

        await expect(studentPage).toHaveURL(new RegExp(`/student/exercises/${sharedState.exerciseId}`));
        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(exerciseTitle);

        // 1. Cambiar a C++ y enviar solución compilada en contenedor gcc
        const langCombo = solver.locator('.lang-selector-group combobox');
        await langCombo.locator('input[role="combobox"]').click();
        await langCombo.locator('[role="option"]', { hasText: /^CPP$/i }).click();
        await expect(solver.locator('.file-label')).toContainText('solucion.cpp');

        const studentEditor = solver.locator('code-editor .cm-content').first();
        await expect(studentEditor).toBeVisible();
        await studentEditor.click();
        await studentPage.keyboard.press('Control+A');
        await studentPage.keyboard.insertText(cppSolution);

        await solver.locator('[data-testid="btn-submit-solution"]').click();
        const resultPanel = solver.locator('[data-testid="judge-result-panel"]');
        await expect(resultPanel).toBeVisible({ timeout: 45000 });
        await expect(resultPanel.locator('verdict-badge').first()).toContainText('AC');
        await expect(resultPanel.locator('.submission-saved-banner')).toContainText('Entrega registrada');

        // 2. Cambiar a Python y enviar solución interpretada
        await langCombo.locator('input[role="combobox"]').click();
        await langCombo.locator('[role="option"]', { hasText: /^PYTHON$/i }).click();
        await expect(solver.locator('.file-label')).toContainText('solucion.python');

        await studentEditor.click();
        await studentPage.keyboard.press('Control+A');
        await studentPage.keyboard.insertText(pythonSolution);

        await solver.locator('[data-testid="btn-submit-solution"]').click();
        await expect(resultPanel.locator('verdict-badge').first()).toContainText('AC', { timeout: 45000 });
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
