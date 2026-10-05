import { expect, test, type Page } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

async function openSecondEditorLink(page: Page) {
    await page.getByTestId('editor-open-parent').click();
    const parent = page.locator('#fixture-editor-parent-modal');
    await expect(parent).toHaveAttribute('open');
    const editor = page.getByTestId('editor-beta');
    await expect(editor.locator('.ProseMirror')).toHaveText('Beta content');
    await editor.locator('button[aria-label="Link"]').first().click();
    const linkDialog = editor.locator('dialog[open]');
    await expect(linkDialog).toHaveCount(1);
    return { parent, editor, linkDialog };
}

for (const dev of [true, false]) {
    test.describe(dev ? 'development editors' : 'production editors', () => {
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

        test('inserts a link only into the second editor inside its parent dialog', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const { parent, editor, linkDialog } = await openSecondEditorLink(page);
            await linkDialog.locator('input[type="text"]').nth(0).fill('https://example.com/beta');
            await linkDialog.locator('input[type="text"]').nth(1).fill('Beta link');
            await linkDialog.locator('button[name="save"]').click();
            await expect(editor.locator('.ProseMirror a')).toHaveAttribute('href', 'https://example.com/beta');
            await expect(editor.locator('.ProseMirror a')).toHaveText('Beta link');
            await expect(page.getByTestId('editor-alpha').locator('.ProseMirror')).toHaveText('Alpha content');
            await expect(page.getByTestId('editor-alpha').locator('.ProseMirror a')).toHaveCount(0);
            await expect(editor.locator('dialog[open]')).toHaveCount(0);
            await expect(parent).toHaveAttribute('open');
            expect(await page.evaluate(() => document.body.style.position)).toBe('fixed');
            await page.getByTestId('editor-close-parent').click();
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => page.evaluate(() => document.body.style.position)).toBe('');
        });

        test('cancels the second link dialog and keeps both editors unchanged', async ({ page }) => {
            await mountTrailModalFixture(page, bundle);
            const { parent, editor, linkDialog } = await openSecondEditorLink(page);
            await linkDialog.locator('input[type="text"]').nth(0).fill('https://example.com/cancelled');
            await linkDialog.locator('input[type="text"]').nth(1).fill('Cancelled link');
            await linkDialog.locator('button').filter({ hasText: /^cancel$/ }).click();
            await expect(editor.locator('dialog[open]')).toHaveCount(0);
            await expect(parent).toHaveAttribute('open');
            await expect(editor.locator('.ProseMirror')).toHaveText('Beta content');
            await expect(page.getByTestId('editor-alpha').locator('.ProseMirror')).toHaveText('Alpha content');
            await expect(page.locator('.ProseMirror a')).toHaveCount(0);
            expect(await page.evaluate(() => document.body.style.position)).toBe('fixed');
            await page.keyboard.press('Escape');
            await expect(page.locator('dialog[open]')).toHaveCount(0);
            await expect.poll(() => page.evaluate(() => document.body.style.position)).toBe('');
        });
    });
}
