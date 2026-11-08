import type { Asset } from "./asset.ts";

// Copy responses identify the selected source so the editor can replace it.
export interface AttachedAsset extends Asset {
    sourceAssetId?: string;
}
