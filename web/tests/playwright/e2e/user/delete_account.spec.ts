import { expect, test } from '../../fixtures';

test.describe('account deletion when the server rejects it', () => {
    test.use({
        browserErrorAllowlist: [{
            source: 'console.error',
            message: /^Failed to load resource: the server responded with a status of 400 \(Bad Request\)$/,
            url: /\/api\/v1\/user\/\w+$/,
            issue: 'https://github.com/open-wanderer/wanderer/issues/1515',
        }],
    });

    test('shows an error and stays on the page', async ({ page }) => {
        // The shared test user is deleted for real by auth.teardown.ts.
        await page.route('**/api/v1/user/*', route => route.request().method() === 'DELETE'
            ? route.fulfill({ status: 400, json: { message: 'Failed to delete record.' } })
            : route.continue());

        await page.goto('/settings/account', { waitUntil: 'networkidle' });
        await page.locator('#delete-account').click();
        await page.locator('dialog[open] button[name="delete"]').click();

        await expect(page.getByRole('alert')).toContainText('Error deleting account');
        await expect(page).toHaveURL('/settings/account');
    });
});
