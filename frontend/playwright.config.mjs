import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.spec.mjs',
  workers: 1,
  use: {
    baseURL: 'http://127.0.0.1:4300',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: 'npm run start -- --host 127.0.0.1 --port 4300',
    url: 'http://127.0.0.1:4300',
    timeout: 120_000,
    reuseExistingServer: !process.env.CI,
  },
});