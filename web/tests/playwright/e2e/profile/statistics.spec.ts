import {
    expect,
    test,
    type Page,
} from '@playwright/test';

const statsApiPattern = /\/api\/v1\/profile\/@test\/stats\?/;

function statisticActivity(text: string) {
    return {
        id: `activity-${text}`,
        collectionId: 'summit_logs',
        collectionName: 'summit_logs',
        source: 'summit_log',
        date: new Date().toISOString().slice(0, 10),
        text,
        photos: [],
        author: 'testactor000001',
        trail: 'privatetrail001',
        distance: 1000,
        duration: 3600,
        elevation_gain: 100,
        elevation_loss: 100,
        expand: {
            trail: {
                id: 'privatetrail001',
                name: 'Private test trail',
                public: false,
                completed: true,
                photos: [],
                tags: [],
                like_count: 0,
                author: 'testactor000001',
            },
        },
    };
}

async function openStatsWithPrivateActivity(
    page: Page,
    options: {
        onPublicRequest?: () => void;
        onPublicResponse?: () => void;
        publicResponseGate?: Promise<void>;
    } = {},
) {
    let requestCount = 0;
    await page.route(statsApiPattern, async (route) => {
        requestCount += 1;
        const cookies = await page.context().cookies();
        const authenticated = cookies.some((cookie) => cookie.name === 'pb_auth');
        if (!authenticated) {
            options.onPublicRequest?.();
            await options.publicResponseGate;
        }
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(
                authenticated
                    ? [statisticActivity('PRIVATE_STATS_SENTINEL')]
                    : [],
            ),
            headers: { 'Cache-Control': 'private, no-store' },
        });
        if (!authenticated) {
            options.onPublicResponse?.();
        }
    });

    await page.goto('/profile/@test', { waitUntil: 'networkidle' });
    await page.locator('a[href="/profile/@test/stats"]').click();
    await expect(page).toHaveURL('/profile/@test/stats');
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('1');

    return () => requestCount;
}

test('does not cache the server-rendered statistics page', async ({ page }) => {
    const response = await page.goto('/profile/@test/stats', {
        waitUntil: 'networkidle',
    });

    expect(await response!.headerValue('cache-control')).toBe(
        'private, no-store',
    );
});

test('converges a rejected client token without an invalidation loop', async ({
    page,
    context,
}) => {
    await page.goto('/profile/@test/stats', { waitUntil: 'networkidle' });

    let dataRequests = 0;
    let statsRequests = 0;
    page.on('request', (request) => {
        const url = new URL(request.url());
        if (url.pathname.endsWith('/__data.json')) {
            dataRequests += 1;
        }
        if (statsApiPattern.test(url.pathname + url.search)) {
            statsRequests += 1;
        }
    });

    const otherPage = await context.newPage();
    await otherPage.goto('/', { waitUntil: 'networkidle' });
    await otherPage.evaluate(() => {
        const encode = (value: object) =>
            btoa(JSON.stringify(value))
                .replaceAll('+', '-')
                .replaceAll('/', '_')
                .replaceAll('=', '');
        const token = `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode({
            exp: Math.floor(Date.now() / 1000) + 3600,
            type: 'auth',
            collectionId: '_pb_users_auth_',
        })}.invalid-signature`;

        localStorage.setItem(
            'pocketbase_auth',
            JSON.stringify({
                token,
                record: {
                    id: 'rejectedclient1',
                    username: 'Rejected client',
                    collectionId: '_pb_users_auth_',
                    collectionName: 'users',
                },
            }),
        );
    });

    await expect(page.getByLabel('Open user menu')).toHaveCount(0);
    await expect(page.getByRole('link', { name: 'Login' }).first()).toBeVisible();
    await expect.poll(() => dataRequests).toBeGreaterThan(0);

    const settledDataRequests = dataRequests;
    const settledStatsRequests = statsRequests;
    await page.waitForTimeout(750);

    expect(dataRequests).toBe(settledDataRequests);
    expect(statsRequests).toBe(settledStatsRequests);
    expect(dataRequests).toBeLessThanOrEqual(2);
    expect(statsRequests).toBeLessThanOrEqual(3);
});

test('does not restore private statistics after logout and browser back', async ({
    page,
}) => {
    await openStatsWithPrivateActivity(page);
    await page.getByLabel('Open user menu').click();
    await page.locator('.menu .menu-item').filter({ hasText: 'Logout' }).click();

    await page.waitForURL('/');

    const cookies = await page.context().cookies();
    const pbAuthCookie = cookies.find((cookie) => cookie.name === 'pb_auth');
    expect(pbAuthCookie).toBeFalsy();

    await page.goBack({ waitUntil: 'networkidle' });
    await expect(page).toHaveURL('/profile/@test/stats');
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');
    await expect(page.getByLabel('Open user menu')).toHaveCount(0);
    await expect(page.getByRole('link', { name: 'Login' }).first()).toBeVisible();
});

test('resynchronizes auth for a simulated BFCache restore', async ({
    page,
    context,
}) => {
    let publicResponseFinished = false;
    await openStatsWithPrivateActivity(page, {
        onPublicResponse: () => (publicResponseFinished = true),
    });

    // Simulate a logout while this page was frozen in the browser's
    // back-forward cache: the cookie is gone, but its JavaScript auth store is
    // still the old authenticated instance until `pageshow` is handled.
    await context.clearCookies({ name: 'pb_auth' });
    await page.evaluate(() => {
        window.dispatchEvent(
            new PageTransitionEvent('pageshow', { persisted: true }),
        );
    });

    await expect.poll(() => publicResponseFinished).toBe(true);
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');
    await expect(page.getByLabel('Open user menu')).toHaveCount(0);
    await expect(page.getByRole('link', { name: 'Login' }).first()).toBeVisible();
});

test('clears private statistics after logout in another tab', async ({
    page,
    context,
}) => {
    let publicRequestStarted = false;
    let publicResponseFinished = false;
    let releasePublicResponse!: () => void;
    const publicResponseGate = new Promise<void>((resolve) => {
        releasePublicResponse = resolve;
    });
    const statsRequestCount = await openStatsWithPrivateActivity(page, {
        onPublicRequest: () => (publicRequestStarted = true),
        onPublicResponse: () => (publicResponseFinished = true),
        publicResponseGate,
    });
    const logoutPage = await context.newPage();
    await logoutPage.goto('/', { waitUntil: 'networkidle' });
    await logoutPage.getByLabel('Open user menu').click();
    await logoutPage
        .locator('.menu .menu-item')
        .filter({ hasText: 'Logout' })
        .click();

    await expect(logoutPage).toHaveURL('/');
    await expect.poll(() => publicRequestStarted).toBe(true);
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');
    await expect(page.getByLabel('Open user menu')).toHaveCount(0);

    releasePublicResponse();
    await expect.poll(statsRequestCount).toBeGreaterThan(1);
    await expect.poll(() => publicResponseFinished).toBe(true);
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');
});

test('does not restore a stale simulated BFCache refresh after cross-tab logout', async ({
    page,
    context,
}) => {
    await openStatsWithPrivateActivity(page);
    await page.unroute(statsApiPattern);

    let privateRefreshStarted = false;
    let privateRefreshFinished = false;
    let publicRefreshFinished = false;
    let releasePrivateRefresh!: () => void;
    const privateRefreshGate = new Promise<void>((resolve) => {
        releasePrivateRefresh = resolve;
    });

    await page.route(statsApiPattern, async (route) => {
        const cookies = await page.context().cookies();
        const authenticated = cookies.some((cookie) => cookie.name === 'pb_auth');

        if (authenticated) {
            privateRefreshStarted = true;
            await privateRefreshGate;
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify([
                    statisticActivity('STALE_PRIVATE_RESPONSE'),
                ]),
            });
            privateRefreshFinished = true;
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: '[]',
        });
        publicRefreshFinished = true;
    });

    await page.evaluate(() => {
        window.dispatchEvent(
            new PageTransitionEvent('pageshow', { persisted: true }),
        );
    });
    await expect.poll(() => privateRefreshStarted).toBe(true);

    const logoutPage = await context.newPage();
    await logoutPage.goto('/', { waitUntil: 'networkidle' });
    await logoutPage.getByLabel('Open user menu').click();
    await logoutPage
        .locator('.menu .menu-item')
        .filter({ hasText: 'Logout' })
        .click();

    await expect(logoutPage).toHaveURL('/');
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');

    releasePrivateRefresh();
    await expect.poll(() => privateRefreshFinished).toBe(true);
    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');
    await expect.poll(() => publicRefreshFinished).toBe(true);
    await page.evaluate(
        () =>
            new Promise<void>((resolve) =>
                requestAnimationFrame(() =>
                    requestAnimationFrame(() => resolve()),
                ),
            ),
    );

    await expect(page.getByTestId('statistics-activity-count')).toHaveText('0');
    await expect(
        page.getByText('STALE_PRIVATE_RESPONSE', { exact: true }),
    ).toHaveCount(0);
});

test('ignores an older statistics response that finishes last', async ({
    page,
}) => {
    await page.goto('/profile/@test/stats', { waitUntil: 'networkidle' });

    let requestNumber = 0;
    let firstRequestStarted = false;
    let firstRequestFinished = false;
    let secondRequestFinished = false;
    let releaseFirstResponse!: () => void;
    const firstResponseGate = new Promise<void>((resolve) => {
        releaseFirstResponse = resolve;
    });

    await page.route(statsApiPattern, async (route) => {
        requestNumber += 1;
        if (requestNumber === 1) {
            firstRequestStarted = true;
            await firstResponseGate;
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify([statisticActivity('OLD_RESPONSE')]),
            });
            firstRequestFinished = true;
            return;
        }

        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify([statisticActivity('LATEST_RESPONSE')]),
        });
        secondRequestFinished = true;
    });

    const period = page.locator('select[name="statistics-period"]');
    await period.selectOption('current_year');
    await expect.poll(() => firstRequestStarted).toBe(true);
    await period.selectOption('current_month');
    await expect.poll(() => secondRequestFinished).toBe(true);
    await expect(page.getByText('LATEST_RESPONSE', { exact: true })).toBeVisible();

    releaseFirstResponse();
    await expect.poll(() => firstRequestFinished).toBe(true);
    await page.evaluate(
        () =>
            new Promise<void>((resolve) =>
                requestAnimationFrame(() =>
                    requestAnimationFrame(() => resolve()),
                ),
            ),
    );

    await expect(page.getByText('OLD_RESPONSE', { exact: true })).toHaveCount(0);
    await expect(page.getByText('LATEST_RESPONSE', { exact: true })).toBeVisible();
});

test('keeps the selected period during an auth-neutral invalidation', async ({
    page,
}) => {
    await page.goto('/profile/@test/stats', { waitUntil: 'networkidle' });

    let requestCount = 0;
    await page.route(statsApiPattern, async (route) => {
        requestCount += 1;
        await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: '[]',
        });
    });

    const period = page.locator('select[name="statistics-period"]');
    await period.selectOption('current_year');
    await expect.poll(() => requestCount).toBe(1);

    await page.evaluate(() => {
        window.dispatchEvent(
            new PageTransitionEvent('pageshow', { persisted: true }),
        );
    });

    await expect.poll(() => requestCount).toBeGreaterThan(1);
    await expect(period).toHaveValue('current_year');
});
