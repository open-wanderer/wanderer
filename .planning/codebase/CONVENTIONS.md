# Coding Conventions

**Analysis Date:** 2026-06-10

## Naming Patterns

**Files:**
- Utility files: snake_case suffixed with `_util` (e.g., `array_util.ts`, `date_util.ts`, `api_util.ts`, `format_util.dart`)
- Store files: snake_case with `_store` suffix (e.g., `user_store.ts`, `theme_store.ts`, `tag_store.ts`)
- Model/type files: snake_case (e.g., `trail.ts`, `tag.ts`, `settings.ts`)
- Entity files (Dart): snake_case with `_entity` suffix (e.g., `trail_entity.dart`, `user_entity.dart`)
- SvelteKit routes: SvelteKit conventions (`+page.svelte`, `+layout.svelte`, `+server.ts`, `+error.svelte`)
- Svelte components: PascalCase (e.g., `NavBar.svelte`, `PhotoCard.svelte`, `ConfirmModal.svelte`)
- Flutter screens/pages: PascalCase (e.g., `ProfileScreen`, `TrailDetailScreen`)
- Dart provider files (codegen): suffixed with `_provider.dart` producing `_provider.g.dart` (e.g., `auth_provider.dart`, `api_provider.dart`)

**Functions and Methods:**
- TypeScript/Svelte: camelCase (e.g., `tags_create()`, `trails_index()`, `handleError()`)
- Dart: camelCase (e.g., `register()`, `login()`, `logout()`, `_updateUserEntity()`)
- Prefix naming for store operations: `{entity}_{operation}` (e.g., `users_create`, `users_update`, `tags_index`)
- Utility functions: verb-first naming (e.g., `range()`, `isToday()`, `dateExistsInList()`, `isSameDay()`)
- Boolean functions: prefix with `is` (e.g., `isToday()`, `isRouteProtected()`)
- HTTP handler functions in routes: Uppercase HTTP method names (`GET`, `PUT`, `POST`, `DELETE`, `PATCH`)
- Private functions in Dart: prefixed with underscore (e.g., `_updateUserEntity()`)

**Variables and Constants:**
- All variable names: camelCase (e.g., `navUser`, `authFile`, `searchParams`)
- Boolean variables: prefix with `is` (e.g., `isToday`, `isRouteProtected`, `isValid`)
- Module-level constants: `const` with camelCase (e.g., `privateRoutes = [...]`, `publicRoutes = [...]`)
- Readonly arrays: explicitly typed as `ReadonlyArray<T>` in TypeScript (e.g., `ReadonlyArray<number>`)
- Enum values: lowercase snake_case (e.g., `Collection.users`, `Language.en`, `TrailDifficulty.easy`)

**Types and Classes:**
- TypeScript interfaces/types: PascalCase (e.g., `User`, `Trail`, `AuthRecord`)
- Dart classes: PascalCase (e.g., `Trail`, `TrailEntity`, `UserEntity`, `Auth`)
- Dart freezed classes: PascalCase with `@freezed` annotation (e.g., `Trail`, `TrailExpand`, `FeedItem`)
- Enum names: PascalCase (e.g., `Collection`, `TrailDifficulty`, `Language`)

## Code Style

**TypeScript:**
- Strict mode enabled: `"strict": true` in `web/tsconfig.json`
- Indentation: 2 spaces
- Quotes: Mixed usage (single and double quotes observed; no enforced convention)
- Semicolons: Used consistently throughout
- Type annotations: Explicit on all function parameters and return types (required by strict mode)
- ESLint: No dedicated `.eslintrc` file; type checking via `svelte-check` and TypeScript compiler

**Dart:**
- Indentation: 2 spaces (follows Dart style guide)
- Strong typing: Type annotations required on all variables and function parameters
- Null safety: Non-nullable by default; use `?` for nullable types
- Follows Dart/Flutter style conventions (via `flutter_lints`)

**CSS/Styling:**
- TailwindCSS for styling: utility-first approach in `web/tailwind.config.js`
- PostCSS for processing: configured in `web/postcss.config.js`
- No dedicated CSS files in components; Tailwind classes used inline in `.svelte` files

## Import Organization

**TypeScript/Svelte:**
1. SvelteKit environment imports (`$app/`)
2. Environment variables (`$env/dynamic/public`, `$env/dynamic/private`)
3. Alias imports (`$lib/`)
4. Third-party packages
5. Relative imports

**Dart:**
1. `dart:` imports (standard library)
2. `package:` imports (pub dependencies)
3. Relative imports
4. Part files (code generation results)

**Path Aliases:**
- TypeScript: `$lib/` → `src/lib/`
- TypeScript: `$app/` → SvelteKit internal
- TypeScript: `$env/` → SvelteKit environment
- Dart: No path aliases; relative/package imports only

## Error Handling

**TypeScript/API Layer:**
- Wrap async operations in try/catch
- Throw `APIError` with status code, message, and optional detail: `new APIError(status, message, detail)`
- Class-based error: `web/src/lib/util/api_util.ts` defines `APIError` constructor
- Store functions catch errors; return empty arrays/defaults rather than throwing
- Components guard against missing data (e.g., `trail?.expand?.waypoints ?? []`)
- Backend error responses: JSON object with `message`, `detail` fields matched to status code

**Error Handler Function:**
- `handleError(e)` in `web/src/lib/util/api_util.ts` distinguishes error types:
  - `ZodError` → 400 Bad Request with validation issues
  - `ClientResponseError` → Pass through PocketBase error with original status
  - `SyntaxError` → 400 Bad Request for invalid JSON
  - Other → 500 Internal Server Error

**Dart Error Handling:**
- `AsyncValue.guard()` for exception wrapping in Riverpod providers
- `.catchError()` for specific error handling with fallback
- Exception-based flows set `state` to `AsyncError()` in providers

## Logging

**TypeScript:**
- `console.warn()` for non-critical issues (e.g., markdown fetch failures in components)
- Minimal console logging observed; logging not extensively used
- No structured logging or log levels implemented
- Error boundaries may use console for debugging

**Dart:**
- Uses `print()` for simple debug output (development)
- Error logging via Riverpod state management
- Provider errors captured in `AsyncError` state without explicit logging

## Comments

**TypeScript/Svelte:**
- JSDoc comments for OpenAPI/Swagger documentation on API endpoints
  - Example: `web/src/routes/api/v1/trail/+server.ts` documents `GET`/`PUT` with @swagger blocks
- Minimal inline comments; code is largely self-documenting
- TODOs found only in vendor code (`src/lib/vendor/`), not in main source
- Field-level documentation in vendor code for complex options

**Dart:**
- Minimal inline comments
- Public APIs documented with triple-slash `///` comments (freezed models)
- Private methods/properties may have `//` comments explaining purpose
- No TODOs observed in main app code

## Function Design

**TypeScript:**
- Typed parameters required (strict mode)
- Generic type parameters common in utility functions (e.g., `list<T>()`, `create<T>()`, `upload<T>()`)
- Explicit return type annotations on all functions
- Use of `Promise<T>` for async functions
- Generics leverage for reusable types (e.g., API functions returning `ListResult<T>`)
- Function overloads used sparingly

**Dart:**
- Strong typing with full type annotations
- Async functions return `Future<T>` or `FutureOr<T>`
- Riverpod-generated functions marked with `@riverpod` annotation for async provider creation
- Constructor parameters use named parameters and `required` keyword
- Factory constructors used for JSON deserialization: `factory Model.fromJson(Map<String, dynamic> json)`
- Freezed generates immutable classes with `.copyWith()` for updates

## Module Design

**TypeScript:**
- Named exports for utilities and functions
- Default exports rarely used
- Multiple named exports from utility modules (e.g., `array_util.ts` exports `range()`)
- Stores use named exports: `currentUser`, `users_create`, `login()`, etc.
- Barrel files (index files) not commonly employed
- Classes used for models: `Tag`, `TrailTag`, `Settings`, `APIError`
- Classes used for Page Objects in E2E tests: `IndexPage`
- Type-only imports using `type` keyword: `import type { User } from "$lib/models/user"`

**Dart:**
- Single public class per file (followed by codegen files)
- Freezed classes: immutable via `@freezed abstract class Model with _$Model`
- Entity classes for local storage: `@Entity()` decorator (ObjectBox ORM)
- Model factory constructors: `factory Model.fromJson()` and `toJson()` via codegen
- No barrel files; imports are explicit and file-specific
- Part files declare codegen: `part 'model.freezed.dart'; part 'model.g.dart';`

## Store Patterns (TypeScript/Svelte)

**Store Creation:**
- Type-annotated on creation: `export const currentUser: Writable<User | null> = writable<User | null>(null)`
- Some stores initialized with default values: `theme: Writable<Theme> = writable(getDefaultTheme())`
- Stores co-located with operations (e.g., `user_store.ts` contains `currentUser` store and user functions)

**Store Operations:**
- `get()` function to extract value: `const currentTheme = get(theme)`
- `.set()` to update: `theme.set(newTheme)`
- Stores in functions manage fetch/update logic with error handling
- Common pattern: async function fetches, throws `APIError`, updates store if successful

**Example Store Pattern (from `tag_store.ts`):**
```typescript
let tags: Writable<Tag[]> = writable([]);

export async function tags_index(name: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/tag?' + new URLSearchParams({
        filter: `name~'${name}'`,
    }), { method: 'GET' });
    
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
    
    const response: ListResult<Tag> = await r.json();
    tags.set(response.items);
    return response;
}
```

## API Conventions (TypeScript)

**Centralized API Utilities:**
- Location: `web/src/lib/util/api_util.ts`
- Generic functions: `list<T>()`, `show<T>()`, `create<T>()`, `update<T>()`, `upload<T>()`, `remove()`
- `Collection` enum maps entity names to API collection paths
- Zod schemas for validation (e.g., `TrailCreateSchema`)

**API Route Handlers:**
- Export `GET`, `PUT`, `POST`, `DELETE`, `PATCH` async functions
- All receive `RequestEvent` parameter
- Centralized error handling via `handleError()`
- Data enrichment/transformation applied before return
- Example: Waypoints sorted by `distance_from_start` in GET trail response

**Validation:**
- Input validation via Zod schemas on all API endpoints
- Schemas enforce request body and query parameter types
- Used: `RecordListOptionsSchema`, `RecordIdSchema`, `RecordOptionsSchema`
- Parse input first; throw `ZodError` on invalid data

## Type Safety

**TypeScript:**
- Input validation via Zod schemas on all API endpoints
- Skipneed LibCheck: `"skipLibCheck": true` in tsconfig to skip dependency type checking
- Sourcemaps enabled: `"sourceMap": true` for debugging
- All files use strict mode

**Dart:**
- Non-nullable types by default; nullable with `?`
- Full type annotations required
- Freezed code generation ensures immutability and equality
- JSON deserialization validated via factory constructors

## Riverpod Patterns (Dart)

**Provider Creation:**
- Use `@riverpod` annotation for async provider functions
- Returns `FutureOr<T>` for async operations
- Access to other providers via `ref.watch()`, `ref.read()`, `ref.listen()`
- Codegen produces `.g.dart` file with provider implementation

**Example (from `auth_provider.dart`):**
```dart
@Riverpod(keepAlive: true)
class Auth extends _$Auth {
  Box<UserEntity> get _box => ref.read(objectBoxProvider).box<UserEntity>();

  @override
  FutureOr<UserEntity?> build() async {
    final store = ref.watch(objectBoxProvider);
    final box = store.box<UserEntity>();
    final savedUserEntity = box.getAll().firstOrNull;
    return savedUserEntity;
  }

  Future<UserEntity?> register(String username, String email, String password) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      // operation here
    });
    return state.value;
  }
}
```

**AsyncValue States:**
- `AsyncData<T>` when successful
- `AsyncLoading()` during fetch
- `AsyncError()` on exception (caught by `AsyncValue.guard()`)

---

*Convention analysis: 2026-06-10*
