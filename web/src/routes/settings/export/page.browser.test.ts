import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { afterEach, expect, it, vi } from "vitest";
import JSZip from "jszip";
import GPX from "#lib/models/gpx/gpx.ts";
import * as fileUtils from "#lib/util/file_util.ts";
import messages from "#lib/i18n/locales/en.json";
import Page from "./+page.svelte";
import { currentUser } from "#lib/stores/user_store.ts";

vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example", PUBLIC_MAP_MAX_POLYLINES: undefined, PUBLIC_PRIVATE_INSTANCE: undefined }));
vi.mock("#lib/stores/toast_store.svelte.ts", () => ({ show_toast: vi.fn() }));
vi.mock("#lib/stores/theme_store.ts", async () => { const { writable } = await import("svelte/store"); return { theme: writable("light") }; });
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); document.body.replaceChildren(); });

it("exports owned account trails and metadata even when an earlier trail has no GPX file", async () => {
    currentUser.set({ id: "owner", actor: "author" } as never);
    addMessages("en", messages);
    init({ fallbackLocale: "en", initialLocale: "en" });
    const showModal = HTMLDialogElement.prototype.showModal;
    const close = HTMLDialogElement.prototype.close;
    HTMLDialogElement.prototype.showModal = function () { this.open = true; };
    HTMLDialogElement.prototype.close = function () { this.open = false; };
    vi.spyOn(window, "scrollTo").mockImplementation(() => {});
    const downloaded = vi.spyOn(fileUtils, "saveAs").mockImplementation(() => {});
    const request = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input), "http://localhost");
        if (url.pathname === "/api/v1/trail") {
            expect(url.searchParams.get("perPage")).toBe("100");
            expect(url.searchParams.get("filter")).toBe('author = "author"');
            expect(url.searchParams.get("photos")).toBe("false");
            expect(url.searchParams.get("expand")).not.toContain("_assets_");
            expect(url.searchParams.get("expand")?.split(",")).toContain("author");
            return Response.json({ items: [
                { id: "no-gpx", collectionId: "trails", name: "Photo hike", gpx: "", author: "author", expand: { author: { preferred_username: "alice" }, waypoints_via_trail: [{ name: "Summit", lat: 47, lon: 8 }], summit_logs_via_trail: [] } },
                { id: "has-gpx", collectionId: "trails", name: "Tracked hike", gpx: "track.gpx", author: "author", expand: { author: { preferred_username: "alice" }, waypoints_via_trail: [], summit_logs_via_trail: [] } },
            ] });
        }
        if (url.pathname === "/api/v1/files/trails/has-gpx/track.gpx") return new Response('<gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="46" lon="7"/></trkseg></trk></gpx>');
        throw new Error(`Unexpected export request ${url}`);
    });
    vi.stubGlobal("fetch", request);
    const target = document.createElement("div");
    document.body.append(target);
    const component = mount(Page, { target });
    try {
        flushSync();
        const open = [...target.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === messages["export-all-trails"]);
        expect(open).toBeDefined();
        open!.click();
        flushSync();
        const save = target.querySelector<HTMLButtonElement>('button[name="save"]')!;
        save.click();
        await vi.waitFor(() => expect(downloaded).toHaveBeenCalledOnce());
        const blob = downloaded.mock.calls[0][0];
        const bytes = await new Promise<ArrayBuffer>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(reader.result as ArrayBuffer); reader.onerror = reject; reader.readAsArrayBuffer(blob); });
        const zip = await JSZip.loadAsync(bytes);
        expect(Object.keys(zip.files).filter((name) => !zip.files[name].dir)).toEqual(["Photo hike/Photo hike.gpx", "Tracked hike/Tracked hike.gpx"]);
        const photoHike = GPX.parse(await zip.file("Photo hike/Photo hike.gpx")!.async("string"));
        expect(photoHike).not.toBeInstanceOf(Error);
        expect((photoHike as GPX).metadata?.name).toBe("Photo hike");
        expect((photoHike as GPX).wpt?.[0].name).toBe("Summit");
        const tracked = GPX.parse(await zip.file("Tracked hike/Tracked hike.gpx")!.async("string"));
        expect((tracked as GPX).trk?.[0].trkseg?.[0].trkpt?.[0].$.lat).toBe(46);
    } finally {
        await unmount(component);
        HTMLDialogElement.prototype.showModal = showModal;
        HTMLDialogElement.prototype.close = close;
    }
});
