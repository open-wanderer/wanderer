import { expect, test, type Page } from '../../fixtures';

// Keep the real editor, stores and API. Only the map style is stubbed, because
// the tile servers are not part of the stack these tests run against.
test.beforeEach(async ({ context }) => {
    await context.route('**/styles/ofm.json', route => route.fulfill({ json: {
        version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }],
    } }));
});

const gpx = (name: string, lat: number, points = 6) => `<?xml version="1.0"?><gpx version="1.1" creator="t" xmlns="http://www.topografix.com/GPX/1/1"><metadata><name>${name}</name></metadata><trk><trkseg>${
    Array.from({ length: points }, (_, i) => `<trkpt lat="${(lat + i * 0.001).toFixed(4)}" lon="11.0000"><ele>${500 + i * 10}</ele><time>2024-05-01T10:${String(i).padStart(2, '0')}:00Z</time></trkpt>`).join('')
}</trkseg></trk></gpx>`;

async function recordedTrail(page: Page, name: string) {
    const upload = await page.request.put('/api/v1/trail/upload', {
        multipart: { file: { name: 'track.gpx', mimeType: 'application/gpx+xml', buffer: Buffer.from(gpx(name, 47)) }, ignoreDuplicates: 'true' },
    });
    expect(upload.ok(), await upload.text()).toBeTruthy();
    const { id } = await upload.json();
    const update = await page.request.post(`/api/v1/trail/form/${id}`, { multipart: { moving_duration: '5000' } });
    expect(update.ok(), await update.text()).toBeTruthy();
    return id as string;
}

const stored = async (page: Page, id: string) => (await page.request.get(`/api/v1/trail/${id}`)).json();

async function openEditor(page: Page, id: string) {
    await page.goto(`/trail/edit/${id}`, { waitUntil: 'networkidle' });
    await page.getByRole('button', { name: 'Edit route' }).click();
}

async function save(page: Page) {
    const response = page.waitForResponse(r => r.url().includes('/api/v1/trail/form/'));
    await page.getByRole('button', { name: 'Save Trail' }).click();
    expect((await response).status()).toBe(200);
}

test('replacing a route with a file saves the track and its metrics', async ({ page }) => {
    const id = await recordedTrail(page, 'Replaced');
    const before = await stored(page, id);
    await openEditor(page, id);

    await page.getByRole('button', { name: 'Reset route' }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Reset route' }).click();
    await page.locator('#fileInput').setInputFiles({ name: 'other.gpx', mimeType: 'application/gpx+xml', buffer: Buffer.from(gpx('Other', 48, 3)) });
    await expect(page.getByRole('button', { name: 'Stop editing' })).toBeVisible();
    await save(page);

    const after = await stored(page, id);
    expect(after.lat).toBe(48);
    expect(after.distance).toBeLessThan(before.distance);
    expect(after.moving_duration).toBe(0);
    expect(after.gpx).not.toBe(before.gpx);
});

test('a validation failure is visible', async ({ page }) => {
    const id = await recordedTrail(page, 'Invalid');
    await page.goto(`/trail/edit/${id}`, { waitUntil: 'networkidle' });
    await page.evaluate(() => {
        const field = document.querySelector<HTMLInputElement>('#trail-form input[name="name"]')!;
        field.value = '';
        field.dispatchEvent(new Event('input', { bubbles: true }));
    });

    await page.getByRole('button', { name: 'Save Trail' }).click();

    await expect(page.getByText(/Error saving trail: name/)).toBeVisible();
});

test('reversing moves the start and clears the moving time', async ({ page }) => {
    const id = await recordedTrail(page, 'Reversed');
    await openEditor(page, id);

    await page.getByRole('button', { name: 'Reverse trail direction' }).click();
    await save(page);

    const after = await stored(page, id);
    expect(after.lat).toBeCloseTo(47.005, 6);
    expect(after.moving_duration).toBe(0);
});

test('undoing a reversal brings the moving time back', async ({ page }) => {
    const id = await recordedTrail(page, 'Undone');
    await openEditor(page, id);

    await page.getByRole('button', { name: 'Reverse trail direction' }).click();
    await page.getByRole('button', { name: 'undo route action' }).click();
    await save(page);

    const after = await stored(page, id);
    expect(after.lat).toBe(47);
    expect(after.moving_duration).toBe(5000);
});

async function crop(page: Page, stepsFromStart: number) {
    await page.getByRole('button', { name: 'crop route' }).click();
    const start = page.locator('.noUi-handle-lower');
    await start.focus();
    for (let i = 0; i < stepsFromStart; i++) await page.keyboard.press('ArrowRight');
    await page.getByRole('button', { name: 'Crop', exact: true }).click();
}

test('cropping moves the start and clears the moving time', async ({ page }) => {
    const id = await recordedTrail(page, 'Cropped');
    await openEditor(page, id);

    await crop(page, 30);
    await save(page);

    const after = await stored(page, id);
    expect(after.lat).toBeGreaterThan(47);
    expect(after.moving_duration).toBe(0);
});

test('cropping from 0% keeps the first point', async ({ page }) => {
    const id = await recordedTrail(page, 'Whole');
    const before = await stored(page, id);
    await openEditor(page, id);

    await crop(page, 0);
    await save(page);

    const after = await stored(page, id);
    expect(after.lat).toBe(47);
    expect(after.distance).toBeCloseTo(before.distance, 6);
});

test('undoing a crop brings the moving time back', async ({ page }) => {
    const id = await recordedTrail(page, 'Crop undone');
    await openEditor(page, id);

    await crop(page, 30);
    await page.getByRole('button', { name: 'undo route action' }).click();
    await save(page);

    const after = await stored(page, id);
    expect(after.lat).toBe(47);
    expect(after.moving_duration).toBe(5000);
});
