import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { beforeAll, afterEach, expect, it, vi } from "vitest";
import JSZip from "jszip";
import { Trail } from "#lib/models/trail.ts";
import en from "#lib/i18n/locales/en.json";
import { trails_show } from "#lib/stores/trail_store.ts";
import { saveAs } from "#lib/util/file_util.ts";
import Dropdown from "./trail_dropdown.svelte";

const share = "a".repeat(32);
vi.mock("$app/state", () => ({ page: { params: {}, url: new URL("https://a.example/trail/view/shared?share=" + "a".repeat(32)) } }));
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));
vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example", PUBLIC_MAP_MAX_POLYLINES: undefined, PUBLIC_PRIVATE_INSTANCE: undefined }));
vi.mock("$env/dynamic/public", () => ({ env: {} }));
vi.mock("svelte/transition", () => ({ fly: () => ({ duration: 0 }), fade: () => ({ duration: 0 }), slide: () => ({ duration: 0 }) }));
vi.mock("#lib/stores/trail_store.ts", () => ({ trails_show: vi.fn(), trails_update: vi.fn(), trails_delete: vi.fn() }));
vi.mock("#lib/stores/list_store.ts", () => ({ lists_index: vi.fn().mockResolvedValue([]), lists_add_trail: vi.fn(), lists_remove_trail: vi.fn() }));
vi.mock("#lib/util/file_util.ts", async (original) => ({ ...(await original<object>()), saveAs: vi.fn() }));
vi.mock("#lib/stores/plugin_store.ts", async () => ({ hasSendCapablePlugin: (await import("svelte/store")).writable(false) }));
vi.mock("../confirm_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("../list/list_search_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("./trail_send_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("./trail_share_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("./trail_duplicate_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("./trail_merge_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("./trail_bulk_edit_modal.svelte", () => ({ default: () => ({}) }));
vi.mock("./trail_merge_dialog.svelte", () => ({ default: () => ({}) }));

beforeAll(() => { addMessages("en", en); init({ fallbackLocale: "en", initialLocale: "en" }); });
afterEach(() => { vi.restoreAllMocks(); document.body.replaceChildren(); });

it("exports a private shared trail after refreshing photos with the same link token", async () => {
    vi.spyOn(window, "scrollTo").mockImplementation(() => {});
    const showModal = HTMLDialogElement.prototype.showModal;
    const close = HTMLDialogElement.prototype.close;
    HTMLDialogElement.prototype.showModal = function () { this.open = true; };
    HTMLDialogElement.prototype.close = function () { this.open = false; };
    const trail = new Trail("Shared hike", { id: "trail0000000001", gpx: "track.gpx" });
    trail.expand = { author: { preferred_username: "walker" } as never };
    vi.mocked(trails_show).mockImplementation(async (_id, _handle, token, loadGPX) => {
        if (token !== share) throw new Error("Share token required");
        return { ...trail, expand: { ...trail.expand, waypoints_via_trail: [],
            ...(loadGPX ? { gpx_data: '<gpx version="1.1"><trk><trkseg><trkpt lat="47" lon="8"/></trkseg></trk></gpx>' } : {}),
        } };
    });
    const target = document.createElement("div"); document.body.append(target);
    const component = mount(Dropdown, { target, props: { mode: "overview", trails: new Set([trail]) } });
    try {
        flushSync();
        target.querySelector<HTMLButtonElement>('button[aria-label="Open dropdown"]')!.click(); flushSync();
        const action = [...target.querySelectorAll<HTMLElement>("li.menu-item")].find((item) => item.textContent?.trim() === en.export);
        expect(action).toBeDefined(); action!.click(); flushSync();
        const checkbox = target.querySelector<HTMLInputElement>('input[id$="-include-photos-checkbox"]')!;
        checkbox.checked = true; checkbox.dispatchEvent(new Event("change", { bubbles: true })); flushSync();
        target.querySelector<HTMLButtonElement>('dialog[open] button[name="save"]')!.click(); flushSync();
        await vi.waitFor(() => expect(trails_show).toHaveBeenCalledExactlyOnceWith(trail.id, "@walker", share, true));
        await vi.waitFor(() => expect(saveAs).toHaveBeenCalledExactlyOnceWith(expect.any(Blob), "Shared hike.zip"));
        const blob = vi.mocked(saveAs).mock.calls[0][0] as Blob;
        const bytes = await new Promise<ArrayBuffer>((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => resolve(reader.result as ArrayBuffer);
            reader.onerror = reject;
            reader.readAsArrayBuffer(blob);
        });
        const zip = await JSZip.loadAsync(bytes);
        expect(await zip.file("Shared hike.gpx")!.async("string")).toContain('<trkpt lat="47" lon="8"');
    } finally {
        await unmount(component);
        HTMLDialogElement.prototype.showModal = showModal;
        HTMLDialogElement.prototype.close = close;
    }
});
