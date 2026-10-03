import { expect, test, type Page } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

function modalCalls(page: Page) {
    return page.evaluate(() => (window as any).modalCalls);
}

async function openSecond(page: Page, kind: string) {
    await page.getByTestId(`beta-${kind}`).click();
    const dialog = page.getByTestId('beta').locator('dialog[open]');
    await expect(dialog).toHaveCount(1);
    return dialog;
}

for (const dev of [true, false]) {
    test.describe(dev ? 'development' : 'production', () => {
        let bundle: string;
        const runtimeErrors: string[] = [];

        test.beforeAll(async () => {
            bundle = await buildTrailModalFixture(dev);
        });
        test.beforeEach(({ page }) => {
            runtimeErrors.length = 0;
            page.on('pageerror', error => runtimeErrors.push(error.message));
        });
        test.afterEach(() => {
            expect(runtimeErrors).toEqual([]);
        });

        test('keeps dialog and input ids unique across cards and modal types', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const ids = await page.locator('[id]').evaluateAll(elements => elements.map(element => element.id));
            expect(new Set(ids).size).toBe(ids.length);
        });

        test('duplicates the second trail with its own photo defaults and options', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'duplicate');
            await expect(dialog).toContainText('Trail Beta');
            await expect(dialog.locator('input').nth(1)).toBeChecked();
            await dialog.locator('input').nth(2).uncheck();
            await dialog.locator('.btn-primary').click();
            expect(await modalCalls(page)).toEqual([{ type: 'duplicate', trailId: 'beta', settings: {
                waypoints: false, summitLogs: false, trailPhotos: true, waypointPhotos: false, summitLogPhotos: false,
            } }]);
            await expect(page.locator('dialog[open]')).toHaveCount(0);
        });

        test('exports the second trail and changes its checkboxes through their labels', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'export');
            await dialog.locator('select').selectOption('json');
            await dialog.locator('label').filter({ hasText: /^photos$/ }).click();
            await dialog.locator('label').filter({ hasText: /^summit-book$/ }).click();
            for (const checkbox of await dialog.locator('input[type=checkbox]').all()) {
                await expect(checkbox).toBeChecked();
            }
            const alphaExport = page.getByTestId('alpha').locator('dialog').filter({ has: page.locator('option[value="json"]') });
            for (const checkbox of await alphaExport.locator('input[type=checkbox]').all()) {
                await expect(checkbox).not.toBeChecked();
            }
            await dialog.locator('.btn-primary').click();
            expect(await modalCalls(page)).toEqual([{ type: 'export', trailId: 'beta', settings: {
                fileFormat: 'json', photos: true, summitLog: true,
            } }]);
            await expect(page.locator('dialog[open]')).toHaveCount(0);
        });

        test('shares the second trail using its own radio group and close callback', async ({ page }) => {
            const writes = await mountTrailModalFixture(page, bundle);
            const radios = page.locator('input[type=radio]');
            expect(new Set(await radios.evaluateAll(inputs => inputs.map(input => input.getAttribute('name')))).size).toBe(2);
            await expect(page.getByTestId('alpha').locator('input[type=radio]').first()).toBeChecked();
            await expect(page.getByTestId('beta').locator('input[type=radio]').first()).toBeChecked();
            const dialog = await openSecond(page, 'share');
            await dialog.locator('label').filter({ hasText: 'everyone-with-the-link' }).click();
            await expect.poll(() => writes).toEqual([{ path: '/api/v1/trail-link-share', body: {
                trail: 'beta', permission: 'view',
            } }]);
            await dialog.locator('.btn-primary').click();
            expect(await modalCalls(page)).toEqual([{ type: 'share', trailId: 'beta' }]);
            await expect(page.locator('dialog[open]')).toHaveCount(0);
        });

        test('sends the second trail to the selected plugin', async ({ page }) => {
            const writes = await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'send');
            await dialog.locator('button').filter({ hasText: 'Test Device' }).click();
            await expect.poll(() => writes).toEqual([{ path: '/api/v1/plugin-system/trail-send', body: {
                pluginId: 'device', trailId: 'beta',
            } }]);
            await expect(page.locator('dialog[open]')).toHaveCount(0);
        });

        test('merges the second selection and scopes all checkbox labels to that dialog', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'merge');
            await expect(dialog.locator('select')).toHaveValue('beta');
            for (const key of ['summit-log', 'photos', 'comment', 'tags', 'likes', 'delete-linked-trails']) {
                const label = dialog.locator('label').filter({ hasText: new RegExp(`^${key}$`) });
                await label.click();
            }
            await dialog.locator('label').filter({ hasText: 'trail-merge-remember-settings-short' }).click();
            await dialog.locator('.btn-primary').click();
            expect(await modalCalls(page)).toEqual([{ type: 'merge', trailId: 'beta', settings: {
                summitLog: false, photos: false, comments: false, delete: false, tags: false, likes: false,
            }, selection: {
                targetTrail: { id: 'beta', name: 'Trail Beta', public: true },
                sourceTrails: [{ id: 'beta-source', name: 'beta source' }],
            } }]);
            expect(await page.evaluate(() => localStorage.getItem('trail_merge_remember'))).toBe('true');
            await expect(page.locator('dialog[open]')).toHaveCount(0);
        });

        test('confirms deletion only for the second trail', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'delete');
            await expect(dialog).toContainText('Delete Trail Beta?');
            await dialog.locator('button[name="delete"]').click();
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => modalCalls(page)).toEqual([{ type: 'delete', trailId: 'beta' }]);
        });

        test('cancels the second deletion without confirming either trail', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'delete');
            await dialog.locator('.btn-secondary').click();
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => modalCalls(page)).toEqual([{ type: 'cancel-delete', trailId: 'beta' }]);
            await openSecond(page, 'delete');
            await page.keyboard.press('Escape');
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => page.evaluate(() => ({
                position: document.body.style.position,
                top: document.body.style.top,
                width: document.body.style.width,
            }))).toEqual({ position: '', top: '', width: '' });
            await expect.poll(() => modalCalls(page)).toEqual([
                { type: 'cancel-delete', trailId: 'beta' },
                { type: 'cancel-delete', trailId: 'beta' },
            ]);
        });

        test('preserves an explicitly supplied confirmation dialog id', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            await page.getByTestId('named-confirm').click();
            const dialog = page.locator('#custom-confirm-dialog[open]');
            await expect(dialog).toHaveCount(1);
            await dialog.locator('button[name="delete"]').click();
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => modalCalls(page)).toEqual([{ type: 'custom-confirm', trailId: 'custom' }]);
        });

        test('changes list membership only for the second trail', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'list');
            await expect(dialog).toContainText('beta list');
            await dialog.locator('button[aria-label="unlink"]').click();
            await expect.poll(() => modalCalls(page)).toEqual([{ type: 'list', trailId: 'beta', settings: { listId: 'beta-list' } }]);
            await expect(dialog).not.toContainText('beta list');
            await expect(page.getByTestId('alpha')).toContainText('alpha list');
            await dialog.locator('.btn-primary').click();
            await expect(page.locator('dialog[open]')).toHaveCount(0);
        });

        test('applies bulk edits only to the second selection', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'bulk-edit');
            await dialog.locator('label').filter({ hasText: /^\s*difficulty\s*$/ }).click();
            await dialog.locator('select').selectOption('difficult');
            await dialog.locator('.btn-primary').click();
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => modalCalls(page)).toEqual([{ type: 'bulk-edit', trailId: 'beta', settings: { difficulty: 'difficult' } }]);
        });

        test('restores body styles and scrolling for every way of closing a dialog', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            await page.evaluate(() => {
                document.body.style.minHeight = '4000px';
                document.body.style.setProperty('position', 'relative', 'important');
                document.body.style.top = '4px';
                document.body.style.width = '83%';
            });
            const bodyState = () => page.evaluate(() => ({
                position: document.body.style.position,
                priority: document.body.style.getPropertyPriority('position'),
                top: document.body.style.top,
                width: document.body.style.width,
                scrollY: window.scrollY,
            }));
            for (const action of ['cancel', 'confirm', 'header', 'escape']) {
                await page.evaluate(() => window.scrollTo({ top: 350, behavior: 'instant' }));
                const before = await bodyState();
                expect(before.scrollY).toBe(350);
                // Avoid Playwright scrolling the off-screen trigger into view.
                await page.getByTestId('beta-delete').evaluate(button => (button as HTMLButtonElement).click());
                const dialog = page.getByTestId('beta').locator('dialog[open]');
                await expect(dialog).toHaveCount(1);
                expect(await page.evaluate(() => document.body.style.position)).toBe('fixed');
                if (action === 'escape') await page.keyboard.press('Escape');
                else if (action === 'confirm') await dialog.locator('button[name="delete"]').click();
                else if (action === 'header') await dialog.locator('.btn-icon').click();
                else await dialog.locator('.btn-secondary').click();
                await expect(page.locator('dialog[open]')).toHaveCount(0);
                await expect.poll(bodyState).toEqual(before);
                await page.evaluate(() => window.scrollTo({ top: 450, behavior: 'instant' }));
                await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(450);
            }
        });

        test('keeps scrolling locked when a dialog reopens before its queued close event', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const dialog = await openSecond(page, 'delete');
            await dialog.evaluate(element => {
                (element as HTMLDialogElement).close();
                const trigger = document.querySelector<HTMLButtonElement>('[data-testid="beta-delete"]')!;
                trigger.click();
                trigger.click();
            });
            // Wait for the old close event before checking the reopened dialog.
            await expect.poll(() => modalCalls(page)).toEqual([{ type: 'cancel-delete', trailId: 'beta' }]);
            await expect(dialog).toHaveCount(1);
            expect(await page.evaluate(() => document.body.style.position)).toBe('fixed');
            await page.keyboard.press('Escape');
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => page.evaluate(() => document.body.style.position)).toBe('');
        });

        test('releases scroll locks when an open parent and child are unmounted', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            await page.evaluate(() => {
                document.body.style.minHeight = '4000px';
                window.scrollTo({ top: 350, behavior: 'instant' });
            });
            await page.getByTestId('editor-open-parent').evaluate(button => (button as HTMLButtonElement).click());
            await page.getByTestId('editor-beta').locator('button[aria-label="Link"]').first().click();
            await expect(page.locator('dialog[open]')).toHaveCount(2);
            expect(await page.evaluate(() => document.body.style.position)).toBe('fixed');
            await page.evaluate(() => (window as any).unmountModalFixture());
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => page.evaluate(() => ({
                position: document.body.style.position,
                top: document.body.style.top,
                width: document.body.style.width,
                scrollY: window.scrollY,
            }))).toEqual({ position: '', top: '', width: '', scrollY: 350 });
            expect(await modalCalls(page)).toEqual([]);
        });
    });
}
