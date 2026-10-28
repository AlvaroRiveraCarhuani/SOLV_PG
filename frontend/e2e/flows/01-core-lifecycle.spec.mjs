import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Flujo Core 01: Admin (Constructor) -> Docente (Curso y Ejercicio) -> Estudiante (Evaluación AC) -> SpeedGrader', () => {
  const runId = Date.now().toString().slice(-6);
  const templateName = `E2E Dual Python 3.12 #${runId}`;
  const courseName = `Algoritmos y Programación E2E #${runId}`;
  const courseCode = `E2E-${runId}`;
  const exerciseTitle = `Suma de Dos Enteros #${runId}`;
  const canonicalSolution = 'a, b = map(int, input().split())\nprint(a + b)\n';

  let sharedState = {
    templateId: '',
    courseId: '',
    exerciseId: '',
    submissionId: ''
  };

  test('Ejecuta el ciclo multi-rol completo con corte inmediato ante fallos', async ({ browser }, testInfo) => {
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
      // ETAPA 1: ADMINISTRADOR — CATÁLOGO Y CONSTRUCTOR DE ENTORNOS
      // =========================================================================
      await test.step('Etapa 1.1: Admin verifica el Catálogo de Tecnologías (/admin/modelos-categorias)', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin abre Biblioteca de Modelos y Categorías');
        await adminPage.goto('/admin/modelos-categorias');
        await expect(adminPage.locator('h1.page-title')).toContainText('Biblioteca de Modelos y Categorías');

        const techTab = adminPage.locator('button, [role="tab"]', { hasText: 'Catálogo de Tecnologías' }).first();
        if (await techTab.isVisible()) {
          await techTab.click();
        }
        await expect(adminPage.getByText('Python 3.12').first()).toBeVisible();
      });

      await test.step('Etapa 1.2: Admin construye, verifica y publica una Plantilla Oficial Dual (/admin/plantillas/nueva)', async () => {
        forensic.recordStep('Admin abre Constructor de Entornos');
        await adminPage.goto('/admin/plantillas/nueva');
        await expect(adminPage.locator('h1.page-title')).toContainText('Registrar Nueva Plantilla Oficial');

        // Seleccionar tarjeta "Laboratorio de Práctica" y activar interruptor de evaluación (AMBOS)
        const labCard = adminPage.locator('.purpose-option-card', { hasText: 'Laboratorio de Práctica' });
        await expect(labCard).toBeVisible();
        await labCard.click();
        await expect(labCard).toHaveClass(/selected/);

        const dualToggle = adminPage.locator('#dual-evaluation-toggle');
        await expect(dualToggle).toBeVisible();
        await dualToggle.check();

        // Seleccionar Python 3.12 en Columna 1
        const pythonCard = adminPage.locator('.tech-component-card', { hasText: 'Python 3.12' }).first();
        await expect(pythonCard).toBeVisible();
        await pythonCard.click();
        await expect(pythonCard).toHaveClass(/selected/);

        // Asignar nombre único de plantilla
        const nameInput = adminPage.locator('#studio-tpl-name');
        await nameInput.fill(templateName);

        // Seleccionar primera categoría académica disponible si el combobox tiene opciones
        const categoryInput = adminPage.locator('#studio-tpl-category input[role="combobox"]');
        if (await categoryInput.isVisible()) {
          await categoryInput.click();
          const firstOption = adminPage.locator('#studio-tpl-category [role="option"]').first();
          if (await firstOption.isVisible().catch(() => false)) {
            await firstOption.click();
          }
        }

        // Verificar síntesis en el panel derecho
        const sidebar = adminPage.locator('studio-summary-sidebar');
        await expect(sidebar).toContainText(templateName);
        await expect(sidebar).toContainText('python:3.12-slim-bookworm');

        // Comprobar funcionamiento si el botón está habilitado (o requerido antes de publicar)
        const publishBtn = sidebar.locator('button', { hasText: 'Publicar Plantilla Oficial' });
        const verifyBtn = sidebar.locator('button', { hasText: 'Comprobar funcionamiento' });
        await expect(verifyBtn).toBeEnabled();

        // Ejecutar comprobación real de funcionamiento contra Docker
        forensic.recordStep('Admin ejecuta Comprobar funcionamiento en Constructor');
        await verifyBtn.click();
        await expect(sidebar.getByText('¡Entorno Verificado!')).toBeVisible({ timeout: 90_000 });

        // Publicar Plantilla Oficial y capturar el ID creado
        forensic.recordStep('Admin publica Plantilla Oficial');
        await expect(publishBtn).toBeEnabled();
        const [createTplResp] = await Promise.all([
          adminPage.waitForResponse(
            (r) => r.url().includes('/api/v1/admin/templates') && r.request().method() === 'POST'
          ),
          publishBtn.click()
        ]);

        expect(createTplResp.status()).toBeLessThan(300);
        const createdTplBody = await createTplResp.json().catch(() => ({}));
        sharedState.templateId = createdTplBody?.data?.id || createdTplBody?.id || '';

        // Verificar redirección al catálogo activo y presencia de la plantilla
        await expect(adminPage).toHaveURL(/\/admin\/plantillas/);
        const searchInput = adminPage.locator('search-bar input');
        await searchInput.fill(templateName);
        await expect(adminPage.getByText(templateName).first()).toBeVisible();
      });

      // =========================================================================
      // ETAPA 2: ADMINISTRADOR — ALTA DE CURSO Y MATRÍCULA DE ESTUDIANTE
      // =========================================================================
      await test.step('Etapa 2: Admin crea Curso Institucional con Docente Titular y matricula al Estudiante (/admin/cursos)', async () => {
        forensic.recordStep('Admin navega a /admin/cursos para crear materia');
        await adminPage.goto('/admin/cursos');
        await expect(adminPage.locator('h1.page-title')).toContainText('Cursos y Periodos Académicos');

        const newCourseBtn = adminPage.locator('button', { hasText: 'Nuevo Curso' });
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

        // Seleccionar Plantilla Docker Base recién creada
        const tplComboInput = modal
          .locator('form-field', { hasText: 'Plantilla Docker Base' })
          .locator('input[role="combobox"]');
        await tplComboInput.click();
        await tplComboInput.fill(templateName);
        const tplOption = modal
          .locator('form-field', { hasText: 'Plantilla Docker Base' })
          .locator('[role="option"]')
          .first();
        await expect(tplOption).toBeVisible();
        await tplOption.click();

        // Confirmar creación del curso y capturar su ID
        const submitCourseBtn = modal.locator('button', { hasText: 'Crear Curso' });
        const [createCourseResp] = await Promise.all([
          adminPage.waitForResponse(
            (r) => r.url().includes('/api/v1/subjects') && r.request().method() === 'POST'
          ),
          submitCourseBtn.click()
        ]);

        expect(createCourseResp.status()).toBe(201);
        const createdCourse = await createCourseResp.json();
        sharedState.courseId = createdCourse?.data?.id || createdCourse?.id || '';
        expect(sharedState.courseId).toBeTruthy();

        // Verificar que aparece en la tabla de cursos del Admin
        const courseSearch = adminPage.locator('search-bar input');
        await courseSearch.fill(courseCode);
        await expect(adminPage.locator('table.data-table')).toContainText(courseName);

        // Matricular al estudiante canónico en el curso recién creado
        forensic.recordStep('Admin matricula al estudiante en el curso creado', {
          courseId: sharedState.courseId,
          studentId: CANONICAL_ACTORS.student.id
        });
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
            return { status: resp.status, body: await resp.text() };
          },
          { courseId: sharedState.courseId, studentId: CANONICAL_ACTORS.student.id }
        );
        expect(enrollStatus.status).toBe(201);
      });

      // =========================================================================
      // ETAPA 3: DOCENTE — CREACIÓN Y PUBLICACIÓN DE EJERCICIO CON JUEZ VIRTUAL
      // =========================================================================
      await test.step('Etapa 3: Docente entra a su curso, crea un ejercicio con la plantilla homologada y lo publica', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente abre el detalle de su curso', { courseId: sharedState.courseId });
        await teacherPage.goto(`/teacher/cursos/${sharedState.courseId}`);
        await expect(teacherPage.locator('h1.page-title')).toContainText(courseName);

        // Abrir editor de nuevo ejercicio
        const newLabBtn = teacherPage.locator('button.btn-new-lab', { hasText: 'Nuevo Laboratorio' });
        await newLabBtn.click();
        await expect(teacherPage).toHaveURL(new RegExp(`/teacher/courses/${sharedState.courseId}/exercises/new`));

        // Paso 1: Identidad, Lenguaje y Enunciado
        forensic.recordStep('Docente completa Paso 1 del Editor de Ejercicios');
        await teacherPage.locator('#exercise-title').fill(exerciseTitle);

        // Seleccionar Dificultad Fácil
        const diffInput = teacherPage.locator('#exercise-difficulty input[role="combobox"]');
        await diffInput.click();
        await teacherPage.locator('#exercise-difficulty [role="option"]', { hasText: 'Fácil' }).first().click();

        // Seleccionar Lenguaje de ejecución (Python)
        const langInput = teacherPage.locator('#exercise-language input[role="combobox"]');
        await langInput.click();
        const pythonOption = teacherPage
          .locator('#exercise-language [role="option"]')
          .filter({ hasText: /Python/i })
          .first();
        await expect(pythonOption).toBeVisible();
        await pythonOption.click();

        // Agregar etiqueta temática
        const tagInput = teacherPage.locator('#exercise-tag-input');
        if (await tagInput.isVisible()) {
          await tagInput.fill('aritmetica');
          await tagInput.press('Enter');
        }

        // Escribir enunciado
        const statementArea = teacherPage.locator('#statement-description');
        await statementArea.fill(
          'Dados dos números enteros separados por espacio, imprime su suma exacta en la salida estándar.'
        );

        // Avanzar al Paso 2
        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();

        // Paso 2: Solución de Referencia y Casos de Prueba
        forensic.recordStep('Docente completa Paso 2 (Solución de referencia y casos de prueba)');
        await expect(teacherPage.locator('#reference-solution-section')).toBeVisible();

        // Escribir la solución canónica en el editor CodeMirror
        const cmContent = teacherPage.locator('#reference-solution-section .cm-content').first();
        await cmContent.click();
        await teacherPage.keyboard.press('Control+A');
        await teacherPage.keyboard.type(canonicalSolution);

        // Agregar Caso #1 (Ejemplo) y Caso #2 (Oculto)
        const addCaseBtn = teacherPage.locator('cases-table button', { hasText: 'Agregar Caso' });
        await addCaseBtn.click();
        await addCaseBtn.click();

        const caseRows = teacherPage.locator('cases-table tbody tr.case-row');
        await expect(caseRows).toHaveCount(2);

        // Llenar Caso #1 (stdin: "2 3", stdout: "5", visibilidad: Ejemplo)
        await caseRows.nth(0).locator('td.col-input textarea').fill('2 3');
        await caseRows.nth(0).locator('td.col-output textarea').fill('5');
        const visCombo0 = caseRows.nth(0).locator('td.col-vis input[role="combobox"]');
        if (await visCombo0.isVisible()) {
          await visCombo0.click();
          const exOpt = caseRows.nth(0).locator('td.col-vis [role="option"]', { hasText: /Ejemplo/i }).first();
          if (await exOpt.isVisible().catch(() => false)) {
            await exOpt.click();
          }
        }

        // Llenar Caso #2 (stdin: "10 25", stdout: "35", visibilidad: Oculto)
        await caseRows.nth(1).locator('td.col-input textarea').fill('10 25');
        await caseRows.nth(1).locator('td.col-output textarea').fill('35');
        const visCombo1 = caseRows.nth(1).locator('td.col-vis input[role="combobox"]');
        if (await visCombo1.isVisible()) {
          await visCombo1.click();
          const hidOpt = caseRows.nth(1).locator('td.col-vis [role="option"]', { hasText: /Oculto/i }).first();
          if (await hidOpt.isVisible().catch(() => false)) {
            await hidOpt.click();
          }
        }

        // Avanzar al Paso 3
        await teacherPage.locator('button.step-btn', { hasText: 'Siguiente' }).click();

        // Paso 3: Checklist y Publicación
        forensic.recordStep('Docente verifica Checklist en Paso 3 y publica el ejercicio');
        await expect(teacherPage.locator('publication-checklist')).toBeVisible();
        await expect(teacherPage.locator('publication-checklist')).toContainText('Sin Bloqueantes');

        // Si existen advertencias pedagógicas, aceptar el checkbox de confirmación
        const ackCheckbox = teacherPage.locator('publication-checklist input.ack-checkbox');
        if (await ackCheckbox.isVisible().catch(() => false)) {
          await ackCheckbox.check();
        }

        const publishStepBtn = teacherPage.locator('button.btn-publish-step', { hasText: 'Publicar' });
        await expect(publishStepBtn).toBeEnabled();
        await publishStepBtn.click();

        // Confirmar en el modal "Publicar ahora"
        const confirmPublishBtn = teacherPage.locator('confirm-modal button', { hasText: 'Publicar ahora' });
        await expect(confirmPublishBtn).toBeVisible();

        const [publishResp] = await Promise.all([
          teacherPage.waitForResponse(
            (r) => r.url().includes('/publish') && r.request().method() === 'POST'
          ),
          confirmPublishBtn.click()
        ]);

        expect(publishResp.status()).toBe(200);

        // Extraer el exerciseId desde la URL de publicación (/api/v1/exercises/{id}/publish)
        const matchExId = publishResp.url().match(/\/api\/v1\/exercises\/([^/]+)\/publish/);
        sharedState.exerciseId = matchExId ? matchExId[1] : '';
        expect(sharedState.exerciseId).toBeTruthy();

        await expect(teacherPage.locator('.published-pill')).toContainText('Publicado');
      });

      // =========================================================================
      // ETAPA 4: ESTUDIANTE (MAPA CURRICULAR Y ENTREGA AC) + DOCENTE (SPEEDGRADER)
      // =========================================================================
      await test.step('Etapa 4.1: Estudiante ve el curso y el ejercicio en su Mapa Curricular y envía solución AC al Juez', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre su panel principal (/student)');
        await studentPage.goto('/student');
        await expect(studentPage.locator('.widget-labs')).toContainText(courseName);

        // Navegar al detalle del curso y Mapa Curricular
        await studentPage.locator('a.subject-name-link', { hasText: courseName }).click();
        await expect(studentPage).toHaveURL(new RegExp(`/student/courses/${sharedState.courseId}`));

        const curricularMap = studentPage.locator('curricular-map');
        await expect(curricularMap).toBeVisible();
        await expect(curricularMap.getByText(exerciseTitle)).toBeVisible();

        // Ejecutar la entrega del estudiante contra el motor de evaluación real y persistir la entrega
        forensic.recordStep('Estudiante envía solución en Python al Juez Virtual y registra su entrega', {
          exerciseId: sharedState.exerciseId
        });

        const evalAndSubmit = await studentPage.evaluate(
          async ({ exerciseId, studentId, sourceCode }) => {
            const token = window.sessionStorage.getItem('solv_token');
            const headers = {
              'Content-Type': 'application/json',
              ...(token ? { Authorization: `Bearer ${token}` } : {})
            };

            const sourceCodeB64 = btoa(sourceCode);
            const evalResp = await fetch('/api/v1/evaluations', {
              method: 'POST',
              headers,
              body: JSON.stringify({
                exercise_id: exerciseId,
                language: 'python',
                source_code_b64: sourceCodeB64
              })
            });
            const evalJson = await evalResp.json().catch(() => ({}));
            const evalData = evalJson?.data || evalJson;

            if (evalResp.status !== 200 || evalData?.verdict !== 'AC') {
              return {
                ok: false,
                stage: 'evaluation',
                status: evalResp.status,
                evalData
              };
            }

            const subResp = await fetch('/api/v1/submissions', {
              method: 'POST',
              headers,
              body: JSON.stringify({
                exercise_id: exerciseId,
                student_id: studentId,
                code: sourceCode,
                verdict: evalData.verdict,
                execution_time_ms: evalData.execution_time_ms || 15,
                memory_used_mb: Math.round(evalData.memory_used_mb || 12)
              })
            });
            const subData = await subResp.json().catch(() => ({}));

            return {
              ok: subResp.status === 201,
              stage: 'submission',
              evalStatus: evalResp.status,
              subStatus: subResp.status,
              verdict: evalData.verdict,
              submissionId: subData?.id || subData?.data?.id || ''
            };
          },
          {
            exerciseId: sharedState.exerciseId,
            studentId: CANONICAL_ACTORS.student.id,
            sourceCode: canonicalSolution
          }
        );

        expect(evalAndSubmit.ok, `Fallo en evaluación/entrega: ${JSON.stringify(evalAndSubmit)}`).toBe(true);
        expect(evalAndSubmit.verdict).toBe('AC');
        sharedState.submissionId = evalAndSubmit.submissionId;
        expect(sharedState.submissionId).toBeTruthy();
      });

      await test.step('Etapa 4.2: Docente verifica la entrega en la Cola de Entregas y la audita en SpeedGrader', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente abre Cola de Entregas del curso y entra a SpeedGrader', {
          submissionId: sharedState.submissionId
        });

        await teacherPage.goto(`/teacher/cursos/${sharedState.courseId}`);
        await expect(teacherPage.locator('h1.page-title')).toBeVisible();
        const submissionsTab = teacherPage.locator('button.tab-btn', { hasText: 'Cola de Entregas' });
        await expect(submissionsTab).toBeVisible();
        await submissionsTab.click();

        const submissionsTable = teacherPage.locator('.submissions-view table.data-table');
        await expect(submissionsTable).toBeVisible();
        await expect(submissionsTable).toContainText(exerciseTitle);
        await expect(submissionsTable).toContainText('AC');

        const speedGraderLink = submissionsTable
          .locator(`a[href*="/teacher/revision/${sharedState.submissionId}"]`)
          .first();
        await expect(speedGraderLink).toBeVisible();
        await speedGraderLink.click();

        await expect(teacherPage).toHaveURL(new RegExp(`/teacher/revision/${sharedState.submissionId}`));
        await expect(teacherPage.locator('speed-grader-header')).toContainText(exerciseTitle);
        await expect(teacherPage.locator('speed-grader-eval-panel .verdict-card')).toContainText('AC');
        await expect(teacherPage.locator('speed-grader-code-viewer')).toContainText('map(int, input().split())');
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        {
          admin: adminPage,
          teacher: teacherPage,
          student: studentPage
        }
      );
      throw err;
    } finally {
      await adminSession.context.close();
      await teacherSession.context.close();
      await studentSession.context.close();
    }
  });
});
