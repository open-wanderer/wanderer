import { expect, test } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Use the real tag field, normalization callback, model and strict API schema.
// The page's map and backend are outside this chip-creation regression.
const source = `
    <script>
        import Combobox from '$lib/components/base/combobox.svelte';
        import { normalizeNewTagName } from '$lib/util/tag_name';
        import { Tag } from '$lib/models/tag';
        import { TagCreateSchema } from '$lib/models/api/tag_schema';

        const existing = Object.freeze({ id: 'tag000000000001', name: '  Existing  ' });
        const canonical = Object.freeze({ id: 'tag000000000002', name: 'Canonical Name' });
        const empty = Object.freeze({ id: 'tag000000000003', name: '' });
        const spaces = Object.freeze({ id: 'tag000000000004', name: '  ' });
        let selected = $state([
            { text: existing.name, value: existing },
            { text: empty.name, value: empty },
        ]);
        // Compare the selected references after Svelte wraps fetched plain data.
        const initialSelectedValues = [selected[0].value, selected[1].value];
        const suggestions = [existing, canonical, spaces].map(value => ({ text: value.name, value }));
        let genericSelected = $state([]);
        let savedNames = $state([]);
        window.inspectTagSelection = () => selected.map(item => ({
            text: item.text,
            name: item.value?.name,
            id: item.value?.id,
            preserved: initialSelectedValues.includes(item.value),
        }));
        window.inspectOriginalTagRecords = () => [existing, canonical, empty, spaces].map(tag => ({ ...tag }));
        window.inspectGenericSelection = () => genericSelected.map(item => item.text);
        function save() {
            savedNames = TagCreateSchema.array().parse(
                selected.map(item => item.value ?? new Tag(item.text)),
            ).map(tag => tag.name);
        }
    </script>
    <section data-testid="tag-field">
        <Combobox multiple chips bind:value={selected} items={suggestions}
            normalizeNewItemText={normalizeNewTagName} />
    </section>
    <section data-testid="generic-field">
        <Combobox multiple chips bind:value={genericSelected} />
    </section>
    <button data-testid="save-tags" onclick={save}>Save tags</button>
    <output data-testid="saved-names">{JSON.stringify(savedNames)}</output>
`;

for (const dev of [true, false]) {
    test.describe(dev ? 'new tags in development' : 'new tags in production', () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, source); });

        test('shows normalized chips before saving the same names', async ({ page }) => {
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            await mountTrailModalFixture(page, bundle);
            const field = page.getByTestId('tag-field');
            const input = field.locator('input');
            const names = ['  Existing  ', '', 'Word Word', 'VT Word', 'FF Word'];
            const rawNames = ['Word\tWord', 'VT\vWord', 'FF\fWord'];
            for (const [index, raw] of rawNames.entries()) {
                await input.fill(raw);
                await expect(input).toHaveValue(raw);
                await input.press('Enter');
                await expect(field.locator('span.text-xs')).toHaveCount(index + 3);
                expect(await field.locator('span.text-xs').nth(index + 2).textContent()).toBe(names[index + 2]);
            }
            await page.getByTestId('save-tags').click();
            await expect(page.getByTestId('saved-names')).toHaveText(JSON.stringify(names));
            const selected = await page.evaluate(() => (window as any).inspectTagSelection());
            expect(selected.slice(0, 2)).toEqual([
                { text: '  Existing  ', name: '  Existing  ', id: 'tag000000000001', preserved: true },
                { text: '', name: '', id: 'tag000000000003', preserved: true },
            ]);
            expect(errors).toEqual([]);
        });

        test('skips blank new names, deduplicates and reuses existing suggestions', async ({ page }) => {
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            await mountTrailModalFixture(page, bundle);
            const field = page.getByTestId('tag-field');
            const input = field.locator('input');
            for (const raw of ['\t', '\u0000', '\u007f', ' \t ']) {
                await input.fill(raw);
                await input.press('Enter');
                await expect(field.locator('span.text-xs')).toHaveCount(2);
            }
            await input.fill('Canonical\tName');
            await input.press('Enter');
            await expect(field.locator('span.text-xs')).toHaveCount(3);
            await input.fill('Canonical\vName');
            await input.press('Enter');
            await expect(field.locator('span.text-xs')).toHaveCount(3);
            // An existing space-only record remains selectable by its raw name.
            await input.fill('  ');
            await input.press('Enter');
            await expect(field.locator('span.text-xs')).toHaveCount(4);
            const selected = await page.evaluate(() => (window as any).inspectTagSelection());
            expect(selected.slice(2).map(({ text, name, id }: { text: string; name: string; id: string }) => ({ text, name, id }))).toEqual([
                { text: 'Canonical Name', name: 'Canonical Name', id: 'tag000000000002' },
                { text: '  ', name: '  ', id: 'tag000000000004' },
            ]);
            expect(await page.evaluate(() => (window as any).inspectOriginalTagRecords())).toEqual([
                { id: 'tag000000000001', name: '  Existing  ' },
                { id: 'tag000000000002', name: 'Canonical Name' },
                { id: 'tag000000000003', name: '' },
                { id: 'tag000000000004', name: '  ' },
            ]);
            expect(errors).toEqual([]);
        });

        test('retains generic combobox behavior without the tag callback', async ({ page }) => {
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            await mountTrailModalFixture(page, bundle);
            const field = page.getByTestId('generic-field');
            const input = field.locator('input');
            for (const raw of ['   ', 'Word\tWord']) {
                await input.fill(raw);
                await input.press('Enter');
            }
            await expect(field.locator('span.text-xs')).toHaveCount(2);
            expect(await page.evaluate(() => (window as any).inspectGenericSelection())).toEqual(['   ', 'Word\tWord']);
            expect(errors).toEqual([]);
        });
    });
}
