# Requirements: Wanderer Instance Federation

**Defined:** 2026-06-27
**Core Value:** An administrator can connect two Wanderer instances so that public content flows between them automatically, using the same ActivityPub machinery already powering user-level federation.

## v1.1 Requirements

### Discovery

- [ ] **DISC-01**: Admin can paste a remote instance URL and receive a preview card (instance name, Wanderer version, user/trail count) before any Follow is sent
- [ ] **DISC-02**: Discovery returns a clear error when the remote is unreachable, not a Wanderer instance, already connected, or resolves to the local instance

### Connection Management

- [ ] **CONN-01**: Admin can initiate an outgoing Follow from the UI; the connection appears as "Outbound / Pending" in the peer list without requiring PocketBase admin panel access
- [ ] **CONN-02**: Admin can approve an inbound pending Follow from the UI; an `Accept{Follow}` is delivered and the connection moves to "Inbound / Accepted"
- [ ] **CONN-03**: Admin can reject an inbound pending Follow from the UI; a `Reject{Follow}` is delivered and the connection is removed
- [ ] **CONN-04**: Admin can disconnect from any peer; outbound connections send `Undo{Follow}`; inbound-only connections send `Reject{Follow}` (not Undo) — direction-aware

### Dashboard

- [ ] **DASH-01**: Admin can view all peer connections with status (pending / accepted / rejected) and direction (Outbound / Inbound / Mutual) in a browser page at `/federation/`
- [ ] **DASH-02**: The page requires PocketBase superuser authentication — unauthenticated requests receive 401; regular Wanderer user tokens are rejected

### Safety

- [ ] **SAFE-05**: Discovery rejects URLs that resolve to the local instance actor (prevents self-follow loops)
- [ ] **SAFE-06**: All outbound HTTP to admin-supplied URLs uses a SSRF-safe client with a ≤10s timeout
- [ ] **SAFE-07**: API handlers only write DB records for follow lifecycle operations; ActivityPub delivery is not called directly — hooks remain the sole delivery path

## v2 Requirements

### Polish

- **UX-01**: Dashboard shows realtime status updates via PocketBase follows subscription (no manual refresh)
- **UX-02**: Refresh peer metadata button re-fetches NodeInfo and updates the preview data for a connected peer
- **UX-03**: Connection activity log showing timeline of Follow/Accept/Reject/Undo events per peer
- **UX-04**: Email notification to admin when an inbound pending Follow arrives

### Discovery

- **DISC-03**: WebFinger resolution for the instance actor (`/.well-known/webfinger`) — enables peers running authorized-fetch mode to discover the instance actor by `@instance@domain` handle

### Content Lifecycle

- **VIS-01**: When a trail's `is_public` flips from `true` to `false`, a `Delete` activity is sent to all peer instances so they remove their cached copy

## Out of Scope

| Feature | Reason |
|---------|--------|
| PocketBase UI extensions for the admin page | Maintainer-stated not production-safe; will break on next PocketBase upgrade (discussion #7612) |
| SvelteKit /settings/federation page | No admin user concept in frontend; PocketBase superusers have no Wanderer account; deferred to when admin role is defined |
| Domain blocklist | v2+ scope; low priority for initial admin UX |
| Auto-accept inbound follows | Mutual approval is a hard requirement from v1.0 trust model |
| Federation with non-Wanderer ActivityPub servers | Unknown content schema; DISC-01 preview step rejects non-Wanderer instances |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| DISC-01 | Phase 5 | Pending |
| DISC-02 | Phase 5 | Pending |
| CONN-01 | Phase 5 | Pending |
| CONN-02 | Phase 5 | Pending |
| CONN-03 | Phase 5 | Pending |
| CONN-04 | Phase 5 | Pending |
| DASH-01 | Phase 6 | Pending |
| DASH-02 | Phase 6 | Pending |
| SAFE-05 | Phase 5 | Pending |
| SAFE-06 | Phase 5 | Pending |
| SAFE-07 | Phase 5 | Pending |

**Coverage:**
- v1.1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-27*
*Last updated: 2026-06-27 after roadmap creation*
