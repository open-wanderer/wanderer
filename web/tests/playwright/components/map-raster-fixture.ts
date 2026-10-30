import { expect, test as base, type Page } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { createServer, type Server } from 'node:http';
import { crc32, deflateSync } from 'node:zlib';

// Raster coverage fixture: actual page, map wrapper and engine. Same setup as
// map-window-fixture.ts `mount()`, but with a raster base style whose tiles are
// served locally, so tile coverage of the visible canvas can be observed.
export const test = base.extend({ context: async ({ browser }, use) => { const context = await browser.newContext({ viewport: null }); await use(context); await context.close(); } });
const dist = fileURLToPath(new URL('../../../node_modules/maplibre-gl/dist/', import.meta.url));

function pngChunk(type: string, data: Buffer) {
    const length = Buffer.alloc(4); length.writeUInt32BE(data.length);
    const body = Buffer.concat([Buffer.from(type, 'ascii'), data]);
    const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(body) >>> 0);
    return Buffer.concat([length, body, crc]);
}
export function solidPng(r: number, g: number, b: number, size = 256) {
    const header = Buffer.alloc(13); header.writeUInt32BE(size, 0); header.writeUInt32BE(size, 4); header[8] = 8; header[9] = 2;
    const row = Buffer.alloc(1 + size * 3);
    for (let i = 0; i < size; i++) { row[1 + i * 3] = r; row[2 + i * 3] = g; row[3 + i * 3] = b; }
    const raw = Buffer.concat(Array.from({ length: size }, () => row));
    return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), pngChunk('IHDR', header), pngChunk('IDAT', deflateSync(raw)), pngChunk('IEND', Buffer.alloc(0))]);
}
let tileServer: Server | undefined;
let tileOrigin = '';
let activeTilePage: Page;
export async function closeTileServer() { await new Promise<void>(resolve => { if (tileServer) { tileServer.closeAllConnections(); tileServer.close(() => resolve()); } else resolve(); }); tileServer = undefined; }
const tile = solidPng(192, 48, 48, 8);
export const background = [218, 229, 223];
const style = { version: 8, sources: { r: { type: 'raster', tiles: ['https://search-page.test/tiles/{z}/{x}/{y}.png'], tileSize: 128, maxzoom: 18 } },
    layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } },
        { id: 'r', type: 'raster', source: 'r', paint: { 'raster-fade-duration': 0 } }] };
// Like the built-in OpenTopoMap/OpenHikingMap styles: raster only, no background layer.
style.layers.shift();

export async function routes(page: Page, extra?: (path: string, url: URL) => unknown) {
    activeTilePage = page;
    if (!tileServer) {
        tileServer = createServer((request, response) => {
            response.writeHead(200, { 'content-type': 'image/png', 'access-control-allow-origin': '*', 'access-control-allow-private-network': 'true', 'cache-control': 'no-store' });
            setTimeout(() => response.end(tile), Number((activeTilePage as any).tileDelay ?? 0));
        });
        await new Promise<void>(resolve => tileServer!.listen(0, '127.0.0.1', resolve));
        tileOrigin = 'http://127.0.0.1:' + (tileServer.address() as { port: number }).port;
    }
    style.sources.r.tiles = [tileOrigin + '/tiles/{z}/{x}/{y}.png'];
    page.on('pageerror', error => { throw error; });
    await page.route(url => url.origin === tileOrigin && !url.pathname.startsWith('/tiles/'), async route => {
        const path = new URL(route.request().url()).pathname;
        if (path.startsWith('/maplibre/')) {
            const name = path.split('/').at(-1)!;
            return route.fulfill({ contentType: name.endsWith('.css') ? 'text/css' : 'text/javascript', body: await readFile(dist + name, 'utf8') });
        }
        if (path.startsWith('/styles/')) return route.fulfill({ json: style });
        if (path.startsWith('/tiles/')) {
            const delay = Number((activeTilePage as any).tileDelay ?? 0);
            if (delay) await new Promise(resolve => setTimeout(resolve, delay));
            return route.fulfill({ contentType: 'image/png', body: tile, headers: { 'cache-control': 'no-store' } }).catch(() => {});
        }
        const handled = await extra?.(path, new URL(route.request().url()));
        if (handled) return route.fulfill(handled as any);
        if (path !== '/map') return route.fulfill({ status: 404, body: '' });
        return route.fulfill({ contentType: 'text/html', body: '<html><head><meta name="viewport" content="width=device-width, initial-scale=1"><link rel="stylesheet" href="/maplibre/maplibre-gl.css"></head><body><script type="module">import * as M from "/maplibre/maplibre-gl.mjs"; window.RealMapLibre = M;</script></body></html>' });
    });
    await page.route(url => url.origin !== tileOrigin, route => route.abort());
    return tileOrigin;
}

// Page-side recorder: tile request start times and, per animation frame, the
// share of a 32x20 grid over the visible map whose point is not covered by any
// renderable raster tile (what MapLibre can draw in that frame).
export async function installRecorder(page: Page) {
    await page.evaluate(() => {
        const w = window as any;
        const probe = w.probe = { tiles: [] as number[], frames: [] as any[], t0: 0, sizeChanges: [] as number[] };
        new PerformanceObserver(list => { for (const entry of list.getEntries()) if (entry.name.includes('/tiles/')) probe.tiles.push(entry.startTime); })
            .observe({ type: 'resource', buffered: true });
        w.coverage = () => {
            const map = w.realMap, container = map.getContainer() as HTMLElement;
            const manager = map.style?.tileManagers?.r;
            if (!manager) return { miss: -1, n: 0 };
            const loaded = manager.getRenderableIds().map((id: string) => manager._inViewTiles.getTileById(id).tileID.canonical);
            const width = container.clientWidth, height = container.clientHeight;
            let miss = 0, n = 0;
            for (let i = 0; i < 32; i++) for (let j = 0; j < 20; j++) {
                const m = w.RealMapLibre.MercatorCoordinate.fromLngLat(map.unproject([(i + .5) * width / 32, (j + .5) * height / 20]));
                n++;
                if (!loaded.some((t: any) => { const s = 2 ** t.z; return m.x * s >= t.x && m.x * s < t.x + 1 && m.y * s >= t.y && m.y * s < t.y + 1; })) miss++;
            }
            return { miss, n, width, height, canvas: map.getCanvas().clientWidth };
        };
        w.startFrames = (duration: number) => {
            probe.frames = []; probe.sizeChanges = [];
            const start = performance.now();
            probe.t0 = start;
            const container = w.realMap.getContainer();
            let last = `${container.clientWidth}x${container.clientHeight}`;
            const loop = () => {
                const size = `${container.clientWidth}x${container.clientHeight}`;
                if (size !== last) { probe.sizeChanges.push(Math.round(performance.now() - start)); last = size; }
                probe.frames.push({ t: Math.round(performance.now() - start), ...w.coverage(), tiles: probe.tiles.filter((t: number) => t >= start).length });
                if (performance.now() - start < duration) requestAnimationFrame(loop);
            };
            requestAnimationFrame(loop);
        };
    });
}

export async function summary(page: Page, wait: number) {
    await page.waitForTimeout(wait);
    return page.evaluate(() => {
        const probe = (window as any).probe;
        const firstChange = probe.sizeChanges[0] ?? 0;
        const frames = probe.frames.filter((f: any) => f.t >= firstChange);
        const firstMissing = frames.find((f: any) => f.miss > 0);
        let lastMissing: any;
        for (const f of frames) if (f.miss > 0) lastMissing = f;
        const firstTile = probe.tiles.filter((t: number) => t >= probe.t0 + firstChange).map((t: number) => Math.round(t - probe.t0 - firstChange))[0];
        return {
            sizeChangeAtMs: firstChange,
            framesAfterChange: frames.length,
            framesWithUncoveredArea: frames.filter((f: any) => f.miss > 0).length,
            maxUncoveredPercent: Math.round(Math.max(0, ...frames.map((f: any) => f.miss / f.n * 100))),
            uncoveredUntilMs: lastMissing ? lastMissing.t - firstChange : 0,
            firstTileRequestAfterChangeMs: firstTile ?? null,
            first: firstMissing, last: frames.at(-1),
        };
    });
}

export async function windowBounds(page: Page, bounds: Record<string, unknown>) {
    const session = await page.context().newCDPSession(page);
    const { windowId } = await session.send('Browser.getWindowForTarget');
    await session.send('Browser.setWindowBounds', { windowId, bounds });
    await session.detach();
}

export async function tilesSettled(page: Page) {
    await expect.poll(() => page.evaluate(() => {
        const map = (window as any).realMap;
        return Boolean(map && !map.isMoving() && map.areTilesLoaded() && (window as any).coverage().miss === 0);
    }), { timeout: 15000 }).toBe(true);
    await page.waitForTimeout(500);
}
