# Testing Patterns

**Analysis Date:** 2026-09-06

## Test Framework Overview

This monorepo uses different testing frameworks per platform:

| Platform | Unit Testing | E2E Testing | Config File | Test Command |
|----------|--------------|------------|-------------|--------------|
| **Web (SvelteKit/TypeScript)** | Vitest | Playwright | `web/vite.config.ts`, `web/playwright.config.ts` | `npm run test:unit`, `npm run test:integration` |
| **Go Backend** | Go testing pkg | (Manual/E2E) | (none) | `go test ./...` |
| **Flutter/Dart App** | flutter_test | (Integration tests) | `app/pubspec.yaml` | `flutter test` |
| **Docs (Astro)** | (none configured) | (none configured) | (none) | N/A |

## Web (TypeScript/SvelteKit)

### Test Framework Configuration

**Vitest (Unit Tests):**
- Framework: Vitest 4.1.9
- Config location: `web/vite.config.ts`
- Test pattern: `src/**/*.{test,spec}.{js,ts}`
- Run: `npm run test:unit` (with `--passWithNoTests` flag)

**Playwright (E2E Tests):**
- Framework: Playwright 1.61.1
- Config location: `web/playwright.config.ts`
- Test directory: `web/tests/playwright/`
- Base URL: `http://localhost:3000`
- Run: `npm run test:integration`

**Configuration Details from `web/playwright.config.ts`:**
```typescript
export default defineConfig({
  testDir: './tests/playwright',
  fullyParallel: false,  // Disabled to prevent auth conflicts
  forbidOnly: !!process.env.CI,  // Fail if test.only left in code
  retries: process.env.CI ? 2 : 0,  // Retry on CI only
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: "http://localhost:3000",
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'setup', testMatch: /.*\.setup\.ts/ },
    {
      name: 'teardown',
      testMatch: /.*\.teardown\.ts/,
      dependencies: ['chromium'],
      use: { storageState: 'playwright/.auth/user.json' }
    },
    {
      name: 'chromium',
      dependencies: ['setup'],
      use: { ...devices['Desktop Chrome'], storageState: 'playwright/.auth/user.json' }
    },
  ],
});
```

### Unit Test Structure (Vitest)

**File Location:**
- Co-located with source files
- Naming: `*.test.ts` or `*.spec.ts`
- Example: `web/src/lib/stores/plugin_store.test.ts` (tests `plugin_store.ts`)

**Test Organization:**
- Use `describe()` for test suites
- Use `it()` for individual tests
- Use `expect()` for assertions

**Example from `web/src/lib/stores/plugin_store.test.ts`:**
```typescript
import { describe, expect, it } from "vitest";
import { externalHttpUrl, plugins_index } from "./plugin_store";

describe("externalHttpUrl", () => {
    it("accepts absolute HTTP(S) URLs", () => {
        expect(externalHttpUrl("https://example.com/donate")).toBe(
            "https://example.com/donate",
        );
    });

    it("rejects URLs without a scheme", () => {
        expect(externalHttpUrl("example.com/donate")).toBeUndefined();
    });

    it("rejects unsafe schemes and non-string values", () => {
        expect(externalHttpUrl("javascript:alert(1)")).toBeUndefined();
        expect(externalHttpUrl({ url: "https://example.com" })).toBeUndefined();
    });
});

describe("plugins_index", () => {
    it("maps and sanitizes optional plugin metadata", async () => {
        const plugin: Record<string, any> = {
            id: "example",
            type: "trails",
            name: "Example",
            version: "1.0.0",
            // ... more fields
        };

        const items = await plugins_index(async () =>
            new Response(JSON.stringify({ items: [plugin] }), {
                status: 200,
                headers: { "content-type": "application/json" },
            }),
        );

        expect(items[0].homepageUrl).toBe("https://example.com/plugin");
        expect(items[0].information).toBeUndefined();
    });
});
```

### E2E Test Structure (Playwright)

**File Location:**
- `web/tests/playwright/` directory
- Setup/teardown: `auth.setup.ts`, `auth.teardown.ts`
- Page objects: `pages/*.ts`
- Test specs: `e2e/**/*.spec.ts`

**Test Organization:**

**1. Setup Phase (`auth.setup.ts`):**
- Creates authenticated session for test runs
- Fills login form and navigates to register if needed
- Stores auth state in `playwright/.auth/user.json` for reuse

**Example from `web/tests/playwright/auth.setup.ts`:**
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

    let responseJson;
    try {
        responseJson = await response.json();
    } catch (e) {
        if (e instanceof Error && e.message.includes("No resource with given identifier found")) {
            console.error("Already logged in!")
        } else {
            throw e
        }
    }

    if (responseJson?.message === "Failed to authenticate.") {
        // Register flow
        await page.goto('/register', { waitUntil: 'networkidle' });
        await page.locator('input[name="username"]').fill('Test');
        await page.locator('input[name="email"]').fill('test@test.de');
        await page.locator('input[name="password"]').fill('password');
        await page.locator('#submit').click();
    }

    await page.waitForURL('/');
    await page.context().storageState({ path: authFile });
});
```

**2. Page Objects:**
- Encapsulate UI element selectors and common actions
- Class-based pattern with Locator properties
- Methods for user interactions

**Example from `web/tests/playwright/pages/index_page.ts`:**
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

**3. Test Specs:**
- Import and use Page Objects
- Inherit authenticated session from setup
- Use Playwright's `test` and `expect` APIs

**Example from `web/tests/playwright/e2e/user/user.spec.ts`:**
```typescript
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

## Go Backend

### Test Framework

**Framework:** Go `testing` package (built-in)
- No external test framework
- Standard `*_test.go` file naming convention
- Run: `go test ./...` (or package-specific)

### Test File Location

- Co-located with source files in same package
- Naming: `*_test.go` suffix
- Examples: `db/pluginsystem/manager_test.go`, `db/routes/plugin_system_test.go`, `db/util/category_test.go`

### Test Structure

**Table-Driven Tests (Common Pattern):**
- Define test cases as struct slices
- Loop through cases with `t.Run()` for sub-tests
- Enables testing multiple scenarios in one function

**Example from `db/util/category_test.go`:**
```go
package util

import "testing"

func TestParseCategoryTranslations(t *testing.T) {
    tests := []struct {
        name    string
        input   types.JSONRaw
        wantErr bool
        want    map[string]CategoryTranslation
    }{
        {
            name:  "valid base locale",
            input: types.JSONRaw(`{"de": {"name": "Wandern", "short_name": "WAND"}}`),
            want:  map[string]CategoryTranslation{"de": {Name: "Wandern", ShortName: "WAND"}},
        },
        {
            name:  "null is nil without error",
            input: types.JSONRaw(`null`),
            want:  nil,
        },
        {
            name:    "region locale rejected",
            input:   types.JSONRaw(`{"pt-BR": {"name": "Caminhada"}}`),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseCategoryTranslations(tt.input)
            if tt.wantErr {
                if err == nil {
                    t.Fatal("ParseCategoryTranslations() error = nil, want error")
                }
                return
            }
            if err != nil {
                t.Fatalf("ParseCategoryTranslations() error = %v", err)
            }
            // Assert result matches expected
        })
    }
}
```

**Sub-tests with `t.Run()`:**
- Groups related test cases within a function
- Better output organization and selective running

**Example from `db/util/category_test.go`:**
```go
func TestNormalizeCategoryName(t *testing.T) {
    t.Run("WhitespaceAndSeparators", func(t *testing.T) {
        for _, value := range []string{"E-Bike", "E Bike", "e-bike"} {
            if got := NormalizeCategoryName(value); got != "e bike" {
                t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", value, got, "e bike")
            }
        }
    })

    t.Run("AccentFolding", func(t *testing.T) {
        for _, value := range []string{"Canoë", "Canoe"} {
            if got := NormalizeCategoryName(value); got != "canoe" {
                t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", value, got, "canoe")
            }
        }
    })

    t.Run("SeparatorsNotRemoved", func(t *testing.T) {
        if got := NormalizeCategoryName("EBike"); got != "ebike" {
            t.Fatalf("NormalizeCategoryName(%q) = %q, want %q", "EBike", got, "ebike")
        }
    })
}
```

### Assertion Pattern

- Direct if/condition checks (no assertion library)
- Use `t.Fatal()` to stop test immediately on failure
- Use `t.Fatalf()` for formatted failure messages
- Use `t.Error()` to log failure but continue
- Use `t.Run()` for sub-test grouping

**Example from `db/pluginsystem/manager_test.go`:**
```go
func TestPluginIssueRecordID(t *testing.T) {
    valid := pluginIssueRecordID(LocalPluginIssue{ID: "komoot", Dir: "/plugins/komoot"})
    if valid != "komoot" {
        t.Fatalf("pluginIssueRecordID(valid) = %q, want komoot", valid)
    }

    first := pluginIssueRecordID(LocalPluginIssue{ID: "@@@", Dir: "/plugins/@@@"})
    second := pluginIssueRecordID(LocalPluginIssue{ID: "***", Dir: "/plugins/***"})
    if first == second {
        t.Fatalf("invalid plugin issue ids collided: %q", first)
    }
}
```

## Flutter/Dart App

### Test Framework

**Framework:** `flutter_test` (Flutter's built-in testing package)
- Part of Flutter SDK (comes with Flutter)
- Config: `app/pubspec.yaml` dev_dependencies
- Linting: `flutter_lints` and `riverpod_lint` (in `analysis_options.yaml`)
- Run: `flutter test` or `flutter test test/`

### Test File Location

- Directory: `app/test/` (mirroring `app/lib/` structure)
- Naming: `*_test.dart` suffix
- Examples:
  - `app/test/store/local_trail_store_test.dart`
  - `app/test/util/format_test.dart`
  - `app/test/provider/online_status_provider_test.dart`

### Test Structure

**Organization with `group()` and `test()`:**
- Use `group()` for test suites (organize related tests)
- Use `test()` for individual test cases
- Use `expect()` for assertions

**Example from `app/test/util/format_test.dart`:**
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/format.dart';

void main() {
  group('formatDistance', () => {
    test('metric (default) formats meters >= 1000 as km', () => {
      expect(formatDistance(1000), '1.00 km');
    });

    test('imperial converts meters to miles (×0.000621371)', () => {
      expect(formatDistance(1000, unit: 'imperial'), '0.62 mi');
    });
  });

  group('formatElevation', () => {
    test('metric (default) formats meters with m suffix', () => {
      expect(formatElevation(100), '100 m');
    });

    test('imperial converts meters to feet (×3.28084)', () => {
      expect(formatElevation(100, unit: 'imperial'), '328 ft');
    });
  });

  group('formatSpeed', () => {
    test('null returns "-"', () => {
      expect(formatSpeed(null), '-');
    });

    test('metric (default) formats to one decimal with km/h suffix', () => {
      expect(formatSpeed(12.34), '12.3 km/h');
    });
  });
}
```

### Async Testing

**Using `async` / `await`:**
- Dart's `test()` function accepts async callbacks
- Return `Future` from test or use `await` for async operations

**Example pattern:**
```dart
test('async operation', () async {
  // Setup
  final result = await someAsyncFunction();
  
  // Assert
  expect(result, expectedValue);
});
```

### Riverpod Provider Testing

**Testing Riverpod Providers:**
- Use `flutter_test`'s `WidgetTester` for integration
- Or use container directly for unit tests (if testing pure Dart providers)
- Example: `app/test/provider/online_status_provider_test.dart`

## Mocking & Fixtures

### TypeScript/Vitest

**Mocking Pattern:**
- Use Vitest's built-in mocking (via Vitest)
- Mock HTTP responses using async functions as fetch replacements

**Example from `web/src/lib/stores/plugin_store.test.ts`:**
```typescript
it("maps and sanitizes optional plugin metadata", async () => {
    const plugin: Record<string, any> = { /* plugin data */ };

    // Mock fetch by providing a test function
    const items = await plugins_index(async () =>
        new Response(JSON.stringify({ items: [plugin] }), {
            status: 200,
            headers: { "content-type": "application/json" },
        }),
    );

    expect(items[0].homepageUrl).toBe("https://example.com/plugin");
});
```

### Playwright E2E

**Mocking/Waiting Patterns:**
- Wait for network responses before assertions
- Intercept responses with `waitForResponse()`

**Example from `web/tests/playwright/pages/index_page.ts`:**
```typescript
async search() {
    // Start waiting for response BEFORE triggering action
    const responsePromise = this.page.waitForResponse(resp =>
      resp.url().includes('/api/v1/search/multi') && resp.status() === 200
    );
    
    // Trigger action that causes the response
    await this.page.locator('input[name="q"]').fill('Munich');
    
    // Wait for response to complete
    await responsePromise;
    
    // Proceed with UI interactions
    await this.page.locator('.menu-item').first().click();
}
```

### Go Tests

**Fixtures/Test Data:**
- Inline in test functions using struct literals
- Example: `LocalPluginIssue{ID: "komoot", Dir: "/plugins/komoot"}`

**Test Helpers:**
- Utility functions for common test setup (if reused)
- Typically package-level functions (lowercase, unexported)

### Flutter Tests

**Fixtures (Test Data):**
- Create test objects directly in test code
- Use Riverpod's `ProviderContainer` for provider testing if needed

**Example pattern:**
```dart
test('format distance with mock trail', () {
  final trail = Trail(
    id: 'test-123',
    distance: 1000,
    // ... other fields
  );
  
  expect(formatDistance(trail.distance), '1.00 km');
});
```

## Coverage

### Web (TypeScript)

**Coverage Requirements:** Not enforced (none detected)
- Vitest can generate coverage with `--coverage` flag
- No coverage config file in repo
- No CI coverage checks configured

### Go

**Coverage:** Not configured
- Can be run with `go test -cover ./...`
- No coverage target enforced

### Flutter

**Coverage:** Not configured
- `flutter test` can generate coverage
- No coverage target enforced

## Test Types

### Web

**Unit Tests (Vitest):**
- Scope: Individual functions/stores (`plugin_store.ts`, etc.)
- Approach: Test pure functions and store logic in isolation
- Location: `web/src/lib/stores/*.test.ts`, `web/src/lib/util/*.test.ts`

**E2E Tests (Playwright):**
- Scope: Full user workflows (login, search, navigation)
- Approach: Browser automation with page objects
- Setup: Shared authentication via `auth.setup.ts`
- Location: `web/tests/playwright/e2e/**/*.spec.ts`

### Go

**Unit Tests:**
- Scope: Package-level functions (utility functions, services)
- Approach: Table-driven tests with assertions
- Location: `db/**/*_test.go` (same directory as source)

### Flutter

**Unit Tests:**
- Scope: Utility functions, formatters, calculations
- Approach: Test pure functions with various inputs
- Location: `app/test/util/*_test.dart`, `app/test/models/*_test.dart`

**Provider Tests (Riverpod):**
- Scope: Provider logic without UI
- Approach: Unit tests for async providers
- Location: `app/test/provider/*_test.dart`

**Integration Tests:**
- Not formally configured (though `flutter test` can run widget tests)
- Widget tests would use `WidgetTester` and material testing utilities

## Test Runs

### Web

**All Tests:**
```bash
npm run test              # Runs unit + integration
npm run test:unit         # Vitest only
npm run test:integration  # Playwright only
```

**Development:**
```bash
# Watch mode for unit tests
npx vitest --watch
```

### Go

```bash
go test ./...             # All packages
go test -v ./db/routes   # Verbose single package
go test -run TestName     # Specific test
```

### Flutter

```bash
flutter test              # All tests
flutter test test/util/   # Specific directory
flutter test -v           # Verbose output
```

---

*Testing analysis: 2026-09-06*
