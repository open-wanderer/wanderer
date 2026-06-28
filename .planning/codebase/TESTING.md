# Testing Patterns

**Analysis Date:** 2026-06-07

## Test Framework

**Runner:**
- Playwright: `^1.58.2` - E2E testing framework
- Vitest: `^4.1.4` - Unit/integration testing (configured but minimal tests observed)
- Config files:
  - `web/playwright.config.ts` - Playwright configuration
  - `web/vite.config.ts` - Vitest configuration embedded (`test: { include: ['src/**/*.{test,spec}.{js,ts}'] }`)

**Assertion Library:**
- Playwright's built-in assertions via `expect()` from `@playwright/test`
- Vitest assertions (standard via `describe`/`it`)

**Run Commands:**
```bash
npm run test                 # Run all tests (integration + unit)
npm run test:integration    # Run Playwright tests only
npm run test:unit           # Run Vitest tests only (vitest --passWithNoTests)
npm run dev                 # Development server
npm run build               # Build for production
npm run check               # Run svelte-check type checking
npm run check:watch         # Watch mode for svelte-check
```

## Test File Organization

**Location:**
- E2E tests: `web/tests/playwright/e2e/` - Separate from source code
- Test fixtures: `web/tests/playwright/fixtures/` - Shared test data (e.g., avatar images)
- Test utilities: `web/tests/playwright/pages/` - Page Object Model (POM) classes
- Setup/teardown: `web/tests/playwright/` root (e.g., `auth.setup.ts`, `auth.teardown.ts`)

**Naming:**
- Test files: `{feature}.spec.ts` (Playwright convention)
- Page Object files: `{page_name}_page.ts`
- Setup/teardown: `{auth|action}.setup.ts`, `{auth|action}.teardown.ts`

**Structure:**
```
tests/playwright/
├── e2e/
│   ├── index/
│   │   └── index.spec.ts
│   ├── list/
│   │   └── list.spec.ts
│   └── user/
│       └── user.spec.ts
├── pages/
│   ├── index_page.ts
│   └── lists_page.ts
├── fixtures/
│   └── avatar.webp
├── auth.setup.ts
└── auth.teardown.ts
```

## Test Structure

**Suite Organization:**

Playwright tests do NOT use explicit `describe` blocks. Each test file contains individual test cases using `test()`:

```typescript
// From web/tests/playwright/e2e/user/user.spec.ts
import { test, expect } from '@playwright/test';

test('logs the user out', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.getByLabel('Open user menu').click();
    await page.locator(".menu .menu-item").filter({ hasText: "Logout" }).click();

    await page.waitForURL('/');
    
    const cookies = await page.context().cookies();
    const pbAuthCookie = cookies.find(cookie => cookie.name === 'pb_auth');
    expect(pbAuthCookie).toBeFalsy();
});
```

**Patterns:**

1. **Setup/Teardown**: Configured at project level in `playwright.config.ts`
   - Setup project: runs `auth.setup.ts` before tests
   - Teardown project: runs `auth.teardown.ts` after tests
   - Dependency chain: teardown depends on chromium project

2. **Authentication State**: Persistent via `storageState`
   - Storage state file: `playwright/.auth/user.json`
   - Chromium project uses: `storageState: 'playwright/.auth/user.json'`

3. **Page Navigation**: Always wait for network idle or specific URL
   ```typescript
   await page.goto('/', { waitUntil: 'networkidle' });
   await page.waitForURL('/map?lat=.*&lon=.*');
   ```

4. **Response Waiting**: Explicit response promise for async operations
   ```typescript
   const responsePromise = this.page.waitForResponse(resp =>
       resp.url().includes('/api/v1/search/multi') && resp.status() === 200
   );
   await this.page.locator('input[name="q"]').fill('Munich');
   await responsePromise;
   ```

## Page Object Model (POM)

**Implementation:**

Classes encapsulate page selectors and navigation logic. Example from `web/tests/playwright/pages/index_page.ts`:

```typescript
import { expect, type Locator, type Page } from '@playwright/test';

export class IndexPage {
  readonly page: Page;
  readonly error: Locator;

  constructor(page: Page) {
    this.page = page;
    this.error = page.getByText("Internal Error");
  }

  async goto() {
    await this.page.goto('/', { waitUntil: 'networkidle' });
  }

  async search() {
    // Start waiting for response before triggering the search
    const responsePromise = this.page.waitForResponse(resp =>
      resp.url().includes('/api/v1/search/multi') && resp.status() === 200
    );
    
    await this.page.locator('input[name="q"]').fill('Munich');
    await responsePromise;
    
    await this.page.locator('.menu-item').first().click();
    await this.page.waitForURL(/\/map\?lat=.*&lon=.*/);
  }

  async hasNoError() {
    await expect(this.error).toHaveCount(0);
  }
}
```

**Usage Pattern:**
- Constructor receives `page` fixture from Playwright
- Locators stored as readonly properties
- Navigation and action methods encapsulate user interactions
- Assertions included in helper methods

## Mocking

**Framework:**
- No mocking library detected (Jest, Sinon, Vitest mock functions not used)
- E2E tests run against real backend via `baseURL: "http://localhost:3000"`

**Patterns:**

1. **No client-side mocking**: E2E tests do NOT mock network calls
2. **Backend dependency**: Tests require running instance of app at `localhost:3000`
3. **Auth state handling**: Uses persistent storage state to avoid re-login between tests

**What to Mock:**
- External APIs (not mocked in current tests)
- File uploads (not mocked; real file upload tested via FormData)

**What NOT to Mock:**
- Backend API calls (tests hit real backend)
- Navigation (use real navigation helpers)
- Authentication (use real auth setup via setup project)

## Test Configuration

**Playwright Config (`web/playwright.config.ts`):**
```typescript
{
  testDir: './tests/playwright',
  fullyParallel: false,                    // Disabled to prevent auth conflicts
  forbidOnly: !!process.env.CI,           // Fail on test.only in CI
  retries: process.env.CI ? 2 : 0,        // Retry failed tests on CI only
  workers: process.env.CI ? 1 : undefined, // Single worker on CI
  reporter: 'html',                        // HTML test report
  use: {
    baseURL: "http://localhost:3000",
    trace: 'on-first-retry'                // Collect trace on first retry
  }
}
```

**Vitest Config (`web/vite.config.ts`):**
```typescript
test: { include: ['src/**/*.{test,spec}.{js,ts}'] }
```

## Test Types

**E2E Tests:**
- Framework: Playwright
- Scope: Full user workflows across UI and backend
- Coverage: User logout, search, navigation
- Location: `web/tests/playwright/e2e/`
- Browser: Chromium (Firefox and WebKit commented out)

**Unit Tests:**
- Framework: Vitest
- Status: Configured but NO tests found in source code
- Would scan: `src/**/*.{test,spec}.{js,ts}`
- Run: `npm run test:unit` (passes with `--passWithNoTests`)

**Integration Tests:**
- Part of overall test suite via Playwright
- Tests API endpoints through UI interactions
- Real backend required

## Async Testing

**Pattern:** All Playwright test functions are async with `await`:

```typescript
test('logs the user out', async ({ page }) => {
    await page.goto('/', { waitUntil: 'networkidle' });
    await page.getByLabel('Open user menu').click();
    await page.locator(".menu .menu-item").filter({ hasText: "Logout" }).click();
    await page.waitForURL('/');
    
    const cookies = await page.context().cookies();
    const pbAuthCookie = cookies.find(cookie => cookie.name === 'pb_auth');
    expect(pbAuthCookie).toBeFalsy();
});
```

**Waiting Strategies:**
- `waitUntil: 'networkidle'` - Wait for network requests to complete
- `waitForURL()` - Wait for navigation to specific URL (supports regex)
- `waitForResponse()` - Wait for specific API response
- Implicit waits: Playwright auto-waits for elements to be visible

## Error Assertion Pattern

```typescript
async hasNoError() {
    await expect(this.error).toHaveCount(0);
}
```

Tests assert error states via element count assertions.

## Coverage

**Requirements:** None explicitly enforced

**View Coverage:**
- Not configured; no coverage tools detected in dependencies
- `npm run test:unit --coverage` would require coverage plugin

## Test Infrastructure

**Authentication Flow:**
1. `auth.setup.ts` runs before all tests - Logs in and saves storage state
2. Tests use saved storage state automatically via `storageState: 'playwright/.auth/user.json'`
3. `auth.teardown.ts` runs after all tests - Cleans up

**BaseURL:** Tests configured to run against local dev server at `http://localhost:3000`

**Parallel Execution:** Disabled (`fullyParallel: false`) to prevent authentication conflicts

**CI Behavior:**
- Retries: 2 attempts per failed test
- Workers: 1 (serial execution)
- Forbid test.only: Enforced
- Fail fast behavior configured

---

*Testing analysis: 2026-06-07*
