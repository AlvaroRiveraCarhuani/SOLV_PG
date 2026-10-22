import { expect, test } from '@playwright/test';
import {
  CANONICAL_ACTORS,
  createAuthenticatedRoleSession
} from '../../fixtures/auth-roles.fixture.mjs';
import { createForensicMonitor } from '../../fixtures/forensic-logger.fixture.mjs';

test.describe.configure({ mode: 'serial' });

test.describe('Categoría 01 (Fácil): Catálogo de Tecnologías y Plantillas Oficiales', () => {
  const runId = Date.now().toString().slice(-6);
  const tplJudgeName = `Python 3.12 Runner #${runId}`;
  const tplWorkspaceName = `Node 20 Fullstack #${runId}`;

  test('Admin explora catálogo y crea plantillas oficiales de Juez y Workspace', async ({ browser }, testInfo) => {
    const forensic = createForensicMonitor();
    const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';

    const adminSession = await createAuthenticatedRoleSession(browser, 'admin', baseURL);
    const { page: adminPage } = adminSession;
    forensic.attachToPage(adminPage, 'admin');

    try {
      // =========================================================================
      // PASO 1: VERIFICAR CATÁLOGO DE TECNOLOGÍAS
      // =========================================================================
      await test.step('Paso 1: Admin verifica el Catálogo de Tecnologías y componentes base', async () => {
        forensic.recordStep('Admin abre Biblioteca de Modelos y Categorías');
        await adminPage.goto('/admin/modelos-categorias');
        await expect(adminPage.locator('h1.page-title')).toContainText('Biblioteca de Modelos y Categorías');

        // Cambiar a la pestaña de Catálogo de Tecnologías
        const techTabBtn = adminPage.locator('button.tab-btn', { hasText: 'Catálogo de Tecnologías' });
        await expect(techTabBtn).toBeVisible();
        await techTabBtn.click();

        // Verificar presencia de tecnologías oficiales
        await expect(adminPage.getByText('Python 3.12').first()).toBeVisible();
      });

      // =========================================================================
      // PASO 2: CREAR PLANTILLA DE EVALUACIÓN (JUEZ_EFIMERO)
      // =========================================================================
      await test.step('Paso 2: Admin crea y homologa plantilla oficial de Juez Efímero', async () => {
        forensic.recordStep('Admin registra plantilla de Juez Efímero');
        await adminPage.goto('/admin/plantillas/nueva');
        await expect(adminPage.locator('h1.page-title')).toContainText('Registrar Nueva Plantilla Oficial');

        // Seleccionar Evaluador Automático de Ejercicios
        const judgeCard = adminPage.locator('.purpose-option-card', { hasText: 'Evaluador Automático de Ejercicios' });
        await expect(judgeCard).toBeVisible();
        await judgeCard.click();

        // Seleccionar Python 3.12
        const pythonCard = adminPage.locator('.tech-component-card', { hasText: 'Python 3.12' }).first();
        await expect(pythonCard).toBeVisible();
        await pythonCard.click();

        // Nombre
        await adminPage.locator('#studio-tpl-name').fill(tplJudgeName);

        // Sidebar
        const sidebar = adminPage.locator('studio-summary-sidebar');
        await expect(sidebar).toContainText(tplJudgeName);

        // Comprobar funcionamiento
        const verifyBtn = sidebar.locator('button', { hasText: 'Comprobar funcionamiento' });
        await expect(verifyBtn).toBeEnabled();
        await verifyBtn.click();
        await expect(sidebar.locator('.verification-status-box')).toContainText('Entorno Verificado', { timeout: 30000 });

        // Publicar
        const publishBtn = sidebar.locator('button', { hasText: 'Publicar Plantilla Oficial' });
        await expect(publishBtn).toBeEnabled();
        await publishBtn.click();

        await expect(adminPage).toHaveURL(/\/admin\/plantillas/);
        await expect(adminPage.getByText(tplJudgeName).first()).toBeVisible();
      });
    } catch (err) {
      await forensic.flushOnTestEnd(
        {
          ...testInfo,
          status: 'failed',
          expectedStatus: 'passed',
          error: { message: err?.message || String(err), stack: err?.stack || '' }
        },
        { admin: adminPage }
      );
      throw err;
    } finally {
      await adminSession.context.close();
    }
  });
});
