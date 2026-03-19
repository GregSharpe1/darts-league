import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  use: {
    baseURL: 'http://127.0.0.1:4174',
    trace: 'on-first-retry',
  },
  webServer: [
    {
      command: 'go run ./cmd/api',
      cwd: '../backend',
      url: 'http://127.0.0.1:8081/healthz',
      reuseExistingServer: false,
      env: {
        HTTP_ADDRESS: '127.0.0.1:8081',
        DATABASE_URL: 'postgres://postgres:postgres@127.0.0.1:1/darts_league?sslmode=disable',
        ADMIN_USERNAME: 'admin',
        ADMIN_PASSWORD: 'change-me',
        ADMIN_SESSION_SECRET: 'playwright-secret',
      },
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 4174',
      cwd: '.',
      url: 'http://127.0.0.1:4174',
      reuseExistingServer: false,
      env: {
        API_PROXY_TARGET: 'http://127.0.0.1:8081',
      },
    },
  ],
})
