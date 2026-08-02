/// D-07's backoff curve and attempt ceiling, expressed as pure, testable
/// policy with no dependency beyond `dart:core`.
///
/// D-07 deliberately rejects a retryable-vs-permanent error taxonomy: the
/// app has no reliable way to classify a failure as "transient" vs
/// "permanent" from a bare [DioException], and misreading a 401 raised
/// mid-token-refresh as permanent would park a perfectly good trail after
/// one unlucky attempt. The only escalation signal this policy uses is the
/// attempt count itself -- every failure, whatever its cause, consumes one
/// attempt, and the curve below decides how long to wait before the next
/// one.
library;

/// The number of failed upload attempts after which a trail's row is parked
/// as [TrailSyncState.failed] and stops retrying automatically (D-07). Must
/// stay greater than 1 -- a ceiling of 1 would park a trail after its very
/// first failure, with no retry at all.
const int kMaxSyncAttempts = 4;

/// How long the drain should wait before retrying a trail that has already
/// failed [attempts] times.
///
/// Returns 30 seconds for attempt 1, 2 minutes for attempt 2, and 10 minutes
/// for attempt 3 and everything higher -- the curve is clamped, never
/// unbounded. `attempts <= 1` (including 0 and negative inputs) is treated
/// the same as attempt 1, so a malformed or not-yet-incremented attempt
/// count never throws and never produces a shorter-than-intended delay.
Duration syncBackoffDelay(int attempts) {
  if (attempts <= 1) return const Duration(seconds: 30);
  if (attempts == 2) return const Duration(minutes: 2);
  return const Duration(minutes: 10);
}
