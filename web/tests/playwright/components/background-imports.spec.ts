import { fileURLToPath } from 'node:url';
import JSZip from 'jszip';
import { expect, test } from '../fixtures';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

const root = fileURLToPath(new URL('../../../', import.meta.url));
let bundle: string;

const gpx = '<gpx version="1.1" creator="fixture" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg><trkpt lat="47" lon="8"/><trkpt lat="47.01" lon="8.01"/></trkseg></trk></gpx>';

test.beforeAll(async () => {
    bundle = await buildTrailModalFixture(false, `
        <script>
            import ExportPage from '${root}src/routes/settings/export/+page.svelte';
            import Dialog from '#lib/components/background_tasks_dialog.svelte';
            import PhotoPicker from '#lib/components/photo/photo_picker.svelte';
            import { uploadStore } from '#lib/stores/upload_store.svelte.ts';
            import { backgroundTaskStore } from '#lib/stores/background_task_store.svelte.ts';
            let photos = $state([]);
            let photoFiles = $state([]);
            let thumbnail = $state(0);
            window.uploadRequests = [];
            window.uploadResolvers = [];
            window.importState = () => ({
                waiting: uploadStore.enqueuedUploads.filter(upload => upload.status === 'enqueued').map(upload => upload.file.name),
                completed: uploadStore.completedUploads.map(upload => ({ name: upload.file.name, status: upload.status })),
                tasks: backgroundTaskStore.tasks.map(task => ({ title: task.title, status: task.status })),
            });
        </script>
        <ExportPage />
        <Dialog />
        <section data-testid="device-photos">
            <PhotoPicker id="device" parent={{ id: 'trail0000000001', collectionId: 'trails' }} bind:photos bind:photoFiles bind:thumbnail />
            <output data-testid="picked-files">{photoFiles.map(file => file.name).join(',')}</output>
        </section>
    `, {
        [root + 'src/lib/stores/trail_store.ts']: `
            import { writable } from 'svelte/store';
            export const trail = writable({});
            export async function trails_show(id) { throw new Error('Unexpected incomplete trail export ' + id); }
            export async function fetchGPX() { return ${JSON.stringify(gpx)}; }
            export function trails_upload(file) {
                window.uploadRequests.push(file.name);
                return new Promise(resolve => window.uploadResolvers.push(() => resolve({ id: file.name })));
            }
        `,
        '\0fixture:i18n': `
            import { writable } from 'svelte/store';
            import messages from '#lib/i18n/locales/en.json';
            export const locale = writable('en');
            export const _ = writable((key, options) => {
                let message = messages[key] ?? key;
                for (const [name, value] of Object.entries(options?.values ?? {})) message = message.replaceAll('{' + name + '}', String(value));
                return message;
            });
        `,
    });
});

test('shows and cancels a GPX import waiting behind the active batch', async ({ page }) => {
    await mountTrailModalFixture(page, bundle);
    await page.locator('#file-input').setInputFiles(['first.gpx', 'second.gpx', 'third.gpx', 'waiting.gpx'].map(name => ({
        name, mimeType: 'application/gpx+xml', buffer: Buffer.from(gpx),
    })));
    const waiting = page.locator('div.bg-menu-item-background-hover').filter({ has: page.getByText('waiting.gpx', { exact: true }) });
    await expect(waiting).toBeVisible();
    await expect(waiting.getByRole('button', { name: 'Cancel upload' })).toBeVisible();
    expect(await page.evaluate(() => (window as any).uploadRequests)).toEqual(['first.gpx', 'second.gpx', 'third.gpx']);
    await waiting.getByRole('button', { name: 'Cancel upload' }).click();
    await expect(waiting.getByRole('button', { name: 'Cancel upload' })).toHaveCount(0);
    await expect.poll(() => page.evaluate(() => (window as any).importState().waiting)).toEqual([]);
    expect(await page.evaluate(() => (window as any).importState().completed)).toContainEqual({ name: 'waiting.gpx', status: 'cancelled' });
    await page.evaluate(() => (window as any).uploadResolvers.forEach((resolve: () => void) => resolve()));
    await expect.poll(() => page.evaluate(() => (window as any).importState().completed.length)).toBe(4);
    expect(await page.evaluate(() => (window as any).uploadRequests)).not.toContain('waiting.gpx');
});

test('exports the account through the actual page and writes the archive', async ({ page }) => {
    await mountTrailModalFixture(page, bundle);
    const requests: URL[] = [];
    await page.route('**/api/v1/trail?*', async route => {
        const url = new URL(route.request().url());
        requests.push(url);
        await route.fulfill({ json: { page: 1, totalPages: 1, items: [{
            id: 'trail0000000001', name: 'Account trail', photos: [], author: 'tester',
            expand: { waypoints_via_trail: [], summit_logs_via_trail: [] },
        }] } });
    });
    await page.getByRole('button', { name: 'Export all trails' }).click();
    const download = page.waitForEvent('download');
    await page.locator('dialog[open] .btn-primary').click();
    const archive = await download;
    expect(archive.suggestedFilename()).toMatch(/^wanderer-export-\d+\.zip$/);
    const path = await archive.path();
    expect(path).not.toBeNull();
    const zip = await JSZip.loadAsync(await (await import('node:fs/promises')).readFile(path!));
    expect(zip.file('Account trail/Account trail.gpx')).not.toBeNull();
    expect(await zip.file('Account trail/Account trail.gpx')!.async('string')).toContain('<trkpt');
    expect(requests).toHaveLength(1);
    expect(requests[0].searchParams.get('export')).toBe('owned');
    expect(requests[0].searchParams.get('perPage')).toBe('100');
});

test('adds and removes device photos through the actual picker', async ({ page }) => {
    await mountTrailModalFixture(page, bundle);
    const picker = page.getByTestId('device-photos');
    const pixel = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aRk0AAAAASUVORK5CYII=', 'base64');
    await picker.locator('input[type="file"]').setInputFiles(['first.png', 'second.png'].map(name => ({ name, mimeType: 'image/png', buffer: pixel })));
    await expect(picker.getByRole('button', { name: 'Delete photo' })).toHaveCount(2);
    await expect(page.getByTestId('picked-files')).toHaveText('first.png,second.png');
    await picker.getByRole('button', { name: 'Delete photo' }).first().click();
    await expect(picker.getByRole('button', { name: 'Delete photo' })).toHaveCount(1);
    await expect(page.getByTestId('picked-files')).toHaveText('second.png');
});
