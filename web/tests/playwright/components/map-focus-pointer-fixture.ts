import { createHash } from 'node:crypto';
import { createServer } from 'node:http';
import { expect, type Page } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { buildMapPageFixture } from './map-page-fixture';

const component = (name: string) => fileURLToPath(new URL(`../../../src/lib/components/trail/${name}.svelte`, import.meta.url));
const dist = fileURLToPath(new URL('../../../node_modules/maplibre-gl/dist/', import.meta.url));

// Keep the real page, map engine, result links, selection and focus handlers.
// Only the two rendering children have fixed, representative card/row sizes.
export async function buildPointerFixture(dev: boolean) {
    const bundle = await buildMapPageFixture(dev, true, true, false, false, {
        [component('trail_card')]: `<script>let { trail } = $props();</script>
            <article data-testid="pointer-card" style="height:386px;border:1px solid #999;border-radius:1rem;box-sizing:border-box">
                <div style="height:192px;background:#9ab;border-radius:1rem 1rem 0 0"></div>
                <h4 style="padding:1rem">{trail.name}</h4>
            </article>`,
        [component('trail_list_item')]: `<script>let { trail } = $props();</script>
            <article data-testid="pointer-row" style="height:96px;padding:12px;box-sizing:border-box;display:flex;gap:12px">
                <div style="width:64px;height:64px;background:#9ab"></div><h4>{trail.name}</h4>
            </article>`,
    });
    console.log('POINTER_BUNDLE', JSON.stringify({
        dev, cache: 'bypassed for overrides',
        appSha256: createHash('sha256').update(await readFile(component('trail_map_results'))).digest('hex'),
        bundleSha256: createHash('sha256').update(bundle).digest('hex'),
    }));
    return bundle;
}

export type Position = { list: number; document: number };
export type PressRecord = Position & { type: string; button: number; tag: string; trail: string | null };
export const position = (page: Page): Promise<Position> => page.evaluate(() => ({
    list: document.querySelector<HTMLElement>('.trail-results')!.scrollTop,
    document: document.scrollingElement!.scrollTop,
}));

export async function mountPointerResults(page: Page, bundle: string, display = 'cards') {
    // Chromium can issue a native background-tab request before Playwright
    // attaches routing to that target. Serve its first document over real HTTP.
    const server = createServer((request, response) => {
        const path = new URL(request.url!, 'http://localhost').pathname;
        response.writeHead(path.startsWith('/map/trail/') ? 200 : 404, { 'Content-Type': 'text/html' });
        response.end(`<!doctype html><title>Opened trail</title><main data-testid="opened-trail">${path}</main>`);
    });
    await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve));
    page.context().once('close', () => server.close());
    const address = server.address() as { port: number };
    const origin = `http://127.0.0.1:${address.port}`;
    const records: PressRecord[] = [];
    page.on('pageerror', error => { throw error; });
    await page.exposeBinding('__recordPointer', (_source, record) => records.push(record));
    await page.addInitScript(() => {
        for (const type of ['mousedown', 'mouseup', 'pointerdown', 'pointerup', 'contextmenu']) {
            document.addEventListener(type, event => {
                const list = document.querySelector<HTMLElement>('.trail-results');
                if (!list) return;
                const target = event.target as Element;
                (window as any).__recordPointer({
                    type, button: (event as MouseEvent).button,
                    list: list.scrollTop, document: document.scrollingElement!.scrollTop,
                    tag: target.tagName, trail: target.closest('a')?.getAttribute('href') ?? null,
                });
            }, true);
        }
    });
    await page.context().route(url => url.origin === origin, async route => {
        const path = new URL(route.request().url()).pathname;
        if (path.startsWith('/map/trail/')) return route.fulfill({
            contentType: 'text/html',
            body: `<!doctype html><title>Opened trail</title><main data-testid="opened-trail">${path}</main>`,
        });
        if (path.startsWith('/maplibre/')) {
            const name = path.split('/').at(-1)!;
            return route.fulfill({
                contentType: name.endsWith('.css') ? 'text/css' : 'text/javascript',
                body: await readFile(dist + name, 'utf8'),
            });
        }
        if (path.startsWith('/styles/')) return route.fulfill({ json: {
            version: 8, sources: {},
            layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }],
        } });
        return route.fulfill({
            contentType: 'text/html',
            body: '<html><head><meta name="viewport" content="width=device-width, initial-scale=1"><link rel="stylesheet" href="/maplibre/maplibre-gl.css"></head><body><script type="module">import * as M from "/maplibre/maplibre-gl.mjs"; window.RealMapLibre=M;</script></body></html>',
        });
    });
    await page.context().route(url => url.origin !== origin, route => route.abort());
    await page.goto(origin + '/map?tl_lat=47.1&tl_lon=8.1&br_lat=46.9&br_lon=7.9');
    await expect.poll(() => page.evaluate(() => Boolean((window as any).RealMapLibre))).toBe(true);
    await page.evaluate(display => localStorage.setItem('mapResultsDisplay', display), display);
    await page.addScriptTag({ content: bundle });
    await expect.poll(() => page.evaluate(() => (window as any).fixture.publications.length)).toBeGreaterThan(0);
    await page.evaluate(() => {
        const fixture = (window as any).fixture;
        const reply = (request: any) => request.channel === 'map'
            ? { mapTrails: [], clusters: { type: 'FeatureCollection', features: [] } }
            : { items: Array.from({ length: 40 }, (_, index) => ({ id: 'pointer-' + index, author: 'tester', name: 'Trail ' + index })), page: 1, totalPages: 1, totalHits: 40 };
        const push = fixture.requests.push.bind(fixture.requests);
        fixture.requests.push = (...requests: any[]) => {
            const count = push(...requests);
            for (const request of requests) request.resolve(reply(request));
            return count;
        };
        for (const request of fixture.requests) request.resolve(reply(request));
    });
    await expect(page.locator('.trail-result')).toHaveCount(40);
    await expect(page.locator('.map-results')).toHaveAttribute('data-layout-ready', 'true');
    if (!(await page.locator('.trail-results').isVisible())) await page.locator('.map-view-toggle').tap();
    await expect(page.locator('.trail-results')).toBeVisible();
    return records;
}

export async function clippedPoint(page: Page, display: 'cards' | 'list', checkbox = false) {
    await page.mouse.move(1, 1);
    await page.evaluate(() => (document.activeElement as HTMLElement)?.blur());
    const point = await page.evaluate(({ display, checkbox }) => {
        const list = document.querySelector<HTMLElement>('.trail-results')!;
        const item = document.querySelectorAll<HTMLElement>('.trail-result')[6];
        const box = list.getBoundingClientRect();
        const top = box.top + list.clientTop;
        const bottom = top + list.clientHeight;
        const first = item.getBoundingClientRect();
        list.scrollTop += display === 'cards' ? first.top - (top - 120) : first.bottom - (bottom + 30);
        const card = item.getBoundingClientRect();
        return {
            x: display === 'cards' ? card.left + 32 : card.right - 32,
            y: display === 'cards' ? top + 32 : bottom - (checkbox ? 2 : 32),
            href: item.querySelector('a')!.getAttribute('href')!,
            top: card.top, bottom: card.bottom, listTop: top, listBottom: bottom,
        };
    }, { display, checkbox });
    expect(await page.evaluate(({ x, y }) => x >= 0 && x < innerWidth && y >= 0 && y < innerHeight, point), 'the native pointer target must be inside the visible window').toBe(true);
    await page.mouse.move(point.x, point.y);
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    return point;
}

export async function pressAt(page: Page, point: { x: number; y: number }, button: 'left' | 'middle' | 'right' = 'left') {
    await page.mouse.move(point.x, point.y);
    const before = await position(page);
    await page.mouse.down({ button });
    const held = await position(page);
    await page.mouse.up({ button });
    expect(held, 'pointer focus must preserve list and document position before release').toEqual(before);
    return before;
}

export async function expectStablePress(records: PressRecord[], before: Position, button = 0) {
    await expect.poll(() => records.filter(event => event.type === 'mouseup' && event.button === button).length).toBeGreaterThan(0);
    const down = records.find(event => event.type === 'mousedown' && event.button === button)!;
    const up = records.find(event => event.type === 'mouseup' && event.button === button)!;
    expect({ list: down.list, document: down.document }).toEqual(before);
    expect({ list: up.list, document: up.document }, 'release must retain the original list and document position').toEqual(before);
}
