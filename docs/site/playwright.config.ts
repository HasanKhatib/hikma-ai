import { defineConfig, devices } from "@playwright/test";

// Tests run against the production build served like GitHub Pages, under /hikma-ai/.
const PORT = 4310;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : "list",
  use: {
    baseURL: `http://localhost:${PORT}/hikma-ai/`,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "phone-chromium", use: { ...devices["Pixel 7"] } },
    // Other engines run weekly and on demand (CROSS_BROWSER=1), not on every change.
    ...(process.env.CROSS_BROWSER
      ? [
          { name: "firefox", use: { ...devices["Desktop Firefox"] } },
          { name: "webkit", use: { ...devices["Desktop Safari"] } },
          { name: "phone-webkit", use: { ...devices["iPhone 14"] } },
        ]
      : []),
  ],
  webServer: {
    command: `npm run docs:build && npx vitepress preview . --port ${PORT} --strictPort`,
    url: `http://localhost:${PORT}/hikma-ai/`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
