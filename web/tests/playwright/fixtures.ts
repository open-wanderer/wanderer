import { test as base } from '@playwright/test';
import { watchBrowserErrors, type BrowserErrorAllowance } from './browser_errors';

export { expect, type Page } from '@playwright/test';

export const test = base.extend<{
    browserErrorAllowlist: BrowserErrorAllowance[];
    browserErrors: void;
}>({
    browserErrorAllowlist: [[], { option: true }],
    browserErrors: [async ({ context, browserErrorAllowlist }, use) => {
        const guard = watchBrowserErrors(context, browserErrorAllowlist);
        try {
            await use();
            guard.assertClean();
        } finally {
            guard.dispose();
        }
    }, { auto: true }],
});
