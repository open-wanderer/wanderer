import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { afterEach, expect, it, vi } from "vitest";
import { backgroundTaskStore } from "#lib/stores/background_task_store.svelte.ts";
import { enqueueUpload, processUploadQueue, uploadStore } from "#lib/stores/upload_store.svelte.ts";
import { APIError } from "#lib/util/api_util.ts";
import { goto } from "$app/navigation";
import messages from "#lib/i18n/locales/en.json";
import germanMessages from "#lib/i18n/locales/de.json";
import Dialog from "./background_tasks_dialog.svelte";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

afterEach(() => {
    uploadStore.enqueuedUploads = [];
    uploadStore.completedUploads = [];
    uploadStore.uploading = false;
    backgroundTaskStore.tasks = [];
    document.body.replaceChildren();
    vi.clearAllMocks();
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
        expect(counts!.classList.contains("text-sm")).toBe(true);
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

it.each(["en", "de"])("links a duplicate name inside the %s message without a separate open action", async (language) => {
    const translations = language === "de" ? germanMessages : messages;
    addMessages(language, translations);
    init({ fallbackLocale: "en", initialLocale: language });
    enqueueUpload({
        file: new File(["gpx"], "duplicate.gpx"), status: "enqueued", progress: 0,
        function: () => Promise.reject(new APIError(409, "Duplicate trail", { id: "existing", domain: "tester", name: "Existing trail" })),
    });
    await processUploadQueue();
    const target = document.createElement("div");
    document.body.append(target);
    const component = mount(Dialog, { target });
    try {
        flushSync();
        const row = target.querySelector(".bg-menu-item-background-hover")!;
        const link = row.querySelector<HTMLAnchorElement>("a");
        expect(link).not.toBeNull();
        expect(link!.textContent).toBe("Existing trail");
        expect(link!.getAttribute("href")).toBe("/trail/view/@tester/existing");
        expect(link!.classList.contains("underline")).toBe(true);
        expect(link!.parentElement!.classList.contains("text-amber-400")).toBe(true);
        expect(link!.parentElement!.textContent?.replace(/\s+/g, " ").trim()).toBe(
            language === "de" ? "Duplikat: Existing trail" : "Duplicate: Existing trail",
        );
        link!.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
        expect(goto).toHaveBeenCalledExactlyOnceWith("/trail/view/@tester/existing");
        expect(row.querySelector(".fa-external-link")).toBeNull();
        expect(row.querySelector('button[aria-label="Existing trail"]')).toBeNull();
        expect(backgroundTaskStore.tasks[0].actions).toEqual([
            { label: translations["upload-force"], icon: "upload", run: expect.any(Function) },
        ]);
        expect(row.querySelector(`button[aria-label="${translations["upload-force"]}"] .fa-upload`)).not.toBeNull();
        const dismiss = row.querySelector<HTMLButtonElement>(`button[aria-label="${translations.dismiss}"]`)!;
        expect(dismiss.querySelector(".fa-close")).not.toBeNull();
        dismiss.click();
        expect(backgroundTaskStore.tasks).toEqual([]);
        expect(uploadStore.completedUploads).toEqual([]);
    } finally {
        await unmount(component);
    }
});
