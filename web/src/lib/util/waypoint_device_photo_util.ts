import { convertAssetUploadFile } from "./asset_upload_util";
import { extractDevicePhotoExif } from "./exif_util";

const devicePhotoTypes = new Set(["image/jpeg", "image/png", "image/vnd.mozilla.apng", "image/webp", "image/heic", "image/heif", "image/heic-sequence", "image/heif-sequence"]);
const devicePhotoExtensions = [".jpg", ".jpeg", ".png", ".apng", ".webp", ".heic", ".heif"];
export const DEVICE_PHOTO_ACCEPT = [...devicePhotoTypes, ...devicePhotoExtensions].join(",");

export function isSupportedDevicePhoto(file: Pick<File, "name" | "type">): boolean {
    if (file.type && file.type !== "application/octet-stream") return devicePhotoTypes.has(file.type.toLowerCase());
    return devicePhotoExtensions.some((extension) => file.name.toLowerCase().endsWith(extension));
}

export async function prepareWaypointDevicePhoto(file: File) {
    if (!isSupportedDevicePhoto(file)) return undefined;
    const { coordinates, metadata, fileType } = await extractDevicePhotoExif(file);
    if (!coordinates) return undefined;
    const contentType = ({ jpeg: "image/jpeg", png: "image/png", webp: "image/webp", heic: "image/heic", heif: "image/heif" } as Record<string, string>)[fileType?.toLowerCase() ?? ""];
    const input = contentType && contentType !== file.type
        ? new File([file], file.name, { type: contentType, lastModified: file.lastModified })
        : file;
    const prepared = contentType && !["image/heic", "image/heif"].includes(contentType)
        ? input : await convertAssetUploadFile(input);
    return { coordinates, metadata, file: prepared };
}

export async function prepareWaypointDevicePhotos(
    files: File[],
    onFailure: (file: File, reason: "unsupported" | "missing_gps" | "conversion") => void,
    onProgress?: (file: File, current: number, total: number) => void,
) {
    const prepared: (NonNullable<Awaited<ReturnType<typeof prepareWaypointDevicePhoto>>> & { index: number })[] = [];
    for (const [index, file] of files.entries()) {
        onProgress?.(file, index + 1, files.length);
        if (!isSupportedDevicePhoto(file)) { onFailure(file, "unsupported"); continue; }
        try {
            const photo = await prepareWaypointDevicePhoto(file);
            if (photo) prepared.push({ ...photo, index });
            else onFailure(file, "missing_gps");
        } catch {
            onFailure(file, "conversion");
        }
    }
    return prepared;
}
