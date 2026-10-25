import { expect, test as setup } from './fixtures';

const authFile = 'playwright/.auth/user.json';

setup('create user', async ({ page, request }) => {
    // Seed the account through the isolated API context. Setup must not trigger
    // an expected failed browser login on every fresh CI database.
    const probe = await request.post('/api/v1/auth/login', {
        data: { username: 'Test', password: 'password' },
    });
    expect([200, 400]).toContain(probe.status());
    if (probe.status() === 400) {
        const created = await request.put('/api/v1/user', {
            data: {
                username: 'Test', email: 'test@test.de',
                password: 'password', passwordConfirm: 'password',
            },
        });
        expect(created.ok(), 'Create the test account').toBeTruthy();
    }
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.locator('input[name="username"]').fill('Test');
    await page.locator('input[name="password"]').fill('password');
    const login = page.waitForResponse('**/api/v1/auth/login');
    await page.getByRole('button', { name: 'Login' }).click();
    // Navigation can discard a successful response body before it is read.
    expect((await login).status()).toBe(200);
    // Wait until the page receives the cookies.
    //
    // Sometimes login flow sets cookies in the process of several redirects.
    // Wait for the final URL to ensure that the cookies are actually set.
    await page.waitForURL('/');

    // End of authentication steps.

    await page.context().storageState({ path: authFile });
});
