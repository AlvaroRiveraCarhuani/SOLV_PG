import { defineConfig } from '@playwright/test';

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:4300';

export default defineConfig({
  testDir: './e2e',
  testMatch: 'typography.spec.mjs',
  workers: 1,
  use: {
    baseURL,
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: 'npm run start -- --host 127.0.0.1 --port 4300',
    url: baseURL,
    // El arranque en frío compila la aplicación y puede superar los dos minutos.
    timeout: 300_000,
    reuseExistingServer: true,
    // El servidor de desarrollo toma PORT por encima de --port; con PORT=0 elige
    // un puerto aleatorio y la espera sobre 4300 nunca se cumple.
    env: { ...process.env, PORT: '4300' },
  },
});