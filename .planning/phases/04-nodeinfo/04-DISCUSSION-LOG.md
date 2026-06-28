# Phase 4: NodeInfo - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-26
**Phase:** 04-nodeinfo
**Areas discussed:** Version string, Post count definition, User count scope

---

## Version String

| Option | Description | Selected |
|--------|-------------|----------|
| Hardcoded constant | `const wandererVersion = "1.0.0"` in nodeinfo.go, updated manually | |
| WANDERER_VERSION env var | `os.Getenv("WANDERER_VERSION")`, fallback `"dev"` | ✓ |
| Go ldflags build variable | Injected via `go build -ldflags` at build time | |

**User's choice:** WANDERER_VERSION env var  
**Notes:** User noted the version is defined in `web/package.json` and wants a single source of truth for both frontend and backend. CI/Docker reads `package.json` and injects `WANDERER_VERSION` at build time.

---

## Post Count Definition

| Option | Description | Selected |
|--------|-------------|----------|
| Trails only | COUNT of `trails` collection | |
| All four content types | SUM of trails + comments + summit_logs + lists | |
| Public trails only | COUNT of `trails` WHERE `public = true` | ✓ |

**User's choice:** Public trails only  
**Notes:** Private trails never leave the instance, so including them in a federation-facing count would be misleading. Public trails are the authoritative "federated posts" count.

---

## User Count Scope

| Option | Description | Selected |
|--------|-------------|----------|
| All rows in `users` collection | COUNT of PocketBase `users` collection | ✓ |
| You decide | Claude picks based on schema | |

**User's choice:** All rows in `users` collection  
**Notes:** The instance actor lives in `activitypub_actors`, not `users`, so no filtering is needed — every row in `users` is a real human account.

---

## Claude's Discretion

- Response content type and caching headers
- Whether to use one handler file or two for the two endpoints
- Whether to include `users.activeMonth` / `users.activeHalfyear` fields (optional NodeInfo fields; likely omit or set to 0)

## Deferred Ideas

- WebFinger extension for instance actor IRI resolution (v2 DISC-01 in REQUIREMENTS.md)
- Active user count fields (`users.activeMonth`, `users.activeHalfyear`) — optional NodeInfo fields, not discussed
