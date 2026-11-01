import { readFile } from "node:fs/promises";
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

for (const mode of ["busy", "idle", "enable"]) {
    test(`manual sync button follows the instance lock (${mode})`, async ({ page }) => {
        const busy = mode === "busy";
        const pageSource = await readFile('src/routes/settings/plugins/+page.svelte', 'utf8');
        const instance = { id: 'example', user: 'tester', plugin_id: 'example', enabled: mode !== 'enable', status: 'configured', busy };
        const source = `
            <script>
                import Settings from 'fixture:plugin-page.svelte';
                const data = {
                    pluginInstances: [${JSON.stringify(instance)}],
                    pluginProviders: [{ id: 'example', name: 'Example', type: 'trails', status: 'available', capabilities: ['list_routes.v1'], auth: { type: 'none' } }],
                    categories: [], subcategories: []
                };
            </script>
            <Settings {data} />
        `;
        const bundle = await buildTrailModalFixture(false, source, { '\0fixture:plugin-page.svelte': pageSource });
        await mountTrailModalFixture(page, bundle);
        let starts = 0;
        await page.route('**/api/v1/plugin-system/sync', async (route) => {
            starts++;
            await route.fulfill({ status: 202, json: { started: true } });
        });
        await page.route('**/api/v1/plugin-instance?*', (route) => route.fulfill({ json: { items: [{ ...instance, enabled: true, busy: false }] } }));
        await page.route('**/api/v1/plugin-instance/example', (route) => route.fulfill({ json: { ...instance, enabled: true } }));
        await page.route('**/api/v1/plugin-system/category-remap/preview', (route) => route.fulfill({ json: { count: 0 } }));
        const button = page.getByRole('button', { name: 'plugin-sync-now', exact: true });
        if (busy) {
            await expect(button).toBeDisabled();
        } else {
            if (mode === "enable") await page.locator('.plugin-card-toggle label').click();
            else await button.click();
            await expect.poll(() => starts).toBe(1);
            await expect(button).toBeDisabled();
        }
        await expect(button).toBeEnabled({ timeout: 10000 });
        expect(starts).toBe(busy ? 0 : 1);
    });
}

for (const change of ['switch-off', 'unmount']) {
    test(`a late sync refresh respects ${change}`, async ({ page }) => {
        const pageSource = await readFile('src/routes/settings/plugins/+page.svelte', 'utf8');
        const instance = { id: 'example', user: 'tester', plugin_id: 'example', enabled: true, status: 'configured', busy: true };
        const source = `
            <script>
                import Settings from 'fixture:plugin-page.svelte';
                import { pluginInstances } from '#lib/stores/plugin_instance_store.ts';
                window.pollUpdates = 0;
                pluginInstances.subscribe((items) => { window.pollUpdates++; window.polledInstances = items; });
                const data = {
                    pluginInstances: [${JSON.stringify(instance)}],
                    pluginProviders: [{ id: 'example', name: 'Example', type: 'trails', status: 'available', capabilities: ['list_routes.v1'], auth: { type: 'none' } }],
                    categories: [], subcategories: []
                };
            </script>
            <Settings {data} />
        `;
        const bundle = await buildTrailModalFixture(false, source, { '\0fixture:plugin-page.svelte': pageSource });
        await mountTrailModalFixture(page, bundle);
        let releasePoll: () => void;
        const pendingPoll = new Promise<void>((resolve) => { releasePoll = resolve; });
        let pollStarted: () => void;
        const started = new Promise<void>((resolve) => { pollStarted = resolve; });
        await page.route('**/api/v1/plugin-instance?*', async (route) => {
            pollStarted();
            await pendingPoll;
            await route.fulfill({ json: { items: [{ ...instance, busy: false }] } });
        });
        await page.route('**/api/v1/plugin-instance/example', (route) => route.fulfill({ json: { ...instance, enabled: false, busy: false } }));
        await started;
        if (change === 'unmount') {
            const aborted = page.waitForEvent('requestfailed', (request) => request.url().includes('/api/v1/plugin-instance?'));
            await page.evaluate(() => (window as any).unmountModalFixture());
            releasePoll!();
            await aborted;
            expect(await page.evaluate(() => (window as any).pollUpdates)).toBe(1);
            return;
        }
        await page.locator('.plugin-card-toggle label').click();
        const toggle = page.locator('.plugin-card-toggle input');
        await expect(toggle).not.toBeChecked();
        await expect(page.getByRole('button', { name: 'plugin-sync-now', exact: true })).toHaveCount(0);
        releasePoll!();
        await page.waitForFunction(() => (window as any).pollUpdates > 1);
        await expect(toggle).not.toBeChecked();
        await expect.poll(() => page.evaluate(() => (window as any).polledInstances[0]?.enabled)).toBe(false);
    });
}

test.describe('refused syncs', () => {
    // The browser logs each refused sync request; this test refuses them on purpose.
    test.use({ browserErrorAllowlist: [{
        source: 'console.error',
        message: /status of (403|429)/,
        url: /\/api\/v1\/plugin-system\/sync$/,
        issue: 'https://github.com/open-wanderer/wanderer/pull/1469',
    }] });

    test('a sync started by switching a plugin on stays quiet when it cannot start', async ({ page }) => {
        const pageSource = await readFile('src/routes/settings/plugins/+page.svelte', 'utf8');
        const instance = { id: 'example', user: 'tester', plugin_id: 'example', enabled: false, status: 'configured', busy: false };
        const source = `
            <script>
                import Settings from 'fixture:plugin-page.svelte';
                const data = {
                    pluginInstances: [${JSON.stringify(instance)}],
                    pluginProviders: [{ id: 'example', name: 'Example', type: 'trails', status: 'available', capabilities: ['list_routes.v1'], auth: { type: 'none' } }],
                    categories: [], subcategories: []
                };
            </script>
            <Settings {data} />
        `;
        const bundle = await buildTrailModalFixture(false, source, { '\0fixture:plugin-page.svelte': pageSource });
        await mountTrailModalFixture(page, bundle);
        let status = 429;
        let starts = 0;
        await page.route('**/api/v1/plugin-system/sync', async (route) => {
            starts++;
            await route.fulfill({ status, json: { message: 'refused', data: { retryAfter: 112 } } });
        });
        await page.route('**/api/v1/plugin-instance/example', (route) => route.fulfill({ json: { ...instance, enabled: true } }));
        await page.route('**/api/v1/plugin-system/category-remap/preview', (route) => route.fulfill({ json: { count: 0 } }));
        const errors = () => page.evaluate(() => ((window as any).toasts ?? []).filter((toast: any) => toast.type === 'error').map((toast: any) => toast.text));

        await page.locator('.plugin-card-toggle label').click();
        await expect.poll(() => starts).toBe(1);
        // A click asks for the sync, so it explains each refusal. Its toast comes
        // after any the switch-on sync would have shown.
        const button = page.getByRole('button', { name: 'plugin-sync-now', exact: true });
        await button.click();
        await expect.poll(errors).toEqual(['plugin-sync-too-soon']);
        status = 403;
        await button.click();
        await expect.poll(errors).toEqual(['plugin-sync-too-soon', 'plugin-sync-off']);
    });
});
