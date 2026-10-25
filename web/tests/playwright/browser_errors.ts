import { expect, type BrowserContext, type ConsoleMessage, type WebError } from '@playwright/test';

type BrowserError = { source: 'pageerror' | 'console.error'; message: string; url: string };

export type BrowserErrorAllowance = {
    source: BrowserError['source'];
    message: RegExp;
    url: RegExp;
    issue: string;
};

// Keep exceptions scoped to an error and resource URL, with a tracking issue.
// The default allowlist is empty: newly observed failures need investigation.
export function watchBrowserErrors(context: BrowserContext, allowlist: BrowserErrorAllowance[] = []) {
    for (const allowance of allowlist) {
        expect(allowance.issue, 'Browser error exceptions need a tracking issue').toMatch(
            /^https:\/\/github\.com\/[^/]+\/[^/]+\/(issues|pull)\/\d+$/,
        );
    }
    const errors: BrowserError[] = [];
    const record = (error: BrowserError) => {
        if (!allowlist.some(allowance => allowance.source === error.source
            && allowance.message.test(error.message) && allowance.url.test(error.url))) {
            errors.push(error);
        }
    };
    const onError = (error: WebError) => record({
        source: 'pageerror', message: error.error().message,
        url: error.location().url || error.page()?.url() || '',
    });
    const onConsole = (message: ConsoleMessage) => {
        if (message.type() === 'error') record({
            source: 'console.error', message: message.text(),
            url: message.location().url || message.page()?.url() || '',
        });
    };
    // Context events include additional pages and popups, not just the page fixture.
    context.on('weberror', onError);
    context.on('console', onConsole);
    return {
        assertClean() {
            expect(errors, 'Unexpected browser errors').toEqual([]);
        },
        dispose() {
            context.off('weberror', onError);
            context.off('console', onConsole);
        },
    };
}
