# 06-SECURITY — Settings Navigation, Language & Units

**Phase:** 06 — settings-navigation-language-units
**ASVS Level:** 1
**Mode:** retroactive-STRIDE (no plan-time threat register existed; register built from implementation)
**Block-on:** critical
**Audited:** 2026-06-20

## Scope

New/changed attack surface introduced by Phase 6:

1. **Settings save flow** — `SettingsLanguageScreen` → `settingsProvider.saveToServer()` → `POST /api/v1/settings/{id}` (user data persistence + authenticated server write).
2. **Locale / unit provider state** — `localeProvider`, `unitProvider` deriving from `settingsProvider`; locale string fed into `MaterialApp.router` and the gen-l10n delegate; unit string fanned out to ~50 `format_util` call sites.
3. **New router sub-routes** — `/settings/privacy`, `/settings/language`, `/settings/notifications` under the existing `/settings` subtree.
4. **12 new locale ARB files + regenerated `AppLocalizations`** — untrusted-at-build-time translation strings rendered into the UI.

Out of scope (pre-existing, unchanged by this phase): PocketBase auth issuance, cookie jar persistence, the global router redirect guard logic itself (only re-verified for the new routes).

## Retroactive STRIDE Threat Register

| ID | STRIDE | Asset / Flow | Disposition | Status |
|----|--------|--------------|-------------|--------|
| T-01 | Tampering / EoP | Settings save body (`unit`, `language`) accepted by `POST /settings/{id}` | mitigate | CLOSED |
| T-02 | Elevation of Privilege | A user saving settings for another user's record id | mitigate | CLOSED |
| T-03 | Tampering (state) | `unit` / `language` provider values driving widget rendering | mitigate | CLOSED |
| T-04 | Information Disclosure | Save-failure error surfaced to the user (no token/PII leakage in toast) | mitigate | CLOSED |
| T-05 | Information Disclosure (transport) | Settings POST over the network (auth cookie + body in transit) | accept | CLOSED (accepted risk) |
| T-06 | Spoofing / EoP | New `/settings/*` sub-routes reachable without authentication | mitigate | CLOSED |
| T-07 | Tampering (injection/XSS-equiv) | 12 ported ARB locale strings rendered as UI text | mitigate | CLOSED |
| T-08 | Denial of Service | Unbounded/missing fields on save (`settings == null`, null value) | mitigate | CLOSED |

## Threat Verification

### T-01 — Tampering / EoP: malformed or unexpected settings fields (mitigate → CLOSED)

The client sends `settings.toJson()` (the full `Settings` model, including `id`/`user`) but the server does not trust it wholesale.

- Body is validated by `SettingsCreateSchema.parse(data)` before any write — `web/src/routes/api/v1/settings/[id]/+server.ts:80` calls `update<Settings>(event, SettingsCreateSchema, Collection.settings)`.
- `update()` parses the body with the Zod schema before the PocketBase call — `web/src/lib/util/api_util.ts:104` (`const safeData = schema.parse(data)`), persisted at `api_util.ts:106`.
- The two Phase-6 fields are whitelisted to fixed enums: `unit: z.enum(["metric","imperial"])` and `language: z.enum(Object.values(Language))` — `web/src/lib/models/api/settings_schema.ts:6-7`. Arbitrary strings are rejected with a ZodError → 400 via `handleError`.
- Client never sends free-form values: `unit` is bound to two literal `RadioListTile<String>` values `'metric'`/`'imperial'` (`app/lib/routes/settings_language_screen.dart:131-138`); `language` is a `Language` enum value (`settings_language_screen.dart:100`, `Language` enum `app/lib/models/settings.dart:6-35`).

Evidence: `web/src/routes/api/v1/settings/[id]/+server.ts:78-85`, `web/src/lib/util/api_util.ts:96-108`, `web/src/lib/models/api/settings_schema.ts:5-7`, `app/lib/routes/settings_language_screen.dart:122-139`.

### T-02 — Elevation of Privilege: writing another user's settings record (mitigate → CLOSED)

The save targets a record id supplied by the client (`/settings/${settings.id}`, `app/lib/provider/settings_provider.dart:36`). Record-level authorization is not done in the SvelteKit layer — it is delegated to PocketBase collection rules under the per-request authenticated client.

- The write uses `event.locals.pb.collection('settings').update(safeParams.id, ...)` — `web/src/lib/util/api_util.ts:106`. `event.locals.pb` is the per-request PocketBase instance bound to the caller's `pb_auth` token, so PocketBase enforces the `settings` collection update rule (owner-scoped) for the given record id.
- The path id is itself validated (`RecordIdSchema.parse(params)`, `api_util.ts:98`) preventing path/filter injection via the id segment.
- The Flutter client only ever submits its own settings id, sourced from the authenticated user's expanded `settings_via_user` record (`app/lib/provider/auth_provider.dart:125-138` → `updateFromServer`).

Disposition rationale: authorization is correctly *transferred to the PocketBase rule engine* and the SvelteKit handler does not bypass it (it uses the auth-bound `locals.pb`, not an admin/superuser client). This is the established project pattern for all collection writes. Verified present, not assumed.

Evidence: `web/src/lib/util/api_util.ts:96-108`, `app/lib/provider/settings_provider.dart:33-39`, `app/lib/provider/auth_provider.dart:120-142`.

### T-03 — Tampering of provider-derived display state (mitigate → CLOSED)

`unit`/`locale` come from `settingsProvider` (server-synced + ObjectBox-backed), not from unauthenticated input, and have safe fallbacks.

- `unitProvider` returns `settings?.unit ?? 'metric'` — `app/lib/provider/local_settings_provider.dart:52-56`. A null settings state or null unit can never produce an undefined unit string.
- `localeProvider` returns `null` (device-locale fallback) when language is null, else `Locale(lang.name)` — `app/lib/provider/local_settings_provider.dart:45-50`. `lang.name` is constrained to the `Language` enum identifiers, all of which exist in `supportedLocales` after Plan 04, so no unsupported-locale crash.
- A locale value can only be one of the 14 enum codes (`app/lib/models/settings.dart:6-35`); it cannot be an attacker-chosen arbitrary string.

Evidence: `app/lib/provider/local_settings_provider.dart:45-56`, `app/lib/models/settings.dart:6-35`.

### T-04 — Information Disclosure via error toast (mitigate → CLOSED)

`saveToServer` has no internal try/catch and can throw `DioException`. The screen catches it and surfaces only a generic localized message — no exception detail, stack, token, or server response is shown.

- `_save` wraps the call in try/catch and on failure adds `ToastMessage(... text: l10n.error_saving_settings)` — `app/lib/routes/settings_language_screen.dart:41-53`. The caught error object is discarded (`catch (_)`), so no `DioException`/response body reaches the UI.
- The null-settings guard path uses the same generic key (`settings_language_screen.dart:88-99`).
- The message key resolves to a static string ("Error saving settings" / German equivalent), confirmed added in Plan 02 (`app_en.arb`/`app_de.arb`).

Evidence: `app/lib/routes/settings_language_screen.dart:33-54, 88-99`.

### T-05 — Information Disclosure in transit (accept → CLOSED, accepted risk)

The settings POST carries the `pb_auth` cookie and body over the network. See the Accepted Risks log below. The app defaults unqualified server URLs to `https://` (`app/lib/routes/server_selection_screen.dart:32-33`) and does **not** weaken TLS — no `badCertificateCallback`, custom `HttpClient`, or cert-bypass exists anywhere in `app/lib` (grep returned no matches). Transport confidentiality therefore depends on the operator-provided server presenting valid TLS, which is the documented accepted posture for this self-hostable app.

Evidence: `app/lib/routes/server_selection_screen.dart:31-34`, `app/lib/provider/api_provider.dart:11-20`; negative grep for `badCertificate|allowBadCert|onHttpClientCreate|HttpClient(` across `app/lib` returned nothing.

### T-06 — Spoofing / EoP: unauthenticated access to new sub-routes (mitigate → CLOSED)

The three new routes are children of `/settings` and inherit the global redirect guard; no new public route was introduced.

- The router `redirect` callback sends any unauthenticated user (`user == null`) hitting a non-auth route to `/welcome` — `app/lib/provider/router_provider.dart:68-99`. The auth-allowlist is exactly `['/login','/register','/welcome','/select-server']` (`router_provider.dart:79-84`); `/settings/privacy`, `/settings/language`, `/settings/notifications` are not in it, so they are gated.
- The new routes are plain authenticated builders with no `extra`/path-param trust issues (`router_provider.dart:177-188`). The stub Privacy/Notifications screens render an empty themed body — no data exposure.

Evidence: `app/lib/provider/router_provider.dart:68-99, 169-194`.

### T-07 — Tampering / injection via ported translation strings (mitigate → CLOSED)

The 12 new ARB files (`cs, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh`) are rendered through Flutter `Text` widgets and the gen-l10n delegate, which treat values as plain text (no HTML/script interpretation; Flutter has no DOM/XSS sink). ICU placeholder names were validated against the English template during the port (Plan 04 placeholder-safety check), so a mismatched placeholder cannot inject an unexpected argument or crash gen-l10n.

- Strings are sourced from the project's own `web/src/lib/i18n/locales/*.json` (first-party translations), not user input.
- gen-l10n compiled all 14 locales with zero errors; `app_localizations.dart` lists 14 locales (Plan 04 verification). Untranslated keys fall back to the English template rather than rendering blank/attacker-controlled text.

Evidence: Plan 04 implementation + verification (`06-04-SUMMARY.md`), `app/lib/routes/settings_language_screen.dart` rendering via `Text(_languageNames[language]!)` and `Text(l10n.*)`.

### T-08 — DoS / null-deref on save with no settings loaded (mitigate → CLOSED)

Both save entry points guard against a null settings model and a null selection before constructing a `copyWith`:

- Language path: `if (value == null) return;` then explicit `if (settings == null)` branch that shows a toast and returns without calling `_save` — `app/lib/routes/settings_language_screen.dart:86-100`.
- Unit path: `if (value == null || settings == null) return;` — `settings_language_screen.dart:124-126`.

No unbounded input is possible (radio/enum-constrained), and the server rejects oversized/extra fields via the Zod schema (T-01).

Evidence: `app/lib/routes/settings_language_screen.dart:84-127`.

## Accepted Risks Log

| ID | Risk | Rationale | Owner | Review |
|----|------|-----------|-------|--------|
| T-05 | Settings POST (auth cookie + body) confidentiality depends on the operator's TLS configuration; the client does not pin certificates. | Wanderer is self-hostable against an arbitrary user-supplied server URL. The client defaults to `https://` and never disables certificate validation, but cannot enforce TLS on a server the operator misconfigures. Certificate pinning is incompatible with the bring-your-own-server model. ASVS L1 scope. | App maintainer | Revisit if a managed first-party backend is introduced. |

## Unregistered Flags

None. No SUMMARY.md (`06-01` through `06-04`) contains a `## Threat Flags` section, and no new attack surface beyond the four scoped items above was detected during implementation. The only cross-cutting change (the `error_saving_settings` ARB key, Plan 02) is a static UI string and introduces no new surface.

## Result

All 8 retroactively-modeled threats resolve to CLOSED (7 mitigated in code, 1 accepted and logged). No open mitigations. No critical findings. Phase 6 clears the security gate.
