# Research Conflicts: Federation Connect UI (v1.1)

**Identified:** 2026-06-27
**Status:** Unresolved -- requires decision in requirements step

---

## Conflict 1: Admin Auth Approach

**Type:** Competing variant (two valid options, different tradeoffs)
**Files in conflict:** ARCHITECTURE.md (recommends Variant A) vs FEATURES.md (recommends Variant B)
**Blocking:** Phase 3 (Admin UI) planning only. Does NOT block Phase 1 (Go API Layer) or Phase 2 (Peer List Query).

---

### Variant A: Custom Go Route + PocketBase Superuser JWT

**Source:** ARCHITECTURE.md

**How it works:**
- All six /federation/* handlers protected by e.HasSuperuserAuth()
- Admin authenticates via the PocketBase admin panel (/_/) to obtain a superuser JWT
- UI is a minimal embedded HTML page (go:embed) or a pure JSON API for curl/Postman
- No SvelteKit changes, no migrations, no new collections

**Auth mechanism:**
  e.HasSuperuserAuth() -- already used in db/routes/plugin_system.go and db/hooks/plugin_instances.go.
  This is a proven, zero-uncertainty pattern in this codebase.

**Implementation cost:**
- db/routes/federation_admin.go: six handlers (all with e.HasSuperuserAuth() guard)
- db/main.go: register six routes
- Optional: one embedded HTML file for browser UI
- Total: ~300-400 lines of Go + optional minimal HTML

**What you do NOT need:**
- No PocketBase migration
- No changes to users collection
- No SvelteKit route or page
- No changes to PocketBase collection API rules

**Advantages:**
- Zero new surface area -- superuser is the natural infrastructure operator credential
- No privilege escalation risk -- superuser auth is the hardest credential in the system
- Consistent with CLAUDE.md constraint: no SvelteKit or Flutter changes for v1 federation logic
- Deployment is a single Go binary; no frontend build step for the admin page
- Admin auth is stateless JWT (same as PocketBase admin panel) -- well-understood

**Disadvantages:**
- Admin must have PocketBase superuser credentials. In many self-hosted deployments, the Wanderer instance admin and the PocketBase DBA are the same person. In managed or shared deployments, they may not be.
- Admin federation dashboard is a separate URL/session from the regular Wanderer web UI. An admin managing federation must use /_/ or the federation-specific page, not their regular Wanderer account.
- If the admin page is HTML-embedded, it is visually distinct from the rest of the Wanderer UI (no shared design system, no SvelteKit navigation)

---

### Variant B: SvelteKit Settings Page + is_admin User Flag

**Source:** FEATURES.md

**How it works:**
- Add is_admin: bool field to the users PocketBase collection via a migration
- New SvelteKit page at /settings/federation -- consistent with existing settings pages (plugins, maintenance, privacy, account)
- +page.server.ts load function checks locals.user.is_admin; returns 403 if false
- SvelteKit API routes at /api/v1/federation/* proxy to the Go backend endpoints (or call PocketBase directly)
- Go endpoints accept the user auth token and verify is_admin via e.Auth.GetBool("is_admin")

**Auth mechanism:**
  Regular Wanderer user session (cookie-based, managed by PocketBase) + is_admin flag check.
  In SvelteKit: locals.user.is_admin.
  In Go handler: e.Auth.GetBool("is_admin").

**Implementation cost:**
- 1 PocketBase migration: add is_admin bool to users collection
- 1 new SvelteKit page: web/src/routes/settings/federation/+page.svelte
- 1 new SvelteKit server page: web/src/routes/settings/federation/+page.server.ts
- Type change: add is_admin?: boolean to User type in web/src/lib/models/user.ts
- PocketBase collection rules change on follows: allow admin users to PATCH instance-actor follow records
- Go route guards: replace e.HasSuperuserAuth() with is_admin check (or keep superuser for Go routes and add a separate SvelteKit-level guard)
- Total: ~150 lines Go + ~200 lines SvelteKit + migration + rule change

**Advantages:**
- Consistent with the existing Wanderer settings page pattern -- same navigation, same design system, same session
- Admin is a Wanderer user, not a PocketBase DBA -- appropriate separation of concerns for managed hosting
- No separate login flow for federation management; admin uses their existing Wanderer account
- Easier to extend: can add admin-only features elsewhere in the SvelteKit app using the same flag

**Disadvantages:**
- More moving parts: migration, type change, SvelteKit page, collection rule change, dual-layer auth guard
- is_admin must be set manually in the PocketBase admin panel or via a first-run promotion flow -- no built-in mechanism
- Conflation risk: Wanderer admins (is_admin) and PocketBase superusers (_superusers) are now two distinct admin concepts. Documentation and onboarding must distinguish them clearly.
- ARCHITECTURE.md notes: PocketBase superusers have no users record; they cannot authenticate to SvelteKit via locals.pb as a regular user. A PocketBase superuser who is also the Wanderer operator must create a separate Wanderer user account and grant themselves is_admin -- two accounts for one person.
- Collection API rules for follows currently restrict PATCH to the record owner. A rule change is required to allow admin users to approve/reject instance-actor follow records. This is a non-trivial rule change that affects security boundaries.

---

### Decision Criteria

The right choice depends on deployment context and operator model:

| Question | If answer is... | Choose |
|----------|----------------|--------|
| Who manages federation? | Always the PocketBase DBA / server operator | Variant A |
| Who manages federation? | A designated Wanderer user (could be different from DBA) | Variant B |
| Is single-operator self-hosting the primary deployment target? | Yes | Variant A |
| Is managed hosting or multi-admin a target for v1.1? | Yes | Variant B |
| Is CLAUDE.md constraint (no SvelteKit changes for federation v1) still active? | Yes | Variant A |
| Has CLAUDE.md constraint been relaxed for v1.1? | Yes | Either |
| How important is UI consistency with the rest of Wanderer settings? | High priority | Variant B |
| How important is minimising implementation surface area? | High priority | Variant A |

**ARCHITECTURE.md final position:** Variant A. The CLAUDE.md constraint (no SvelteKit/Flutter changes for v1 federation logic) is explicitly cited. The superuser is the natural operator for infrastructure-level settings. The implementation is proven and zero-risk.

**FEATURES.md final position:** Variant B. The settings page pattern is established and consistent. Wanderer admin != PocketBase superuser is the right model. The migration is simple.

**Synthesis recommendation:** Resolve in requirements. If the CLAUDE.md v1 constraint applies to v1.1, Variant A is mandatory. If the constraint is relaxed for v1.1 (the UI layer does not change the protocol), either variant is valid and the choice is a product decision about operator model.

---

*Conflict identified by: GSD research synthesizer*
*Date: 2026-06-27*
*Resolution required before: Phase 3 planning*
