import { expect, test } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { build } from 'vite';

const root = fileURLToPath(new URL('../../../', import.meta.url));
const webfonts = root + 'node_modules/@fortawesome/fontawesome-free/webfonts/';

// Marker images are drawn from Font Awesome's `::before` glyphs. Use the real
// stylesheet and fonts so a Font Awesome update that changes them fails here.
test('draws the category glyph into trailhead marker images', async ({ page }) => {
    const css = readFileSync(root + 'node_modules/@fortawesome/fontawesome-free/css/all.min.css', 'utf8')
        .replace(/url\(\.\.\/webfonts\/([^)]+)\)/g, (_match, file: string) => file.endsWith('.woff2')
            ? `url(data:font/woff2;base64,${readFileSync(webfonts + file).toString('base64')})` : 'url()');
    const result = await build({ root, configFile: false, logLevel: 'error', resolve: { alias: { '#lib': root + 'src/lib' } },
        build: { write: false, minify: false, lib: { entry: root + 'src/lib/util/trailhead_marker.ts', formats: ['iife'], name: 'trailheadMarker' } } });
    if (!Array.isArray(result)) throw new Error('Expected a single bundle');
    await page.setContent(`<style>${css}</style>`);
    await page.addScriptTag({ content: result[0].output[0].code });

    const glyphPixels = await page.evaluate(async () => {
        const { withTrailheadIcons, ensureTrailheadImages } = (window as any).trailheadMarker;
        const images: Record<string, ImageData> = {};
        const map = { style: {}, hasImage: (name: string) => name in images, addImage: (name: string, image: ImageData) => { images[name] = image; } };
        const point = { type: 'Point', coordinates: [7, 46] };
        await ensureTrailheadImages(map, withTrailheadIcons({ type: 'FeatureCollection', features: [
            { type: 'Feature', properties: { category_icon: 'person-hiking' }, geometry: point },
            { type: 'Feature', properties: { category_icon: 'bicycle' }, geometry: point },
            { type: 'Feature', properties: {}, geometry: point },
        ] }));
        // Count white glyph pixels inside the circle, leaving out the white border.
        return Object.fromEntries(Object.entries(images).map(([name, image]) => {
            let white = 0;
            const size = image.width;
            for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) {
                const index = (y * size + x) * 4;
                const inside = Math.hypot(x + .5 - size / 2, y + .5 - size / 2) < size / 2 - 4;
                if (inside && image.data[index] > 200 && image.data[index + 1] > 200 && image.data[index + 2] > 200) white++;
            }
            return [name, white];
        }));
    });

    expect(Object.keys(glyphPixels).sort()).toEqual(['trailhead-fa-bicycle', 'trailhead-fa-person-hiking', 'trailhead-fa-shapes']);
    // A missing font draws the same placeholder box for every icon.
    expect(new Set(Object.values(glyphPixels)).size).toBe(3);
    for (const pixels of Object.values(glyphPixels)) expect(pixels).toBeGreaterThan(40);
});
