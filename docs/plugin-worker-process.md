# Plugin Worker Process Implementation

## Goal

Before the first release of the plugin system, Wanderer should stop executing
plugins in-process and run each sync/send job in a separate worker process.

This document is the implementation baseline.

## Final Decision

The first release uses:

- a separate worker process per sync/send job
- stdio-based RPC between backend and worker
- Extism/WASM execution inside the worker
- host functions executed in the backend only

This first release does **not** require OS-specific network isolation.

What this design must guarantee:

- plugin crash or hang does not crash the backend process
- plugin memory is not shared with backend memory
- plugin-controlled host HTTP still goes through backend policy code only
- timeouts and message-size limits are enforced by the backend

What this design does **not** claim:

- hard OS-level network isolation
- hard memory or CPU isolation beyond process separation

Network isolation continues to depend on the current Extism/WASI capability
model unless stronger isolation is added later.

## Scope

Included in this implementation:

- worker process launch and lifecycle
- RPC protocol for export execution and host-function callbacks
- moving Extism runtime execution out of the backend process
- backend-side handling of `wanderer.http_request`
- timeout handling
- RPC size limits
- worker stderr capture
- hostile test coverage for crash, hang, and oversized messages

Explicitly out of scope for this release:

- Linux-only network namespaces
- seccomp / cgroups / AppContainer / container isolation
- worker pooling
- additional host functions beyond what already exists, except for the RPC
  bridge

## Runtime Model

Current model:

- backend calls `ExtismRuntime.Call(...)` directly
- host functions run in-process

Target model:

1. Backend starts a worker process.
2. Backend sends one or more `call_export` requests for the same job.
3. Worker loads the plugin and executes each export in sequence.
4. If plugin code calls `wanderer.http_request`, the worker sends an RPC
   request back to the backend.
5. Backend runs `ExecuteHostRequest(...)` and returns the result.
6. Worker returns each plugin result.
7. Backend sends `shutdown` when the job is complete.
8. Worker exits and the backend reaps it.

Important invariant:

- the worker contains no network policy logic
- the backend remains the only place where plugin-controlled HTTP is executed

## Concrete Design

## Worker lifecycle

First release policy:

- spawn one worker per sync/send job
- reuse the same worker for all exports inside that job
- no long-lived worker pool

Reason:

- avoids process startup for every detail export
- still gives one clear isolation boundary per job
- keeps cleanup and timeout handling manageable

Protocol consequence:

- each worker may receive multiple `call_export` messages for one job
- all `call_export` messages in one worker session must refer to the same plugin
  instance/job context
- the backend sends `shutdown` when no more exports remain for that job
- the worker exits after `shutdown` or after a fatal protocol/runtime error

## Transport

RPC transport:

- backend <-> worker over stdin/stdout
- framed JSON messages

Do not use:

- localhost TCP sockets
- Unix sockets as the first implementation

Reason:

- stdio is local by construction
- no extra listener
- easiest cross-platform process pair setup

## Message types

Backend -> worker:

- `call_export`
- `shutdown`
- `host_http_response`

Worker -> backend:

- `host_http_request`
- `call_result`
- `error`

Backend -> worker responses:

- none beyond the explicit messages above; responses are direction-specific RPC
  messages, not a third transport direction

Minimum `call_export` payload:

- plugin WASM path
- export name
- serialized input bytes
- resolved `RequestPolicyContext`
- optional stable job/session identifier for diagnostics

First release assumption:

- the backend passes an absolute plugin WASM path
- the worker process runs with filesystem access sufficient to open that path

The first implementation should not rely on:

- matching current working directories
- relative plugin paths

Passing WASM bytes instead of a path is possible, but is not the chosen first
implementation because it increases RPC payload size and complexity.

## Runtime loading

First implementation:

- worker loads the WASM file once at session start
- worker registers Extism host functions that proxy back to the backend
- worker reuses that loaded runtime for all exports within the same job

Do not optimize beyond per-job reuse yet.

## Plugin state within one worker session

Because the worker reuses one loaded plugin runtime for all exports in the same
job, module-level memory and globals may be shared between those export calls.

This should be treated as:

- allowed for intra-job caching and other performance optimizations
- not part of the correctness contract between host and plugin

In practice this means:

- a plugin may cache derived data within one job, for example parsed auth
  material or provider metadata
- a plugin must not require that previous exports have populated in-memory state
  in order to behave correctly
- every export must still be correct when evaluated only from its explicit host
  input plus any provider calls it performs through host functions

Reason:

- the backend may retry work after worker failure
- future implementations may recreate the worker session mid-job
- preserving functional correctness from explicit inputs keeps the ABI easier to
  reason about

## Backend limits

These limits must exist before release:

- per-export timeout
- per-worker-session timeout
- maximum RPC request size
- maximum RPC response size
- maximum captured stderr size

These limits should fail closed:

- oversized message -> fail plugin call
- export timeout -> kill worker and fail the whole job/session
- session timeout -> kill worker and fail the whole job/session
- malformed RPC -> fail plugin call

Failure policy within one worker session:

- any export failure, trap, malformed RPC, or timeout makes the current worker
  session invalid
- the backend must terminate that worker session and must not send additional
  `call_export` messages to the same worker afterward
- if the surrounding sync/send job should continue, it must do so with a fresh
  worker session

Recommended first-release backend policy:

- failure in a list export fails that capability/job path immediately
- failure in a detail export invalidates the current worker session, records the
  item failure, and may continue remaining items with a new worker session if
  the sync logic chooses to continue

This keeps the worker protocol simple while leaving higher-level sync behavior
under backend control.

If the worker receives a malformed or unexpected first message, it should write
a short diagnostic to stderr and exit with a non-zero status instead of trying
to continue the RPC session.

If the worker receives an invalid message after a session has started, it should
also fail the session, write a short diagnostic to stderr, and exit non-zero.

## Files

## New files

### `db/pluginsystem/worker.go`

Responsibilities:

- start worker process
- wire stdin/stdout/stderr
- manage one worker session per sync/send job
- enforce per-export timeout
- enforce per-session timeout
- kill worker on timeout or protocol failure
- return final plugin output or structured error

### `db/pluginsystem/worker_rpc.go`

Responsibilities:

- RPC message structs
- framing/encoding/decoding helpers
- message size checks

### `db/pluginsystem/worker_test.go`

Required tests:

- worker crash -> backend survives
- worker hang -> timeout + kill
- oversized RPC payload -> bounded failure
- worker killed externally -> clean error

### `db/cmd/pluginworker/`

Responsibilities:

- worker process entry point
- receive `call_export` and `shutdown`
- instantiate Extism plugin
- proxy host-function calls over RPC

## Modified files

### `db/pluginsystem/runtime.go`

Change:

- replace direct in-process `ExtismRuntime.Call(...)` execution path with a
  worker-backed runtime implementation
- pass an absolute plugin WASM path to the worker instead of relying on process
  working-directory behavior
- support a worker session that can serve multiple export calls for one job

Required outcome:

- backend no longer loads/runs the plugin directly in the request path

### `db/pluginsystem/host_http.go`

Change:

- expose backend-side handling that can serve worker RPC `host_http_request`
  messages

Required outcome:

- worker never executes host HTTP directly
- backend remains the single HTTP chokepoint

### `db/pluginsystem/protocol.go`

Change:

- only if shared protocol/helper types are needed by worker/backend code

Keep this minimal. Do not overload plugin ABI types with internal worker RPC
transport concerns unless that clearly reduces duplication.

### `db/main.go`

Change:

- register or bundle worker entrypoint if needed by the chosen binary layout

### `docs/src/content/docs/develop/plugin-system.md`

Change:

- document that released plugin execution uses a separate worker process
- document that host HTTP still runs in the backend

### `plugins/README.md`

Change:

- document worker process boundary for plugin developers

### `plugins/sdk/README.md`

Change:

- document that SDK host-function helpers are proxied through the backend via
  the worker process

## Suggested binary layout

Preferred first implementation:

- keep one backend binary
- add a worker subcommand, for example:
  - `./pocketbase plugin-worker`

Reason:

- easiest deployment
- no second artifact to package
- easiest version alignment between backend and worker

Alternative:

- separate worker binary

Only choose this if packaging stays simple.

## Implementation Steps

1. Create worker RPC types and framing.
2. Add worker subcommand/binary entrypoint.
3. Implement backend worker launcher.
4. Implement worker session lifecycle for one sync/send job.
5. Implement worker-side Extism execution.
6. Implement backend-side sequencing so multiple exports for one job reuse the
   same worker.
7. Implement worker-side `wanderer.http_request` shim.
8. Implement backend-side RPC handling that calls `ExecuteHostRequest(...)`.
9. Add timeout, kill, stderr capture, and message-size enforcement.
10. Switch runtime/job call paths to worker-backed execution.
11. Add crash/hang/oversize tests.
12. Update plugin documentation.

## Acceptance Criteria

This work is done when all of the following are true:

1. Plugin execution no longer happens in-process in normal runtime paths.
2. All exports within one sync/send job run in the same worker session.
3. A plugin panic, trap, or worker exit does not crash the backend.
4. A hung plugin or worker session is terminated by timeout.
5. Both per-export and per-session timeouts are enforced.
6. `wanderer.http_request` still works through backend policy enforcement.
7. Oversized RPC messages are rejected safely.
8. Worker stderr is captured and surfaced for debugging.
9. Documentation reflects the worker-based runtime model.

## Verification

Minimum release-blocking verification:

1. Hostile plugin that traps immediately.
   Expected:
   - backend survives
   - call fails cleanly

2. Hostile plugin that hangs forever.
   Expected:
   - export timeout or session timeout fires
   - worker is killed
   - backend survives

3. One sync job that performs a list export followed by multiple detail exports.
   Expected:
   - all exports for that job reuse the same worker session
   - the backend still controls sequencing and deduplication

4. Plugin that uses `wanderer.http_request`.
   Expected:
   - request succeeds only according to host policy

5. Plugin or worker that emits oversized RPC data.
   Expected:
   - bounded failure
   - backend survives

6. Worker process killed externally during execution.
   Expected:
   - backend detects failure
   - call fails cleanly

## Non-goals

Do not block first release on:

- worker pools
- process sandboxing beyond process separation
- Linux-only network namespaces
- cross-platform hard network isolation

Those can follow later if needed. The first release goal is a correct and
robust worker-process boundary.
