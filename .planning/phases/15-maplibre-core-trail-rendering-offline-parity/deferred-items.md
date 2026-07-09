# Deferred Items — Phase 15

Out-of-scope discoveries logged during execution (not fixed — see SCOPE BOUNDARY).

## 15-03

- **Pre-existing dead code in `app/lib/components/trail/trail_dropdown.dart:124-126`**
  — `_allowDelete()` unconditionally `return false;` before its real body
  (`final user = ref.watch(authProvider).value; return user != null && ...`),
  so `flutter analyze` reports one `dead_code` warning at line 126. This predates
  15-03 (the download-delete permission gate is intentionally hard-disabled) and
  is untouched by this plan's glyph-cache wiring. Left as-is; whoever re-enables
  trail deletion should remove the early `return false;`.
