import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const RESULTS_DIR = path.resolve(__dirname, '../test-results');
export const FORENSIC_FAILURE_PATH = path.join(RESULTS_DIR, 'forensic-last-failure.json');

export function createForensicMonitor() {
  const httpFailures = [];
  const consoleErrors = [];
  const pageExceptions = [];
  const stepTrail = [];

  function attachToPage(page, roleLabel) {
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push({
          timestamp: new Date().toISOString(),
          role: roleLabel,
          url: page.url(),
          text: msg.text()
        });
      }
    });

    page.on('pageerror', (err) => {
      pageExceptions.push({
        timestamp: new Date().toISOString(),
        role: roleLabel,
        url: page.url(),
        message: err.message,
        stack: err.stack || ''
      });
    });

    page.on('response', async (response) => {
      const status = response.status();
      if (status < 400) return;
      const req = response.request();
      const url = response.url();
      // Ignore expected font/static aborts if any
      if (url.includes('fonts.googleapis.com') || url.includes('fonts.gstatic.com')) return;

      let responseBody = '';
      try {
        responseBody = await response.text();
      } catch {
        responseBody = '<unreadable response body>';
      }

      let requestBody = '';
      try {
        requestBody = req.postData() || '';
      } catch {
        requestBody = '';
      }

      httpFailures.push({
        timestamp: new Date().toISOString(),
        role: roleLabel,
        method: req.method(),
        url,
        status,
        requestBody: requestBody.slice(0, 2000),
        responseBody: responseBody.slice(0, 4000)
      });
    });
  }

  function recordStep(stepName, details = {}) {
    stepTrail.push({
      timestamp: new Date().toISOString(),
      step: stepName,
      ...details
    });
  }

  async function flushOnTestEnd(testInfo, activePages = {}) {
    if (testInfo.status === testInfo.expectedStatus) {
      return;
    }

    fs.mkdirSync(RESULTS_DIR, { recursive: true });

    const pageSnapshots = {};
    for (const [role, page] of Object.entries(activePages)) {
      if (!page || page.isClosed()) continue;
      try {
        const screenshotFile = path.join(RESULTS_DIR, `failure-${role}.png`);
        await page.screenshot({ path: screenshotFile, fullPage: true });
        pageSnapshots[role] = {
          url: page.url(),
          title: await page.title().catch(() => ''),
          screenshot: screenshotFile
        };
      } catch {
        pageSnapshots[role] = { url: page.url() };
      }
    }

    const report = {
      failedAt: new Date().toISOString(),
      testTitle: testInfo.title,
      status: testInfo.status,
      error: testInfo.error
        ? {
            message: testInfo.error.message,
            stack: testInfo.error.stack
          }
        : null,
      lastSteps: stepTrail.slice(-15),
      httpFailures,
      consoleErrors,
      pageExceptions,
      pageSnapshots
    };

    fs.writeFileSync(FORENSIC_FAILURE_PATH, JSON.stringify(report, null, 2), 'utf8');
  }

  return {
    attachToPage,
    recordStep,
    flushOnTestEnd,
    getHttpFailures: () => httpFailures,
    getConsoleErrors: () => consoleErrors,
    getPageExceptions: () => pageExceptions
  };
}
