# Testing Patterns

**Analysis Date:** 2026-06-10

## Test Framework

**Web (TypeScript/Svelte):**

**E2E Testing:**
- Runner: Playwright 1.58.2
- Config: `web/playwright.config.ts`
- Scope: Full application flow testing (login, navigation, API calls)
- Run Command: `npm run test:integration`

**Unit Testing:**
- Runner: Vitest 4.1.4 (via Vite)
- Config: Declared in `web/vite.config.ts` with pattern `src/**/*.{test,spec}.{js,ts}`
- Assertion Library: Not configured (uses Vitest defaults or Playwright assertions for E2E)
- Run Command: `npm run test:unit`

**Combined:**
- Run all tests: `npm test` (runs both integration and unit)

**Mobile (Dart/Flutter):**

**Unit Testing:**
- Framework: `flutter_test` (built-in to Flutter SDK)
- Test file location: `app/test/models/` (co-located in test directory)
- Assertion Library: Dart's built-in `expect()` from `flutter_test`
- Example: `app/test/models/feed_item_test.dart`

## Test File Organization

**Web E2E:**
- Location: `web/tests/playwright/e2e/`
- Structure:
  ```
  tests/playwright/
  ├── auth.setup.ts          # Setup phase: user registration/login
  ├── auth.teardown.ts       # Teardown phase: cleanup
  ├── pages/                 # Page Object Pattern
  │   └── index_page.ts
  ├── e2e/                   # Test suites
  │   ├── index/
  │   │   └── index.spec.ts
  │   ├── list/
  │   │   └── list.spec.ts
  │   ├── user/
  │   │   └── user.spec.ts
  ├── fixtures/              # Test data (e.g., avatar.webp image)
  ```

**Web Unit:**
- Location: `src/**/*.{test,spec}.ts` (co-located with source files)
- Not currently populated in codebase

**Dart:**
- Location: `app/test/models/` (separate test directory)
- Naming: `{module}_test.dart` (e.g., `feed_item_test.dart`)

## Test Structure

**Playwright E2E Test Suite (from `index.spec.ts`):**
```typescript
import { expect, test } from '@playwright/test';
import { IndexPage } from '../../pages/index_page';

test('index page does not show error', async ({ page }) => {
  const indexPage = new IndexPage(page);
  await indexPage.goto()
  await indexPage.hasNoError()
});

test('location search works', async ({ page }) => {
  const indexPage = new IndexPage(page);
  await indexPage.goto()
  await indexPage.search()
});
```

**Patterns:**
- Each test is a single `test()` call with description and async handler
- Page Object Pattern for reusable test logic (see `IndexPage` class)
- Setup/teardown handled by Playwright's phase system (`auth.setup.ts` runs before tests)
- No test grouping (no `describe()` blocks observed)

**Flutter/Dart Test Suite (from `feed_item_test.dart`):**
```dart
void main() {
  group('FeedItem.fromJson', () {
    test('type "trail" returns FeedItemTrail with TrailSearchResult', () {
      final json = {
        'id': 'feed-1',
        'actor': 'actor-1',
        'type': 'trail',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': {
          'item': _minimalTrailJson(),
        },
      };

      final result = FeedItem.fromJson(json);

      expect(result, isA<FeedItemTrail>());
      final trailItem = result as FeedItemTrail;
      expect(trailItem.id, 'feed-1');
      expect(trailItem.actor, 'actor-1');
    });

    test('absent expand.item throws FormatException', () {
      final json = {
        'id': 'feed-3',
        'actor': 'actor-1',
        'type': 'trail',
        'created': '2024-01-01 00:00:00.000Z',
        'expand': <String, dynamic>{},
      };

      expect(() => FeedItem.fromJson(json), throwsA(isA<FormatException>()));
    });
  });
}
```

**Patterns:**
- Tests organized in `group()` for related functionality
- Factory constructors tested for correct parsing and error handling
- Minimal test data created via helper functions (e.g., `_minimalTrailJson()`)
- Matchers for type checking and exception verification (e.g., `isA<T>()`, `throwsA()`)

## Mocking

**Playwright E2E:**
- No explicit mocking library used
- Real API calls made to running backend during tests
- Authentication handled via `playwright/.auth/user.json` (persistent storage state)
- Response waiting: `page.waitForResponse()` for API responses
- URL waiting: `page.waitForURL()` for navigation confirmation

**Example (from `index_page.ts`):**
```typescript
async search() {
  const responsePromise = this.page.waitForResponse(resp =>
    resp.url().includes('/api/v1/search/multi') && resp.status() === 200
  );
  
  await this.page.locator('input[name="q"]').fill('Munich');
  await responsePromise;
  
  await this.page.locator('.menu-item').first().click();
  await this.page.waitForURL(/\/map\?lat=.*&lon=.*/);
}
```

**Dart Unit Tests:**
- No mocking library explicitly imported in `feed_item_test.dart`
- Focuses on factory deserialization testing, not service mocking
- Uses built-in `expect()` for assertions and exception matching

## Fixtures and Factories

**Playwright Test Data:**
- Location: `web/tests/playwright/fixtures/`
- Current data: `avatar.webp` image file for test assertions
- Strategy: Minimal fixtures; mostly API-driven test setup via register/login flow

**Playwright Fixture Code Pattern:**
- Auth setup in `auth.setup.ts` creates a test user via register API
- Saves auth state to `playwright/.auth/user.json`
- Teardown in `auth.teardown.ts` cleans up test session

**Example Setup (from `auth.setup.ts`):**
```typescript
import { test as setup } from '@playwright/test';

const authFile = 'playwright/.auth/user.json';

setup('create user', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.locator('input[name="username"]').fill('Test');
    await page.locator('input[name="password"]').fill('password');
    
    const loginPromise = page.waitForResponse('**/api/v1/auth/login')
    await page.getByRole('button', { name: 'Login' }).click();
    const response = await loginPromise;

    // ... handle response/register if needed ...

    await page.waitForURL('/');
    await page.context().storageState({ path: authFile });
});
```

**Dart Test Fixtures:**
- Helper functions for minimal JSON: `_minimalTrailJson()`, `_minimalListJson()`
- Lightweight factories creating just required fields
- Data-driven by test case; no separate fixture files

**Example (from `feed_item_test.dart`):**
```dart
Map<String, dynamic> _minimalTrailJson() => {
  'id': 'trail-1',
  'author': 'user-1',
  'author_name': 'Alice',
  'author_avatar': 'avatar.jpg',
  'name': 'Test Trail',
  // ... other required fields ...
};

// Used in test:
final json = {
  'id': 'feed-1',
  'actor': 'actor-1',
  'type': 'trail',
  'created': '2024-01-01 00:00:00.000Z',
  'expand': {
    'item': _minimalTrailJson(),
  },
};
```

## Coverage

**Web (TypeScript/Svelte):**
- Requirements: No coverage target enforced
- Unit tests minimal; primarily E2E coverage
- Coverage tool available (Playwright HTML reports) but not actively generated

**Dart (Flutter):**
- Requirements: No coverage target enforced
- Single test file covers model deserialization
- Coverage command: `flutter test --coverage` (standard Flutter command; not observed in scripts)

## Test Types

**Web E2E Tests:**
- **Scope:** Full user flows from UI interaction to API response
- **Approach:**
  - Page Object Pattern for reusable component selectors
  - Setup/teardown phases for auth state management
  - Response waiting for async operations
  - URL/navigation assertions
- **Current Coverage:**
  - Index page: error checking, location search
  - User page: profile viewing
  - List page: list operations
- **Example (index.spec.ts):**
  ```typescript
  test('index page does not show error', async ({ page }) => {
    const indexPage = new IndexPage(page);
    await indexPage.goto()
    await indexPage.hasNoError()
  });
  ```

**Web Unit Tests:**
- **Scope:** Currently not populated in codebase
- **Approach:** When added, will use Vitest with `src/**/*.{test,spec}.ts` pattern
- **Potential targets:** Utility functions, store logic, form validation

**Dart Unit Tests:**
- **Scope:** Model factory constructors and JSON deserialization
- **Approach:**
  - Test factory methods (e.g., `FeedItem.fromJson()`)
  - Verify type matching and field assignment
  - Test error handling for malformed JSON
- **Current Coverage (feed_item_test.dart):**
  - Trail feed items: correct deserialization to `FeedItemTrail` type
  - List feed items: correct deserialization to `FeedItemList` type
  - Missing expand.item: throws `FormatException` (defensive contract)
  - Unknown types: throws `UnsupportedError`

## Configuration Details

**Playwright Config (web/playwright.config.ts):**
- Test directory: `./tests/playwright`
- Parallel execution: Disabled by default (`fullyParallel: false`) to prevent auth conflicts
- CI behavior: Retries 2 times, workers = 1 (sequential), forbid `test.only`
- Development: No retries, default worker count (parallel)
- Reporter: HTML report
- Base URL: `http://localhost:3000`
- Trace collection: On first retry
- Projects: Setup phase, chromium browser, teardown phase
- Storage state: `playwright/.auth/user.json` for persistent auth

**Vite Config (web/vite.config.ts):**
- Unit test pattern: `src/**/*.{test,spec}.{js,ts}`
- Pass with no tests: `--passWithNoTests` flag
- No coverage configuration present

**Dart Test Setup:**
- Framework: Built-in `flutter_test`
- Run command: `flutter test` (standard; not in pubspec scripts)
- Generate coverage: `flutter test --coverage` (manual)

## Common Patterns

**Playwright - Async/Network Waiting:**
```typescript
// Wait for API response before continuing
const responsePromise = this.page.waitForResponse(resp =>
  resp.url().includes('/api/v1/endpoint') && resp.status() === 200
);

await this.page.locator('button').click();
await responsePromise;

// Or wait for navigation
await this.page.waitForURL(/\/route\?param=value/);
```

**Playwright - Page Object Methods:**
```typescript
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

  async hasNoError() {
    await expect(this.error).toHaveCount(0);
  }
}
```

**Dart - Model Testing with Factory Methods:**
```dart
test('type "trail" returns FeedItemTrail with TrailSearchResult', () {
  final json = {
    'id': 'feed-1',
    'actor': 'actor-1',
    'type': 'trail',
    'created': '2024-01-01 00:00:00.000Z',
    'expand': {
      'item': _minimalTrailJson(),
    },
  };

  final result = FeedItem.fromJson(json);

  expect(result, isA<FeedItemTrail>());
  final trailItem = result as FeedItemTrail;
  expect(trailItem.id, 'feed-1');
});
```

**Dart - Exception Testing:**
```dart
test('absent expand.item throws FormatException', () {
  final json = {
    'id': 'feed-3',
    'actor': 'actor-1',
    'type': 'trail',
    'created': '2024-01-01 00:00:00.000Z',
    'expand': <String, dynamic>{},
  };

  expect(() => FeedItem.fromJson(json), throwsA(isA<FormatException>()));
});
```

## What to Test

**E2E Priority:**
- Critical user flows: authentication, core CRUD operations, navigation
- Integration points: API responses, real-time updates, error handling
- UI assertions: error messages, data display, loading states

**Unit Priority:**
- Utility functions with complex logic
- Data transformations (serialization/deserialization)
- Error boundaries and validation schemas
- Store operations with side effects

**What NOT to Test:**
- Third-party library behavior
- Trivial getters/setters
- UI rendering without business logic
- Code that's already tested at E2E layer

---

*Testing analysis: 2026-06-10*
