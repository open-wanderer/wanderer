import { expect, test as base } from '@playwright/test';
import { cases, label } from './fixtures';
import type { BrowserInput, BrowserObservation } from './types';

const test = base.extend({
    baseURL: async ({}, use) => {
        if (!process.env.SRCH0_BROWSER_URL) throw new Error('Der eigene SRCH0-Browserserver wurde nicht gestartet.');
        await use(process.env.SRCH0_BROWSER_URL);
    },
});

for (const fixture of cases<BrowserInput, BrowserObservation>(['browser'])) {
    test(label(fixture), async ({ page, context }) => {
        const observed = fixture.observed.browser_state;
        const searches: any[] = [];
        page.on('request', request => {
            if (new URL(request.url()).pathname === '/api/v1/search/trails' && request.method() === 'POST') searches.push(request.postDataJSON());
        });
        // Raster tiles and geolocation are not part of this contract. An empty
        // MapLibre style makes real map lifecycle deterministic and offline.
        await context.route('**/*', async route => {
            const url = new URL(route.request().url());
            if (url.pathname.startsWith('/styles/')) return route.fulfill({ json: { version: 8, sources: {}, layers: [] } });
            if (url.hostname === '127.0.0.1' || url.hostname === 'localhost') return route.continue();
            if (url.pathname.endsWith('.json')) return route.fulfill({ json: { version: 8, sources: {}, layers: [] } });
            return route.abort();
        });
        await context.addInitScript(storage => {
            if (!sessionStorage.getItem('srch0-initialized')) {
                localStorage.clear();
                for (const [key, value] of Object.entries(storage)) localStorage.setItem(key, String(value));
                sessionStorage.setItem('srch0-initialized', 'true');
            }
        }, fixture.input.storage);
        await page.goto(fixture.input.url);
        await expect.poll(() => searches.length).toBeGreaterThan(0);

        if (fixture.input.navigation === 'history-back') {
            // Use a real application link: SvelteKit invokes beforeNavigate,
            // captures the snapshot and restores it on browser Back.
            await page.locator('a[href="/login"]').first().click();
            await expect(page).toHaveURL(/\/login/);
            expect(await page.evaluate(() => localStorage.getItem('trailListFilter'))).toBeNull();
            await page.goBack();
            await expect.poll(() => searches.length).toBeGreaterThan(1);
        }

        if (fixture.context.surface === 'map') {
            await expect.poll(() => [...new URL(page.url()).searchParams.keys()].sort()).toEqual(observed.query_keys);
            expect(new URL(page.url()).pathname).toBe(observed.url_path);
            expect(searches.at(-1).options.sort).toEqual([observed.sort]);
            await expect(page.locator('.maplibregl-canvas').first()).toBeVisible();
        } else {
            await expect.poll(() => new URL(page.url()).pathname + new URL(page.url()).search).toBe(observed.url);
            await expect.poll(() => page.evaluate(() => JSON.parse(localStorage.getItem('trailListFilter') ?? 'null'))).toEqual(observed.filter);
            expect(searches.at(-1).options.hitsPerPage).toBe(observed.page_size);
            expect(searches.at(-1).q).toBe(observed.filter?.q);
        }
    });
}
