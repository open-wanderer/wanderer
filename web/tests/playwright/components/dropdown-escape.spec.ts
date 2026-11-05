import { expect, test } from '../fixtures';
import { resolve } from 'node:path';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';
const source = `<script>
    import { page } from '$app/state';
    page.data.notifications = {items: [], page: 0, totalPages: 0};
    import Dropdown from '#lib/components/base/dropdown.svelte';
    import Notifications from '#lib/components/notification/notification_dropdown.svelte';
    import Modal from '#lib/components/base/modal.svelte';
    let modal; let escapes = $state(0);
</script>
<input aria-label="Outside" onkeydown={event => { if (event.key === 'Escape') escapes++; }} /><output>{escapes}</output>
<Dropdown items={[{text: 'Profile', value: 'profile'}]}>
    {#snippet children({toggleMenu})}<button onclick={toggleMenu}>User menu</button>{/snippet}
</Dropdown><Notifications />
<button onclick={() => modal.openModal()}>Dialog</button>
<Modal id="test" title="Test" bind:this={modal}>{#snippet content()}<input aria-label="Dialog input" />{/snippet}</Modal>`;
for (const dev of [true, false]) test.describe(`dropdown Escape (${dev ? 'development' : 'production'})`, () => {
    let bundle: string;
    test.beforeAll(async () => { bundle = await buildTrailModalFixture(dev, source, {
        [resolve('src/lib/stores/notification_store.ts')]: '<script module>export async function notifications_index() { return {items: [], page: 1, totalPages: 1}; } export async function notifications_mark_as_seen() {}</script>',
    });
    });
    for (const name of ['User menu', 'Toggle notification dropdown']) {
        test(`${name}: closes from body and returns focus`, async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const trigger = page.getByRole('button', {name, exact: true});
            await trigger.click(); await expect(page.locator('ul.menu')).toBeVisible();
            await page.evaluate(() => (document.activeElement as HTMLElement).blur());
            await page.keyboard.press('Escape');
            await expect(page.locator('ul.menu')).toHaveCount(0); await expect(trigger).toBeFocused();
            await trigger.click(); await page.keyboard.press('Escape');
            await expect(page.locator('ul.menu')).toHaveCount(0); await expect(trigger).toBeFocused();
        });
        test(`${name}: external field and dialog own Escape`, async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            await page.getByRole('button', {name, exact: true}).click();
            await page.getByRole('textbox', {name: 'Outside', exact: true}).focus();
            await page.keyboard.press('Escape');
            await expect(page.locator('ul.menu')).toBeVisible(); await expect(page.locator('output')).toHaveText('1');
            await page.getByRole('button', {name: 'Dialog', exact: true}).evaluate((node: HTMLButtonElement) => node.click());
            await page.getByRole('textbox', {name: 'Dialog input', includeHidden: true}).focus();
            await page.keyboard.press('Escape');
            await expect(page.locator('dialog')).not.toHaveAttribute('open'); await expect(page.locator('ul.menu')).toBeVisible();
        });
    }
});
