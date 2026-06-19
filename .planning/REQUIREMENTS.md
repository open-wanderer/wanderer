# Requirements: Wanderer Trail Navigation

**Defined:** 2026-06-12
**Milestone:** v1.2 Settings Screens
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1.2 Requirements

Requirements for the Settings Screens milestone. Each maps to roadmap phases.

### Account & Profile

- [ ] **ACCT-01**: User can view and update their avatar from the Account settings screen
- [ ] **ACCT-02**: User can view and edit their bio from the Account settings screen
- [ ] **ACCT-03**: User can change their email address from the Account settings screen
- [ ] **ACCT-04**: User can change their password from the Account settings screen
- [ ] **ACCT-05**: User can delete their account from the Account settings screen with a confirmation step

### Privacy

- [ ] **PRIV-01**: User can set their account visibility to public or private
- [ ] **PRIV-02**: User can set their trails default visibility to public or private
- [ ] **PRIV-03**: User can set their lists default visibility to public or private

### Language & Units

- [ ] **LANG-01**: User can select their preferred language from the 14 supported locales
- [ ] **LANG-02**: User can toggle between metric and imperial units

### Notifications

- [ ] **NOTIF-01**: User can toggle web and email notifications for trail comments
- [ ] **NOTIF-02**: User can toggle web and email notifications for new followers
- [ ] **NOTIF-03**: User can toggle web and email notifications for trail shares
- [ ] **NOTIF-04**: User can toggle web and email notifications for trail likes
- [ ] **NOTIF-05**: User can toggle web and email notifications for list shares
- [ ] **NOTIF-06**: User can toggle web and email notifications for summit log creates
- [ ] **NOTIF-07**: User can toggle web and email notifications for trail mentions
- [ ] **NOTIF-08**: User can toggle web and email notifications for comment mentions
- [ ] **NOTIF-09**: User can toggle web and email notifications for summit log mentions

### Settings Navigation

- [ ] **SETNAV-01**: Settings screen lists entries for Account, Privacy, Language, Notifications, and Appearance

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
| SETNAV-01 | Phase 6 | Pending |
| LANG-01 | Phase 6 | Pending |
| LANG-02 | Phase 6 | Pending |
| PRIV-01 | Phase 7 | Pending |
| PRIV-02 | Phase 7 | Pending |
| PRIV-03 | Phase 7 | Pending |
| ACCT-01 | Phase 8 | Pending |
| ACCT-02 | Phase 8 | Pending |
| ACCT-03 | Phase 8 | Pending |
| ACCT-04 | Phase 8 | Pending |
| ACCT-05 | Phase 8 | Pending |
| NOTIF-01 | Phase 9 | Pending |
| NOTIF-02 | Phase 9 | Pending |
| NOTIF-03 | Phase 9 | Pending |
| NOTIF-04 | Phase 9 | Pending |
| NOTIF-05 | Phase 9 | Pending |
| NOTIF-06 | Phase 9 | Pending |
| NOTIF-07 | Phase 9 | Pending |
| NOTIF-08 | Phase 9 | Pending |
| NOTIF-09 | Phase 9 | Pending |

**Coverage:**
- v1.2 requirements: 20 total
- Mapped to phases: 20
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-12*
*Last updated: 2026-06-19 — milestone v1.2 roadmap created; all 20 requirements mapped to Phases 6-9*
