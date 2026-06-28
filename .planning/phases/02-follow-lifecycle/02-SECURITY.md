---
phase: 02-follow-lifecycle
audited_at: 2026-06-25
auditor: gsd-security-auditor (claude-sonnet-4-6)
asvs_level: 1
threats_total: 16
threats_closed: 16
threats_open: 0
verdict: SECURED
---

# Security Audit — Phase 02: Follow Lifecycle

## Summary

All 16 threats in the Phase 02 register are CLOSED. Three accepted risks require no code
verification. Thirteen mitigate-disposition threats each have confirmed implementation
evidence in the cited files. No unregistered threat flags were identified.

## Threat Verification

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-02-01 | Tampering | mitigate | CLOSED | `db/migrations/1782290001_add_rejected_to_follows_status.go:27-32` — idempotent guard iterates existing values and skips append if `"rejected"` already present; down-migration rebuilds filtered slice without `"rejected"` |
| T-02-02 | Denial of Service | accept | CLOSED | Auto-closed per accepted risk: PocketBase startup aborts on migration failure (fail-closed) |
| T-02-SC | Tampering | mitigate | CLOSED | No new Go packages or npm packages introduced across all four plans; all imports (`pocketbase/core`, `pocketbase/migrations`, `go-ap/activitypub`, `pocketbase/dbx`, `pocketbase/tools/security`) confirmed already in go.mod; TypeScript imports (`activitypub-types`, `@sveltejs/kit`) already in package.json per prior phase |
| T-02-03 | Spoofing | mitigate | CLOSED | `db/federation/instance.go:161-164` — `util.VerifySignature(e.App, e.Request, actor.GetString("public_key"))` called before any activity processing; `err != nil \|\| !verified` returns `e.UnauthorizedError("Invalid http signature", err)` |
| T-02-04 | Tampering | mitigate | CLOSED | `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts:44` — `originalHeaders['X-Forwarded-Path'] = event.url.pathname` set by SvelteKit layer from `event.url.pathname`, not from any client-supplied header; `db/federation/instance.go:136` — Go handler reads `e.Request.Header.Get("X-Forwarded-Path")` from the internal forwarded request only |
| T-02-05 | Elevation of Privilege | mitigate | CLOSED | `db/federation/follow.go:109-131` — branch `if object.GetString("actor_type") == "instance" && object.GetBool("is_local")` sets `followRecord.Set("status", "pending")` and returns early before Accept delivery; filters on the trusted local object record, not on remote actor |
| T-02-06 | Information Disclosure | accept | CLOSED | Auto-closed per accepted risk: inbox returns 200 with processor error embedded; no private actor fields exposed |
| T-02-07 | Repudiation | mitigate | CLOSED | `db/federation/follow.go:116-129` — inside the instance-actor branch, `activitypub_activities` record created with `iri`, `type=FollowType`, `actor=actor.GetString("iri")` (remote sender), `object=object.GetString("iri")` (instance actor IRI), `published=time.Now()` before early return |
| T-02-08 | Elevation of Privilege | mitigate | CLOSED | `db/hooks/follow.go:16,34,174` — `isInstanceFollow(e.App, e.Record)` called as first statement in `CreateFollowHandler`, `DeleteFollowHandler`, and `InstanceFollowDeleteHandler`; `db/hooks/follow.go:146` — `InstanceFollowUpdateHandler` checks `!isInstanceFollow \|\| isOutboundInstanceFollow`; `db/hooks/follow.go:101` — `InstanceFollowCreateHandler` checks `!isOutboundInstanceFollow`; all return `e.Next()` early for non-instance follows |
| T-02-09 | Spoofing | mitigate | CLOSED | `db/federation/follow.go:251` (`CreateAcceptFollowActivity`) and `:314` (`CreateRejectFollowActivity`) — both call `PostActivity(app, followeeActor, ...)` signing as the local instance actor (`followeeActor`); activities constructed from locally-stored `activitypub_activities` Follow record, not from attacker-controlled input (`FindFirstRecordByFilter` with `actor`, `object`, `type` params) |
| T-02-10 | Repudiation | mitigate | CLOSED | `db/federation/follow.go:241-244` (`CreateAcceptFollowActivity`) and `:307` (`CreateRejectFollowActivity`) — both persist an `activitypub_activities` record with `type=AcceptType`/`RejectType`, `actor=followeeActor.GetString("iri")`, `object=followActivity`, `published=time.Now()` before returning; `CreateFollowActivity` at `:57` and `CreateUnfollowActivity` (undo.go) similarly persist outgoing activities |
| T-02-11 | Denial of Service | accept | CLOSED | Auto-closed per accepted risk: PostActivity is fire-and-forget goroutine; slow/unreachable remote does not block admin save |
| T-02-12 | Tampering | mitigate | CLOSED | `db/hooks/follow.go:159-162` — `case "reject":` branch in `InstanceFollowUpdateHandler` calls `federation.CreateRejectFollowActivity(e.App, e.Record)`; `instanceFollowAction("pending", "rejected")` returns `"reject"` (verified at `:85-87`); `db/migrations/1782290001_add_rejected_to_follows_status.go:33` — schema prerequisite `"rejected"` appended to select values |
| T-02-13 | Denial of Service | mitigate | CLOSED | `db/federation/follow.go:324-327` — `followActivity, ok := activity.Object.(*pub.Activity)` with immediate `if !ok { return fmt.Errorf("ProcessAcceptActivity: object is not *pub.Activity, got %T", activity.Object) }` as first statement; returns error before any DB call, no panic possible |
| T-02-14 | Spoofing / Elevation of Privilege | mitigate | CLOSED | `db/hooks/follow.go:48-55` — `isOutboundInstanceFollow` loads only the `follower` actor and compares `followerActor.GetString("iri") == instanceIRI`; `db/hooks/follow.go:101` — `InstanceFollowCreateHandler` opens with `if !isOutboundInstanceFollow(e.App, e.Record) { return e.Next() }`, preventing execution for inbound follows where instance is followee |
| T-02-15 | Repudiation | mitigate | CLOSED | `db/federation/instance.go:175,179,183` — each processor case returns `e.BadRequestError(...)` on non-nil error; `db/federation/instance.go:186` — `default:` case returns `e.BadRequestError("Unsupported activity type", nil)`; `db/federation/instance.go:189` — success path returns `e.JSON(http.StatusOK, map[string]bool{"success": true})` (non-nil body); `var procErr error` and `e.JSON(http.StatusOK, procErr)` pattern confirmed absent |
| T-02-16 | Tampering | mitigate | CLOSED | `db/hooks/follow.go:16` — `CreateFollowHandler` checks `if isInstanceFollow(e.App, e.Record) { return e.Next() }` before `federation.CreateFollowActivity`; `db/hooks/follow.go:34` — `DeleteFollowHandler` checks `if isInstanceFollow(e.App, e.Record) { return e.Next() }` before `federation.CreateUnfollowActivity`; AfterSuccess handlers are the single delivery path for instance follows |

## Unregistered Flags

None. SUMMARY.md `## Threat Flags` sections across all four plans do not identify any new
attack surface that lacks a threat register mapping. The SUMMARY.md files for plans 02-02,
02-03, and 02-04 reference only the registered threat IDs (T-02-03 through T-02-16).

## Accepted Risks Log

| Threat ID | Category | Risk | Rationale |
|-----------|----------|------|-----------|
| T-02-02 | Denial of Service | PocketBase aborts startup on migration failure | Fail-closed behavior; migration errors are caught at deploy time, not request time |
| T-02-06 | Information Disclosure | Inbox returns 200 with processor error embedded | Consistent with existing user-inbox contract (ActivitypubActivityProcess); no private actor fields exposed |
| T-02-11 | Denial of Service | PostActivity goroutine fire-and-forget; unreachable remote drops activity | Online-only constraint by design; matches existing user federation behavior |

## Files Audited

- `db/migrations/1782290001_add_rejected_to_follows_status.go`
- `db/hooks/follow.go`
- `db/hooks/follow_test.go`
- `db/federation/follow.go`
- `db/federation/instance.go`
- `db/federation/instance_inbox_test.go`
- `web/src/routes/api/v1/activitypub/instance/inbox/+server.ts`
- `db/main.go` (hook registrations and route registration)
