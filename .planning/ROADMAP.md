# Roadmap: Wanderer Instance Federation

## Milestones

- ✅ **v1.0 Instance Federation** — Phases 1–4 (shipped 2026-06-26)
- 🚧 **v1.1 Federation Connect UI** — Phases 5–6 (in progress)

## Phases

<details>
<summary>✅ v1.0 Instance Federation (Phases 1–4) — SHIPPED 2026-06-26</summary>

- [x] Phase 1: Instance Actor (1/1 plans) — completed 2026-06-25
- [x] Phase 2: Follow Lifecycle (4/4 plans) — completed 2026-06-25
- [x] Phase 3: Fanout and Safety (3/3 plans) — completed 2026-06-26
- [x] Phase 4: NodeInfo (1/1 plan) — completed 2026-06-26

Full archive: `.planning/milestones/v1.0-ROADMAP.md`

</details>

### 🚧 v1.1 Federation Connect UI (In Progress)

**Milestone Goal:** Give an admin a browser-accessible UI to manage peer instance connections — discover a remote instance, initiate/approve/reject follows, disconnect, and view all peers — without touching the PocketBase admin panel.

- [ ] **Phase 5: Federation Admin API** - Go route handlers for all peer management operations behind superuser auth guard
- [ ] **Phase 6: Admin Browser UI** - Embedded HTML page at `/federation/` for the connection dashboard and all admin actions

## Phase Details

### Phase 5: Federation Admin API
**Goal**: An admin can perform all peer management operations (discover, connect, approve, reject, disconnect, list) via a JSON API without touching the PocketBase admin panel
**Depends on**: Phase 4
**Requirements**: DISC-01, DISC-02, CONN-01, CONN-02, CONN-03, CONN-04, SAFE-05, SAFE-06, SAFE-07
**Success Criteria** (what must be TRUE):
  1. Admin can POST a remote instance URL to `/federation/discover` and receive a preview card (instance name, Wanderer version, user/trail counts) or a clear error if the URL is unreachable, not a Wanderer instance, already connected, or resolves to the local instance
  2. Admin can POST to `/federation/follow` and the local follows record is created with status "pending"; a Follow activity is delivered to the remote via the existing hook with no double-delivery
  3. Admin can POST to `/federation/approve/:id` and an Accept{Follow} is delivered; the connection moves to accepted status
  4. Admin can POST to `/federation/reject/:id` to reject an inbound follow (Reject{Follow} delivered) or `/federation/disconnect/:id` to undo an outbound follow (Undo{Follow} delivered) — direction-aware with no incorrect Undo on inbound-only connections
  5. All six endpoints reject requests lacking a valid PocketBase superuser token with 401; any admin-supplied URL is fetched through the SSRF-safe client with a 10-second timeout
**Plans**: TBD

### Phase 6: Admin Browser UI
**Goal**: Admin can manage all peer connections from a browser page at `/federation/` using their PocketBase superuser credentials — no curl or PocketBase admin panel required
**Depends on**: Phase 5
**Requirements**: DASH-01, DASH-02
**Success Criteria** (what must be TRUE):
  1. Navigating to `/federation/` in a browser shows a dashboard listing all peer connections with status (pending / accepted / rejected) and direction (Outbound / Inbound / Mutual)
  2. The dashboard provides inline forms for the full connection workflow: paste-URL discovery with preview, Connect, Approve, Reject, and Disconnect — all without leaving the page
  3. Unauthenticated requests to `/federation/` return 401; a valid Wanderer user token that is not a PocketBase superuser is also rejected
**Plans**: TBD
**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Instance Actor | v1.0 | 1/1 | Complete | 2026-06-25 |
| 2. Follow Lifecycle | v1.0 | 4/4 | Complete | 2026-06-25 |
| 3. Fanout and Safety | v1.0 | 3/3 | Complete | 2026-06-26 |
| 4. NodeInfo | v1.0 | 1/1 | Complete | 2026-06-26 |
| 5. Federation Admin API | v1.1 | 0/? | Not started | - |
| 6. Admin Browser UI | v1.1 | 0/? | Not started | - |
