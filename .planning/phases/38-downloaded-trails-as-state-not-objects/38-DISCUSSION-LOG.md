# Phase 38: Downloaded Trails as State, Not Objects - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-04
**Phase:** 38-downloaded-trails-as-state-not-objects
**Areas discussed:** Remove-download wording & confirm, Menu shape for Update / Remove, How Edit refuses without a server copy, Fate of the forceOffline flag

---

## Remove-download wording & confirm

### Q1 — What confirm does "Remove download" get?

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror the regions dialog | Title + body modelled on `regions_delete_confirm_*`: name what is removed, state re-download is needed, red confirm action. Already shipped for offline regions, translated in all 14 locales, avoids the false "cannot be undone" claim. | ✓ |
| No confirm — just remove it | Reversible with one tap, so a dialog is friction. Against: AllTrails confirms anyway, and removing while offline is not reversible until signal returns. | |
| Keep reusing `delete_trail_confirm` | Zero new strings. Against: that string says "cannot be undone", which is false for an un-download — already flagged wrong in a code comment. | |

**User's choice:** Mirror the regions dialog
**Notes:** Precedent surfaced during the discussion at `settings_offline_regions_screen.dart:1022-1041`; its body copy ("You'll need to download it again to use it offline") is the model.

### Q2 — Removing a download while offline can't be reversed until signal returns. Handle that?

| Option | Description | Selected |
|--------|-------------|----------|
| Body names the cost, same everywhere | One dialog whose body already states re-download is needed. No branching on connectivity. | ✓ |
| Extra warning line when offline | Adds a sentence when there's no connection. Most honest at the moment it matters, costs a second string and a connectivity check in the dialog. | |
| Refuse it while offline | Blocks removal without a connection. Safest against data loss, but paternalistic — freeing space is legitimate in the field. | |

**User's choice:** Body names the cost, same everywhere

### Q3 — New l10n keys, or compose from the translated ones?

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse `remove` + `available_offline` | Both exist and are translated in all 14 locales; only the confirm body is new copy. Minimises what lands English-only. | ✓ |
| Mint dedicated keys for everything | Clearest copy, but every new string ships English-only in 13 locales and the pending todo forbids machine-translating destructive copy. | |
| You decide | Claude's discretion per string. | |

**User's choice:** Reuse `remove` + `available_offline`
**Notes:** Also surfaced — `trail_dropdown.dart:179` hardcodes the English literal `'Available offline'` despite a translated key existing.

---

## Menu shape for Update / Remove

### Q1 — How do Update and Remove download appear in the overflow menu?

| Option | Description | Selected |
|--------|-------------|----------|
| Two flat items replacing the inert one | Update + Remove download as flat items. No sub-sheet pattern exists in the app; keeps every action one tap deep. | ✓ (after follow-up) |
| Keep one item, tapping opens a sheet | Shorter menu, but a new interaction pattern and both actions two taps deep. | |
| Split across menu and action bar | Update promoted to the bottom bar. Most discoverable, risks crowding Navigate. | |

**User's initial response:** *"Why do we need the 'update' action? I thought we update the local trail automatically on edit?"*

**Notes:** Legitimate challenge — the roadmap entry had conflated two cases. Clarified that automatic reconciliation covers only the hiker's own edit made on *this* device (response already in hand, no network), whereas Update covers changes this device did not make: the trail's author correcting the route, or the hiker editing on the web app or a second device. Also established that Update is nearly free to build because `TrailDownloadService` already overwrites in place, and that today's only refresh workaround (Remove → Download) has a window where the hiker holds neither copy.

### Q1b — Keep Update in scope? (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Keep it — two flat menu items | Covers author-side and cross-device changes; cheap given in-place overwrite; removes the neither-copy window. | ✓ |
| Keep it, but only as stale-copy recovery | Would require the staleness detection already rejected as unsupported by any researched app. | |
| Drop it — auto-reconcile is enough | Simplest menu. Accepts that an author-edited trail stays stale indefinitely with no affordance. | |

**User's choice:** Keep it — two flat menu items

### Q2 — What goes in the bottom action bar once downloaded?

| Option | Description | Selected |
|--------|-------------|----------|
| Nothing — Navigate keeps full width | Today's behaviour. Navigate is the primary action; Update stays a deliberate, less prominent choice. | ✓ |
| Download button becomes an Update button | Stable layout across states, but gives a maintenance action the same weight as Navigate. | |

**User's choice:** Nothing — Navigate keeps the full width

### Q3 — Fold the existing "Offline" pill's re-gating into this phase?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — re-gate on library membership | The pill at `trail_panel.dart:214` has the same `isLocal` defect; re-gate so badge and menu agree. | ✓ |
| No — leave the pill alone | Keeps the phase narrower, but the pill keeps flickering with network conditions and can contradict the menu. | |

**User's choice:** Yes — re-gate on library membership

---

## How Edit refuses without a server copy

### Q1 — How does Edit behave when the screen is showing a cached copy?

| Option | Description | Selected |
|--------|-------------|----------|
| Stay enabled; fetch the server copy on tap | Editor opens on success, toast on failure. Structurally kills the photo bug — the editor can never receive a cached model. Follows the existing `trail_uploaded_reopen_to_edit` toast precedent. | ✓ |
| Disable and grey it out | Matches "Show on map" and drain-in-flight Delete, but both are silent — exactly what the D-17 comment calls out as reading like a broken app. | |
| Hide it entirely | Matches D-17's choice, but Edit vanishing from your own trail reads as a permissions problem. | |

**User's choice:** Stay enabled; fetch the server copy on tap
**Notes:** Four competing precedents exist in `trail_dropdown.dart` — hide (D-17), disable-silently (×2), and toast-on-refusal. The toast precedent won.

### Q2 — The refusal message when the fetch fails

| Option | Description | Selected |
|--------|-------------|----------|
| Mint one new string modelled on `delete_needs_connection` | One new key, English-only in 13 locales exactly as its sibling already is. | ✓ |
| Reuse the generic offline copy | Zero i18n debt, but doesn't say the edit specifically needs the server copy. | |
| You decide | Claude's discretion. | |

**User's choice:** Mint one new string modelled on `delete_needs_connection`

---

## Fate of the forceOffline flag

### Q1 — What happens to forceOffline?

| Option | Description | Selected |
|--------|-------------|----------|
| Retire the flag, keep the behaviour in the provider | Disk-first whenever a library row exists, uniformly. One instance, no plumbing. | |
| Keep it as an explicit display-source flag | Today's behaviour on `feature/app`. Narrowest, but keeps the forwarding footgun and a per-navigation-path notion of identity. | |
| Retire it entirely — always prefer the server | Simplest model; disk only as fallback. Costs a round trip per library open. | |

**User's response:** *"We cannot allow a downloaded trail to be stale when we are online. However I understand your thought process. What do you recommend?"*

**Notes:** The constraint ruled out both disk-first variants. Recommendation given: retire entirely, on the reasoning that the flag's only job was making the Library's Delete deterministic — now handled by deriving destructive actions from library membership and authorship. The bandwidth principle was reconciled by reframing a download as existing for *offline availability, not online data-saving*: nothing re-downloads automatically, and both Download and Update stay explicit. Also recorded: the family-key footgun (three call sites needed threading in one sitting; any future one that forgets breaks quietly).

### Q1b — Confirm the forceOffline decision (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Retire it + opportunistic metadata refresh | Delete flag and plumbing; any successful fetch of a downloaded trail refreshes the stored row's metadata at zero network cost. | ✓ |
| Retire it, nothing more | Smallest phase; stored metadata can sit old until Update. | |
| Retire it, and auto-refresh GPX on change too | Safety argument for the track specifically; the only option that spends bytes unasked. | |

**User's choice:** Retire it + opportunistic metadata refresh

---

## Post-discussion correction — scope of the automatic refresh

Raised by the user after CONTEXT.md was first written: *"When we open a downloaded trail online, do
we not automatically download the GPX already?"*

**Correct, and it invalidated the reasoning behind D-14's original narrow scope.** Verified in code:
`TrailNotifier.build()` fetches the GPX file on every online open of every trail, and
`TrailDownloadService` never fetches the GPX at all — it takes an already-fetched model and pulls
only photos. So the fresh track is already in hand on every online view, and restricting the
opportunistic refresh to metadata was based on a false premise about where the bytes go.

The user's follow-up: *"At that point just refresh the whole thing including photos. Should we keep
the update button? I tend to say yes. It's essentially the same just without opening the trail."*

Photos were then priced explicitly, since they are the one asset **not** already in the response —
each is a separate file download, and refreshing them on every view would be the app's single most
expensive automatic behaviour.

### Q — How far does the automatic refresh go on an online view?

| Option | Description | Selected |
|--------|-------------|----------|
| Everything, photos by filename diff | Metadata + GPX free; photos compared against the server's filename list, fetching only genuine changes. Requires retaining server filenames, untangling Phase 36's D-10 `photos`/`localPhotos` overloading. | |
| Everything, re-fetch all photos every view | Simplest; an online view fully restores the download. Cost: every online open re-downloads every full-size photo automatically. | |
| Metadata + GPX only; photos on Update | Draws the line at free-vs-costly. Nothing automatic ever spends bytes. A trail whose author swapped photos shows old ones offline until Update. | ✓ |

**User's choice:** Metadata + GPX only; photos on Update
**Notes:** Update retained, with its purpose sharpened per the user's own framing — "the same refresh
without opening the trail". Captured as D-12a, D-14, D-14a and D-23. The filename-diff approach was
recorded as a deferred idea rather than discarded.

---

## Claude's Discretion

- Exact wording of the new confirm body and the new edit-refusal string, within the agreed shapes.
- Whether Update and Remove download sit under one divider or two, and their order.
- Whether the menu reads library membership via the existing `availableOffline` prop or directly.

## Deferred Ideas

- **"Go to source" menu item** — the user's own initial proposal, dropped once the single-object model removed the gap it bridged.
- **Staleness / "update available" indicator** — no researched app does this.
- **Automatic GPX re-fetch when the track changed** — rejected as the only option spending bytes unasked; revisit if stale tracks prove real.
- **Server-side cleanup of already-duplicated photos** — the fix stops new duplication but nothing prunes existing duplicates; needs its own decision about touching user data.
- **Interim mitigation before this phase lands** — pulling the Edit gate forward as a standalone fix was offered and not taken up.
