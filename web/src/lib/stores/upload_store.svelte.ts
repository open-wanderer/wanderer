import { APIError } from "$lib/util/api_util";
import { _ } from "svelte-i18n";
import { get } from "svelte/store";
import {
    createBackgroundTaskId,
    removeBackgroundTask,
    upsertBackgroundTask,
    type BackgroundTaskAction,
} from "./background_task_store.svelte";

export type Upload = {
    file: File;
    status: "enqueued" | "uploading" | "cancelled" | "success" | "error" | "duplicate";
    error?: string;
    duplicate?: { id: string, domain: string, name: string };
    progress: number;
    taskId?: string;
    function: (f: File, ignoreDuplicates?: boolean, onProgress?: (p: number) => void) => Promise<unknown>
};

class UploadStore {
    enqueuedUploads: Upload[] = $state([]);
    completedUploads: Upload[] = $state([]);
    uploading: boolean = $state(false);
}

export const uploadStore = new UploadStore();

function t(key: string, values?: Record<string, string | number | boolean | Date | null | undefined>) {
    return get(_)(key, values ? { values } : undefined);
}

function dismissUpload(u: Upload) {
    const index = uploadStore.completedUploads.indexOf(u);
    if (index >= 0) {
        uploadStore.completedUploads.splice(index, 1);
    }
    if (u.taskId) {
        removeBackgroundTask(u.taskId);
    }
}

function cancelUpload(u: Upload) {
    const index = uploadStore.enqueuedUploads.indexOf(u);
    u.status = "cancelled";
    if (index >= 0) {
        uploadStore.enqueuedUploads.splice(index, 1);
    }
    uploadStore.completedUploads.push(u);
    syncUploadTask(u);
}

function reUpload(u: Upload, ignoreDuplicates: boolean = false) {
    const index = uploadStore.completedUploads.indexOf(u);
    u.status = "enqueued";
    u.progress = 0;
    u.error = undefined;
    u.duplicate = undefined;
    if (index >= 0) {
        uploadStore.completedUploads.splice(index, 1);
    }
    uploadStore.enqueuedUploads.push(u);
    syncUploadTask(u);
    processUploadQueue(undefined, ignoreDuplicates);
}

export function syncUploadTask(u: Upload) {
    if (!u.taskId) {
        u.taskId = createBackgroundTaskId("upload");
    }
    const actions: BackgroundTaskAction[] = [];
    if (u.status === "enqueued") {
        actions.push({ label: t("upload-cancel"), icon: "stop", run: () => cancelUpload(u) });
    }
    if (u.status === "error" || u.status === "cancelled") {
        actions.push({ label: t("upload-retry"), icon: "redo", run: () => reUpload(u) });
    }
    if (u.status === "duplicate") {
        actions.push({ label: t("upload-force"), icon: "upload", run: () => reUpload(u, true) });
        if (u.duplicate) {
            actions.push({
                label: u.duplicate.name,
                icon: "external-link",
                href: `/trail/view/@${u.duplicate.domain}/${u.duplicate.id}`,
            });
        }
    }
    upsertBackgroundTask({
        id: u.taskId,
        title: u.file.name,
        status:
            u.status === "uploading"
                ? "running"
                : u.status === "duplicate"
                  ? "warning"
                  : u.status,
        progress: u.progress,
        detail: u.error ?? (u.duplicate ? t("upload-duplicate-detail", { name: u.duplicate.name }) : undefined),
        actions,
        onDismiss: () => dismissUpload(u),
    });
}

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
            syncUploadTask(b);
            uploadPromises.push(
                b.function(b.file, ignoreDuplicates, (p: number) => {
                    b.progress = p
                    syncUploadTask(b);
                })
            );
        }
        const results = await Promise.all(
            uploadPromises.map((p) =>
                p.then(
                    () => ({ ok: true as const }),
                    (error) => ({ ok: false as const, error }),
                ),
            ),
        );
        results.forEach((result, i) => {
            const u = batch[i];
            if (result.ok) {
                u.status = "success"
            } else if (result.error instanceof APIError && result.error.message == "Duplicate trail") {
                u.status = "duplicate"
                u.duplicate = { id: result.error.detail.id, domain: result.error.detail.domain, name: result.error.detail.name };
            } else if (result.error instanceof APIError) {
                u.status = "error"
                u.error = result.error.message
            } else if (result.error instanceof Error) {
                u.status = "error"
                u.error = result.error.message
            } else {
                u.status = "error"
                u.error = String(result.error || "Upload failed")
            }
            uploadStore.completedUploads.push(u);
            syncUploadTask(u);
        });
        uploadStore.enqueuedUploads.splice(0, batchSize)
    }
    uploadStore.uploading = false;
}
