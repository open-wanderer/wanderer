import type { Asset } from "#lib/models/asset.ts";

export const mediaFileExtensions: Record<string, string> = {
    "image/jpeg": "jpg", "image/png": "png", "image/vnd.mozilla.apng": "png", "image/webp": "webp",
    "image/gif": "gif", "image/avif": "avif", "image/svg+xml": "svg", "image/heic": "heic", "image/heif": "heif",
    "image/bmp": "bmp", "image/x-ms-bmp": "bmp", "image/x-icon": "ico", "image/vnd.microsoft.icon": "ico", "image/tiff": "tif",
    "video/mp4": "mp4", "video/webm": "webm", "video/ogg": "ogg",
};
const mediaExtensions = new Set([...Object.values(mediaFileExtensions), "jpeg", "jpe", "jfif", "apng", "tiff", "m4v", "f4v", "ogv"]);

export function canonicalMediaFilename(candidate: string, contentType?: string): string {
    let name = candidate.replaceAll("\\", "/").split("/").pop()?.trim().replace(/[\x00-\x1f\x7f:*?"<>|]/g, "-") || "photo";
    if (name === "." || name === "..") name = "photo";
    const extension = mediaFileExtensions[contentType?.split(";", 1)[0].trim().toLowerCase() ?? ""];
    if (!extension) return name;
    const dot = name.lastIndexOf(".");
    const existing = dot > 0 ? name.slice(dot + 1).toLowerCase() : "";
    if (mediaExtensions.has(existing)) return `${name.slice(0, dot)}.${extension}`;
    return `${name.replace(/\.$/, "")}.${extension}`;
}

export function assetCopyFilename(asset: Pick<Asset, "id" | "file" | "metadata">, contentType?: string): string {
    const remote = asset.metadata?.remote as { filename?: unknown } | undefined;
    const original = [remote?.filename]
        .find((value): value is string => typeof value === "string" && Boolean(value.trim()));
    // PocketBase appends one random suffix when storing an uploaded file.
    // Re-uploading that stored name would accumulate another suffix per copy.
    const storedName = asset.file?.replace(/_[a-z0-9]{10}(?=\.[^.]+$)/, "");
    return canonicalMediaFilename(original || storedName || asset.id, contentType);
}
