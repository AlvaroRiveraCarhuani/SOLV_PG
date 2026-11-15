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
  // Los flujos levantan contenedores reales y algunos encadenan diez pasos
  // (J04, J05). Cuatro minutos quedaban justos cuando la máquina venía de las
  // suites de integración; seis dan margen sin relajar ninguna aserción.
  timeout: 360_000,
  expect: {
    // El conjunto corre con contenedores reales y la máquina viene cargada de
    // las suites de integración. Quince segundos dejaban sin margen a las
    // pantallas que resuelven datos contra la API después del primer render.
    timeout: 30_000
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
