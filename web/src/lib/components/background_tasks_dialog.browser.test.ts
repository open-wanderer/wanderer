import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { afterEach, expect, it, vi } from "vitest";
import { backgroundTaskStore } from "#lib/stores/background_task_store.svelte.ts";
import { enqueueUpload, processUploadQueue, uploadStore } from "#lib/stores/upload_store.svelte.ts";
import { APIError } from "#lib/util/api_util.ts";
import messages from "#lib/i18n/locales/en.json";
import Dialog from "./background_tasks_dialog.svelte";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

afterEach(() => {
    uploadStore.enqueuedUploads = [];
    uploadStore.completedUploads = [];
    uploadStore.uploading = false;
    backgroundTaskStore.tasks = [];
    document.body.replaceChildren();
});

it("keeps mixed import result counts in the header when minimized", async () => {
    addMessages("en", messages);
    init({ fallbackLocale: "en", initialLocale: "en" });
    const outcomes = [
        { name: "success.gpx", run: () => Promise.resolve() },
        { name: "duplicate.gpx", run: () => Promise.reject(new APIError(409, "Duplicate trail", { id: "duplicate", domain: "tester", name: "Existing trail" })) },
        { name: "invalid.gpx", run: () => Promise.reject(new APIError(400, "Invalid GPX")) },
        { name: "offline.gpx", run: () => Promise.reject(new TypeError("Failed to fetch")) },
    ];
    for (const { name, run } of outcomes) {
        enqueueUpload({ file: new File(["gpx"], name), status: "enqueued", progress: 0, function: run });
    }
    await processUploadQueue();
    backgroundTaskStore.tasks.push(
        { id: "upload-cancelled", title: "Cancelled import", status: "cancelled" },
        { id: "other-success", title: "Other completed work", status: "success" },
        { id: "other-error", title: "Other failed work", status: "error" },
        { id: "other-warning", title: "Other warning", status: "warning" },
    );
    const target = document.createElement("div");
    document.body.append(target);
    const component = mount(Dialog, { target });
    try {
        flushSync();
        expect(uploadStore.completedUploads.map((upload) => upload.status)).toEqual(["success", "duplicate", "error", "error"]);
        const counts = target.querySelector("p > span.text-emerald-400")?.parentElement;
        expect(counts).toBeTruthy();
        expect(counts!.textContent?.replace(/\s+/g, " ").trim()).toBe("Uploaded 1 - Error 2 - Duplicates 1");
        expect([...counts!.querySelectorAll("span")].map((span) => ({ count: span.textContent, color: span.className }))).toEqual([
            { count: "1", color: "text-emerald-400" },
            { count: "2", color: "text-red-400" },
            { count: "1", color: "text-amber-500" },
        ]);
        const list = target.querySelector(".max-h-96")!;
        expect(list.classList.contains("hidden")).toBe(false);
        target.querySelector<HTMLButtonElement>(`button[aria-label="${messages.minimize}"]`)!.click();
        flushSync();
        expect(list.classList.contains("hidden")).toBe(true);
        expect(counts!.isConnected).toBe(true);
        expect(list.contains(counts!)).toBe(false);
        expect(counts!.textContent?.replace(/\s+/g, " ").trim()).toBe("Uploaded 1 - Error 2 - Duplicates 1");
    } finally {
        await unmount(component);
    }
});
