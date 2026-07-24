---
status: complete
phase: 26-trail-download-guard
source: [26-VERIFICATION.md]
started: 2026-07-24T12:44:19Z
updated: 2026-07-24T13:10:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Fully-covered / updateAvailable trail starts immediately
expected: Tapping download on a trail whose overlapping regions are all downloaded/updateAvailable starts the download instantly with no sheet.
result: pass

### 2. Missing-coverage sheet appearance and dismiss-abort
expected: Sheet lists missing region(s) with name/size and Vector(checked)/DEM(unchecked) checkboxes; dismissing starts nothing.
result: pass

### 3. No-region-gap warning
expected: An info/warning toast appears for a trail outside every configured region, and the trail download still proceeds.
result: pass

### 4. Multi-region parallel download + unified notification
expected: Trail + all selected packages download in parallel; one id-42 notification shows a single combined, advancing progress bar (not one notification per download).
result: issue
reported: "The download bar resets 3 times to different values during download and finally finishes before 100%. The trail is successfully downloaded."
severity: major

### 5. Guard does not re-fire after a just-completed region download (CR-02 live regression check)
expected: Trigger the sheet, download a region's Vector package, wait for it to finish, then re-tap download on the same/overlapping trail without navigating away (keep Settings/Offline Regions mounted) — the guard recognizes the region as now covered and does not re-show it in the sheet.
result: pass

### 6. Unified notification stays on aggregate copy through tile generation (WR-01 live check)
expected: On a multi-region trail with packages selected, the id-42 notification does not flash back to plain trail-name/"Generating..." copy during the tile-generation phase.
result: pass

### 7. Download button never permanently stranded (CR-01 live check)
expected: After any download attempt (including one that errors early), the trail's download button re-enables; it never stays permanently disabled for the app session.
result: pass

## Summary

total: 7
passed: 6
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "One id-42 notification shows a single combined, advancing progress bar (not one notification per download) during a multi-region parallel trail+package download"
  status: failed
  reason: "User reported: The download bar resets 3 times to different values during download and finally finishes before 100%. The trail is successfully downloaded."
  severity: major
  test: 4
  root_cause: ""
  artifacts: []
  missing: []
  debug_session: ""
