// Keep uploads aligned with the assets file field. HEIF variants are converted
// to JPEG by both device-photo upload paths before uploading.
const assetUploadTypes = new Set([
    "image/jpeg", "image/png", "image/vnd.mozilla.apng", "image/webp", "image/svg+xml",
    "image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence",
    "video/mp4", "video/webm", "video/ogg",
]);
const assetUploadExtensions = [".jpg", ".jpeg", ".png", ".apng", ".webp", ".svg", ".heic", ".heif", ".mp4", ".webm", ".ogg"];
export const ASSET_UPLOAD_ACCEPT = [...assetUploadTypes, ...assetUploadExtensions].join(",");

export function isSupportedAssetUpload(file: Pick<File, "name" | "type">): boolean {
    if (file.type && file.type !== "application/octet-stream") {
        return assetUploadTypes.has(file.type.toLowerCase());
    }
    const name = file.name.toLowerCase();
    return assetUploadExtensions.some((extension) => name.endsWith(extension));
}

export function isAssetUploadImage(file: Pick<File, "name" | "type">): boolean {
    return file.type.toLowerCase().startsWith("image/") || /\.(jpe?g|png|apng|webp|svg|heic|heif)$/i.test(file.name);
}

export async function convertAssetUploadFile(file: File): Promise<File> {
    const heifTypes = ["image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence"];
    if (!heifTypes.includes(file.type.toLowerCase()) && !/\.(heic|heif)$/i.test(file.name)) return file;

    const heic2any = (await import("heic2any")).default;
    const converted = await heic2any({ blob: file, toType: "image/jpeg" });
    const name = file.name.replace(/\.(heic|heif)$/i, ".jpg");
    return new File([Array.isArray(converted) ? converted[0] : converted], name === file.name ? `${name}.jpg` : name, {
        type: "image/jpeg",
        lastModified: file.lastModified,
    });
}
