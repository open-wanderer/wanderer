import { APIError } from "$lib/util/api_util";

export type Upload = {
    file: File;
    status: "enqueued" | "uploading" | "cancelled" | "success" | "error" | "duplicate";
    error?: string;
    duplicate?: { id: string, domain: string, name: string, author?: string };
    progress: number;
    function: (f: File, ignoreDuplicates?: boolean, onProgress?: (p: number) => void) => Promise<unknown>
};

class UploadStore {
    enqueuedUploads: Upload[] = $state([]);
    completedUploads: Upload[] = $state([]);
    uploading: boolean = $state(false);
}

export const uploadStore = new UploadStore();


export async function processUploadQueue(batchSize: number = 3, ignoreDuplicates: boolean = false) {
    if (uploadStore.uploading) {
        return;
    }
    uploadStore.uploading = true;

    while (uploadStore.enqueuedUploads.length > 0) {
        const batch = uploadStore.enqueuedUploads.slice(0, batchSize);
        const uploadPromises: Promise<unknown>[] = [];
        for (const b of batch) {
            b.status = "uploading";
            uploadPromises.push(
                b.function(b.file, ignoreDuplicates, (p: number) => {
                    b.progress = p
                })
            );
        }
        const results = await Promise.all(
            uploadPromises.map((p) => p.catch((e) => e)),
        );
        results.forEach((r, i) => {
            const u = batch[i];
            if (r instanceof APIError && r.message == "Duplicate trail") {
                u.status = "duplicate"
                u.duplicate = { id: r.detail.id, domain: r.detail.domain, name: r.detail.name, author: r.detail.author };
            } else if (r instanceof APIError) {
                u.status = "error"
                u.error = r.message
            } else {
                u.status = "success"
            }
            uploadStore.completedUploads.push(u);
        });
        uploadStore.enqueuedUploads.splice(0, batchSize)
    }
    uploadStore.uploading = false;
}
