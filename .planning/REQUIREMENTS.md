# Requirements: Wanderer Trail Navigation

**Defined:** 2026-06-12
**Milestone:** v1.2 Settings Screens
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1.2 Requirements

Requirements for the Settings Screens milestone. Each maps to roadmap phases.

### Account & Profile

- [x] **ACCT-01**: User can view and update their avatar from the Account settings screen
- [x] **ACCT-02**: User can view and edit their bio from the Account settings screen
- [x] **ACCT-03**: User can change their email address from the Account settings screen
- [x] **ACCT-04**: User can change their password from the Account settings screen
- [x] **ACCT-05**: User can delete their account from the Account settings screen with a confirmation step

### Privacy

- [x] **PRIV-01**: User can set their account visibility to public or private
- [x] **PRIV-02**: User can set their trails default visibility to public or private
- [x] **PRIV-03**: User can set their lists default visibility to public or private

### Language & Units

- [x] **LANG-01**: User can select their preferred language from the 14 supported locales
- [x] **LANG-02**: User can toggle between metric and imperial units

### Notifications

- [x] **NOTIF-01**: User can toggle web and email notifications for trail comments
- [x] **NOTIF-02**: User can toggle web and email notifications for new followers
- [x] **NOTIF-03**: User can toggle web and email notifications for trail shares
- [x] **NOTIF-04**: User can toggle web and email notifications for trail likes
- [x] **NOTIF-05**: User can toggle web and email notifications for list shares
- [x] **NOTIF-06**: User can toggle web and email notifications for summit log creates
- [x] **NOTIF-07**: User can toggle web and email notifications for trail mentions
- [x] **NOTIF-08**: User can toggle web and email notifications for comment mentions
- [x] **NOTIF-09**: User can toggle web and email notifications for summit log mentions

### Settings Navigation

- [x] **SETNAV-01**: Settings screen lists entries for Account, Privacy, Language, Notifications, and Appearance

## Future Requirements

### Account

- **ACCT-F01**: API token management (generate, view, delete tokens) — deferred (mobile use case unclear)

## Out of Scope

| Feature | Reason |
|---------|--------|
| API token management | Mobile clients don't need API tokens; web-only feature |
| Favourite sport picker | Being removed from web; not porting to mobile |
| Export settings | Data export is a desktop workflow |
| Integrations (Strava, Komoot) | Complex OAuth flows; separate milestone |
| Maintenance screens | Admin/developer tooling, not user-facing |
| Map settings | Not yet needed on mobile |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SETNAV-01 | Phase 6 | Complete |
| LANG-01 | Phase 6 | Complete |
| LANG-02 | Phase 6 | Complete |
| PRIV-01 | Phase 7 | Complete |
| PRIV-02 | Phase 7 | Complete |
| PRIV-03 | Phase 7 | Complete |
| ACCT-01 | Phase 8 | Complete |
| ACCT-02 | Phase 8 | Complete |
| ACCT-03 | Phase 8 | Complete |
| ACCT-04 | Phase 8 | Complete |
| ACCT-05 | Phase 8 | Complete |
| NOTIF-01 | Phase 9 | Complete |
| NOTIF-02 | Phase 9 | Complete |
| NOTIF-03 | Phase 9 | Complete |
| NOTIF-04 | Phase 9 | Complete |
| NOTIF-05 | Phase 9 | Complete |
| NOTIF-06 | Phase 9 | Complete |
| NOTIF-07 | Phase 9 | Complete |
| NOTIF-08 | Phase 9 | Complete |
| NOTIF-09 | Phase 9 | Complete |

**Coverage:**

- v1.2 requirements: 20 total
- Mapped to phases: 20
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-12*
*Last updated: 2026-06-19 — milestone v1.2 roadmap created; all 20 requirements mapped to Phases 6-9*
