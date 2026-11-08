import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { addMessages, init } from "svelte-i18n";
import en from "#lib/i18n/locales/en.json";
import { backgroundTaskStore } from "./background_task_store.svelte.ts";
import { enqueueUpload, processUploadQueue, uploadStore, type Upload } from "./upload_store.svelte.ts";

beforeAll(() => { addMessages("en", en); init({ initialLocale: "en", fallbackLocale: "en" }); });
beforeEach(() => { vi.useFakeTimers(); uploadStore.enqueuedUploads = []; uploadStore.completedUploads = []; uploadStore.uploading = false; backgroundTaskStore.tasks = []; });
afterEach(() => { backgroundTaskStore.tasks.forEach((task) => task.onDismiss?.()); vi.useRealTimers(); vi.unstubAllGlobals(); });

it("shows every queued upload immediately and cancels a waiting upload before it starts", async () => {
    const upload = vi.fn(async (_file: File) => {});
    const queued = Array.from({ length: 7 }, (_, i) => ({ file: new File(["gpx"], `hike-${i}.gpx`), status: "enqueued" as const, progress: 0, function: upload }));
    queued.forEach(enqueueUpload);
    expect(backgroundTaskStore.tasks).toHaveLength(7);
    const last = backgroundTaskStore.tasks.at(-1)!;
    expect(last.actions?.some(action => action.label === en["upload-cancel"])).toBe(true);
    await last.actions!.find(action => action.label === en["upload-cancel"])!.run!();
    expect(uploadStore.enqueuedUploads).toHaveLength(6);
    expect(uploadStore.completedUploads[0].status).toBe("cancelled");
    expect(upload).not.toHaveBeenCalled();
    await processUploadQueue();
    expect(upload).toHaveBeenCalledTimes(6);
    expect(upload.mock.calls.flat()).not.toContain(queued[6].file);
});

it.each([new Error("Upload interrupted"), "Upload interrupted", undefined])("reports a rejected upload as an error", async (failure) => {
    const entry: Upload = { file: new File(["gpx"], "outing.gpx"), status: "enqueued", progress: 0, function: vi.fn().mockRejectedValue(failure) };
    enqueueUpload(entry);
    await processUploadQueue();
    expect(uploadStore.completedUploads[0].status).toBe("error");
    expect(backgroundTaskStore.tasks[0].status).toBe("error");
    expect(backgroundTaskStore.tasks[0].detail).toBe(failure === undefined ? "Upload failed" : "Upload interrupted");
});
