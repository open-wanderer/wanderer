import { randomUUID } from 'node:crypto';
import { expect, test } from '../../fixtures';

// New accounts start with "en", so the unsupported ja-JP fallback is only
// distinguishable from a skipped save by the settings write itself.
for (const [locale, language] of [['de-CH', 'de'], ['fr-FR', 'fr'], ['ja-JP', 'en']] as const) {
    test.describe(`registration with ${locale}`, () => {
        test.use({ locale, storageState: { cookies: [], origins: [] } });

        test('creates an account, signs in and persists the browser language', async ({ page, request }) => {
            const username = `register${randomUUID().replaceAll('-', '').slice(0, 12)}`;
            const password = 'registration-password';
            let userId: string | undefined;
            try {
                await page.goto('/register', { waitUntil: 'networkidle' });
                await page.locator('input[name="username"]').fill(username);
                await page.locator('input[name="email"]').fill(`${username}@example.com`);
                await page.locator('input[name="password"]').fill(password);
                const created = page.waitForResponse(response =>
                    new URL(response.url()).pathname === '/api/v1/user'
                    && response.request().method() === 'PUT',
                );
                const savedLanguage = page.waitForResponse(response =>
                    new URL(response.url()).pathname.startsWith('/api/v1/settings/')
                    && response.request().method() === 'POST',
                );
                await page.locator('#submit').click();
                const response = await created;
                expect(response.ok(), 'Registration creates the account').toBeTruthy();
                userId = (await response.json()).id;
                const write = await savedLanguage;
                expect(write.ok(), 'Registration saves the language').toBeTruthy();
                expect(write.request().postDataJSON().language).toBe(language);
                await page.waitForURL('/');
                await expect(page.getByLabel('Open user menu')).toBeVisible();
                expect((await page.context().cookies()).some(cookie => cookie.name === 'pb_auth')).toBe(true);

                // Check the persisted preference through a separate authenticated
                // API context; a translated page alone does not prove it was saved.
                const login = await request.post('/api/v1/auth/login', { data: { username, password } });
                expect(login.ok()).toBeTruthy();
                const user = await request.get(`/api/v1/user/${userId}?expand=settings_via_user`);
                expect(user.ok()).toBeTruthy();
                const saved = await user.json();
                expect(saved.username).toBe(username);
                expect(saved.expand.settings_via_user).toMatchObject({ user: userId, language });
            } finally {
                if (userId) {
                    await page.waitForLoadState('networkidle');
                    const login = await request.post('/api/v1/auth/login', { data: { username, password } });
                    expect(login.ok(), 'Authenticate cleanup of the registration account').toBeTruthy();
                    const deleted = await request.delete(`/api/v1/user/${userId}`);
                    expect(deleted.ok(), 'Delete the registration account').toBeTruthy();
                }
            }
        });
    });
}

test.describe('registration when the language cannot be saved', () => {
    test.use({
        locale: 'de-CH', storageState: { cookies: [], origins: [] },
        browserErrorAllowlist: [{
            source: 'console.error',
            message: /^Failed to load resource: the server responded with a status of 503 \(Service Unavailable\)$/,
            // Filled with the exact URL before returning the injected 503.
            url: /^$/,
            issue: 'https://github.com/open-wanderer/wanderer/pull/948',
        }],
    });

    test('shows the error on the homepage and keeps the new user signed in', async ({ page, request, browserErrorAllowlist }) => {
        const username = `register${randomUUID().replaceAll('-', '').slice(0, 12)}`;
        const password = 'registration-password';
        let userId: string | undefined;
        let failedWrites = 0;
        await page.route('**/api/v1/settings/*', async route => {
            const write = route.request();
            if (write.method() !== 'POST') return route.continue();
            expect(write.postDataJSON().language).toBe('de');
            failedWrites++;
            // Only the Chromium diagnostic for this intentional HTTP failure is
            // expected. Other console errors and every pageerror still fail.
            browserErrorAllowlist[0].url = new RegExp(`^${write.url().replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`);
            await route.fulfill({ status: 503, json: { message: 'settings write failed' } });
        });
        try {
            await page.goto('/register', { waitUntil: 'networkidle' });
            await page.locator('input[name="username"]').fill(username);
            await page.locator('input[name="email"]').fill(`${username}@example.com`);
            await page.locator('input[name="password"]').fill(password);
            const created = page.waitForResponse(response =>
                new URL(response.url()).pathname === '/api/v1/user'
                && response.request().method() === 'PUT',
            );
            await page.locator('#submit').click();
            const response = await created;
            expect(response.ok()).toBeTruthy();
            userId = (await response.json()).id;

            await page.waitForURL('/', { timeout: 5000 });
            expect(failedWrites).toBe(1);
            await expect(page.getByText('Error saving settings', { exact: true })).toBeVisible();
            // Use the browser's existing cookies, without authenticating again.
            const user = await page.request.get(`/api/v1/user/${userId}?expand=settings_via_user`);
            expect(user.ok()).toBeTruthy();
            const saved = await user.json();
            expect(saved.id).toBe(userId);
            expect(saved.username).toBe(username);
            expect(saved.expand.settings_via_user).toMatchObject({ user: userId, language: 'en' });

            await page.reload({ waitUntil: 'networkidle' });
            await expect(page.getByText('Error saving settings', { exact: true })).toHaveCount(0);
        } finally {
            if (userId) {
                await page.waitForLoadState('networkidle');
                const login = await request.post('/api/v1/auth/login', { data: { username, password } });
                expect(login.ok()).toBeTruthy();
                const deleted = await request.delete(`/api/v1/user/${userId}`);
                expect(deleted.ok()).toBeTruthy();
            }
        }
    });
});
