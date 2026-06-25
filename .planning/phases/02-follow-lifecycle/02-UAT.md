---
status: complete
phase: 02-follow-lifecycle
source: [02-VERIFICATION.md]
started: 2026-06-25T16:00:00Z
updated: 2026-06-25T16:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Confirm reverse proxy routing for instance inbox

expected: |
  In the deployment environment, POST /api/v1/activitypub/instance/inbox is
  routed to the SvelteKit app identically to how POST /api/v1/activitypub/user/{handle}/inbox
  is routed. The SvelteKit adapter-node / nginx / caddy config does NOT strip or
  rewrite X-Forwarded-Path before it reaches the Go handler. Remote instances can
  POST HTTP-signed activities to the inbox IRI and the Go handler reconstructs the
  signed path correctly to verify the signature.
result: skipped
reason: cannot test now

## Summary

total: 1
passed: 0
issues: 0
pending: 0
skipped: 1
blocked: 0

## Gaps
