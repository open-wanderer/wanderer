import { readFile } from 'node:fs/promises';
import { expect, test, type Page } from '../../fixtures';
import type { APIRequestContext } from '@playwright/test';
import { buildAssetStoresFixture } from './asset-stores-fixture';

const jpeg = new URL('../../fixtures/asset-upload.jpg', import.meta.url);
const heic = new URL('../../fixtures/asset-upload.heic', import.meta.url);
const expand = 'trail_assets_via_trail.asset,waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail.summit_log_assets_via_summit_log.asset';
let storeBundle: string;

declare global {
    interface Window {
        moveSavedWaypointPhoto: (id: string, removeWaypoint: boolean) => Promise<{ photos: string[]; waypoints: { photos?: string[] }[]; photo: string }>;
    }
}

test.beforeAll(async () => { storeBundle = await buildAssetStoresFixture(); });
test.beforeEach(async ({ context }) => {
    await context.route('**/styles/ofm.json', route => route.fulfill({ json: {
        version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }],
    } }));
});

async function actor(api: APIRequestContext, username = 'Test', password = 'password') {
    const response = await api.post('/api/v1/auth/login', { data: { username, password } });
    expect(response.ok(), await response.text()).toBeTruthy();
    const profile = await api.get(`/api/v1/profile/@${encodeURIComponent(username)}`);
    expect(profile.ok(), await profile.text()).toBeTruthy();
    const id = (await profile.json()).actor.id as string;
    expect(id).toMatch(/^[a-z0-9]{15}$/);
    return id;
}

async function createTrail(api: APIRequestContext, author: string, name: string, isPublic = false) {
    const response = await api.put('/api/v1/trail', { data: { name, public: isPublic, completed: true, difficulty: 'easy', author } });
    expect(response.ok(), await response.text()).toBeTruthy();
    return (await response.json()).id as string;
}

async function attachPhoto(api: APIRequestContext, target: Record<string, string>) {
    const bytes = await readFile(jpeg);
    const response = await api.put('/api/v1/assets', { multipart: { ...target, files: { name: 'outing.jpg', mimeType: 'image/jpeg', buffer: bytes } } });
    expect(response.ok(), await response.text()).toBeTruthy();
    return (await response.json())[0] as { id: string; author: string };
}

async function stored(page: Page, id: string) {
    const response = await page.request.get(`/api/v1/trail/${id}?${new URLSearchParams({ expand })}`);
    expect(response.ok(), await response.text()).toBeTruthy();
    return response.json();
}

async function saveEditor(page: Page, creating = false): Promise<string> {
    const metadata = page.waitForResponse(response => new URL(response.url()).pathname.startsWith('/api/v1/trail/form') && response.request().method() === (creating ? 'PUT' : 'POST'));
    await page.getByRole('button', { name: 'Save Trail', exact: true }).click();
    const response = await metadata;
    expect(response.ok(), await response.text()).toBeTruthy();
    const { id } = await response.json();
    await expect(page.getByText('Trail saved successfully', { exact: true })).toBeVisible();
    return id as string;
}

test('creates and edits a trail using real JPEG and HEIC device photos', async ({ page }) => {
    await page.goto('/trail/edit/new', { waitUntil: 'networkidle' });
    await page.locator('#trail-form input[name="name"]').fill('Device photo outing');
    await page.locator('#trail-photo-input').setInputFiles([
        { name: 'outing.jpg', mimeType: 'image/jpeg', buffer: await readFile(jpeg) },
        { name: 'outing.heic', mimeType: 'image/heic', buffer: await readFile(heic) },
    ]);
    const picker = page.locator('#trail-photo-input').locator('..');
    await expect(picker.getByRole('button', { name: 'Delete photo' })).toHaveCount(2);
    const id = await saveEditor(page, true);
    const created = await stored(page, id);
    expect(created.photos).toHaveLength(2);
    expect(created.expand.trail_assets_via_trail.every((link: any) => /\.jpe?g$/i.test(link.expand.asset.file))).toBe(true);

    await page.goto(`/trail/edit/${id}`, { waitUntil: 'networkidle' });
    await picker.getByRole('button', { name: 'Delete photo' }).first().click();
    await page.locator('#trail-photo-input').setInputFiles({ name: 'edited.HEIC', mimeType: 'image/heic', buffer: await readFile(heic) });
    await expect(picker.getByRole('button', { name: 'Delete photo' })).toHaveCount(2);
    await saveEditor(page);
    const edited = await stored(page, id);
    expect(edited.photos).toHaveLength(2);
    expect(edited.photos).not.toContain(created.photos[0]);
    for (const url of edited.photos) expect((await page.request.get(url)).ok()).toBe(true);
});

test('creates a waypoint from a real HEIC device photo with its original GPS position', async ({ page }) => {
    await page.goto('/trail/edit/new', { waitUntil: 'networkidle' });
    await page.locator('#trail-form input[name="name"]').fill('HEIC waypoint outing');
    await page.locator('#waypoint-photo-input').setInputFiles({ name: 'waypoint.heic', mimeType: 'image/heic', buffer: await readFile(new URL('../../fixtures/asset-waypoint.heic', import.meta.url)) });
    const points = page.getByRole('heading', { name: 'Waypoints', exact: true }).locator('xpath=following-sibling::ul[1]');
    await expect(points.locator(':scope > li')).toHaveCount(1);
    const id = await saveEditor(page, true);
    const saved = await stored(page, id);
    expect(saved.expand.waypoints_via_trail).toHaveLength(1);
    const point = saved.expand.waypoints_via_trail[0];
    expect(point).toMatchObject({ lat: 46.5, lon: -7.25 });
    expect(point.photos).toHaveLength(1);
    expect(point.expand.waypoint_assets_via_waypoint[0].expand.asset.file).toMatch(/\.jpe?g$/i);
});

for (const removeWaypoint of [false, true]) {
    test(removeWaypoint ? 'keeps a photo when its waypoint is deleted during a gallery move' : 'moves a saved waypoint photo to the gallery in one save', async ({ page }) => {
        const author = await actor(page.request);
        const id = await createTrail(page.request, author, 'Moving photo outing');
        const waypoint = await page.request.put('/api/v1/waypoint', { data: { name: 'Summit', lat: 47, lon: 8, icon: 'circle', trail: id, author } });
        expect(waypoint.ok(), await waypoint.text()).toBeTruthy();
        const point = await waypoint.json();
        const asset = await attachPhoto(page.request, { trail: id, waypoint: point.id });
        await page.goto(`/trail/edit/${id}`, { waitUntil: 'networkidle' });
        await page.addScriptTag({ content: storeBundle });
        const result = await page.evaluate(({ id, removeWaypoint }) => window.moveSavedWaypointPhoto(id, removeWaypoint), { id, removeWaypoint });
        expect(result.photos).toHaveLength(1);
        expect(result.photos[0]).toContain(asset.id);
        expect(result.waypoints).toHaveLength(removeWaypoint ? 0 : 1);
        if (!removeWaypoint) expect(result.waypoints[0].photos).toEqual([]);
        const bytes = await page.request.get(result.photos[0]);
        expect(bytes.ok(), await bytes.text()).toBeTruthy();
        expect(await bytes.body()).toEqual(await readFile(jpeg));
    });
}

test('removes the last real photo through the editor', async ({ page }) => {
    const author = await actor(page.request);
    const id = await createTrail(page.request, author, 'Cleared photo outing');
    await attachPhoto(page.request, { trail: id });
    await page.goto(`/trail/edit/${id}`, { waitUntil: 'networkidle' });
    await page.locator('#trail-photo-input').locator('..').getByRole('button', { name: 'Delete photo' }).click();
    await saveEditor(page);
    expect((await stored(page, id)).photos).toEqual([]);
});

test('shows summit-book photos when the first entry has no photo', async ({ page }) => {
    const author = await actor(page.request);
    const id = await createTrail(page.request, author, 'Summit book outing');
    for (const [date, text, photo] of [['2026-10-01', 'First outing without a photo', false], ['2026-10-02', 'Second outing with a photo', true]] as const) {
        const response = await page.request.put('/api/v1/summit-log/form', { multipart: { trail: id, author, date, text } });
        expect(response.ok(), await response.text()).toBeTruthy();
        if (photo) await attachPhoto(page.request, { trail: id, summit_log: (await response.json()).id });
    }
    await page.goto(`/trail/view/@Test/${id}`, { waitUntil: 'networkidle' });
    await expect(page.getByText('First outing without a photo')).toBeVisible();
    await expect(page.getByText('Second outing with a photo')).toBeVisible();
    await expect(page.locator('img[src*="/api/v1/files/"]')).toHaveCount(1);
    const response = await page.request.get(`/api/v1/summit-log?${new URLSearchParams({ filter: `trail='${id}'`, perPage: '-1', expand: 'summit_log_assets_via_summit_log.asset' })}`);
    const entries = (await response.json()).items;
    expect(entries[0].photos).toEqual([]);
    expect(entries[1].photos).toHaveLength(1);
});

test('duplicates another author\'s photo through the editor without downloading and uploading it again', async ({ page, playwright, baseURL }) => {
    const ownActor = await actor(page.request);
    const contributor = await playwright.request.newContext({ baseURL });
    const username = `PhotoContributor${Date.now()}`;
    const password = 'photo-password';
    try {
        await contributor.put('/api/v1/user', { data: { username, email: `${username.toLowerCase()}@a.example`, password, passwordConfirm: password } });
        const contributorActor = await actor(contributor, username, password);
        const original = await createTrail(contributor, contributorActor, 'Contributor outing', true);
        const source = await attachPhoto(contributor, { trail: original });
        await page.goto(`/trail/edit/new?${new URLSearchParams({ orig: original, copyTrailPhotos: 'true' })}`, { waitUntil: 'networkidle' });
        await page.locator('#trail-form input[name="name"]').fill('Copied contributor outing');
        const id = await saveEditor(page, true);
        const copied = await stored(page, id);
        const copy = copied.expand.trail_assets_via_trail[0].expand.asset;
        expect(copy.id).not.toBe(source.id);
        expect(copy.author).toBe(ownActor);
        const unchanged = await contributor.get(`/api/v1/trail/${original}?${new URLSearchParams({ expand })}`);
        const linkedSource = (await unchanged.json()).expand.trail_assets_via_trail[0].expand.asset;
        expect(linkedSource).toMatchObject({ id: source.id, author: contributorActor });
    } finally { await contributor.dispose(); }
});

test.describe('editor retry after a failed refresh', () => {
    // Only this injected HTTP failure and the editor's caught diagnostic are
    // expected. Each allowance is bound to its exact request or script URL.
    test.use({ browserErrorAllowlist: [[{
        source: 'console.error',
        message: /^Failed to load resource: the server responded with a status of 503 \(Service Unavailable\)$/,
        url: /^$/,
        issue: 'https://github.com/open-wanderer/wanderer/pull/948',
    }, {
        source: 'console.error',
        message: /^[\w$]+: Injected editor refresh failure(?:\n|$)/,
        url: /^$/,
        issue: 'https://github.com/open-wanderer/wanderer/pull/948',
    }], { scope: 'test' }] });
    test('keeps an uploaded photo after a failed editor refresh and a second save', async ({ page, browserErrorAllowlist }) => {
        const author = await actor(page.request);
        const id = await createTrail(page.request, author, 'Retried photo outing');
        await page.goto(`/trail/edit/${id}`, { waitUntil: 'networkidle' });
        const nodeScripts = await page.evaluate(() => performance.getEntriesByType('resource')
            .map(entry => entry.name).filter(url => /\/_app\/immutable\/nodes\/[^/]+\.js$/.test(url)));
        const editorScripts: string[] = [];
        for (const url of new Set(nodeScripts)) {
            const code = await (await page.request.get(url)).text();
            if (code.includes('trail-form') && code.includes('error-saving-trail')) editorScripts.push(url);
        }
        expect(editorScripts).toHaveLength(1);
        browserErrorAllowlist[1].url = new RegExp(`^${editorScripts[0].replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
        await page.locator('#trail-photo-input').setInputFiles({ name: 'retry.jpg', mimeType: 'image/jpeg', buffer: await readFile(jpeg) });
        await expect(page.locator('#trail-photo-input').locator('..').getByRole('button', { name: 'Delete photo' })).toHaveCount(1);
        let failed = false;
        let uploads = 0;
        page.on('request', request => { if (new URL(request.url()).pathname === '/api/v1/assets' && request.method() === 'PUT') uploads++; });
        await page.route(`**/api/v1/trail/${id}?**`, async route => {
            if (!failed && route.request().method() === 'GET') {
                failed = true;
                browserErrorAllowlist[0].url = new RegExp(`^${route.request().url().replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
                await route.fulfill({ status: 503, json: { message: 'Injected editor refresh failure' } });
            } else await route.continue();
        });
        await page.getByRole('button', { name: 'Save Trail', exact: true }).click();
        await expect(page.getByText('Error saving trail', { exact: true })).toBeVisible();
        expect(failed).toBe(true);
        const afterFailure = await stored(page, id);
        expect(afterFailure.photos).toHaveLength(1);
        await saveEditor(page);
        expect(uploads).toBe(1);
        expect((await stored(page, id)).photos).toEqual(afterFailure.photos);
        await page.reload({ waitUntil: 'networkidle' });
        await expect(page.locator('#trail-photo-input').locator('..').getByRole('button', { name: 'Delete photo' })).toHaveCount(1);
        const photoResponse = await page.request.get(afterFailure.photos[0]);
        expect(photoResponse.ok(), await photoResponse.text()).toBe(true);
    });
});
