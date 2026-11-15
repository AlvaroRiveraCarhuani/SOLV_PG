import { defineConfig } from '@playwright/test';

const baseURL = process.env.E2E_BASE_URL || 'http://localhost:4200';
const slowMo = Number(process.env.E2E_SLOW_MO || 0);

export default defineConfig({
  testDir: './e2e/flows',
  testMatch: '**/*.spec.mjs',
  forbidOnly: !!process.env.CI,
  // Los archivos corren en paralelo; los pasos de un mismo flujo, no: cada
  // flujo levanta contenedores y comparte actores canónicos.
  fullyParallel: false,
  // En local, un solo worker. Tres navegadores en paralelo, más el servidor de
  // Angular y los contenedores del juez, agotan la memoria de la estación y el
  // navegador muere a mitad de un flujo: el error aparece como "Channel closed"
  // o "Target page, context or browser has been closed", que no describe la
  // causa real. Los runners de CI tienen más memoria y conservan el paralelismo.
  workers: Number(process.env.E2E_WORKERS || (process.env.CI ? 3 : 1)),
  // Una falla no aborta el resto: una corrida tiene que informar todos los
  // flujos que fallan, no el primero.
  maxFailures: 0,
  // Un fallo aislado por tiempos se repite solo y queda registrado como
  // inestable. En local el reintento duplica el coste de cada flujo caído, así
  // que solo CI reintenta.
  retries: process.env.CI ? 1 : 0,
  // Los flujos levantan contenedores reales y algunos encadenan diez pasos
  // (J04, J05). Cuatro minutos no alcanzan: J04 se cortó a los 240 s en la
  // corrida completa. Sin reintento local, el costo máximo por flujo es este.
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
    trace: process.env.CI ? 'retain-on-failure' : 'off',
    screenshot: 'only-on-failure',
    video: process.env.CI ? 'retain-on-failure' : 'off',
    launchOptions: {
      ...(slowMo > 0 ? { slowMo } : {}),
      args: [
        '--disable-dev-shm-usage',
        '--disable-gpu',
        '--no-sandbox',
        '--js-flags=--max-old-space-size=512'
      ]
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
    // El arranque en frío compila la aplicación completa y en esta estación
    // supera los dos minutos; con 120 s la corrida fallaba antes de ejecutar el
    // primer flujo. El servidor ya compilado arranca en segundos.
    timeout: 300_000,
    reuseExistingServer: true,
    // El servidor de desarrollo toma la variable PORT por encima de --port. Con
    // PORT=0 en el entorno elige un puerto aleatorio, la espera de webServer
    // sobre 4200 vence y ningún flujo llega a ejecutarse. Se fija explícita.
    env: { ...process.env, PORT: '4200' }
  }
});
