# Coding Conventions

**Analysis Date:** 2026-06-07

## Naming Patterns

**Files:**
- Utility files: snake_case suffixed with purpose (e.g., `array_util.ts`, `date_util.ts`, `api_util.ts`)
- Store files: snake_case with `_store` suffix (e.g., `user_store.ts`, `theme_store.ts`, `search_store.ts`)
- Model/type files: snake_case (e.g., `settings.ts`, `trail.ts`, `tag.ts`)
- Route components: `+page.svelte`, `+layout.svelte`, `+server.ts`, `+error.svelte` (SvelteKit convention)
- Svelte components: PascalCase (e.g., `Scene.svelte`, `TrailCard.svelte`, `FeedCard.svelte`)

**Functions:**
- camelCase for all function names
- Prefix naming convention for store functions: `{entity}_{operation}` (e.g., `users_create`, `users_update`, `users_auth_methods`, `users_delete`)
- Utility functions: verb-first naming (e.g., `range`, `isToday`, `dateExistsInList`, `isSameDay`, `isRouteProtected`)
- Handler functions in routes: HTTP method names (`GET`, `PUT`, `POST`, `DELETE`, `PATCH`)

**Variables:**
- camelCase for all variable names
- Const for module-level constants: `const privateRoutes = [...]`, `const publicRoutes = [...]`
- Readonly arrays: explicitly typed as `ReadonlyArray<T>` (e.g., `ReadonlyArray<number>`)
- Boolean variables: prefix with `is` (e.g., `isToday`, `isRouteProtected`, `isValid`)

**Types:**
- PascalCase for all types and interfaces
- Enum values: lowercase snake_case (e.g., `Collection.users`, `Language.en`)
- Classes: PascalCase (e.g., `APIError`, `Tag`, `TrailTag`, `Settings`, `IndexPage`)

## Code Style

**Formatting:**
- TypeScript with strict mode enabled (`"strict": true` in tsconfig.json)
- No explicit linter config (no `.eslintrc` or `.prettierrc` detected)
- Code uses semicolons consistently
- Indentation: 2 spaces (observed in all TypeScript files)
- String quotes: mix of single and double quotes (no enforced convention)

**Linting:**
- No dedicated linter config file present
- TypeScript compiler provides type checking via `svelte-check`
- No Prettier or ESLint configuration detected

## Import Organization

**Order:**
1. External library imports (e.g., `import { browser } from '$app/environment'`)
2. Environmental imports (e.g., `import { env } from '$env/dynamic/public'`)
3. Relative path imports to models (e.g., `import type { User } from "$lib/models/user"`)
4. Relative path imports to utilities (e.g., `import { getPb } from "$lib/pocketbase"`)
5. Relative path imports to stores (e.g., `import { currentUser } from "$lib/stores/user_store"`)

**Path Aliases:**
- `$app/` - SvelteKit environment and navigation
- `$env/dynamic/public` - Public environment variables
- `$env/dynamic/private` - Private environment variables (server-side only)
- `$lib/` - Alias for `src/lib/` (configured by SvelteKit)

**Example from `src/lib/stores/user_store.ts`:**
```typescript
import type { User, UserAnonymous } from "$lib/models/user";
import { getPb } from "$lib/pocketbase";
import { APIError } from "$lib/util/api_util";
import { type AuthMethodsList } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
```

## Error Handling

**Patterns:**

1. **Custom Error Class**: `APIError` extends `Error` with `status`, `message`, and `detail` properties
   ```typescript
   export class APIError extends Error {
       status: number;
       message: string;
       detail: any;
       
       constructor(status: number, message: string, detail?: any) {
           super();
           this.status = status;
           this.message = message;
           this.detail = detail
       }
   }
   ```

2. **Error Handling in Store Functions**: Throw `APIError` on non-ok responses
   ```typescript
   if (!r.ok) {
       const response = await r.json();
       throw new APIError(r.status, response.message, response.detail)
   }
   ```

3. **Centralized Error Handling**: `handleError()` function converts different error types to JSON responses (`src/lib/util/api_util.ts`)
   - `ZodError` → 400 Bad Request
   - `ClientResponseError` → Preserve original status code
   - `SyntaxError` → 400 Bad Request
   - Other errors → 500 Internal Server Error

4. **Route Error Handling**: Try-catch with `handleError()` return
   ```typescript
   export async function GET(event: RequestEvent) {
       try {
           // ...
       } catch (e: any) {
           return handleError(e);
       }
   }
   ```

5. **Server Hook Error Handling**: Use SvelteKit `error()` function or `throw error(status, message)`

## Logging

**Framework:** `console` (no dedicated logging library)

**Patterns:**
- `console.warn()` for non-critical issues (e.g., markdown fetch failures in components)
- Minimal console logging observed; logging not extensively used
- No structured logging or log levels implemented

## Comments

**When to Comment:**
- JSDoc comments for OpenAPI/Swagger documentation (see `src/routes/api/v1/trail/+server.ts`)
- Minimal inline comments; code is largely self-documenting
- TODOs found only in vendor code (`src/lib/vendor/`), not in main source

**JSDoc/TSDoc:**
- OpenAPI/Swagger annotations in JSDoc blocks for API endpoints
- Example: API route handlers document parameters, request bodies, and responses
- Field-level documentation in vendor code for complex options

## Function Design

**Size:** Functions range from 2-30 lines; no giant functions detected

**Parameters:** 
- Typed parameters required (TypeScript strict mode)
- Generic type parameters common in utility functions (e.g., `list<T>`, `create<T>`, `upload<T>`)
- Function overloads used sparingly

**Return Values:**
- Explicit return type annotations on all functions
- Use of `Promise<T>` for async functions
- Generics leverage for reusable types (e.g., API functions returning `ListResult<T>`)

**Example from `src/lib/util/array_util.ts`:**
```typescript
export function range(to: number, from: number = 0): ReadonlyArray<number> {
    return [...Array(to - from).keys()].map(i => i + from);
}
```

**Example from `src/lib/util/date_util.ts`:**
```typescript
export function isSameDay(d1: Date, d2: Date) {
    return d1.getFullYear() === d2.getFullYear() &&
        d1.getMonth() === d2.getMonth() &&
        d1.getDate() === d2.getDate();
}
```

## Module Design

**Exports:**
- Named exports for utilities and functions
- Default export rarely used
- Multiple named exports from utility modules (e.g., `array_util.ts` exports `range`)
- Stores use named exports: `currentUser`, `users_create`, `login`, etc.

**Barrel Files:**
- Not extensively used; most imports reference specific files
- Index files not commonly employed

**Class and Type Patterns:**
- Classes used for models: `Tag`, `TrailTag`, `Settings`, `APIError`
- Classes used for Page Objects in E2E tests: `IndexPage`
- Type-only imports using `type` keyword (e.g., `import type { User } from "$lib/models/user"`)

## Store Patterns (Svelte)

**Store Creation:**
- Use `writable<T>()` for mutable stores
- Type-annotated on creation: `export const currentUser: Writable<User | null> = writable<User | null>()`
- Some stores initialized with default values (e.g., `theme: Writable<Theme> = writable(getDefaultTheme())`)

**Store Usage:**
- `get()` function to extract value: `const currentTheme = get(theme)`
- `.set()` to update: `theme.set(newTheme)`
- Stores typically co-located with operations (e.g., `user_store.ts` contains `currentUser` store and user operations)

**Example from `src/lib/stores/theme_store.ts`:**
```typescript
import { get, writable, type Writable } from "svelte/store";

type Theme = "dark" | "light"

export const theme: Writable<Theme> = writable(getDefaultTheme());

export function toggleTheme() {
    const currentTheme = get(theme);
    const newTheme = currentTheme === "light" ? "dark" : "light";
    document.documentElement.classList.remove(currentTheme)
    document.documentElement.classList.add(newTheme)
    theme.set(newTheme)
    localStorage.setItem("theme", newTheme);
}
```

## API Conventions

**Collection-Based CRUD:**
- Centralized API utilities in `src/lib/util/api_util.ts`
- Generic functions: `list<T>()`, `show<T>()`, `create<T>()`, `update<T>()`, `upload<T>()`, `remove()`
- `Collection` enum maps entity names to API collection paths
- Zod schemas for validation (e.g., `TrailCreateSchema`)

**Route Handler Pattern:**
- Export `GET`, `PUT`, `POST`, `DELETE`, `PATCH` async functions
- All receive `RequestEvent` parameter
- Centralized error handling via `handleError()`
- Data enrichment/transformation applied before return

**Example from `src/routes/api/v1/trail/+server.ts`:**
```typescript
export async function GET(event: RequestEvent) {
    try {
        const r = await list<Trail>(event, Collection.trails);
        // ... transform data
        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}
```

## Type Safety

**ZodSchema Validation:**
- Input validation via Zod schemas on all API endpoints
- Schemas enforce request body and query parameter types
- Example: `RecordListOptionsSchema`, `RecordIdSchema` used in `api_util.ts`

**TypeScript Strict Mode:**
- All files use strict mode
- `skipLibCheck: true` to skip type checking of dependencies
- `sourceMap: true` for debugging

---

*Convention analysis: 2026-06-07*
