import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:8088'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  retries: 1,
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'desktop-1440', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    { name: 'mobile-390', use: { ...devices['Pixel 5'], viewport: { width: 390, height: 844 } } },
  ],
})
