export interface PhotoLibraryCandidate {
    source?: "plugin" | "wanderer";
    providerId?: string;
    pluginId?: string;
    externalProvider?: string;
    externalId?: string;
    assetId: string;
    originalFileName: string;
    takenAt: string;
    lat: number;
    lon: number;
    pointLat?: number;
    pointLon?: number;
    distance: number;
    distanceFromStart: number;
    city?: string;
    country?: string;
    thumbnailUrl?: string;
}

export interface PhotoLibraryPluginLink {
    pluginId: string;
    assetIds: string[];
}
