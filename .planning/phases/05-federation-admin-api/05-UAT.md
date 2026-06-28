---
status: testing
phase: 05-federation-admin-api
source: [05-VERIFICATION.md]
started: 2026-06-27T18:05:00Z
updated: 2026-06-27T18:05:00Z
---

## Current Test

number: 1
name: FederationDiscover — live preview card
expected: |
  POST /federation/discover with a live Wanderer instance URL returns HTTP 200
  with JSON body containing actor_id, domain, version, user_count, trail_count
  all populated from the remote NodeInfo 2.1 payload and actor record
awaiting: user response

## Tests

### 1. FederationDiscover — live preview card
expected: POST /federation/discover against a live (or test-double) Wanderer instance returns a preview card with actor_id, domain, version, user_count, trail_count as non-empty values
result: [pending]

### 2. FederationDiscover — non-Wanderer rejection
expected: POST /federation/discover with a reachable non-Wanderer URL (e.g., a Mastodon instance) returns HTTP 400 with error "not a Wanderer instance"
result: [pending]

### 3. FederationFollow — hook fires, no double-delivery
expected: POST /federation/follow with a valid actor_id creates a follows record with status=pending; one Follow activity is delivered via hook; no direct CreateFollowActivity call from handler layer
result: [pending]

### 4. FederationApprove — Accept{Follow} delivered
expected: POST /federation/approve/:id for an inbound pending follow moves record to status=accepted; one Accept{Follow} activity delivered to remote instance via hook
result: [pending]

### 5. FederationDisconnect — correct verb per direction
expected: Outbound follow disconnect deletes record + sends Undo{Follow}; inbound-only follow disconnect moves record to rejected + sends Reject{Follow} (not Undo)
result: [pending]

### 6. 401 guard on all six endpoints
expected: Every endpoint (discover, follow, approve/:id, reject/:id, disconnect/:id, peers) returns HTTP 401 without a PocketBase superuser token or with a regular user token
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
