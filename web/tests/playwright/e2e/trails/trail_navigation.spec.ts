import type { Page } from '@playwright/test';
import { expect, test } from '../../fixtures';

// Notifications link to /trail/view, so the map page is left through an in-app
// link instead, like the trail search in the navbar.
const navigations = [
    {
        prefix: '/trail/view',
        via: 'a notification',
        navigate: async (page: Page, destination: { id: string; name: string }) => {
            await page.getByRole('button', { name: 'Toggle notification dropdown' }).click();
            await page.locator('.dropdown li').filter({ hasText: destination.name }).click();
        },
    },
    {
        prefix: '/map/trail',
        via: 'an in-app link',
        navigate: async (page: Page, destination: { id: string; name: string }) => {
            await page.locator('main').evaluate((main, href) => {
                const link = document.createElement('a');
                link.href = href;
                link.textContent = 'Next trail';
                main.append(link);
            }, `/map/trail/@Test/${destination.id}`);
            await page.getByRole('link', { name: 'Next trail', exact: true }).click();
        },
    },
];

for (const { prefix, via, navigate } of navigations) {
    test(`refreshes trail details on ${prefix} after navigating through ${via}`, async ({ page, context }) => {
        const trails: { id: string; name: string }[] = [];
        try {
            for (const name of ['Navigation source', 'Navigation destination']) {
                const upload = await page.request.put('/api/v1/trail/upload', { multipart: {
                    file: { name: 'track.gpx', mimeType: 'application/gpx+xml', buffer: Buffer.from(
                        `<?xml version="1.0"?><gpx version="1.1" creator="t" xmlns="http://www.topografix.com/GPX/1/1"><metadata><name>${name}</name></metadata><trk><trkseg><trkpt lat="47" lon="11"><ele>500</ele></trkpt><trkpt lat="47.001" lon="11"><ele>510</ele></trkpt></trkseg></trk></gpx>`,
                    ) }, ignoreDuplicates: 'true',
                } });
                expect(upload.ok(), await upload.text()).toBeTruthy();
                trails.push({ id: (await upload.json()).id, name });
            }
            const [source, destination] = trails;
            await context.route('**/api/v1/notification?*', route => route.fulfill({ json: {
                items: [{
                    id: 'navigation', type: 'trail_share', seen: false, created: new Date().toISOString(),
                    expand: { author: { username: 'Test', preferred_username: 'Test' } },
                    metadata: { author: '@Test', id: destination.id, trail: destination.name },
                }], page: 1, perPage: 10, totalItems: 1, totalPages: 1,
            } }));
            await context.route('**/api/v1/notification/navigation', route => route.fulfill({ json: { seen: true } }));
            await context.route('**/styles/ofm.json', route => route.fulfill({ json: {
                version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#dae5df' } }],
            } }));
            await context.route('https://api.dicebear.com/**', route => route.fulfill({
                contentType: 'image/svg+xml', body: '<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"/>',
            }));

            const panel = page.locator('.trail-info-panel');
            await page.goto(`${prefix}/@Test/${source.id}`, { waitUntil: 'networkidle' });
            await expect(panel.getByRole('heading', { name: source.name, exact: true })).toBeVisible();
            const document = await page.evaluate(() => performance.timeOrigin);

            await navigate(page, destination);

            await expect(page).toHaveURL(`${prefix}/@Test/${destination.id}`);
            await expect(panel.getByRole('heading', { name: destination.name, exact: true })).toBeVisible();
            await expect(panel).not.toContainText(source.name);
            expect(await page.evaluate(() => performance.timeOrigin)).toBe(document);
        } finally {
            for (const { id } of trails) {
                try {
                    const deleted = await page.request.delete(`/api/v1/trail/${id}`);
                    expect.soft(deleted.ok(), `Delete trail ${id}: ${await deleted.text()}`).toBeTruthy();
                } catch (error) {
                    expect.soft(error, `Delete trail ${id}`).toBeUndefined();
                }
            }
        }
    });
}
