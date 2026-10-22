import { defineConfig } from '@playwright/test';

const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';
const slowMo = Number(process.env.E2E_SLOW_MO || 0);

export default defineConfig({
  testDir: './e2e/flows',
  testMatch: '**/*.spec.mjs',
  forbidOnly: !!process.env.CI,
  fullyParallel: false,
  workers: 1,
  maxFailures: 1,
  retries: 0,
  timeout: 240_000,
  expect: {
    timeout: 15_000
  },
  outputDir: './e2e/test-results/artifacts',
  reporter: [
    ['list'],
    ['json', { outputFile: './e2e/test-results/playwright-report.json' }]
  ],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    launchOptions: {
      ...(slowMo > 0 ? { slowMo } : {}),
      args: ['--enable-features=NetworkServiceInProcess']
    }
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' }
    }
  ],
  webServer: {
    command: 'npm run start -- --host localhost --port 4200',
    url: baseURL,
    timeout: 120_000,
    reuseExistingServer: true
  }
});
