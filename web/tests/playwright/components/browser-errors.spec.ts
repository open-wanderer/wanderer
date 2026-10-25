import { test, expect } from '../fixtures';
import { watchBrowserErrors } from '../browser_errors';

for (const source of ['pageerror', 'console.error'] as const) {
    test(`guard rejects ${source} from a second page`, async ({ browser }) => {
        const context = await browser.newContext();
        const guard = watchBrowserErrors(context);
        try {
            await context.newPage();
            const page = await context.newPage();
            if (source === 'pageerror') {
                const error = page.waitForEvent('pageerror');
                await page.evaluate(() => { setTimeout(() => { throw new Error('guard sentinel'); }, 0); });
                await error;
            } else {
                const error = page.waitForEvent('console', message => message.type() === 'error');
                await page.evaluate(() => console.error('guard sentinel'));
                await error;
            }
            expect(() => guard.assertClean()).toThrow('guard sentinel');
        } finally {
            guard.dispose();
            await context.close();
        }
    });
}

test('guard scopes an exception to its source, message and URL', async ({ browser }) => {
    const context = await browser.newContext();
    const guard = watchBrowserErrors(context, [{
        source: 'console.error', message: /^expected sentinel$/, url: /^about:blank$/,
        issue: 'https://github.com/open-wanderer/wanderer/issues/948',
    }]);
    try {
        const page = await context.newPage();
        await page.evaluate(() => console.error('expected sentinel'));
        guard.assertClean();
        await page.evaluate(() => console.error('unexpected sentinel'));
        expect(() => guard.assertClean()).toThrow('unexpected sentinel');
    } finally {
        guard.dispose();
        await context.close();
    }
});
