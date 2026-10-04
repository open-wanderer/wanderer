import { expect, test } from '@playwright/test';
import { buildTrailModalFixture } from './trail-modal-fixture';

const source = `
    <script>
        import Combobox from '$lib/components/base/combobox.svelte';
        let items = $state(window.fixtureItems);
        let selected = $state([]);
        let single = $state('');
        function update() { items = [...items]; }
    </script>
    <section data-testid="multiple">
        <Combobox multiple chips bind:value={selected} {items} onupdate={update} />
    </section>
    <section data-testid="single">
        <Combobox bind:value={single} {items} onupdate={update} />
    </section>
`;

for (const dev of [true, false]) {
    test.describe(dev ? 'combobox development' : 'combobox production', () => {
        let bundle: string;
        test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, source); });

        async function mount(page: import('@playwright/test').Page, items: string[]) {
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            await page.route('http://combobox.test/**', route => route.fulfill({
                contentType: 'text/html', body: '<!doctype html><html><body></body></html>',
            }));
            await page.goto('http://combobox.test/');
            await page.evaluate(items => {
                Object.assign(window, {
                    fixtureItems: items.map(text => ({ text, value: text })),
                    comboboxExecuted: 0,
                });
            }, items);
            await page.addScriptTag({ content: bundle });
            await expect(page.locator('input[type=search]')).toHaveCount(2);
            return errors;
        }

        test('renders raw markup and entities literally in suggestions and selected chips', async ({ page }) => {
            const attack = '<img src="/missing" onerror="window.comboboxExecuted++">';
            const items = ['payload ' + attack, 'payload ' + attack.replaceAll('<', '&lt;').replaceAll('>', '&gt;'), 'payload &amp; Grün'];
            const errors = await mount(page, items);
            const multiple = page.getByTestId('multiple');
            await multiple.locator('input').fill('payload');
            await expect(multiple.locator('.menu-item p')).toHaveText(items);
            await expect(multiple.locator('.menu-item strong')).toHaveText(['payload', 'payload', 'payload']);
            await expect(page.locator('img, script[src], svg, iframe')).toHaveCount(0);
            expect(await page.evaluate(() => (window as any).comboboxExecuted)).toBe(0);

            await multiple.locator('.menu-item').nth(1).click();
            await expect(multiple.getByText(items[1], { exact: true })).toBeVisible();
            await expect(page.locator('img')).toHaveCount(0);
            expect(await page.evaluate(() => (window as any).comboboxExecuted)).toBe(0);
            expect(errors).toEqual([]);
        });

        test('highlights regex metacharacters as literal text without runtime errors', async ({ page }) => {
            const errors = await mount(page, ['Grüezi [route].* (🌍) \\ $']);
            const single = page.getByTestId('single');
            for (const query of ['[route]', '.*', '(', '\\', '$', 'grüezi']) {
                await single.locator('input').fill(query);
                await expect(single.locator('.menu-item strong')).toHaveText(query === 'grüezi' ? 'Grüezi' : query);
                await expect(single.locator('.menu-item p')).toHaveText('Grüezi [route].* (🌍) \\ $');
            }
            expect(errors).toEqual([]);
        });

        test('updates its own highlighting without mutating another menu', async ({ page }) => {
            const errors = await mount(page, ['Alpine alpine']);
            await page.evaluate(() => {
                const menu = document.createElement('ul');
                menu.className = 'menu';
                menu.dataset.testid = 'unrelated-menu';
                const item = document.createElement('li');
                const text = document.createElement('p');
                text.textContent = 'Unrelated menu text';
                item.append(text);
                menu.append(item);
                document.body.prepend(menu);
            });
            const multiple = page.getByTestId('multiple');
            await multiple.locator('input').fill('alpine');
            await expect(multiple.locator('.menu-item strong')).toHaveText(['Alpine', 'alpine']);
            await expect(page.getByTestId('unrelated-menu')).toHaveText('Unrelated menu text');
            expect(errors).toEqual([]);
        });
    });
}
