import { expect, test } from '../fixtures';
import { buildDevicePhotoFixture, devicePhoto } from './device-photo-fixture';

declare global {
    interface Window {
        photoToasts: { type: string; icon: string; text: string; duration: number }[];
        readDevicePhoto: (bytes: number[]) => Promise<{ coordinates?: { lat: number; lon: number }; fileType?: string }>;
        photoReads: string[];
    }
}

test('skips an oversized PNG before reading EXIF and keeps the exact size limit', async ({ page }) => {
    const clusteredNames: string[] = [];
    await page.route('http://device-photos.test/**', async route => {
        if (route.request().method() === 'GET') {
            await route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
            return;
        }
        const body = route.request().postDataJSON();
        clusteredNames.push(...body.photos.map((photo: { id: string }) => photo.id));
        await route.fulfill({ json: { mergeEnabled: false, mergeRadius: 0, clusters: body.photos.map((photo: { id: string; lat: number; lon: number }) => ({ lat: photo.lat, lon: photo.lon, photos: [photo.id] })) } });
    });
    await page.goto('http://device-photos.test/');
    await page.addScriptTag({ content: bundle });
    await expect(page.locator('body[data-photo-fixture-ready="true"]')).toBeAttached();
    await page.evaluate(() => {
        window.photoReads = [];
        const read = File.prototype.arrayBuffer;
        File.prototype.arrayBuffer = function () {
            window.photoReads.push(this.name);
            return read.call(this);
        };
    });
    const png = devicePhoto('png');
    const atLimit = Buffer.concat([png, Buffer.alloc(20 * 1024 * 1024 - png.length)]);
    await page.locator('#waypoint-photo-input').setInputFiles([
        { name: 'oversized.png', mimeType: 'image/png', buffer: Buffer.concat([atLimit, Buffer.from([0])]) },
        { name: 'at-limit.png', mimeType: 'image/png', buffer: atLimit },
    ]);

    await expect(page.getByTestId('photo-waypoint')).toHaveText('at-limit.png');
    expect(await page.evaluate(() => window.photoReads)).toEqual(['at-limit.png']);
    expect(clusteredNames).toEqual(['1']);
    expect(await page.evaluate(() => window.photoToasts)).toEqual([
        { type: 'warning', icon: 'warning', text: 'waypoint-photos-skipped-limits {"n":1,"size":20,"max":99}', duration: 10000 },
    ]);
});

test('enforces the photo limit across saved, pending and repeated clusters for one waypoint', async ({ page }) => {
    const seededBundle = await buildDevicePhotoFixture({ saved: 50, pending: 47 });
    await page.route('http://device-photos.test/**', async route => {
        if (route.request().method() === 'GET') {
            await route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
            return;
        }
        const body = route.request().postDataJSON();
        const waypoint = body.waypoints[0];
        await route.fulfill({ json: { mergeEnabled: true, mergeRadius: 50, clusters: [
            { ...waypoint, waypoint: waypoint.id, photos: [body.photos[0].id] },
            { ...waypoint, waypoint: waypoint.id, photos: body.photos.slice(1).map((photo: { id: string }) => photo.id) },
        ] } });
    });
    await page.goto('http://device-photos.test/');
    await page.addScriptTag({ content: seededBundle });
    await expect(page.locator('body[data-photo-fixture-ready="true"]')).toBeAttached();
    await page.locator('#waypoint-photo-input').setInputFiles(Array.from({ length: 100 }, (_, index) => {
        const format = index % 2 ? 'webp' : 'jpeg';
        return { name: `incoming-${index}.${format}`, mimeType: `image/${format}`, buffer: devicePhoto(format) };
    }));

    const waypoint = page.getByTestId('photo-waypoint');
    await expect(waypoint).toHaveCount(1);
    await expect(waypoint).toHaveAttribute('data-saved-count', '50');
    await expect(waypoint).toHaveAttribute('data-pending-count', '49');
    expect(await waypoint.locator('span').allTextContents()).toEqual([
        ...Array.from({ length: 47 }, (_, index) => `pending-${index}.jpg`),
        'incoming-0.jpeg', 'incoming-1.webp',
    ]);
    expect(await page.evaluate(() => window.photoToasts)).toEqual([
        { type: 'warning', icon: 'warning', text: 'waypoint-photos-skipped-limits {"n":98,"size":20,"max":99}', duration: 10000 },
    ]);
});

test('limits a new waypoint to 99 photos from one cluster', async ({ page }) => {
    await page.route('http://device-photos.test/**', async route => {
        if (route.request().method() === 'GET') {
            await route.fulfill({ contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
            return;
        }
        const body = route.request().postDataJSON();
        expect(body.waypoints).toEqual([]);
        await route.fulfill({ json: { mergeEnabled: true, mergeRadius: 50, clusters: [{
            lat: body.photos[0].lat,
            lon: body.photos[0].lon,
            photos: body.photos.map((photo: { id: string }) => photo.id),
        }] } });
    });
    await page.goto('http://device-photos.test/');
    await page.addScriptTag({ content: bundle });
    await expect(page.locator('body[data-photo-fixture-ready="true"]')).toBeAttached();
    await page.locator('#waypoint-photo-input').setInputFiles(Array.from({ length: 100 }, (_, index) => ({
        name: `photo-${index}.png`, mimeType: 'image/png', buffer: devicePhoto('png'),
    })));

    const waypoint = page.getByTestId('photo-waypoint');
    await expect(waypoint).toHaveCount(1);
    await expect(waypoint).toHaveAttribute('data-saved-count', '0');
    await expect(waypoint).toHaveAttribute('data-pending-count', '99');
    expect(await waypoint.locator('span').allTextContents()).toEqual(
        Array.from({ length: 99 }, (_, index) => `photo-${index}.png`),
    );
    expect(await page.evaluate(() => window.photoToasts)).toEqual([
        { type: 'warning', icon: 'warning', text: 'waypoint-photos-skipped-limits {"n":1,"size":20,"max":99}', duration: 10000 },
    ]);
});

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
