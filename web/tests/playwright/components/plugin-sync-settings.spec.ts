import { expect, test } from '../fixtures';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

const source = `
    <script>
        import Settings from '#lib/components/settings/plugins/plugin_instance_settings_modal.svelte';
        const plugin = { id: 'example', name: 'Example', type: 'trails', status: 'available', auth: { type: 'oauth2' } };
        const instance = { id: 'example', user: 'tester', plugin_id: 'example', enabled: true, status: 'syncing' };
        function save(value) { window.modalCalls.push(value); }
    </script>
    <Settings {plugin} {instance} onsave={save} />
`;

for (const dev of [true, false]) {
    test(`saving settings during a sync keeps OAuth connected (${dev ? 'development' : 'production'})`, async ({ page }) => {
        const bundle = await buildTrailModalFixture(dev, source);
        await mountTrailModalFixture(page, bundle);
        const dialog = page.locator('dialog');
        await dialog.evaluate((element: HTMLDialogElement) => element.showModal());
        await expect(dialog).toContainText('plugin-oauth-connected-hint');
        await expect(dialog).not.toContainText('plugin-oauth-needs-connect-hint');
        await dialog.locator('button[name=save]').click();
        await expect.poll(() => page.evaluate(() => (window as any).modalCalls)).toMatchObject([
            { id: 'example', enabled: true, status: 'syncing' },
        ]);
        await expect(dialog).not.toHaveAttribute('open');
    });
}
