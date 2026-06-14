# Phase 5: Cache Write + Fallback + UI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-14
**Phase:** 05-cache-write-fallback-ui
**Areas discussed:** isOffline propagation, Offline indicator design, Shape source for download cache

---

## isOffline Propagation

### How should isOffline be passed to NavigationScreen?

| Option | Description | Selected |
|--------|-------------|----------|
| New constructor param | Add `isOffline: bool` to NavigationScreen alongside response param. Minimal change, no new types. | ✓ |
| NavigationArgs wrapper class | Create a freezed class with response and isOffline fields. Cleaner if args list ever grows. | |
| You decide | Claude picks the approach. | |

**User's choice:** New constructor param

---

### How should the router extra be structured to carry both response and isOffline?

| Option | Description | Selected |
|--------|-------------|----------|
| Dart record — (NavigateResponse, bool) | `extra: (response, false)`. Router unpacks with record destructuring. No new type, Dart 3 idiomatic. | ✓ |
| Simple Map — {'response': ..., 'isOffline': ...} | Flexible but untyped — cast required in router. | |
| You decide | Claude picks based on existing router pattern. | |

**User's choice:** Dart record — (NavigateResponse, bool)

---

## Offline Indicator Design

### Where should the offline indicator appear in NavigationScreen?

| Option | Description | Selected |
|--------|-------------|----------|
| Inside the maneuver banner | Small wifi-off icon at trailing edge of existing `_buildBanner` widget. Always visible, no new layout layer. | ✓ |
| Separate positioned overlay | New Positioned widget in the Stack, e.g., top-left corner. More prominent, slightly more code. | |
| You decide | Claude picks the approach. | |

**User's choice:** Inside the maneuver banner

---

### What icon/widget for the offline indicator?

| Option | Description | Selected |
|--------|-------------|----------|
| FaIcon(FontAwesomeIcons.wifiSlash) | Matches FontAwesome icons already used throughout NavigationScreen. No new dependency. | ✓ |
| Icon(Icons.wifi_off) | Material icon. Less consistent with the rest of the screen that uses FontAwesome. | |
| You decide | Claude picks based on existing icon usage. | |

**User's choice:** FaIcon(FontAwesomeIcons.wifiSlash)

---

## Shape Source for Download Cache

### What shape source should downloadTrail use for the Valhalla cache call?

| Option | Description | Selected |
|--------|-------------|----------|
| trail.expand?.gpx only — skip silently if null | Mirrors exactly what launchNavigation does. Best-effort, consistent. | ✓ |
| gpx first, waypointsViaTrail as fallback | More resilient but adds a secondary code path that launchNavigation doesn't use. | |
| You decide | Claude picks based on consistency. | |

**User's choice:** trail.expand?.gpx only — skip silently if null

---

### Should downloadTrail reuse the same downsampling logic as launchNavigation?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — extract shared helper | Move downsampling into a shared function (e.g., in gpx_util.dart). Both paths use the same code. DRY. | ✓ |
| Yes — duplicate the logic | Inline same downsampling in downloadTrail. Simpler short-term but paths can drift. | |
| You decide | Claude picks the refactoring approach. | |

**User's choice:** Yes — extract shared helper

---

## Claude's Discretion

No areas deferred to Claude — user selected recommended options throughout.

## Deferred Ideas

None — discussion stayed within phase scope.
