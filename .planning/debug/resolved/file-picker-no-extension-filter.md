---
slug: file-picker-no-extension-filter
status: resolved
trigger: "The file picker allows picking any file type on my Pixel 6 despite the allowsExtensions being defined here. If this an issue that can only be fixed upstream leave it."
created: 2026-09-05
updated: 2026-09-05
resolution: wont-fix-upstream-platform-limitation
---

# File picker shows all file types on Android despite `allowedExtensions`

## Symptoms

- **Expected:** the Import picker on `trail_source_select_screen.dart` shows only
  `.gpx / .kml / .kmz / .tcx / .fit`.
- **Actual:** every file on the device is selectable.
- **Device:** Pixel 6 (oriole), Android. Not reproduced/checked on iOS.
- **Reproduction:** Trail source screen → Import.

## Current Focus

hypothesis: (resolved) `allowedExtensions` is discarded on Android because the
extensions have no MIME mapping.
next_action: none — closed as an upstream/platform limitation.

## Evidence

- Call site: [trail_source_select_screen.dart:174-177](app/lib/routes/trail_source_select_screen.dart#L174-L177)
  passes `FileType.custom` + `trailImportExtensions`
  (`['gpx','kml','kmz','tcx','fit']`, [import_trail_file.dart:26](app/lib/actions/import_trail_file.dart#L26)).
- file_picker 11.0.2 (`pubspec.lock`). Android path:
  `FilePickerPlugin.kt:147` → `FileUtils.getMimeTypes(allowedExtensions)`.
- `FileUtils.kt:361-391` — for each extension it calls
  `MimeTypeMap.getSingleton().getMimeTypeFromExtension(ext)`. On the **first**
  `null` it logs a warning and does
  `return ArrayList(listOf("*/*"))` — aborting the whole filter, not just
  skipping that one extension.
- `FileUtils.kt:201-211` — the resulting list is put into
  `Intent.EXTRA_MIME_TYPES` on `ACTION_OPEN_DOCUMENT`. With `*/*`, SAF shows
  everything.
- **On-device confirmation** (Pixel 6, logcat, picker opened once):
  ```
  W FilePickerUtils: Custom file type 'gpx' is unsupported and will not be filtered.
  ```
  `gpx` — the very first and most important extension — has no entry in
  Android's `MimeTypeMap`, so the bail-out fires immediately.

## Eliminated

- hypothesis: wrong `FileType` passed at the call site — no,
  `FileType.custom` is correct and required for `allowedExtensions`.
- hypothesis: the extension list is malformed (leading dots, uppercase) — no,
  bare lowercase extensions, exactly the documented format.
- hypothesis: a Dart-side MIME escape hatch exists — no. `FilePicker.pickFiles`
  in 11.0.2 exposes only `type` + `allowedExtensions`; there is no
  `allowedMimeTypes` parameter to bypass the broken mapping.

## Root Cause

Android's Storage Access Framework filters **by MIME type only** — it has no
concept of extension filtering. `file_picker` therefore has to translate
`allowedExtensions` into MIME types via `MimeTypeMap`, and Android's map has no
entry for `.gpx` (nor, almost certainly, `.tcx` or `.fit`). file_picker's
`getMimeTypes` then gives up entirely and sends `*/*`.

Two independent problems, and only one is a package bug:

1. **Package bug (fixable upstream):** bailing to `*/*` on the *first* unmapped
   extension instead of filtering with the ones that did resolve
   (`kml`/`kmz` do have AOSP MIME entries).
2. **Platform limitation (not fixable at all):** even with that fixed, `.gpx`,
   `.tcx` and `.fit` have no MIME type on Android, so SAF cannot filter for
   them. Fixing the package would only make the picker show
   *kml/kmz only* — which would hide the primary `.gpx` format. Strictly worse.

## Resolution

**Closed as won't-fix.** No app-side change. Per the user's instruction, an
upstream-only issue is left alone — and here even the upstream fix would not
produce the desired behaviour.

The correct mitigation is already in place: `importTrailFile` re-validates the
extension after the pick and shows `trail_source_import_error` for anything
unsupported ([import_trail_file.dart:74-79](app/lib/actions/import_trail_file.dart#L74-L79)),
and both call sites are documented as treating `allowedExtensions` as a hint.

files_changed: none
