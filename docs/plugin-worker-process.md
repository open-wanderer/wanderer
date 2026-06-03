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
- plugin code never receives backend policy objects or decrypted host secrets
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
5. Backend runs `ExecuteHostRequest(...)` with the session-bound
   `RequestPolicyContext` and returns the result.
6. Worker returns each plugin result.
7. Backend sends `shutdown` when the job is complete.
8. Worker exits and the backend reaps it.

Important invariant:

- the worker contains no network policy logic
- the worker is never sent `RequestPolicyContext`, decrypted host auth, or other
  backend-only policy material
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
- length-prefixed JSON messages

Wire format:

- 4-byte unsigned big-endian frame length
- UTF-8 JSON envelope of exactly that length
- binary fields, including plugin input/output bytes and HTTP bodies, are
  base64-encoded inside the JSON envelope
- frame length is checked against the configured maximum before allocation or
  full read

Do not use:

- localhost TCP sockets
- Unix sockets as the first implementation

Reason:

- stdio is local by construction
- no extra listener
- easiest cross-platform process pair setup

The worker's stdout is reserved exclusively for RPC frames. The worker must not
enable Extism/WASI behavior that writes plugin stdout directly to `os.Stdout`,
including `EXTISM_ENABLE_WASI_OUTPUT`. Plugin stdout must be discarded, captured
separately, or redirected away from the RPC pipe. Any Extism log/observe output
must also stay off stdout.

Worker stderr may contain both worker diagnostics and plugin/WASI stderr. The
backend captures stderr with a fixed maximum size and treats it as diagnostic
text only, never as protocol data.

## Message types

Backend -> worker:

- `call_export`
- `shutdown`
- `host_http_response`

Worker -> backend:

- `host_http_request`
- `call_result`
- `error`

`call_result` is used for every export invocation that reached a normal Extism
return path. It can represent:

- success: exit code 0 and output bytes
- recoverable plugin error: exit code 0 with a plugin ABI response containing
  `output.Error`, or non-zero exit code with or without a parseable
  `PluginError` from `GetErrorWithContext(...)`. A missing or invalid error
  payload becomes a generic non-fatal call error; the export still returned
  through a normal Extism path, so the session stays valid

`error` is reserved for session-fatal failures:

- worker crash or unexpected worker exit
- Extism trap/panic or `CallWithContext(...)` Go error
- malformed RPC
- oversized frame or payload
- timeout
- stdout/RPC corruption

The backend must reconstruct the same public error shapes used today:

- an ABI response with `output.Error` remains output bytes; existing capability
  decoding turns it into `PluginCapabilityError`
- a non-zero export result with a valid JSON `PluginError` becomes
  `PluginCallError{PluginID, Export, PluginError}`
- a non-zero export result without a parseable `PluginError` becomes a generic
  non-fatal `PluginCallError`, for example with code `plugin_error`; the session
  stays valid
- a session-fatal `error` becomes an infrastructure/runtime error and invalidates
  the worker session

This distinction is required so routine detail-item failures do not churn worker
processes.

Minimum `call_export` payload:

- plugin WASM path
- export name
- serialized input bytes
- optional stable job/session identifier for diagnostics

`call_export` must not include `RequestPolicyContext`, `HostAuth`, connector
policy, TLS CA bundles, decrypted tokens, or other backend-only policy data. The
backend keeps policy state attached to the worker session and uses that state
when servicing `host_http_request` messages.

Minimum `host_http_request` payload:

- the exact request bytes the plugin passed to `wanderer.http_request`

The worker does not parse or validate those bytes beyond RPC size enforcement.
It forwards them verbatim to the backend. The backend decodes them as the current
`HostRequestSpec`, validates policy, executes `ExecuteHostRequest(...)`, and
returns a `host_http_response`.

Minimum `host_http_response` payload:

- the same JSON shape currently written by `writeHostHTTPResponse(...)`:
  `status`, `headers`, `bodyBase64`, and `error`

The worker-side host function writes that response JSON back into WASM memory
and returns the pointer to the plugin, matching the current in-process host
function behavior.

First release assumption:

- the backend passes an absolute plugin WASM path
- the worker process runs with filesystem access sufficient to open that path

The first implementation should not rely on:

- matching current working directories
- relative plugin paths

Passing WASM bytes instead of a path is possible, but is not the chosen first
implementation because it increases RPC payload size and complexity.

## Runtime loading

Preferred first implementation:

- worker loads the WASM file once at session start
- worker registers Extism host functions that proxy back to the backend
- worker reuses that loaded runtime for all exports within the same job

Do not optimize beyond per-job reuse yet.

This intentionally changes current behavior: today each `Runtime.Call(...)`
creates a fresh Extism instance and closes it after one export. Reusing one
loaded runtime within a worker session allows module-level memory/globals to
survive between exports in the same job. That trade-off is accepted for the
first worker-process implementation because it avoids repeated startup during
detail export loops and makes send/auth-refresh sequencing straightforward.

If this state sharing becomes a correctness problem, a later implementation may
keep one worker process per job while compiling once and instantiating fresh per
export. The plugin ABI must not depend on either behavior.

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
- Extism calls remain strictly sequential within one worker session; do not run
  concurrent exports against one loaded plugin instance

Reason:

- the backend may retry work after worker failure
- future implementations may recreate the worker session mid-job
- preserving functional correctness from explicit inputs keeps the ABI easier to
  reason about

## Runtime session API

The existing stateless runtime shape:

- `Runtime.Call(ctx, plugin, export, input, policy)`

is not enough to express one worker session that serves multiple exports for one
sync/send job.

The worker-backed runtime should expose a session-oriented shape such as:

- `session := runtime.OpenSession(ctx, plugin, policy)`
- `session.Call(ctx, export, input)`
- `session.Close(ctx)`

The exact Go names can vary, but the contract must be explicit:

- one session maps to one worker process
- the backend owns `RequestPolicyContext` for that session
- all sync list/detail exports for the job use that session until it fails or is
  closed
- send-route `prepare_send_route` and any token-refresh export triggered by
  auth injection use the same session
- recoverable plugin errors do not invalidate the session
- after a trap, protocol error, timeout, oversized message, stdout/RPC
  corruption, or killed worker, the session is invalid and must be closed/reaped

## Send flow

The send path is different from sync and must be modeled explicitly.

Expected send sequence:

1. Backend opens one worker session for the send job.
2. Backend calls `prepare_send_route` through that session.
3. Worker returns an upload plan.
4. Backend calls `InjectHostRequestAuth(...)` for the plan.
5. If auth injection requires a token refresh export, that refresh export is
   called through the same worker session.
6. Backend executes the upload itself with `ExecuteHostRequest(...)`.
7. Backend sends `shutdown`, closes pipes, waits for the worker, and records the
   result.

Important send invariants:

- the actual upload is backend-side work, not a `host_http_request` emitted by
  the worker
- GPX bytes do not need to be sent to the worker unless an explicit plugin ABI
  requires that in the future
- auth refresh is a normal plugin export call on the live session, even though
  it is triggered by backend auth-injection logic after `prepare_send_route`
  returns

## RPC sequencing

Calls are strictly sequential within one worker session.

During a `call_export`:

- the backend writes `call_export`
- the worker enters `instance.Call(...)`
- if the plugin calls `wanderer.http_request`, the worker-side host function
  writes `host_http_request` to stdout and waits inline for
  `host_http_response` on stdin
- while the backend waits for `call_result`, it must continue reading worker
  messages and service each `host_http_request`
- after the export finishes, the worker writes exactly one terminal
  `call_result` or `error`

Because exports are sequential and host calls are handled inline, the first
implementation does not need correlation IDs. If concurrent calls or pipelining
are added later, correlation IDs become mandatory.

## Backend limits

These limits must exist before release:

- per-export timeout
- per-worker-session timeout
- maximum RPC request size
- maximum RPC response size
- maximum captured stderr size

Recommended first-release defaults:

- per-export timeout: 2 minutes
- per-worker-session timeout: 15 minutes
- worker-slot acquisition timeout: 30 seconds, bounded by the caller context
- maximum backend -> worker frame size: 32 MiB
- maximum worker -> backend frame size: 64 MiB
- maximum captured stderr size: 64 KiB
- maximum concurrent worker sessions: configurable; default to `max(2, NumCPU)`

These values should be configuration-backed. In particular, the worker ->
backend response limit must allow legitimate list-export batches with many
items, while still bounding allocation before a frame is read.

These limits should fail closed:

- oversized message -> fail current call and invalidate the session
- export timeout -> kill worker and fail the whole job/session
- session timeout -> kill worker and fail the whole job/session
- malformed RPC -> fail current call and invalidate the session

Timeouts must be enforced by killing the worker OS process when needed. Context
cancellation alone is not the hard guarantee, especially for CPU-bound WASM
loops. The hostile hang test must include a busy-looping plugin, not only a
sleeping plugin.

Failure policy within one worker session:

- recoverable plugin errors do not make the current worker session invalid
- traps, malformed RPC, oversized messages, stdout/RPC corruption, timeouts, and
  worker death make the current worker session invalid
- after a session-fatal failure, the backend must terminate that worker session
  and must not send additional `call_export` messages to the same worker
  afterward
- if the surrounding sync/send job should continue, it must do so with a fresh
  worker session

Recommended first-release backend policy:

- failure in a list export fails that capability/job path immediately
- recoverable failure in a detail export records the item failure and continues
  remaining items on the same worker session
- session-fatal failure in a detail export invalidates the current worker
  session, records the item failure, and may continue remaining items with a new
  worker session if the sync logic chooses to continue

This keeps the worker protocol simple while leaving higher-level sync behavior
under backend control.

If the worker receives a malformed or unexpected first message, it should write
a short diagnostic to stderr and exit with a non-zero status instead of trying
to continue the RPC session.

If the worker receives an invalid message after a session has started, it should
also fail the session, write a short diagnostic to stderr, and exit non-zero.

If stdin reaches EOF before `shutdown`, the worker should assume the backend is
gone and exit promptly. This prevents orphaned workers after backend crashes.

The backend must always close pipes and `Wait()` for worker exit after normal
shutdown, timeout kill, protocol failure, or external worker death. In-flight
host HTTP handling and reader goroutines must observe session cancellation and
terminate cleanly.

The backend must also enforce a maximum number of concurrent worker sessions
with a process-wide semaphore. Acquiring a worker slot happens before spawning
the process. If no slot is available, the caller waits until either a slot is
released, the caller context is cancelled, or the worker-slot acquisition
timeout expires. On timeout/cancellation, the job fails before spawning a worker.
Do not create an unbounded in-memory worker queue. Worker pooling remains out of
scope, but this cap is required to avoid unbounded process creation when many
sync/send jobs run at once.

## Files

## New files

### `db/pluginsystem/worker.go`

Responsibilities:

- start worker process
- wire stdin/stdout/stderr
- manage one worker session per sync/send job
- keep `RequestPolicyContext` and host secrets backend-side for that session
- enforce the global concurrent-worker cap
- enforce per-export timeout
- enforce per-session timeout
- kill worker on timeout or protocol failure
- return final plugin output or structured error

### `db/pluginsystem/worker_rpc.go`

Responsibilities:

- RPC message structs
- framing/encoding/decoding helpers
- message size checks
- base64 handling for binary message fields

### `db/pluginsystem/worker_test.go`

Required tests:

- worker crash -> backend survives
- worker CPU busy-loop hang -> timeout + kill
- oversized RPC payload -> bounded failure
- worker killed externally -> clean error
- plugin stdout cannot corrupt the RPC stream
- backend policy/secret data is not present in worker RPC payloads
- recoverable plugin detail errors do not restart the worker session
- non-zero export with or without JSON `PluginError` is reconstructed as
  non-fatal `PluginCallError`
- send auth-refresh export runs in the same worker session as
  `prepare_send_route`

### `db/cmd/pluginworker/`

Responsibilities:

- worker process entry point
- receive `call_export` and `shutdown`
- instantiate Extism plugin
- proxy host-function calls over RPC
- keep stdout dedicated to RPC frames
- exit promptly on stdin EOF

## Modified files

### `db/pluginsystem/runtime.go`

Change:

- replace the direct in-process `ExtismRuntime.Call(...)` execution path with a
  worker-backed, session-oriented runtime implementation
- pass an absolute plugin WASM path to the worker instead of relying on process
  working-directory behavior
- support a worker session that can serve multiple export calls for one job
- keep the session policy in the backend and never serialize it to the worker

Required outcome:

- backend no longer loads/runs the plugin directly in the request path

### `db/pluginsystem/host_http.go`

Change:

- expose backend-side handling that can serve worker RPC `host_http_request`
  messages
- make auth injection able to call refresh exports through an existing runtime
  session

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
4. Implement the session-oriented runtime API.
5. Implement worker session lifecycle for one sync/send job.
6. Implement worker-side Extism execution with stdout separated from RPC.
7. Implement backend-side sequencing so multiple exports for one job reuse the
   same worker.
8. Implement worker-side `wanderer.http_request` shim.
9. Implement backend-side RPC handling that calls `ExecuteHostRequest(...)`.
10. Wire send-route auth refresh through the existing worker session.
11. Add timeout, kill, stderr capture, concurrency-cap, and message-size
    enforcement.
12. Switch runtime/job call paths to worker-backed execution.
13. Add crash/hang/oversize/stdout/send-refresh tests.
14. Update plugin documentation.

## Acceptance Criteria

This work is done when all of the following are true:

1. Plugin execution no longer happens in-process in normal runtime paths.
2. All exports within one sync/send job run in the same worker session.
3. A plugin panic, trap, or worker exit does not crash the backend.
4. A hung plugin or worker session is terminated by timeout.
5. Both per-export and per-session timeouts are enforced.
6. `wanderer.http_request` still works through backend policy enforcement.
7. Backend policy objects, decrypted auth, and TLS material are never serialized
   to the worker.
8. Plugin stdout cannot corrupt the RPC stream.
9. Send-route auth refresh runs through the live worker session.
10. Recoverable plugin errors preserve the worker session.
11. Non-zero export results with JSON `PluginError` reconstruct
    `PluginCallError`.
12. Oversized RPC messages are rejected safely.
13. Worker stderr is captured and surfaced for debugging.
14. Concurrent worker creation is bounded with acquire timeout/cancellation.
15. Documentation reflects the worker-based runtime model.

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
   - the test uses a CPU busy loop, not only sleep

3. One sync job that performs a list export followed by multiple detail exports.
   Expected:
   - all exports for that job reuse the same worker session
   - the backend still controls sequencing and deduplication
   - a recoverable detail `output.Error` increments skipped items and remaining
     detail exports continue on the same worker session

4. Plugin returns non-zero with and without valid JSON `PluginError`.
   Expected:
   - backend reconstructs `PluginCallError`
   - worker session remains valid
   - a sync detail call may continue on the same session

5. One send job that performs `prepare_send_route` and requires auth refresh.
   Expected:
   - `prepare_send_route` and refresh export use the same worker session
   - the final upload runs in the backend
   - GPX bytes are not sent to the worker unless the plugin ABI explicitly
     requires them

6. Plugin that uses `wanderer.http_request`.
   Expected:
   - request succeeds only according to host policy
   - policy/secrets are not present in worker frames
   - request bytes are forwarded verbatim to the backend
   - response JSON matches the current `hostHTTPResponse` shape

7. Plugin writes to stdout.
   Expected:
   - RPC stream remains valid
   - stdout output is discarded, separately captured, or redirected away from
     the RPC pipe

8. Plugin or worker that emits oversized RPC data.
   Expected:
   - bounded failure
   - backend survives

9. Worker process killed externally during execution.
   Expected:
   - backend detects failure
   - call fails cleanly

10. Backend closes worker stdin without `shutdown`.
   Expected:
   - worker exits promptly

11. Concurrent sync/send jobs exceed the worker cap.
   Expected:
   - jobs wait for a worker slot
   - jobs fail before spawning if their context is cancelled or the acquisition
     timeout expires

## Non-goals

Do not block first release on:

- worker pools
- process sandboxing beyond process separation
- Linux-only network namespaces
- cross-platform hard network isolation

Those can follow later if needed. The first release goal is a correct and
robust worker-process boundary.
