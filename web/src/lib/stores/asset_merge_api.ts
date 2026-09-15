import { APIError } from "$lib/util/api_util";

export interface AssetMergeLinkCounts {
    trails: number;
    waypoints: number;
    summitLogs: number;
    total: number;
}

export interface AssetMergeAsset {
    id: string;
    collectionId: string;
    collectionName: string;
    created?: string;
    updated?: string;
    type: "photo";
    file?: string;
    storageMode?: "copy" | "link_private";
    remoteStatus?: "available" | "missing" | "inaccessible";
    externalProvider?: string;
    externalId?: string;
    takenAt?: string;
    lat?: number;
    lon?: number;
    originalFileName: string;
    thumbnailUrl: string;
    links: AssetMergeLinkCounts;
}

export interface AssetMergeSuggestGroup {
    groupId: string;
    assetIds: string[];
    targetAssetId: string;
    reason: string;
    matchReason: string;
    score: number;
    assets: AssetMergeAsset[];
}

export interface AssetMergeSuggestResponse {
    groups: AssetMergeSuggestGroup[];
}

export interface AssetMergeResponse {
    acknowledged: boolean;
    targetAssetId: string;
    mergedAssetIds: string[];
    deletedAssetIds: string[];
    reassignedLinks: number;
}

export async function asset_merge_suggest_groups() {
    const response = await fetch("/api/v1/asset-merge/suggest", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({}),
    });

    if (!response.ok) {
        const error = await response.json();
        throw new APIError(response.status, error.message, error.detail);
    }

    return await response.json() as AssetMergeSuggestResponse;
}

export async function asset_merge(sourceAssetIds: string[], targetAssetId: string) {
    const response = await fetch("/api/v1/asset-merge", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            sourceAssetIds,
            targetAssetId,
        }),
    });

    if (!response.ok) {
        const error = await response.json();
        throw new APIError(response.status, error.message, error.detail);
    }

    return await response.json() as AssetMergeResponse;
}
