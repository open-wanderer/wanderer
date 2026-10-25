---
status: awaiting_human_verify
trigger: "Have a look at #1346 and the reviewers comment. Setup a proper testing environment. If you need me to test OIDC workflows, let me know. You are currently on the app branch, because gsd is not available on dev/main."
created: 2026-10-14
updated: 2026-10-14
---

# Debug: PR #1346 review findings (generic OIDC email/username)

## Context

- PR: https://github.com/open-wanderer/wanderer/pull/1346 (base `dev`, head `fix/oidc-unverified-email-claims`, commit 5baff13cb). Fixes issue #1342 (Cloudflare Access OIDC: sends `sub`, `email`, `name`, no `email_verified`, no `preferred_username`).
- PR adds `OAuth2EmailHandler` (puts raw `email` claim into `e.CreateData` for new records) and a username fallback to `OAuth2User.Name` in `OAuth2UsernameHandler` (db/hooks/users.go, registered in db/main.go).
- Reviewer (slothful-vassal) requested changes after reproducing with a fake IdP sending the #1342 claims.
- IMPORTANT: code changes go on the PR branch, NOT feature/app. Use a git worktree, e.g. `git worktree add ../wanderer-oidc fix/oidc-unverified-email-claims` (local branch exists; fast-forward/rebase onto origin as needed). This session file stays on feature/app.
- Go tests that run migrations: use `GOTOOLCHAIN=go1.26.1` (Go 1.27 breaks PocketBase tests with a JSON stack overflow).
- Bash sandbox blocks localhost; curl/HTTP against local servers needs the sandbox disabled.
- Do not push. Commit locally on the PR branch with headline-only messages (no body, no Co-Authored-By). The user decides on pushing/replying to the reviewer.

## Symptoms

expected: Cloudflare-Access-style OIDC users (no email_verified, no preferred_username) get a usable username and their email, repeated logins from multiple devices keep all sessions valid, confirming the email/admin-verify does not unlink the provider, and non-OIDC providers (Google, Microsoft, Apple, ...) keep their previous username behaviour.

actual (reviewer's reproduction, logins without existing session):
| Step | dev | this PR |
|---|---|---|
| 1st login | `users078255`, no email, verified | `David_Schwartz`, `david@example.com`, not verified |
| 2nd login | token from 1st login stays valid | token from 1st login is invalid |
| Email confirmed | - | Cloudflare link removed |
| 3rd login | same account | new account `David_Schwartz_2` without email |

1. Repeated OIDC logins without a session invalidate other sessions: PocketBase sees an unverified record on unauthenticated OAuth2 login, assumes it may be pre-created, sets a random password -> tokenKey refresh. hooks.server.ts refreshes tokens per request, so phone login logs out browser and vice versa. Wanderer sends no verification mail for OAuth2 accounts and has no UI for it.
2. Confirming email (verify, password reset, or admin ticking verified — the #1342 workaround) makes PocketBase delete all external auths of the record. Next Cloudflare login has no email to match -> new empty account; trails stranded.
3. Username fallback applies to all providers: Google/Microsoft/Apple/Facebook/Spotify send no username either, so new accounts now get the real full name, which becomes part of the actor IRI and federated handle and cannot be changed in the UI.

errors: none (behavioural)
timeline: introduced by PR #1346 commit 5baff13cb; dev behaviour is the baseline
reproduction: fake OIDC IdP returning {sub, email, name} without email_verified/preferred_username, configured as PocketBase `oidc` provider; log in repeatedly without an existing session.

Reviewer's suggested direction:
- Per-provider opt-in to trust the email of a generic OIDC provider (not one global switch).
- Set `e.OAuth2User.Email` instead of `CreateData` -> PocketBase creates the account verified; neither problem occurs.
- Opt-in only overrides an ABSENT `email_verified` claim; never an explicit false, null or malformed value.
- Keep existing-account-with-that-address check, excluding the account currently being authenticated.
- Opt-in should also apply to existing accounts (verified, no email, created before fix): PocketBase fills the email on next login; existing-account check needed there too, else "Value must be unique". Their usernames stay.
- Limit username fallback to `oidc`, `oidc2`, `oidc3` (or document intent).

## Goals for this session

1. Build a reproducible local test environment: a minimal fake OIDC IdP (scriptable claims: with/without email_verified, explicit false/null, preferred_username present/absent) plus a scratch PocketBase instance running the wanderer db code with the `oidc` provider pointed at it, and a scripted end-to-end flow (auth URL -> fake authorize -> code -> authWithOAuth2 / auth-with-oauth2 endpoint) that can reproduce all reviewer table rows on the current PR head. Keep the harness outside the repo or in an uncommitted location unless it is a proper Go test; document how to run it in this file.
2. Reproduce findings 1-3 against the PR head (and baseline dev) with that harness.
3. Fix per the reviewer's direction; add Go unit tests; re-run the e2e harness to show all rows fixed (multiple logins keep tokens valid, verify does not unlink, 3rd login same account, non-oidc providers unchanged, explicit email_verified=false untouched, existing verified-no-email accounts get email filled, uniqueness conflict handled).
4. Open design decision for the user (ask via checkpoint before implementing): how the per-provider opt-in is configured (e.g. env var listing provider names such as `POCKETBASE_OAUTH2_TRUST_EMAIL=oidc,oidc2`, vs. other mechanism) and whether docs need updating.
5. Only ask the user to test real OIDC (Cloudflare Access) if the fake IdP cannot cover something.

## Current Focus

hypothesis: CONFIRMED. PocketBase's unverified-record handling on OAuth2 login (triggered because the PR creates the account unverified) causes findings 1 and 2; the provider-agnostic Name fallback and the provider-agnostic, claim-agnostic email handler cause finding 3 and the extra cases found by the harness (see Resolution.root_cause).
test: harness (see "Test harness" section) - baseline binary 85/85 against its own expectations; PR head reproduces every reviewer table row.
expecting: after the fix, `-expect trust` and `-expect default` both 0 failures on a binary built from the PR worktree. RESULT: trust 90/0, default 88/0.
next_action: Fix is committed locally on fix/oidc-unverified-email-claims (da4c45285), not pushed. Waiting for the user: review the commit, decide on pushing and on the reply to the reviewer (draft in the session return). Optional real-world check with Cloudflare Access (set OIDC_TRUST_EMAIL=true). After the user confirms, archive this session to .planning/debug/resolved/.

### Decision (goal 4, answered by the user 2026-10-14)

DATA_START
1. Opt-in mechanism: per-slot boolean env vars OIDC_TRUST_EMAIL / OIDC2_TRUST_EMAIL / OIDC3_TRUST_EMAIL (mirroring the OIDC_SCOPES convention in db/main.go).
2. Username fallback: only for oidc/oidc2/oidc3 slots that have the opt-in enabled.
3. Docs: document the opt-in in environment-configuration.md and auth-providers.md. No changelog entry.
DATA_END

Consequence for the harness: the `default` column of the username checks in s1, s3 and s8 becomes `generated` (no opt-in -> no name fallback). `trust` stays: oidc opted in -> name; oidc2 and google not opted in -> generated.

### reasoning_checkpoint

hypothesis: (see Resolution.root_cause) PocketBase's unverified-record handling is triggered because the PR hands the email over via CreateData; the fix is to hand it over as `OAuth2User.Email` (so the account is created verified and `Record.Email() == OAuth2User.Email`), gated per slot, and only when `email_verified` is absent.
confirming_evidence:
  - PocketBase v0.38.0 oauth2Submit: new-record branch marks verified iff `Record.Email() == OAuth2User.Email`; existing-record branch only randomises the password / deletes external auths when `!Record.Verified()`; the unverified->verified hook deletes external auths. A verified record hits none of these (baseline binary 85/85 shows it).
  - Harness on PR head reproduces each reviewer row; every verified=false account has its older tokens killed, every verified=true account keeps them.
  - The account lookup (external auth, then email) runs before the hooks, so setting OAuth2User.Email in a hook cannot link accounts by email; identity stays `provider + sub`.
falsification_test: with the fix and `OIDC_TRUST_EMAIL=true` for `oidc` only, the harness `-expect trust` must show s1 verified=true with email, all tokens valid across logins 2/2b, confirm keeps 1 external auth, login 3 same account; s3 explicit false/null/"maybe"/"false" untouched; s7 google and s8 oidc2 as baseline. If any verified=false account still appears for the opt-in slot, the hypothesis (verified-at-creation removes both effects) is wrong.
fix_rationale: root-cause fix, not a symptom patch: it removes the unverified state instead of working around PocketBase's reaction to it. Per-slot opt-in + absent-claim-only + uniqueness check with self-exclusion cover the reviewer's security/privacy points; gating the Name fallback behind the same opt-in stops the real-name-as-handle leak for every other provider.
blind_spots: real Cloudflare Access claim set / UX (fake IdP reproduces the #1342 claims exactly, userinfo path only; the id_token path feeds the same RawUser); behaviour with a PocketBase version other than 0.38; accounts that a pre-fix PR build already created unverified are only healed on the next login of an opted-in slot.

### Fix plan (implemented in da4c45285 on top of 5baff13cb; text below is the plan as written before the work, deviations are listed under Resolution)

All in db/hooks/users.go (+ db/main.go wiring, + docs):

1. `OAuth2EmailHandler(trusted func(providerName string) bool)` (or a set), for `e.ProviderName` in the opt-in only:
   - `e.OAuth2User != nil && e.OAuth2User.Email == ""`.
   - `_, present := e.OAuth2User.RawUser["email_verified"]`; if `present` (false, null, "maybe", anything) -> `e.Next()` untouched.
   - email = trimmed string `RawUser["email"]`, must satisfy `is.EmailFormat`.
   - another account with that address, EXCLUDING `e.Record` (the account being authenticated) -> `e.Next()` untouched; `sql.ErrNoRows` from `FindAuthRecordByEmail` means free; any other error is returned.
   - then `e.OAuth2User.Email = email` (NOT `CreateData`). Applies to new records AND existing ones (drop the `IsNewRecord` gate). Keep the "client submitted `createData.email` takes precedence" rule for new records.
2. `OAuth2UsernameHandler`: only fall back to `OAuth2User.Name` when the provider is `oidc`, `oidc2` or `oidc3` (and, depending on the decision, only when that slot has the opt-in).
3. Unit tests in db/hooks/users_test.go: absent vs explicit false/null/"maybe", non-opted provider, existing record without email, uniqueness incl. self-exclusion, `IsNewRecord` true/false, username fallback per provider.
4. Recommended recurrence guard: one Go test that drives the real `auth-with-oauth2` route (`tests.NewTestApp` + `tests.ApiScenario`, `httptest` fake IdP as the provider `userInfoURL`) through login 1 / login 2 / verify / login 3. The existing unit tests call the hook with a hand-built event and therefore never reached PocketBase's `oauth2Submit`, which is where both bugs live.
5. Docs: document the opt-in next to `OIDC_SCOPES` in docs/src/content/docs/run/environment-configuration.md and docs/src/content/docs/run/backend-configuration/auth-providers.md.

Predictions the harness must confirm after the fix (falsifiable): s1 login 1 verified=true with email; login 2/2b keep all tokens valid; confirm keeps 1 external auth; login 3 same account. s3 all four explicit variants untouched. s4 email filled, token valid. s5b no "Value must be unique". s6 legacy account verified on login 1 (needs the self-exclusion, otherwise the address "belongs to another account"). s7/s8 unchanged for google and oidc2 email.

## Test harness (goals 1 and 2, done)

Location (uncommitted, on feature/app, own go.mod, stdlib only): `/Users/christianbeutel/Documents/svelte/wanderer/.planning/debug/oidc-pr1346-harness/` (`main.go` runner and check table, `idp.go` fake IdP, `pb.go` PocketBase/Meilisearch lifecycle and API helpers, `scenarios.go` s1-s8, `results/` saved outputs).

What it does: starts a private Meilisearch (the `init_meilisearch` migration needs one; the user's real Meilisearch on :7700 is never touched), a scratch PocketBase from the given binary on a fresh data dir, enables `oidc`, `oidc2` and `google` on the `users` collection pointed at an in-process fake IdP (`/authorize`, `/token` with PKCE S256 check, `/userinfo` returning the scripted claims; nil = JSON null, missing key = claim absent), then plays each login through the real HTTP flow: `GET auth-methods` -> `GET authURL+redirect` (no redirect following, read `code`) -> `POST auth-with-oauth2`. Token validity = `POST auth-refresh` (what hooks.server.ts does on each request). Account state/external auths via superuser API.

Binaries are built from the sources, outside the repo (scratchpad of the creating session; rebuild if it is gone):

```sh
SP=/private/tmp/claude-501/-Users-christianbeutel-Documents-svelte-wanderer/95a5f5ef-e707-4504-a881-7365a4cca125/scratchpad
mkdir -p $SP/bin
# PR head / fixed worktree
cd /Users/christianbeutel/Documents/svelte/wanderer-oidc/db && GOTOOLCHAIN=go1.26.1 go build -o $SP/bin/pb-pr .
# baseline = PR parent f27a7ca41
mkdir -p $SP/base-src && cd /Users/christianbeutel/Documents/svelte/wanderer-oidc && git archive HEAD~1 db | tar -x -C $SP/base-src   # run while HEAD is 5baff13cb
cd $SP/base-src/db && GOTOOLCHAIN=go1.26.1 go build -o $SP/bin/pb-base .
# harness
cd /Users/christianbeutel/Documents/svelte/wanderer/.planning/debug/oidc-pr1346-harness && go build -o $SP/bin/oidcharness .
```

Run (needs the Bash sandbox disabled; ports 18090 PocketBase, 18091 IdP, 17701 Meilisearch; every run starts from an empty database and stops its child processes, and refuses to start if a port still answers):

```sh
$SP/bin/oidcharness -bin $SP/bin/pb-base -work $SP/run-base -expect baseline          # harness sanity: must be 0 failures
$SP/bin/oidcharness -bin $SP/bin/pb-pr   -work $SP/run-pr   -expect baseline          # PR head vs dev behaviour
$SP/bin/oidcharness -bin $SP/bin/pb-pr   -work $SP/run-pr   -expect trust -env OIDC_TRUST_EMAIL=true   # fixed, opt-in on for "oidc" only
$SP/bin/oidcharness -bin $SP/bin/pb-pr   -work $SP/run-pr   -expect default        # fixed, no opt-in configured
# -only s1,s3 selects scenarios; -expect observe prints without asserting
```

Expectation columns printed on every line: `baseline` = dev before the PR, `trust` = fixed code with the opt-in enabled for `oidc` only (not oidc2/google), `default` = fixed code, no opt-in. The name fallback for the username is gated by the same opt-in (decision 2026-10-14), so the `default` column of the username checks in s1, s3 and s8 is `generated` and s8 (oidc2, not opted in) expects `generated` in `trust` as well. Expectation `-` = not asserted in that mode. Saved outputs of the final runs: results/04-fixed-expect-trust.txt (90 passed, 0 failed) and results/05-fixed-expect-default.txt (88 passed, 0 failed); results/03-fixed-expect-trust-first-run.txt is the first run with one wrong expectation (see Evidence, s4).

Gaps the fake IdP cannot cover: Cloudflare Access' real claim set and login UX (the issue #1342 claims are reproduced exactly: sub, email, name only). Not needed for correctness: PocketBase reads claims from `userInfoURL` or from the id_token, both feed the same `RawUser` the hooks use. Real-world confirmation by the user is optional, after the fix.

## Evidence

- timestamp: 2026-10-14
  checked: PocketBase v0.38.0 `apis/record_auth_with_oauth2.go` `oauth2Submit`, `tools/auth/oidc.go`, `core/external_auth_model.go`
  found: (a) OIDC provider sets `AuthUser.Email` only if `cast.ToBool(email_verified)`; `RawUser` holds the full claim map. (b) New-record branch: payload email = `CreateData.email` if non-empty else `OAuth2User.Email`; the account is marked verified only if `Record.Email() == OAuth2User.Email`. (c) Existing-record branch: for a record that is not verified and is not the logged-in session record -> `SetRandomPassword()` (tokenKey refresh, other sessions die) and `DeleteAllExternalAuthsByRecord`; verified only if record email is empty or equals `OAuth2User.Email`. (d) `OnRecordUpdateExecute` (priority 99) deletes all external auths and refreshes the token key whenever a record goes unverified -> verified. (e) The account lookup (external auth, then `FindAuthRecordByEmail(OAuth2User.Email)`) runs BEFORE the OnRecordAuthWithOAuth2Request hooks, so a hook cannot cause linking by email; it can only change what is written afterwards.
  implication: PR head hands the email via CreateData, so `Record.Email() ("david@..") != OAuth2User.Email ("")` -> account stays unverified -> every later login hits (c) and verifying hits (d). Exactly the reviewer's findings 1 and 2. Setting `OAuth2User.Email` instead makes the account verified at creation and bypasses (c) and (d).

- timestamp: 2026-10-14
  checked: harness, baseline binary (PR parent f27a7ca41) with `-expect baseline`
  found: 85 passed, 0 failed. s1: 1st login `users503880`, email empty, verified true; token of login 1 valid after login 2 and 2b; confirm keeps 1 external auth; login 3 same account. Matches the reviewer's `dev` column.
  implication: the harness is a faithful oracle for dev behaviour (results/01-baseline-binary-expect-baseline.txt).

- timestamp: 2026-10-14
  checked: harness, PR head 5baff13cb with `-expect baseline`
  found: 51 passed, 34 failed (results/02-pr-head-expect-baseline.txt). s1 reproduces the reviewer table row by row: login 1 `David_Schwartz` / `david.schwartz@example.com` / verified=false; login 2 token of login 1 INVALID (also login 2b); after confirm (verified=true via admin) external auths = 0; login 3 different account, isNew=true, username `David_Schwartz_2`, email empty.
  implication: findings 1 and 2 reproduced against the PR head with the real HTTP flow. Every verified=false account in every scenario has invalid older tokens; every verified=true account keeps them (correlation matches code path (c)).

- timestamp: 2026-10-14
  checked: harness s3, PR head
  found: `email_verified` explicitly `false`, `null`, `"maybe"` and `"false"` all get the email assigned, account unverified, older tokens invalidated on the next login (plus the Name username).
  implication: the PR overrides explicit negative claims too; the reviewer's "only an ABSENT claim" requirement is a real defect, not just style. Check `_, present := RawUser["email_verified"]`.

- timestamp: 2026-10-14
  checked: harness s7 (provider `google`) and s8 (`oidc2`), PR head
  found: Google login with verified email gets username `Gerda_Google` (baseline: `users025693`-style); Google without `email_verified` gets username `Gustav_Google`, the email assigned, verified=false, token of login 1 invalid on login 2. oidc2 behaves like oidc: name username, email taken, unverified, tokens die.
  implication: finding 3 reproduced, and the email handler is equally provider-agnostic: non-OIDC providers lose their previous behaviour too, and an opt-in per provider must not leak into oidc2/google.

- timestamp: 2026-10-14
  checked: harness s4 and s6, PR head with `-expect trust`
  found: s4: an existing verified account without email does NOT get the email on the next login (hook is gated by `IsNewRecord`). s6: a PR-head-style leftover (unverified, with email, linked) stays unverified after login (every login keeps killing older tokens). s5 (address already used, new and existing identity) passes only because the PR's hook does nothing for existing accounts and skips taken addresses for new ones.
  implication: matches the reviewer's "also apply to existing accounts" point. s5b ("existing identity + taken address") and s6 (self-exclusion) become real tests once the hook acts on existing records: a naive "email exists -> skip" check would either raise PocketBase's "Value must be unique" (5b, if not checked) or never heal the leftover (s6, if self is not excluded).

- timestamp: 2026-10-14
  checked: web/src/routes/api/v1/auth/oauth/+server.ts, web/src/hooks.server.ts
  found: login calls `locals.pb.collection('users').authWithOAuth2Code(name, code, codeVerifier, redirectURL)` with no `createData` (so the `CreateData` precedence rule in the PR never triggers from the web app). hooks.server.ts calls `authRefresh` on every request with a valid authStore and clears the store/cookies on failure.
  implication: a token invalidated by another device's OAuth2 login logs that session out on its next request, as the reviewer described. The harness' `auth-refresh` probe is the same call.

- timestamp: 2026-10-14
  checked: existing unit tests `GOTOOLCHAIN=go1.26.1 go test ./hooks/ -run OAuth2` at PR head
  found: pass. They build a `RecordAuthWithOAuth2RequestEvent` by hand and assert only `e.CreateData["email"]`.
  implication: (prevention) nothing exercised `oauth2Submit`; the bugs only exist in the interaction with PocketBase's handler. A route-level Go test is the recurrence guard.

- timestamp: 2026-10-14
  checked: harness process hygiene
  found: first PR-head run exited via `os.Exit(1)`, skipping deferred cleanup; the orphaned PocketBase on :18090 served the next run, which produced a spurious s2 failure ("validation_not_unique" on external auth). Fixed in the harness (all exits go through `exit()`, cleanup is idempotent, startup aborts if a port still answers); orphans (pids 91070, 91072, mine) were killed; results above are from the fixed harness.
  implication: not a PR behaviour. Recorded so the s2 output of an earlier run is not misread.

- timestamp: 2026-10-14
  checked: PocketBase admin UI bundle (`ui/dist/assets/index-*.js`), docs
  found: the provider panel exposes only `extra.jwksURL` and `extra.issuers` for OIDC; arbitrary `extra` keys cannot be entered in the UI. Existing per-slot configuration precedent: `OIDC_SCOPES` / `OIDC2_SCOPES` / `OIDC3_SCOPES` (db/main.go `oidcScopesEnv`, documented in docs/src/content/docs/run/environment-configuration.md and backend-configuration/auth-providers.md).
  implication: input for the opt-in decision (see checkpoint).

- timestamp: 2026-10-14
  checked: fixed binary (da4c45285), harness `-expect trust -env OIDC_TRUST_EMAIL=true`, first run
  found: 86 passed, 1 failed. s4: when the email of an EXISTING verified account is first filled in, the token of the previous login is invalidated (my prediction "token valid" was wrong). Cause: PocketBase core `onRecordSaveExecute` (core/record_model.go) refreshes `tokenKey` on every email change of an existing record; the stock "fill empty email from a verified provider" path does exactly the same. Subsequent logins keep all tokens valid (new harness check: login 3 keeps tokens of login 2 and 3).
  implication: not a defect of the fix. A one-time sign-out of other devices for pre-fix accounts when the opt-in is first enabled; documented in auth-providers.md. Harness expectation corrected (trust column "false" for that single check) and a "no further rotation" check added; final runs: trust 90/0, default 88/0.

- timestamp: 2026-10-14
  checked: mutation check of the new route-level Go test (db/hooks/users_oauth2_flow_test.go)
  found: temporarily reverting the handler to the PR-head hand-over (email via CreateData instead of OAuth2User.Email) makes three subtests fail (account not created verified, existing account email not filled, legacy account not healed); restored afterwards, all pass. `go test ./...` in db passes (GOTOOLCHAIN=go1.26.1).
  implication: the recurrence guard actually reaches PocketBase's oauth2Submit and would have caught the original bug.

- timestamp: 2026-10-14
  checked: PocketBase `FindAuthRecordByEmail` and the users email index
  found: the lookup is case-insensitive only if the unique email index has `COLLATE NOCASE`; wanderer's `__pb_users_auth__email_idx` does not, so addresses are compared exactly (also by PocketBase's own `Record.Email() == OAuth2User.Email` verified check).
  implication: the handler compares exactly as well (a claim `David@x` against a stored `david@x` is a different address, no link, no error). A case-insensitive test expectation I first wrote was wrong and was removed.

## Eliminated

- hypothesis: the unverified-account effects are an artefact of the reviewer's fake IdP, not of PocketBase/Wanderer code.
  evidence: reproduced with an independently written fake IdP and the real wanderer binary; the mechanism is in unmodified PocketBase 0.38 (`oauth2Submit` + `external_auth_model.go`), and the baseline binary run through the identical harness passes 85/85.
  timestamp: 2026-10-14

## Resolution

root_cause: Three contributing defects in PR head 5baff13cb (findings 1 and 2 are an AND-gate: the PR's behaviour AND PocketBase's unverified-record handling are both required).
  (1) `OAuth2EmailHandler` passes the raw email via `CreateData`, so PocketBase creates the account UNVERIFIED (`Record.Email() != OAuth2User.Email`). PocketBase then treats every later unauthenticated OAuth2 login of an unverified record as a possible pre-hijack: random password (tokenKey refresh, other sessions invalid) and external auth rebuild; and when the email is later confirmed, the unverified->verified hook deletes all external auths, so the next login (no email to match) creates a new empty account.
  (2) The email handler ignores the `email_verified` claim (acts on explicit false, null and malformed values) and applies to every provider (google, oidc2, ...) with no opt-in; it also skips existing accounts, so pre-fix verified-no-email accounts are never healed.
  (3) The `OAuth2User.Name` username fallback is not restricted to the oidc slots, so Google, Microsoft, Apple, ... get real full names as usernames (and therefore as federated handles).
fix: Commit da4c45285 on fix/oidc-unverified-email-claims (worktree /Users/christianbeutel/Documents/svelte/wanderer-oidc, local only, not pushed), on top of PR head 5baff13cb.
  - `OAuth2EmailHandler(trusted)`: for opted-in slots only, absent `email_verified` only (present with any value, null included, is untouched), trimmed claim must be a valid address, sets `e.OAuth2User.Email` (not CreateData) so PocketBase creates the account verified; applies to new AND existing accounts (IsNewRecord gate dropped); an existing account that holds a different address is untouched; an address used by another account (self excluded) is not assigned (Warn log without the address), any other lookup error is returned; the client-submitted createData.email precedence is kept for new accounts.
  - `OAuth2UsernameHandler(trusted)`: the `OAuth2User.Name` fallback only for opted-in slots; the sanitising of provider usernames from before the PR is unchanged for every provider.
  - db/main.go: `OIDC_TRUST_EMAIL` / `OIDC2_TRUST_EMAIL` / `OIDC3_TRUST_EMAIL` (strconv.ParseBool, invalid values logged and ignored, default off), `oidcTrustedProviders()` handed to both hooks; the built-in providers cannot be enabled.
  - Docs: environment-configuration.md (3 rows) and auth-providers.md (new section "Providers that do not send `email_verified`"). No changelog entry (decision).
  Deviation from the plan: the opt-in also gates the username fallback (decision 2). The "also apply to existing accounts" step additionally needed "existing account with a different address is untouched" so the handler cannot rewrite a deliberately changed email.
verification:
  - Unit tests: `GOTOOLCHAIN=go1.26.1 go test ./hooks/ -run OAuth2` pass (username handler 17 scenarios, email handler 25 scenarios incl. absent/false/"true"/"false"/null/"maybe", other slot, google, taken address, self-exclusion, existing accounts, createData precedence); `TestOIDCTrustedProviders` in db/main_test.go; full `go test ./...` in db passes.
  - Route-level Go test `TestOAuth2OIDCLoginFlow` (db/hooks/users_oauth2_flow_test.go, real auth-with-oauth2 + auth-refresh through PocketBase's router against an httptest IdP): 9 flows; fails on the PR-head hand-over (mutation check), passes with the fix.
  - End-to-end harness, binary built from the fixed worktree: `-expect trust -env OIDC_TRUST_EMAIL=true` 90 passed / 0 failed; `-expect default` 88 passed / 0 failed. Both include: s1 (verified, email, name username, three logins keep every token, confirm keeps 1 external auth, login 3 same account), s3 (false/null/"maybe"/"false" untouched), s4 (email filled; older token rotated once by PocketBase core, no further rotation), s5 (taken address: no linking, no unique error), s6 (legacy unverified account healed), s7 (google unchanged), s8 (oidc2 not opted in unchanged).
  - Not covered: real Cloudflare Access (claims reproduced exactly, userinfo path); other PocketBase versions than 0.38.0.
files_changed:
  - db/hooks/users.go
  - db/hooks/users_test.go
  - db/hooks/users_oauth2_flow_test.go (new)
  - db/main.go
  - db/main_test.go
  - docs/src/content/docs/run/backend-configuration/auth-providers.md
  - docs/src/content/docs/run/environment-configuration.md
  (all in the worktree /Users/christianbeutel/Documents/svelte/wanderer-oidc; paths relative to the repo root)
prevention: why not caught: the existing hook tests built the OAuth2 event by hand and so never ran PocketBase's oauth2Submit, where the verified/unverified handling lives; guard: db/hooks/users_oauth2_flow_test.go drives the real auth-with-oauth2 route (login, login, confirm, login).
