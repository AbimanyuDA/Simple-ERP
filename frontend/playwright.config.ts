import { defineConfig, devices } from "@playwright/test";

const browserExecutable = process.env.E2E_BROWSER_EXECUTABLE;

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    launchOptions: browserExecutable ? { executablePath: browserExecutable } : undefined,
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "mobile", use: { ...devices["Pixel 7"] } },
  ],
  webServer: process.env.CI
    ? undefined
    : { command: "npm run dev", url: "http://localhost:3000", reuseExistingServer: true },
});
