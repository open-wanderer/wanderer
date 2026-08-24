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

export function photoLibraryCandidateKey(candidate: PhotoLibraryCandidate): string {
    return `${candidate.source ?? "plugin"}:${candidate.providerId ?? candidate.pluginId ?? ""}:${candidate.assetId}`;
}

export function photoLibraryPluginLinks(candidates: PhotoLibraryCandidate[]): PhotoLibraryPluginLink[] {
    const grouped = new Map<string, string[]>();
    for (const candidate of candidates) {
        const pluginId = candidate.pluginId ?? candidate.providerId;
        if (!pluginId) {
            continue;
        }
        grouped.set(pluginId, Array.from(new Set([
            ...(grouped.get(pluginId) ?? []),
            candidate.assetId,
        ])));
    }
    return Array.from(grouped.entries()).map(([pluginId, assetIds]) => ({
        pluginId,
        assetIds,
    }));
}

export function mergePhotoLibraryPluginLinks(
    existing: PhotoLibraryPluginLink[] | undefined,
    incoming: PhotoLibraryPluginLink[] | undefined,
): PhotoLibraryPluginLink[] | undefined {
    const grouped = new Map<string, string[]>();
    for (const link of [...(existing ?? []), ...(incoming ?? [])]) {
        grouped.set(link.pluginId, Array.from(new Set([
            ...(grouped.get(link.pluginId) ?? []),
            ...link.assetIds,
        ])));
    }
    const merged = Array.from(grouped.entries())
        .map(([pluginId, assetIds]) => ({ pluginId, assetIds }))
        .filter((link) => link.pluginId && link.assetIds.length);
    return merged.length ? merged : undefined;
}
