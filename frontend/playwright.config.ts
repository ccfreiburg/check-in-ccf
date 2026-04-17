import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  timeout: 30_000,
  retries: 0,

  use: {
    baseURL: 'http://localhost:5180',
    trace: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    command: 'node_modules/.bin/vite --port 5180',
    url: 'http://localhost:5180',
    reuseExistingServer: true,
    timeout: 120_000,
    env: {
      VITE_HTTPS: 'false',
      VITE_POLL_INTERVAL: '9999999',
    },
  },
})
