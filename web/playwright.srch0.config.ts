import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
    testDir: './tests/srch0',
    testMatch: 'browser.spec.ts',
    outputDir: 'test-results/srch0-browser-artifacts',
    fullyParallel: false,
    workers: 1,
    globalSetup: './tests/srch0/browser-setup.ts',
    timeout: 60000,
    expect: { timeout: 15000 },
    reporter: [['list'], ['json', { outputFile: 'test-results/srch0-browser.json' }]],
    use: {
        ...devices['Desktop Chrome'],
        locale: 'en', timezoneId: 'Europe/Zurich',
        launchOptions: { args: ['--use-gl=angle', '--use-angle=swiftshader', '--enable-unsafe-swiftshader'] },
        trace: 'retain-on-failure',
    },
});
