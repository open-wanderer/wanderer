import type { Asset } from "$lib/models/asset";
import { assetIdFromPhotoURL, assetPhotoURL } from "$lib/util/asset_link_util";
import type { GPSCoordinates, PhotoExifMetadata } from "$lib/util/exif_util";
import { APIError } from "$lib/util/api_util";

interface AssetTarget {
    trail?: string;
    waypoint?: string;
    summit_log?: string;
}

export interface AssetPluginLink {
    pluginId: string;
    assetIds: string[];
}

interface AssetPluginImportResult {
    asset?: Asset;
}

interface AssetPluginImportResponse {
    imported?: AssetPluginImportResult[];
    omitted?: { assetId: string; reason: string }[];
}

interface AssetAttachmentInput {
    files?: File[];
    assetIds?: string[];
    pluginLinks?: AssetPluginLink[];
    target: AssetTarget;
    existingPhotos?: string[];
    f?: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>;
}

const generatedAssetKinds = new WeakMap<File, string>();
const assetFileMetadata = new WeakMap<File, PhotoExifMetadata>();

export function markGeneratedAssetFile(file: File, kind: string): File {
    generatedAssetKinds.set(file, kind);
    return file;
}

export function markAssetFileCoordinates(file: File, coordinates: GPSCoordinates | undefined): File {
    if (coordinates) {
        markAssetFileMetadata(file, { coordinates });
    }
    return file;
}

export function markAssetFileMetadata(file: File, metadata: PhotoExifMetadata | undefined): File {
    if (metadata?.coordinates || metadata?.takenAt) {
        assetFileMetadata.set(file, metadata);
    }
    return file;
}

export function assetFileCoordinatesFor(file: File | undefined): GPSCoordinates | undefined {
    return file ? assetFileMetadata.get(file)?.coordinates : undefined;
}

export async function assets_create(
    files: File[],
    target: AssetTarget,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
): Promise<Asset[]> {
    if (!files.length) {
        return [];
    }

    const formData = new FormData();
    for (const file of files) {
        formData.append("files", file);
    }
    const fileMetadata = files.map((file) => assetFileMetadata.get(file) ?? null);
    if (fileMetadata.some((metadata) => metadata !== null)) {
        formData.append("fileMetadata", JSON.stringify(fileMetadata));
    }
    const generatedKind = files.length === 1 ? generatedAssetKind(files[0]) : undefined;
    if (generatedKind) {
        formData.append("metadata", JSON.stringify({
            generated: {
                kind: generatedKind,
            },
        }));
    }
    for (const [key, value] of Object.entries(target)) {
        if (value !== undefined && value !== null) {
            formData.append(key, value.toString());
        }
    }

    const r = await f("/api/v1/assets", {
        method: "PUT",
        body: formData,
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return await r.json();
}

export async function assets_link(
    assetIds: string[] | undefined,
    target: AssetTarget,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
): Promise<Asset[]> {
    if (!assetIds?.length) {
        return [];
    }

    const formData = new FormData();
    for (const assetId of assetIds) {
        formData.append("assetIds", assetId);
    }
    for (const [key, value] of Object.entries(target)) {
        if (value !== undefined && value !== null) {
            formData.append(key, value.toString());
        }
    }

    const r = await f("/api/v1/assets", {
        method: "PUT",
        body: formData,
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return await r.json();
}

export async function assets_import_plugin_links(
    links: AssetPluginLink[] | undefined,
    target: AssetTarget,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
): Promise<Asset[]> {
    if (!links?.length || !target.trail) {
        return [];
    }

    const assets: Asset[] = [];
    for (const link of links) {
        if (!link.assetIds.length) {
            continue;
        }
        const plugin = encodeURIComponent(link.pluginId);
        const r = await f(`/api/v1/plugins/assets/${plugin}/import-to-target`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                trailId: target.trail,
                waypointId: target.waypoint,
                summitLogId: target.summit_log,
                assetIds: link.assetIds,
            }),
        });
        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail);
        }
        const response: AssetPluginImportResponse = await r.json();
        assets.push(...(response.imported ?? []).map((result) => result.asset).filter((asset): asset is Asset => Boolean(asset)));
    }
    return assets;
}

export function has_asset_attachments(input: Pick<AssetAttachmentInput, "files" | "assetIds" | "pluginLinks">): boolean {
    return Boolean(
        input.files?.length ||
        input.assetIds?.length ||
        input.pluginLinks?.some((link) => link.assetIds.length),
    );
}

export async function assets_attach_to_target(input: AssetAttachmentInput): Promise<string[]> {
    const f = input.f ?? fetch;
    const photos = [...(input.existingPhotos ?? [])];
    const appendAssets = (assets: Asset[]) => {
        photos.push(...assets.map(asset_photo_url));
    };

    appendAssets(await assets_create(input.files ?? [], input.target, f));
    appendAssets(await assets_link(input.assetIds, input.target, f));
    appendAssets(await assets_import_plugin_links(input.pluginLinks, input.target, f));

    return uniquePhotoURLs(photos);
}

export async function assets_delete_removed(
    oldPhotos: string[] | undefined,
    newPhotos: string[] | undefined,
    target: AssetTarget,
): Promise<void> {
    const removedAssetIds = (oldPhotos ?? [])
        .filter((oldPhoto) => !(newPhotos ?? []).find((newPhoto) => newPhoto === oldPhoto))
        .map(assetIdFromPhotoUrl)
        .filter((id): id is string => !!id);

    const searchParams = new URLSearchParams();
    for (const key of ["trail", "waypoint", "summit_log"] as const) {
        if (target[key]) {
            searchParams.set(key, target[key]);
        }
    }
    const query = searchParams.toString();

    for (const assetId of removedAssetIds) {
        const r = await fetch(`/api/v1/assets/${assetId}${query ? `?${query}` : ""}`, { method: "DELETE" });
        if (!r.ok && r.status !== 404) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail);
        }
    }
}

export async function assets_set_trail_thumbnail(
    trailId: string,
    photo: string | undefined,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
): Promise<void> {
    const asset = photo ? assetIdFromPhotoUrl(photo) : null;
    const r = await f(`/api/v1/trail/${trailId}/thumbnail`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ asset }),
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }
}

export const asset_photo_url = assetPhotoURL;
export const assetIdFromPhotoUrl = assetIdFromPhotoURL;

function generatedAssetKind(file: File): string | undefined {
    return generatedAssetKinds.get(file);
}

function uniquePhotoURLs(photos: string[]): string[] {
    return [...new Set(photos)];
}
