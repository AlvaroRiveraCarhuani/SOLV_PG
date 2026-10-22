import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';
import {
  authorJudgeExercise,
  createCourseViaAdminUI,
  enrollStudentViaAPI,
  typeInCodeMirror
} from '../../fixtures/authoring.fixture.mjs';

test.describe.configure({ mode: 'serial' });

/**
 * Barrera de seguridad AST.
 *
 * Comportamiento real del motor (verificado contra el stack vivo, no supuesto):
 *
 *  1. Las "Restricciones Específicas" que el docente registra con el botón
 *     "Prohibir" se persisten en `ast_rules.custom_rules` y el pre-chequeo las
 *     evalúa (GOB-SEGURO-02 cerrado): la comparación es por identificador
 *     completo y descarta comentarios y literales, de modo que `is_sorted` no
 *     dispara la prohibición de `sorted`.
 *  2. El veredicto AST_BLOCKED también lo emite el pre-chequeo Semgrep propio de
 *     la plataforma (`internal/infrastructure/semgrep/rules`), que corre antes
 *     que las reglas personalizadas y bloquea `import os`, `import subprocess`,
 *     `eval`, `exec` y `__import__` antes de levantar el contenedor.
 *  3. La política "Restringir Módulos del Sistema Operativo" se aplica en el
 *     filtro regex del analizador estático y produce un veredicto distinto:
 *     AST_VIOLATION, que la UI presenta con la misma semántica de bloqueo
 *     ("Regla de código"), nunca como "Sin evaluar" (GOB-SEGURO-03).
 *
 * Por eso este spec cubre los tres caminos: el bloqueo por regla personalizada
 * del docente, el bloqueo por la barrera de plataforma Semgrep y el
 * AST_VIOLATION de la política de módulos del sistema.
 */
test.describe('J04 — Barrera de Seguridad AST (código bloqueado antes de la ejecución)', () => {
  const runId = Date.now().toString().slice(-6);
  const courseName = `Barrera AST #${runId}`;
  const courseCode = `AST-${runId}`;
  const exerciseTitle = `Doble con Módulos Prohibidos #${runId}`;
  const policyExerciseTitle = `Doble con Política de Sistema #${runId}`;
  const customRuleExerciseTitle = `Doble con sorted Prohibido #${runId}`;

  const referenceSolution = 'n = int(input())\nprint(n * 2)\n';
  const exampleCase = { input: '5', expectedOutput: '10' };
  const hiddenCases = [{ input: '909090', expectedOutput: '1818180' }];
  const hiddenValues = hiddenCases.flatMap((c) => [c.input, c.expectedOutput]);

  const sharedState = {
    courseId: '',
    exerciseId: '',
    policyExerciseId: '',
    customRuleExerciseId: ''
  };

  test('Docente declara módulos peligrosos y el estudiante es bloqueado con AST_BLOCKED sin ejecutar el contenedor', async ({ browser }, testInfo) => {
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
      await test.step('Setup (Admin UI): Crear materia y asignar al docente titular', async () => {
        await adminPage.bringToFront();
        forensic.recordStep('Admin crea la materia para la barrera AST');

        sharedState.courseId = await createCourseViaAdminUI(adminPage, {
          courseName,
          courseCode,
          teacherEmail: CANONICAL_ACTORS.teacher.email
        });
        expect(sharedState.courseId).toBeTruthy();
      });

      await test.step('Setup: Matricular al estudiante canónico (no existe UI de matrícula)', async () => {
        await enrollStudentViaAPI(studentPage, sharedState.courseId, CANONICAL_ACTORS.student.id);
      });

      await test.step('Docente UI: Publicar el ejercicio declarando módulos peligrosos en el ast-rules-builder', async () => {
        await teacherPage.bringToFront();
        forensic.recordStep('Docente declara os, subprocess y eval como prohibidos y publica');

        sharedState.exerciseId = await authorJudgeExercise(teacherPage, {
          courseId: sharedState.courseId,
          courseName,
          title: exerciseTitle,
          statement:
            '### Problema\n\n' +
            'Dado un entero `n`, imprime su doble.\n\n' +
            '### Restricciones de Código\n\n' +
            'Esta prohibido usar modulos del sistema operativo (os, subprocess) y funciones dinamicas (eval).',
          referenceSolution,
          exampleCase,
          hiddenCases,
          astRules: { prohibitedPatterns: 'os, subprocess, eval' }
        });
        expect(sharedState.exerciseId).toBeTruthy();
      });

      await test.step('Docente UI: Publicar un segundo ejercicio con la política de módulos del sistema activa', async () => {
        forensic.recordStep('Docente activa la política Restringir Módulos del Sistema Operativo');

        sharedState.policyExerciseId = await authorJudgeExercise(teacherPage, {
          courseId: sharedState.courseId,
          courseName,
          title: policyExerciseTitle,
          statement:
            '### Problema\n\n' +
            'Dado un entero `n`, imprime su doble.\n\n' +
            '### Restricciones de Código\n\n' +
            'Esta prohibido usar modulos del sistema operativo.',
          referenceSolution,
          exampleCase,
          hiddenCases,
          astRules: { blockSystemModules: true }
        });
        expect(sharedState.policyExerciseId).toBeTruthy();
      });

      await test.step('Docente UI: Publicar un tercer ejercicio con una regla personalizada fuera del alcance de Semgrep', async () => {
        forensic.recordStep('Docente declara sorted como prohibido (regla personalizada)');

        sharedState.customRuleExerciseId = await authorJudgeExercise(teacherPage, {
          courseId: sharedState.courseId,
          courseName,
          title: customRuleExerciseTitle,
          statement:
            '### Problema\n\n' +
            'Dado un entero `n`, imprime su doble.\n\n' +
            '### Restricciones de Código\n\n' +
            'Esta prohibido usar la funcion `sorted`.',
          referenceSolution,
          exampleCase,
          hiddenCases,
          astRules: { prohibitedPatterns: 'sorted' }
        });
        expect(sharedState.customRuleExerciseId).toBeTruthy();
      });

      const openExerciseAsStudent = async (exerciseId, exerciseTitleToFind) => {
        await studentPage.goto(`/student/courses/${sharedState.courseId}`);
        const curricularMap = studentPage.locator('curricular-map');
        await expect(curricularMap).toBeVisible({ timeout: 45000 });

        const publicPayload = studentPage.waitForResponse(
          (res) =>
            res.request().method() === 'GET' &&
            new RegExp(`/api/v1/exercises/${exerciseId}(\\?|$)`).test(res.url())
        );

        const exerciseRow = curricularMap.locator('.exercise-row', { hasText: exerciseTitleToFind });
        await expect(exerciseRow).toBeVisible({ timeout: 45000 });
        await exerciseRow.locator('a.btn-solve').click();

        const [publicRes] = await Promise.all([publicPayload]);
        const publicJson = await publicRes.json();
        const publicBody = publicJson?.data ?? publicJson;
        const publicRaw = JSON.stringify(publicBody);

        // Aislamiento: sin solución de referencia, sin casos de prueba y sin datos ocultos.
        expect(publicBody).not.toHaveProperty('reference_solution');
        expect(publicBody).not.toHaveProperty('test_cases');
        expect(publicRaw).not.toContain('"visibility":"hidden"');
        expect(publicRaw).not.toContain('print(n * 2)');
        for (const value of hiddenValues) {
          expect(publicRaw).not.toContain(value);
        }

        const solver = studentPage.locator('student-exercise-solver');
        await expect(solver.locator('h1.page-title')).toContainText(exerciseTitleToFind, { timeout: 45000 });
        await expect(solver.locator('.example-box')).toHaveCount(1, { timeout: 45000 });
        await expect(studentPage.locator('body')).not.toContainText('print(n * 2)');
        for (const value of hiddenValues) {
          await expect(studentPage.locator('body')).not.toContainText(value);
        }
      };

      await test.step('Estudiante abre el ejercicio protegido y el payload público no filtra nada', async () => {
        await studentPage.bringToFront();
        forensic.recordStep('Estudiante abre el ejercicio con restricciones AST');

        await openExerciseAsStudent(sharedState.exerciseId, exerciseTitle);
      });

      const submitAndExpect = async (code, expectedVerdict) => {
        const solver = studentPage.locator('student-exercise-solver');
        const editor = solver.locator('code-editor .cm-content').first();
        await expect(editor).toBeVisible({ timeout: 45000 });

        await typeInCodeMirror(studentPage, editor, code);

        const evalResponse = studentPage.waitForResponse(
          (res) => res.url().includes('/api/v1/evaluations') && res.request().method() === 'POST',
          { timeout: 120000 }
        );

        await solver.locator('[data-testid="btn-submit-solution"]').click();

        const [evalRes] = await Promise.all([evalResponse]);
        expect(evalRes.ok()).toBeTruthy();

        const evalJson = await evalRes.json();
        const evalBody = evalJson?.data ?? evalJson;
        const evalRaw = JSON.stringify(evalBody);

        expect(evalBody.verdict).toBe(expectedVerdict);
        expect(evalRaw).not.toContain('print(n * 2)');
        for (const value of hiddenValues) {
          expect(evalRaw).not.toContain(value);
        }

        const resultPanel = solver.locator('[data-testid="judge-result-panel"]');
        await expect(resultPanel).toBeVisible({ timeout: 45000 });
        await expect(resultPanel.locator('.verdict-acronym').first()).toHaveText(expectedVerdict, {
          timeout: 45000
        });

        return evalBody;
      };

      await test.step('Bloqueo AST_BLOCKED: import de módulo peligroso detectado antes de ejecutar', async () => {
        forensic.recordStep('Estudiante usa import os -> AST_BLOCKED sin llegar a ejecutar el contenedor');

        const result = await submitAndExpect("import os\nos.system('ls')\n", 'AST_BLOCKED');

        // El mensaje identifica la regla violada.
        expect(result.message).toContain('Violación de seguridad AST (Semgrep)');
        expect(result.message).toContain("Import of 'os' module is forbidden");

        // El contenedor nunca se levantó: 0 ms de ejecución y ningún caso evaluado.
        expect(result.execution_time_ms).toBe(0);
        expect(result.case_results ?? []).toHaveLength(0);
        expect(result.actual_json).toContain('solv-python-forbidden-import-os');
      });

      await test.step('Bloqueo AST_BLOCKED: eval() también queda detenido', async () => {
        forensic.recordStep('Estudiante usa eval() -> AST_BLOCKED');

        const result = await submitAndExpect('n = eval(input())\nprint(n * 2)\n', 'AST_BLOCKED');
        expect(result.message).toContain('Use of eval() is forbidden');
        expect(result.execution_time_ms).toBe(0);
      });

      await test.step('Control: la solución limpia sin imports prohibidos es aceptada', async () => {
        forensic.recordStep('Control: solución correcta sin módulos prohibidos -> AC');

        await submitAndExpect(referenceSolution, 'AC');
      });

      await test.step('AST_VIOLATION: la política de módulos del sistema del docente también bloquea', async () => {
        forensic.recordStep('Ejercicio con política de sistema: import os -> AST_VIOLATION');

        await openExerciseAsStudent(sharedState.policyExerciseId, policyExerciseTitle);

        const result = await submitAndExpect('import os\nn = int(input())\nprint(n * 2)\n', 'AST_VIOLATION');
        expect(result.message).toContain("Importación prohibida de 'os'");
        expect(result.execution_time_ms).toBe(0);

        // GOB-SEGURO-03: la UI presenta el bloqueo con la misma semántica de
        // AST_BLOCKED y nunca como "Sin evaluar".
        const resultPanel = studentPage.locator('[data-testid="judge-result-panel"]');
        await expect(resultPanel).toBeVisible({ timeout: 45000 });
        await expect(resultPanel.locator('.verdict-human-label').first()).toHaveText('Regla de código', {
          timeout: 45000
        });
        await expect(resultPanel.locator('verdict-badge .verdict-badge.verdict-ast').first()).toBeVisible({ timeout: 45000 });
        await expect(resultPanel).not.toContainText('Sin evaluar');
      });

      await test.step('GOB-SEGURO-02: la regla personalizada del docente bloquea con AST_BLOCKED', async () => {
        forensic.recordStep('Estudiante usa sorted() -> AST_BLOCKED por regla personalizada');

        await openExerciseAsStudent(sharedState.customRuleExerciseId, customRuleExerciseTitle);

        const result = await submitAndExpect(
          'valores = [3, 1, 2]\nprint(sorted(valores))\n',
          'AST_BLOCKED'
        );

        // El mensaje identifica la regla personalizada del docente.
        expect(result.message).toContain('regla AST declarada por el docente');
        expect(result.message).toContain('sorted');

        // El payload estructurado permite auditar qué regla se violó.
        expect(result.actual_json).toContain('custom_rule:python:function:sorted');
        expect(result.execution_time_ms).toBe(0);
      });

      await test.step('Control: la solución limpia del ejercicio con regla personalizada es aceptada', async () => {
        forensic.recordStep('Control: solución sin la función prohibida -> AC');

        await submitAndExpect(referenceSolution, 'AC');
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
