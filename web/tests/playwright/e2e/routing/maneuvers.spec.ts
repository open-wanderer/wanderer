import { expect, test } from '@playwright/test';

const MANEUVER_ENDPOINT = '/api/v1/routing/maneuvers';

test('forwards maneuver requests authenticated by pb_auth', async ({ page }) => {
    const response = await page.request.post(MANEUVER_ENDPOINT, {
        data: { trailId: '' },
    });

    expect(response.status()).toBe(400);
    await expectStableHostError(response, 'invalid_request');
});

test('forwards maneuver requests authenticated by wanderer_key', async ({ page, playwright }) => {
    const login = await page.request.post('/api/v1/auth/login', {
        data: { username: 'Test', password: 'password' },
    });
    expect(login.ok()).toBeTruthy();
    const auth = await login.json() as { record: { id: string } };

    const created = await page.request.put('/api/v1/api-token', {
        data: {
            name: `Maneuver integration ${Date.now()}`,
            user: auth.record.id,
        },
    });
    expect(created.ok()).toBeTruthy();
    const token = await created.json() as { id: string; rawToken: string };

    const api = await playwright.request.newContext({
        baseURL: 'http://localhost:3000',
        extraHTTPHeaders: { Authorization: `Bearer ${token.rawToken}` },
    });
    try {
        const response = await api.post(MANEUVER_ENDPOINT, {
            data: { trailId: '' },
        });
        expect(response.status()).toBe(400);
        await expectStableHostError(response, 'invalid_request');
    } finally {
        await api.dispose();
        const removed = await page.request.delete(`/api/v1/api-token/${token.id}`);
        expect(removed.ok()).toBeTruthy();
    }
});

async function expectStableHostError(
    response: { json(): Promise<unknown> },
    code: string,
) {
    const body = await response.json() as {
        data?: { code?: string };
        message?: string;
    };
    expect(body.data?.code, body.message).toBe(code);
}
