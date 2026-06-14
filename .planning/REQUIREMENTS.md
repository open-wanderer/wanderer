# Requirements: Wanderer Trail Navigation

**Defined:** 2026-06-12
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1.0 Requirements (Complete)

### Navigation Screen

- [x] **NAV-01**: User can launch navigation from the trail detail screen
- [x] **NAV-02**: User can launch navigation from the trail detail map screen
- [x] **NAV-03**: Navigation screen shows a full-screen map centered on user's current GPS position
- [x] **NAV-04**: Navigation screen displays the current Valhalla maneuver instruction at the top of the screen
- [x] **NAV-05**: Map has a toggle button to switch between north-up and heading-up orientation
- [x] **NAV-06**: Navigation advances through maneuvers automatically as the user moves along the trail
- [x] **NAV-07**: User can exit navigation and return to the originating trail screen
- [x] **NAV-08**: A red polyline trace is drawn on the map showing the actual path the user has walked/biked during the current navigation session

### Stats Sheet

- [x] **STATS-01**: Bottom DraggableScrollableSheet shows live stats in a collapsed/expanded state
- [x] **STATS-02**: Collapsed sheet shows time elapsed, distance covered, and elevation gain
- [x] **STATS-03**: Expanded sheet adds elevation loss, current speed, and average speed
- [x] **STATS-04**: Button switches sheet content to the reused trail elevation-profile chart and back
- [x] **STATS-05**: Button row provides elevation profile, pause/resume, and exit controls

### Backend API

- [x] **API-01**: New POST /api/v1/valhalla/navigate endpoint accepts a trail (GPX or waypoint array) and returns a structured maneuver list
- [x] **API-02**: Each maneuver object includes instruction text, distance to next turn, and bearing

## v1.1 Requirements

### Offline Navigation

- [ ] **OFFLINE-01**: Trail download also caches Valhalla navigation instructions to ObjectBox so navigation is available without a network connection
- [ ] **OFFLINE-02**: `launchNavigation` falls back to cached instructions when the network request fails (DioException catch → ObjectBox read)
- [ ] **OFFLINE-03**: After a successful online navigation fetch, the local cache is silently updated in ObjectBox
- [ ] **OFFLINE-04**: NavigationScreen shows an offline indicator icon in the AppBar when operating from cached instructions

> **Infrastructure prerequisite:** `NavigateResponse.toJson()` must include `@JsonSerializable(explicitToJson: true)` — the generated code currently does not call `.toJson()` on nested `NavigateManeuver` elements, causing `jsonEncode` to throw at runtime. Fix is required before any ObjectBox caching can work; verified with a roundtrip unit test.

## v2 Requirements

### Audio

- **AUDIO-01**: Navigation screen reads maneuver instructions aloud via text-to-speech
- **AUDIO-02**: User can toggle TTS on/off with a mute/unmute button during navigation

## Out of Scope

| Feature | Reason |
|---------|--------|
| Route from user position to trailhead | Assume user is already at trail start |
| Re-routing when off-trail | Complex; separate future phase |
| Text-to-speech in v1 | Adds Flutter TTS package + audio session complexity — deferred to v2 |
| Stale cache warnings | Trail maneuvers are low-volatility; silent re-cache on next online session is sufficient |
| User-initiated cache refresh | Anti-feature; caching should be invisible |
| Offline re-routing | Out of scope entirely |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| API-01 | Phase 1 (v1.0) | Complete |
| API-02 | Phase 1 (v1.0) | Complete |
| NAV-01–NAV-08 | Phase 2 (v1.0) | Complete |
| STATS-01–STATS-05 | Phase 3 (v1.0) | Complete |
| OFFLINE-01 | Phase 4 (v1.1) | Pending |
| OFFLINE-02 | Phase 4 (v1.1) | Pending |
| OFFLINE-03 | Phase 4 (v1.1) | Pending |
| OFFLINE-04 | Phase 4 (v1.1) | Pending |

**Coverage:**

- v1.1 requirements: 4 total
- Mapped to phases: 4
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-12*
*Last updated: 2026-06-14 — v1.1 requirements added (OFFLINE-01 through OFFLINE-04)*
