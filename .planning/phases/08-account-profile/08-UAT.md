---
status: testing
phase: 08-account-profile
source: [08-VERIFICATION.md]
started: 2026-06-20T00:00:00Z
updated: 2026-06-20T00:00:00Z
---

## Current Test

number: 1
name: Avatar gallery upload on device
expected: |
  Tapping the avatar or camera icon opens the iOS photo library picker. Selecting an image uploads it via multipart POST to the backend. After upload, authProvider.refresh() is called and the avatar updates to show the new image without restarting the app.
awaiting: user response

## Tests

### 1. Avatar gallery upload on device
expected: Tapping the avatar or camera icon opens the iOS photo library picker. Selecting an image uploads it via multipart POST to the backend. After upload, authProvider.refresh() is called and the avatar updates to show the new image without restarting the app.
result: [pending]

### 2. Email change rejection with wrong password
expected: Entering a wrong current password in EmailChangeSheet and submitting results in the sheet staying open and showing the server's error message in a toast (not a generic message). The email is NOT changed.
result: [pending]

### 3. Password change full flow
expected: Entering the correct current password and a valid new password in PasswordChangeSheet results in a success toast and the sheet dismissing. PocketBase validates oldPassword and the change takes effect.
result: [pending]

### 4. Account deletion navigates to /login
expected: Tapping "Delete Account", confirming in the AlertDialog, triggers account deletion via the API and calls logout(). The go_router auth guard redirects to /login with no errors.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
