import { flushSync, mount, unmount } from "svelte";
import { fromStore, writable } from "svelte/store";
import { addMessages, init } from "svelte-i18n";
import { afterEach, beforeAll, expect, it, vi } from "vitest";
import { SummitLog } from "#lib/models/summit_log.ts";
import { Waypoint } from "#lib/models/waypoint.ts";
import * as files from "#lib/util/file_util.ts";
import SummitLogCard from "../summit_log/summit_log_card.svelte";
import WaypointCard from "../waypoint/waypoint_card.svelte";

vi.mock("$app/env/public", () => ({
    PUBLIC_POCKETBASE_URL: "https://a.example",
    PUBLIC_MAP_MAX_POLYLINES: undefined,
    PUBLIC_PRIVATE_INSTANCE: undefined,
}));

vi.mock("#lib/stores/theme_store.ts", async () => {
    const { writable } = await import("svelte/store");
    return { theme: writable("light") };
});

const mounted: ReturnType<typeof mount>[] = [];
beforeAll(() => {
    addMessages("en", { edit: "Edit", delete: "Delete" });
    init({ fallbackLocale: "en", initialLocale: "en" });
});
afterEach(async () => {
    await Promise.all(mounted.splice(0).map((component) => unmount(component)));
    document.body.replaceChildren();
    vi.restoreAllMocks();
});
function target() {
    const element = document.createElement("div");
    document.body.append(element);
    return element;
}
function photos(count = 30) {
    return Array.from({ length: count }, (_, index) => new File(["photo"], `photo-${index}.jpg`));
}

it("reads one upload for a summit log thumbnail", async () => {
    const reader = vi.spyOn(files, "readAsDataURLAsync").mockResolvedValue("data:photo");
    const log = new SummitLog("2026-10-03");
    log._photos = photos();
    const element = target();
    mounted.push(mount(SummitLogCard, { target: element, props: { log, mode: "edit" } }));
    flushSync();
    await vi.waitFor(() => expect(element.querySelector("#header-img")?.getAttribute("src")).toBe("data:photo"));
    expect(reader).toHaveBeenCalledExactlyOnceWith(log._photos[0]);
});

it("keeps the summit log thumbnail while a replacement file preview is pending", async () => {
    let finishReplacement!: (preview: string) => void;
    const reader = vi.spyOn(files, "readAsDataURLAsync")
        .mockResolvedValueOnce("data:first-photo")
        .mockImplementationOnce(() => new Promise((resolve) => { finishReplacement = resolve; }));
    const log = new SummitLog("2026-10-03");
    log._photos = [new File(["first"], "first.jpg")];
    const storedLog = writable(log);
    const value = fromStore(storedLog);
    const element = target();
    mounted.push(mount(SummitLogCard, { target: element, props: { get log() { return value.current; }, mode: "edit" } }));
    flushSync();
    await vi.waitFor(() => expect(element.querySelector("#header-img")?.getAttribute("src")).toBe("data:first-photo"));

    const replacement = new SummitLog(log.date);
    replacement._photos = [new File(["replacement"], "replacement.jpg")];
    storedLog.set(replacement);
    flushSync();

    expect(reader).toHaveBeenCalledTimes(2);
    expect(element.querySelector("#header-img")?.getAttribute("src")).toBe("data:first-photo");
    finishReplacement("data:replacement-photo");
    await vi.waitFor(() => expect(element.querySelector("#header-img")?.getAttribute("src")).toBe("data:replacement-photo"));
});

it("reads only the three visible waypoint uploads and shares previews between cards", async () => {
    const reader = vi.spyOn(files, "readAsDataURLAsync").mockImplementation(async (file) => `data:${file.name}`);
    const waypoint = new Waypoint(47, 8);
    waypoint._photos = photos(5);
    const first = target();
    mounted.push(mount(WaypointCard, { target: first, props: { waypoint, mode: "edit" } }));
    flushSync();
    await vi.waitFor(() => expect(first.querySelectorAll("img")).toHaveLength(3));
    expect([...first.querySelectorAll("img")].map((image) => image.getAttribute("src"))).toEqual([
        "data:photo-0.jpg", "data:photo-1.jpg", "data:photo-2.jpg",
    ]);
    expect(reader.mock.calls.map(([file]) => file)).toEqual(waypoint._photos.slice(0, 3));
    const second = target();
    mounted.push(mount(WaypointCard, { target: second, props: { waypoint, mode: "edit" } }));
    flushSync();
    await vi.waitFor(() => expect(second.querySelectorAll("img")).toHaveLength(3));
    expect([...second.querySelectorAll("img")].map((image) => image.getAttribute("src"))).toEqual([
        "data:photo-0.jpg", "data:photo-1.jpg", "data:photo-2.jpg",
    ]);
    expect(reader).toHaveBeenCalledTimes(3);
});
