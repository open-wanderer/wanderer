import { afterEach, beforeAll, expect, it, vi } from "vitest";
import { addMessages, init } from "svelte-i18n";
import en from "#lib/i18n/locales/en.json";
import { backgroundTaskStore } from "./background_task_store.svelte.ts";
import { enqueueUpload, processUploadQueue, uploadStore, type Upload } from "./upload_store.svelte.ts";

beforeAll(() => {
    addMessages("en", en);
    init({ initialLocale: "en", fallbackLocale: "en" });
});
afterEach(() => {
    uploadStore.enqueuedUploads = [];
    uploadStore.completedUploads = [];
    uploadStore.uploading = false;
    backgroundTaskStore.tasks = [];
});

it("does not start a cancelled import after the active batch finishes", async () => {
    let finishActive!: () => void;
    const active = new Promise<void>((resolve) => { finishActive = resolve; });
    const upload = vi.fn((file: File) => file.name === "active.gpx" ? active : Promise.resolve());
    for (const name of ["active.gpx", "waiting.gpx"]) {
        const entry: Upload = { file: new File(["gpx"], name), status: "enqueued", progress: 0, function: upload };
        enqueueUpload(entry);
    }
    const processing = processUploadQueue(1);
    try {
        const waiting = backgroundTaskStore.tasks.find((task) => task.title === "waiting.gpx")!;
        await waiting.actions!.find((action) => action.label === en["upload-cancel"])!.run!();
        expect(uploadStore.enqueuedUploads.map((entry) => entry.file.name)).toEqual(["active.gpx"]);
        expect(uploadStore.completedUploads.map((entry) => ({ name: entry.file.name, status: entry.status })))
            .toEqual([{ name: "waiting.gpx", status: "cancelled" }]);
    } finally {
        finishActive();
        await processing;
    }
    expect(upload.mock.calls.map(([file]) => file.name)).toEqual(["active.gpx"]);
});
