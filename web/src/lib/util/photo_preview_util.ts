import { readAsDataURLAsync } from "./file_util";

const filePreviews = new WeakMap<File, Promise<string>>();

export function waypointFilePreviews(files: File[]): Promise<string[]> {
    return Promise.all(files.slice(0, 3).map(photoFilePreview));
}

export function photoFilePreview(file: File): Promise<string> {
    let preview = filePreviews.get(file);
    if (!preview) {
        preview = readAsDataURLAsync(file).catch((error) => {
            filePreviews.delete(file);
            throw error;
        });
        filePreviews.set(file, preview);
    }
    return preview;
}
