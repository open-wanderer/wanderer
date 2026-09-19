export interface Asset {
    id: string;
    collectionId: string;
    collectionName: string;
    created?: string;
    updated?: string;
    type: "photo";
    file?: string;
    storage_mode?: "copy" | "link_private";
    remote_status?: "available" | "missing" | "inaccessible";
    remote_checked_at?: string;
    remote_missing_since?: string;
    remote_error?: string;
    author: string;
    external_provider?: string;
    external_id?: string;
    taken_at?: string;
    lat?: number;
    lon?: number;
    metadata?: Record<string, unknown>;
}

export interface AssetLink {
    id?: string;
    collectionId?: string;
    collectionName?: string;
    asset: string;
    trail?: string;
    waypoint?: string;
    summit_log?: string;
    is_thumbnail?: boolean;
    expand?: {
        asset?: Asset;
    };
}
