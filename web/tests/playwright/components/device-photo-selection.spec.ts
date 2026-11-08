import { expect, test } from '../fixtures';
import { buildDevicePhotoFixture, devicePhoto } from './device-photo-fixture';

declare global {
    interface Window {
        photoToasts: { type: string; icon: string; text: string; duration: number }[];
        readDevicePhoto: (bytes: number[]) => Promise<{ coordinates?: { lat: number; lon: number }; fileType?: string }>;
    }
}

let bundle: string;
test.beforeAll(async () => { bundle = await buildDevicePhotoFixture(); });

for (const format of ['jpeg', 'png', 'webp', 'heic', 'heif'] as const) {
    test(`selects device photos by ${format} content`, async ({ page }) => {
        const errors: string[] = [];
        page.on('pageerror', error => errors.push(error.message));
        await page.route('http://device-photos.test/**', async route => {
            if (route.request().method() === 'GET') {
                await route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
                return;
            }
            const body = route.request().postDataJSON();
            await route.fulfill({ json: { mergeEnabled: false, mergeRadius: 0, clusters: body.photos.map((photo: { id: string; lat: number; lon: number }) => ({ lat: photo.lat, lon: photo.lon, photos: [photo.id] })) } });
        });
        await page.goto('http://device-photos.test/');
        await page.addScriptTag({ content: bundle });
        await expect(page.locator('body[data-photo-fixture-ready="true"]')).toBeAttached();
        const bytes = devicePhoto(format);
        const metadata = await page.evaluate(bytes => window.readDevicePhoto(bytes), [...bytes]);
        expect(metadata.coordinates).toEqual({ lat: 46.5, lon: -7.25 });
        const supported = ['jpeg', 'png', 'webp'].includes(format);
        const name = `position.${format}`;
        // Accepted content has an unrelated declared type; HEIC declares JPEG.
        await page.locator('#waypoint-photo-input').setInputFiles({ name, mimeType: supported ? 'image/heic' : 'image/jpeg', buffer: bytes });
        if (supported) {
            await expect(page.getByTestId('photo-waypoint')).toHaveText(name);
            await expect(page.getByTestId('photo-waypoint')).toHaveAttribute('data-lat', '46.5');
            await expect(page.getByTestId('photo-waypoint')).toHaveAttribute('data-lon', '-7.25');
            expect(await page.evaluate(() => window.photoToasts)).toEqual([]);
        } else {
            await expect.poll(() => page.evaluate(() => window.photoToasts)).toEqual([
                { type: 'warning', icon: 'warning', text: `${name}: no-gps-data-in-image`, duration: 10000 },
            ]);
            await expect(page.getByTestId('photo-waypoint')).toHaveCount(0);
        }
        expect(errors).toEqual([]);
    });
}
