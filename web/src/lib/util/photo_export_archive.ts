import { get } from "svelte/store";
import { _ } from "svelte-i18n";
import type JSZip from "jszip";
import { fetchPhotoForExport, getFileURL, photoExportFilename } from "./file_util";

export interface MissingExportPhoto { trail: string; filename: string }

// Both single-trail and account exports use this loop. Photo failures cannot
// discard the GPX/JSON and the photos already written to the archive.
export async function appendExportPhotos(
    folder: JSZip | null | undefined,
    trail: { name: string; photos?: string[]; collectionId?: string; id?: string },
    usedNames = new Set<string>(),
    request: typeof fetch = fetch,
): Promise<MissingExportPhoto[]> {
    if (!folder) throw new Error("Photo export folder unavailable");
    const missing: MissingExportPhoto[] = [];
    for (const photo of trail.photos ?? []) {
        try {
            const blob = await fetchPhotoForExport(getFileURL(trail, photo), request);
            folder.file(photoExportFilename(photo, blob.type, usedNames), new Uint8Array(await blob.arrayBuffer()));
        } catch {
            missing.push({ trail: trail.name, filename: photoExportFilename(photo) });
        }
    }
    return missing;
}

export function missingExportPhotoNames(missing: MissingExportPhoto[]): string {
    const names = missing.slice(0, 5).map((photo) => {
        const filename = /^[a-z0-9]{15}\.bin$/.test(photo.filename) ? `${get(_)("export-photo-name-unavailable")} (${photo.filename.slice(0, 15)})` : photo.filename;
        return `${photo.trail}: ${filename}`.slice(0, 120);
    }).join("; ");
    return missing.length > 5 ? `${names}; ${get(_)("photo-names-more", { values: { n: missing.length - 5 } })}` : names;
}

export function addMissingPhotoReport(zip: JSZip, missing: MissingExportPhoto[]): void {
    if (missing.length) zip.file("missing-photos.txt", missing.map((photo) => `${photo.trail}: ${photo.filename}`).join("\n") + "\n");
}
