import { expect, test } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Suggestions stay empty to reproduce Enter before a suggestion response.
// Saving uses the real model/store and a synthetic HTTP lookup response.
const source = `
    <script>
        import Combobox from '$lib/components/base/combobox.svelte';
        import { normalizeNewTagName } from '$lib/util/tag_name';
        import { Tag } from '$lib/models/tag';
        import { tags_create } from '$lib/stores/tag_store';

        const existing = { id: 'existingtag0001', name: 'Berg Tour' };
        const requests = [];
        let selected = $state([]);
        let saved = $state(null);
        let error = $state('');
        window.inspectTagRequests = () => requests;
        window.fetch = async (url, options) => {
            const data = JSON.parse(options.body);
            requests.push({ url: String(url), method: options.method, data });
            if (String(url) === '/api/v1/tag/lookup') {
                return Response.json({ items: data.name === existing.name ? [existing] : [] });
            }
            return Response.json({ id: 'createdtag00001', ...data });
        };
        async function save() {
            try {
                saved = await tags_create(new Tag(selected[0].text));
            } catch (e) {
                error = e.message;
            }
        }
    </script>
    <Combobox multiple chips bind:value={selected} items={[]}
        normalizeNewItemText={normalizeNewTagName} />
    <button data-testid="save-tag" onclick={save}>Save tag</button>
    <output data-testid="saved-tag">{JSON.stringify(saved)}</output>
    <output data-testid="error">{error}</output>
`;

for (const dev of [true, false]) {
    test.describe(dev ? 'tag reuse in development' : 'tag reuse in production', () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, source); });

        for (const raw of ['Berg\tTour', 'Berg Tour']) {
            test(`reuses a saved tag without suggestions for ${JSON.stringify(raw)}`, async ({ page }) => {
                await mountTrailModalFixture(page, bundle);
                await page.locator('input').fill(raw);
                await page.locator('input').press('Enter');
                await expect(page.locator('span.text-xs')).toHaveText('Berg Tour');
                await page.getByTestId('save-tag').click();
                await expect(page.getByTestId('saved-tag')).toHaveText(JSON.stringify({ id: 'existingtag0001', name: 'Berg Tour' }));
                await expect(page.getByTestId('error')).toBeEmpty();
                expect(await page.evaluate(() => (window as any).inspectTagRequests())).toEqual([
                    { url: '/api/v1/tag/lookup', method: 'POST', data: { name: 'Berg Tour' } },
                ]);
            });
        }
    });
}
