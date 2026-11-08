import { flushSync, mount, tick, unmount } from "svelte";
import { fromStore, get, writable } from "svelte/store";
import { addMessages, init } from "svelte-i18n";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { Trail } from "#lib/models/trail.ts";
import * as files from "#lib/util/file_util.ts";
import Harness, { type PhotoPickerFormApi } from "./photo_picker_form_harness.svelte";
import PhotoPicker from "./photo_picker.svelte";

vi.mock("$app/env/public", () => ({
    PUBLIC_POCKETBASE_URL: "https://a.example",
    PUBLIC_MAP_MAX_POLYLINES: undefined,
    PUBLIC_PRIVATE_INSTANCE: undefined,
}));

const first = "first.jpg";
const second = "second.jpg";
const mounted: ReturnType<typeof mount>[] = [];

beforeAll(() => {
    addMessages("en", { "delete": "Delete", "make-thumbnail": "Make thumbnail" });
    init({ fallbackLocale: "en", initialLocale: "en" });
});

afterEach(async () => {
    await Promise.all(mounted.splice(0).map((component) => unmount(component)));
    document.body.replaceChildren();
    vi.restoreAllMocks();
});

function mountForm() {
    const saved = new Trail("Saved", { id: "trail0000000001", photos: [first, second] });
    const api = {} as PhotoPickerFormApi;
    const target = document.createElement("div");
    document.body.append(target);
    mounted.push(mount(Harness, { target, props: { api, saved } }));
    flushSync();
    api.save();
    flushSync();
    return { api, saved, target };
}

async function removePhoto(target: HTMLElement, index: number) {
    (target.querySelectorAll('button[aria-label="Delete photo"]')[index] as HTMLButtonElement).click();
    flushSync();
    await tick();
}

function mountUploads(uploads: File[]) {
    const formData = writable({ photos: [] as string[], _photos: uploads });
    const values = fromStore(formData);
    const target = document.createElement("div");
    document.body.append(target);
    mounted.push(mount(PhotoPicker, { target, props: {
        id: "uploads",
        get parent() { return values.current; },
        get photos() { return values.current.photos; },
        set photos(photos) { formData.update((value) => ({ ...value, photos })); },
        get photoFiles() { return values.current._photos; },
        set photoFiles(_photos) { formData.update((value) => ({ ...value, _photos: _photos ?? [] })); },
    } }));
    flushSync();
    return {
        target,
        uploads: () => get(formData)._photos,
        replace(next: File[]) {
            formData.update((value) => ({ ...value, _photos: next }));
            flushSync();
        },
    };
}

function controlledPreviews() {
    const pending: { file: File; resolve: (value: string) => void }[] = [];
    const reader = vi.spyOn(files, "readAsDataURLAsync").mockImplementation((file) =>
        new Promise<string>((resolve) => pending.push({ file, resolve })),
    );
    return {
        reader,
        takeReads: () => pending.splice(0),
        async finish(reads: typeof pending) {
            for (const { file, resolve } of reads) resolve(`data:image/jpeg;base64,${file.name}`);
            await new Promise((resolve) => setTimeout(resolve, 0));
            flushSync();
            await tick();
        },
    };
}

describe("PhotoPicker photo removal", () => {
    it("preserves the stored baseline when felte shares the saved photo array", async () => {
        const { api, saved, target } = mountForm();
        expect(api.formPhotos()).toBe(saved.photos);

        await removePhoto(target, 0);

        expect(api.formPhotos()).toEqual([second]);
        expect(api.storedPhotos()).toEqual([first, second]);
    });

    it("removes an upload without mutating a plain caller-owned file array", async () => {
        const files = [new File(["first"], "first.jpg"), new File(["second"], "second.jpg")];
        const target = document.createElement("div");
        document.body.append(target);
        mounted.push(mount(PhotoPicker, { target, props: { id: "plain", parent: {}, photos: [], photoFiles: files } }));
        flushSync();
        await vi.waitFor(() => expect(target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(2));

        await removePhoto(target, 0);

        expect(files).toHaveLength(2);
        expect(files.map((file) => file.name)).toEqual(["first.jpg", "second.jpg"]);
        await vi.waitFor(() => expect(target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(1));
    });

    it("keeps a store-bound picker empty when an older upload read finishes after both deletes", async () => {
        const previews = controlledPreviews();
        const uploads = [new File(["first"], "first.jpg"), new File(["second"], "second.jpg")];
        const form = mountUploads(uploads);
        await previews.finish(previews.takeReads());
        expect(form.target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(2);

        await removePhoto(form.target, 0);
        const olderReads = previews.takeReads();
        await removePhoto(form.target, 0);
        await previews.finish(previews.takeReads());
        await previews.finish(olderReads);

        expect(form.uploads()).toEqual([]);
        expect(form.target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(0);
        expect(previews.reader).toHaveBeenCalledTimes(2);
    });

    it("keeps the remaining store-bound upload aligned when older reads finish last", async () => {
        const previews = controlledPreviews();
        const uploads = [new File(["first"], "first.jpg"), new File(["second"], "second.jpg"), new File(["third"], "third.jpg")];
        const form = mountUploads(uploads);
        await previews.finish(previews.takeReads());
        expect(form.target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(3);

        await removePhoto(form.target, 0);
        const olderReads = previews.takeReads();
        await removePhoto(form.target, 0);
        await previews.finish(previews.takeReads());
        await previews.finish(olderReads);

        expect(form.uploads()).toEqual([uploads[2]]);
        expect(form.target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(1);
        expect(form.target.querySelector('[style*="background-image"]')?.getAttribute("style")).toContain("third.jpg");
        await removePhoto(form.target, 0);
        expect(form.uploads()).toEqual([]);
        expect(previews.reader).toHaveBeenCalledTimes(3);
    });

    it("ignores an earlier upload preview after the store replaces the selection", async () => {
        const previews = controlledPreviews();
        const oldUpload = new File(["old"], "old.jpg");
        const newUpload = new File(["new"], "new.jpg");
        const form = mountUploads([oldUpload]);
        const olderReads = previews.takeReads();

        form.replace([newUpload]);
        await previews.finish(previews.takeReads());
        await previews.finish(olderReads);

        expect(previews.reader.mock.calls.map(([file]) => file)).toEqual([oldUpload, newUpload]);
        expect(form.uploads()).toEqual([newUpload]);
        expect(form.target.querySelectorAll('button[aria-label="Delete photo"]')).toHaveLength(1);
        expect(form.target.querySelector('[style*="background-image"]')?.getAttribute("style")).toContain("new.jpg");
    });
});
