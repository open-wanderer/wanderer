/// The upload/sync lifecycle of a locally-captured trail.
///
/// **Enum order is load-bearing.** `synced` MUST stay at index 0: it is
/// persisted on [TrailEntity] as a `@Transient()` shadow (`dbSyncState`),
/// and ObjectBox defaults a new int column to `0` on every row that existed
/// before this phase. Reordering these values would make every
/// already-downloaded trail read back as some other sync state and show a
/// sync badge it never earned. The order is append-only for the same reason:
/// the index is what gets persisted, so a new value goes LAST.
///
/// `draft` means "durably stored on this device, deliberately never uploaded
/// automatically". A draft becomes `pending` only through an explicit save in
/// the edit form.
enum TrailSyncState { synced, pending, uploading, failed, draft }

/// Whether [state] represents a trail that has not (yet, or not
/// successfully) reached the server.
///
/// A single named predicate so later plans compare against one call site
/// instead of scattering `state != TrailSyncState.synced` inequality checks.
///
/// A [TrailSyncState.draft] counts as unsynced ON PURPOSE: the sign-out
/// warning, delete gating, the live-capture checks and the orphan-photo sweep
/// must all protect a draft.
bool isUnsyncedState(TrailSyncState state) => state != TrailSyncState.synced;
