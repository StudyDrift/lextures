import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost:5173'
const apiURL = process.env.E2E_API_URL ?? 'http://localhost:8080'

export default defineConfig({
  globalSetup: './global-setup.ts',
  testDir: './tests',
  // Spec files use unique users via fixtures; safe to parallelize tests within files.
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  // Local: single worker for easier debugging. CI: default (~50% of cores per shard job).
  ...(process.env.CI ? {} : { workers: 1 }),
  reporter: process.env.CI
    ? [
        ['blob'],
        ['github'],
      ]
    : 'list',
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  outputDir: 'test-results/',
})

export { baseURL, apiURL }
