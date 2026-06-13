# Requirements: Wanderer Trail Navigation

**Defined:** 2026-06-12
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1 Requirements

### Navigation Screen

- [ ] **NAV-01**: User can launch navigation from the trail detail screen
- [ ] **NAV-02**: User can launch navigation from the trail detail map screen
- [ ] **NAV-03**: Navigation screen shows a full-screen map centered on user's current GPS position
- [ ] **NAV-04**: Navigation screen displays the current Valhalla maneuver instruction at the top of the screen
- [ ] **NAV-05**: Map has a toggle button to switch between north-up and heading-up orientation
- [ ] **NAV-06**: Navigation advances through maneuvers automatically as the user moves along the trail
- [ ] **NAV-07**: User can exit navigation and return to the originating trail screen
- [ ] **NAV-08**: A red polyline trace is drawn on the map showing the actual path the user has walked/biked during the current navigation session

### Stats Sheet

- [x] **STATS-01**: Bottom DraggableScrollableSheet shows paginated stats (horizontal swipe between pages)
- [x] **STATS-02**: Page 1 — distance remaining to trail end
- [x] **STATS-03**: Page 2 — distance covered and ETA
- [x] **STATS-04**: Page 3 — cumulative elevation gain and loss so far
- [x] **STATS-05**: Page 4 — current GPS speed and average speed since navigation start

### Backend API

- [x] **API-01**: New POST /api/v1/valhalla/navigate endpoint accepts a trail (GPX or waypoint array) and returns a structured maneuver list
- [x] **API-02**: Each maneuver object includes instruction text, distance to next turn, and bearing

## v2 Requirements

### Audio

- **AUDIO-01**: Navigation screen reads maneuver instructions aloud via text-to-speech
- **AUDIO-02**: User can toggle TTS on/off with a mute/unmute button during navigation

### Offline

- **OFFLINE-01**: Valhalla navigation instructions are pre-fetched and cached when a trail is downloaded for offline use
- **OFFLINE-02**: Offline navigation falls back to cached instructions when no network is available

## Out of Scope

| Feature | Reason |
|---------|--------|
| Route from user position to trailhead | Assume user is already at trail start for v1 |
| Re-routing when off-trail | Complex and out of scope for initial nav core |
| Text-to-speech in v1 | Adds Flutter TTS package + audio session complexity — deferred to v2 |
| Offline nav in v1 | Requires download workflow changes — separate future phase |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| API-01 | Phase 1 | Complete |
| API-02 | Phase 1 | Complete |
| NAV-01 | Phase 2 | Pending |
| NAV-02 | Phase 2 | Pending |
| NAV-03 | Phase 2 | Pending |
| NAV-04 | Phase 2 | Pending |
| NAV-05 | Phase 2 | Pending |
| NAV-06 | Phase 2 | Pending |
| NAV-07 | Phase 2 | Pending |
| NAV-08 | Phase 2 | Pending |
| STATS-01 | Phase 3 | Complete |
| STATS-02 | Phase 3 | Complete |
| STATS-03 | Phase 3 | Complete |
| STATS-04 | Phase 3 | Complete |
| STATS-05 | Phase 3 | Complete |

**Coverage:**

- v1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-12*
*Last updated: 2026-06-12 after roadmap creation — phase assignments corrected*
