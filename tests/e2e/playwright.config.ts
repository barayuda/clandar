import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './specs',
  timeout: 30000,
  use: {
    baseURL: 'http://localhost:8080',
    headless: true,
  },
  webServer: {
    command: 'go run ./cmd/server',
    url: 'http://localhost:8080/health',
    reuseExistingServer: true,
    cwd: '../../',
    timeout: 30000,
  },
});
