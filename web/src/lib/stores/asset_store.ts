import type { Asset } from "#lib/models/asset.ts";
import type { AttachedAsset } from "#lib/models/attached_asset.ts";
import { assetIdFromPhotoURL, assetPhotoURL } from "#lib/util/asset_link_util.ts";
import type { GPSCoordinates, PhotoExifMetadata } from "#lib/util/exif_util.ts";
import { APIError } from "#lib/util/api_util.ts";

interface AssetTarget {
    trail?: string;
    waypoint?: string;
    summit_log?: string;
}

export type AssetRemovalDeferral = (remove: () => Promise<void>) => void;

interface AssetAttachmentInput {
    files?: File[];
    assetIds?: string[];
    target: AssetTarget;
    existingPhotos?: string[];
    onLinked?: (assets: AttachedAsset[]) => void;
    f?: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>;
}

export interface CompletedAssetAttachments {
    files: boolean;
    uploadedFiles?: number;
    assetIds: boolean;
}

const assetFileMetadata = new WeakMap<File, PhotoExifMetadata>();

export function markAssetFileMetadata(file: File, metadata: PhotoExifMetadata | undefined): File {
    if (metadata?.coordinates) {
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

    const uploaded: Asset[] = [];
    let completedFileCount = 0;
    for (const batch of assetUploadBatches(files)) {
        try {
            const formData = new FormData();
            for (const file of batch) {
                formData.append("files", file);
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
            uploaded.push(...await r.json());
            completedFileCount += batch.length;
        } catch (cause) {
            const error = cause instanceof APIError ? cause : new APIError(502, "asset_import_failed", { cause });
            error.detail = {
                ...error.detail,
                uploadedAssets: [...uploaded, ...(error.detail?.uploadedAssets ?? [])],
                attachedAssets: [...uploaded, ...(error.detail?.attachedAssets ?? [])],
                completedFileCount: completedFileCount + (error.detail?.completedFileCount ?? 0),
            };
            throw error;
        }
    }
    return uploaded;
}

// Leave ample room for multipart framing below the server's 1 GiB limit.
const ASSET_UPLOAD_BATCH_BYTES = 256 * 1024 * 1024;

function* assetUploadBatches(files: File[]): Generator<File[]> {
    let batch: File[] = [];
    let bytes = 0;
    for (const file of files) {
        if (batch.length && bytes + file.size > ASSET_UPLOAD_BATCH_BYTES) {
            yield batch;
            batch = [];
            bytes = 0;
        }
        batch.push(file);
        bytes += file.size;
    }
    if (batch.length) yield batch;
}

export async function assets_link(
    assetIds: string[] | undefined,
    target: AssetTarget,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
): Promise<AttachedAsset[]> {
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
    const response = await r.json();
    if (!r.ok) {
        throw new APIError(r.status, response.message, response.detail);
    }

    return response;
}

export function has_asset_attachments(input: Pick<AssetAttachmentInput, "files" | "assetIds">): boolean {
    return Boolean(
        input.files?.length ||
        input.assetIds?.length,
    );
}

export async function assets_attach_to_target(input: AssetAttachmentInput): Promise<string[]> {
    const f = input.f ?? fetch;
    const photos = [...(input.existingPhotos ?? [])];
    const appendAssets = (assets: Asset[]) => {
        photos.push(...assets.map(asset_photo_url));
    };
    const appendLinkedAssets = (assets: AttachedAsset[]) => {
        input.onLinked?.(assets);
        for (const asset of assets) {
            if (!asset.sourceAssetId) continue;
            for (const [index, photo] of photos.entries()) {
                if (assetIdFromPhotoURL(photo) === asset.sourceAssetId) photos[index] = asset_photo_url(asset);
            }
        }
        appendAssets(assets);
    };
    const completedAttachments: CompletedAssetAttachments = { files: false, assetIds: false };
    try {
        try {
            appendAssets(await assets_create(input.files ?? [], input.target, f));
        } catch (cause) {
            if (cause instanceof APIError) {
                appendAssets(cause.detail?.uploadedAssets ?? []);
                completedAttachments.uploadedFiles = cause.detail?.completedFileCount ?? 0;
            }
            throw cause;
        }
        completedAttachments.files = true;
        try {
            appendLinkedAssets(await assets_link(input.assetIds, input.target, f));
        } catch (cause) {
            if (cause instanceof APIError) appendLinkedAssets(cause.detail?.attachedAssets ?? []);
            throw cause;
        }
        completedAttachments.assetIds = true;
    } catch (cause) {
        const error = cause instanceof APIError ? cause : new APIError(502, "asset_import_failed", { cause });
        error.detail = { ...error.detail, completedAttachments, attachedPhotos: uniquePhotoURLs(photos) };
        throw error;
    }

    return uniquePhotoURLs(photos);
}

export async function assets_delete_removed(
    oldPhotos: string[] | undefined,
    newPhotos: string[] | undefined,
    target: AssetTarget,
): Promise<void> {
    if (newPhotos === undefined) return;
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

function uniquePhotoURLs(photos: string[]): string[] {
    return [...new Set(photos)];
}
