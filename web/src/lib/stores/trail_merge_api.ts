import type { MergeSettings } from "$lib/components/trail/trail_merge_modal.svelte";
import { APIError } from "$lib/util/api_util";

export interface TrailMergeSuggestCandidate {
    trailId: string;
    score: number;
    reason: string;
    warnings: string[];
    selectable: boolean;
}

export interface TrailMergeSuggestResponse {
    targetTrailId: string;
    reason: string;
    warnings: string[];
    candidates: TrailMergeSuggestCandidate[];
}

export interface TrailMergeSuggestGroup {
    groupId: string;
    trailIds: string[];
    targetTrailId: string;
    reason: string;
    score: number;
    indirect: boolean;
}

export interface TrailMergeSuggestGroupsResponse {
    groups: TrailMergeSuggestGroup[];
}

export async function trail_merge_suggest_manual(trailIds: string[]) {
    const response = await fetch("/api/v1/trail-merge/suggest", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            mode: "manual-selection",
            trailIds,
        }),
    });

    if (!response.ok) {
        const error = await response.json();
        throw new APIError(response.status, error.message, error.detail);
    }

    return await response.json() as TrailMergeSuggestResponse;
}

export async function trail_merge_suggest_auto(sourceTrailId: string) {
    const response = await fetch("/api/v1/trail-merge/suggest", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            mode: "auto-discovery",
            sourceTrailId,
        }),
    });

    if (!response.ok) {
        const error = await response.json();
        throw new APIError(response.status, error.message, error.detail);
    }

    return await response.json() as TrailMergeSuggestResponse;
}

export async function trail_merge_suggest_groups() {
    const response = await fetch("/api/v1/trail-merge/suggest", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            mode: "maintenance-groups",
        }),
    });

    if (!response.ok) {
        const error = await response.json();
        throw new APIError(response.status, error.message, error.detail);
    }

    return await response.json() as TrailMergeSuggestGroupsResponse;
}

export async function trail_merge(sourceTrailId: string, targetTrailId: string, settings: MergeSettings) {
    const response = await fetch("/api/v1/trail-merge", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            sourceTrailId,
            targetTrailId,
            settings,
        }),
    });

    if (!response.ok) {
        const error = await response.json();
        throw new APIError(response.status, error.message, error.detail);
    }

    return await response.json();
}
