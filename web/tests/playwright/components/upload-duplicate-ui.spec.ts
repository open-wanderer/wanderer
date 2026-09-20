import { expect, test, type Page } from '@playwright/test';
import { buildUploadDuplicateFixture, mountUploadDuplicateFixture } from './upload-duplicate-fixture';

const publicToggle = (page: Page) => page.getByRole('checkbox', { name: "Include other users' public trails" });
const sharedToggle = (page: Page) => page.getByRole('checkbox', { name: 'Include trails shared with me' });
const saveButton = (page: Page) => page.getByRole('button', { name: 'Save', exact: true });

async function expectScope(page: Page, includePublic: boolean, includeShared: boolean) {
    await expect(publicToggle(page)).toBeChecked({ checked: includePublic });
    await expect(sharedToggle(page)).toBeChecked({ checked: includeShared });
}

for (const dev of [true, false]) {
    test.describe(dev ? 'upload duplicates development' : 'upload duplicates production', () => {
        let bundle: string;
        const runtimeErrors: string[] = [];

        test.beforeAll(async () => { bundle = await buildUploadDuplicateFixture(dev); });
        test.beforeEach(async ({ page }) => {
            runtimeErrors.length = 0;
            page.on('pageerror', error => runtimeErrors.push(error.message));
            page.on('console', message => {
                if (/ownership_invalid_(mutation|binding)/.test(message.text())) runtimeErrors.push(message.text());
            });
        });
        test.afterEach(() => { expect(runtimeErrors).toEqual([]); });

        test('defaults to own trails and independently persists both options across remounts', async ({ page }) => {
            const fixture = await mountUploadDuplicateFixture(page, bundle);
            await expectScope(page, false, false);
            await expect(saveButton(page)).toBeDisabled();
            for (const [includePublic, includeShared] of [[true, false], [false, true], [true, true], [false, false]]) {
                await publicToggle(page).setChecked(includePublic);
                await sharedToggle(page).setChecked(includeShared);
                await saveButton(page).click();
                await expect(saveButton(page)).toBeDisabled();
                await expect.poll(() => fixture.server.settings.uploadDuplicateCheck).toEqual({ includePublic, includeShared });
                await expect.poll(() => page.evaluate(() => window.uploadDuplicateFixture.toasts.length)).toBe(fixture.saves.length);
                expect(fixture.saves.at(-1)).toEqual({ id: 'settings-a', uploadDuplicateCheck: { includePublic, includeShared } });
                await fixture.remount();
                await expectScope(page, includePublic, includeShared);
                await expect(saveButton(page)).toBeDisabled();
            }
        });

        test('keeps an unsaved draft after a failed save and allows retry', async ({ page }) => {
            const fixture = await mountUploadDuplicateFixture(page, bundle, { id: 'settings-a', uploadDuplicateCheck: null });
            await expectScope(page, false, false);
            await publicToggle(page).check();
            fixture.server.status = 500;
            await saveButton(page).click();
            await expect.poll(() => page.evaluate(() => window.uploadDuplicateFixture.toasts.at(-1)?.text)).toBe('Error saving settings');
            await expectScope(page, true, false);
            await expect(saveButton(page)).toBeEnabled();
            expect(fixture.server.settings.uploadDuplicateCheck).toBeNull();
            expect(await page.evaluate(() => window.uploadDuplicateFixture.invalidations)).toBe(0);
            fixture.server.status = 200;
            await saveButton(page).click();
            await expect.poll(() => page.evaluate(() => window.uploadDuplicateFixture.toasts.at(-1)?.text)).toBe('Settings saved');
            await fixture.remount();
            await expectScope(page, true, false);
        });

        test('refreshes clean settings, preserves local edits, and resets drafts for another settings ID', async ({ page }) => {
            const fixture = await mountUploadDuplicateFixture(page, bundle);
            await fixture.refresh({ id: 'settings-a', uploadDuplicateCheck: { includeShared: true } });
            await expectScope(page, false, true);
            await expect(saveButton(page)).toBeDisabled();
            await publicToggle(page).check();
            await fixture.refresh({ id: 'settings-a', uploadDuplicateCheck: { includePublic: true } });
            await expectScope(page, true, true);
            await expect(saveButton(page)).toBeEnabled();
            await fixture.refresh({ id: 'settings-b', uploadDuplicateCheck: { includeShared: true } });
            await expectScope(page, false, true);
            await expect(saveButton(page)).toBeDisabled();
            expect(fixture.saves).toEqual([]);
        });

        test('ignores a late save after switching settings IDs and preserves the new draft', async ({ page }) => {
            const fixture = await mountUploadDuplicateFixture(page, bundle);
            let release!: () => void;
            fixture.server.gate = new Promise<void>(resolve => { release = resolve; });
            const invalidated = page.waitForResponse('**/fixture/settings');
            try {
                await publicToggle(page).check();
                await saveButton(page).click();
                await expect.poll(() => fixture.saves.length).toBe(1);
                await expect(publicToggle(page)).toBeDisabled();
                await expect(sharedToggle(page)).toBeDisabled();
                await fixture.refresh({ id: 'settings-b', uploadDuplicateCheck: { includeShared: true } });
                await expectScope(page, false, true);
                await expect(publicToggle(page)).toBeEnabled();
                await publicToggle(page).check();
            } finally {
                release();
            }
            await expect.poll(() => page.evaluate(() => window.uploadDuplicateFixture.invalidations)).toBe(1);
            await (await invalidated).finished();
            await expectScope(page, true, true);
            await expect(saveButton(page)).toBeEnabled();
            expect(await page.evaluate(() => window.uploadDuplicateFixture.toasts)).toEqual([]);
            expect(fixture.saves).toEqual([{ id: 'settings-a', uploadDuplicateCheck: { includePublic: true, includeShared: false } }]);
        });

        for (const duplicate of [
            { id: 'own-private', name: 'Private morning walk', author: 'own-actor', domain: 'alice', ownerText: 'Your trail' },
            { id: 'remote-trail', name: 'Mountain walk', author: 'other-actor', domain: 'bob@remote.example', ownerText: 'Trail by @bob@remote.example' },
        ]) {
            test(`shows ${duplicate.id} ownership and forwards force upload as true`, async ({ page }) => {
                const fixture = await mountUploadDuplicateFixture(page, bundle);
                fixture.server.duplicate = duplicate;
                await page.locator('#file-input').setInputFiles({ name: 'walk.gpx', mimeType: 'application/gpx+xml', buffer: Buffer.from('<gpx/>') });
                await expect(page.getByText('Similar trail found:', { exact: false })).toBeVisible();
                await expect(page.getByText(duplicate.ownerText, { exact: true })).toBeVisible();
                await page.getByRole('button', { name: duplicate.name, exact: true }).click();
                expect(await page.evaluate(() => window.uploadDuplicateFixture.navigations)).toEqual([`/trail/view/@${duplicate.domain}/${duplicate.id}`]);
                expect(fixture.uploads).toEqual([{ filename: 'walk.gpx', ignoreDuplicates: false }]);
                await page.getByRole('button', { name: 'Upload as your own trail despite duplicate', exact: true }).click();
                await expect.poll(() => page.evaluate(() => window.uploadDuplicateFixture.uploads().map(upload => upload.status))).toEqual(['success']);
                expect(fixture.uploads).toEqual([
                    { filename: 'walk.gpx', ignoreDuplicates: false },
                    { filename: 'walk.gpx', ignoreDuplicates: true },
                ]);
                await expect(page.getByText(duplicate.ownerText, { exact: true })).toHaveCount(0);
            });
        }
    });
}
