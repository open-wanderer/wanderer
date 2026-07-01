import { get } from "svelte/store";
import { _ } from "svelte-i18n";

const mergeErrorCodes = new Set([
    "trail_merge_auth_required",
    "trail_merge_actor_not_found",
    "trail_merge_invalid_request",
    "trail_merge_unknown_suggest_mode",
    "trail_merge_missing_actor",
    "trail_merge_missing_trail_id",
    "trail_merge_same_source_target",
    "trail_merge_requires_multiple_trails",
    "trail_merge_missing_source_trail_id",
    "trail_merge_source_actor_mismatch",
    "trail_merge_source_not_found",
    "trail_merge_target_not_found",
    "trail_merge_not_allowed",
]);

export function translateTrailMergeError(message: string): string {
    if (mergeErrorCodes.has(message)) {
        return get(_)(message);
    }

    return message;
}
