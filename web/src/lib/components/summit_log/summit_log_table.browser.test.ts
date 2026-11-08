import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { beforeAll, expect, it, vi } from "vitest";
import { SummitLog } from "#lib/models/summit_log.ts";
import { currentUser } from "#lib/stores/user_store.ts";
import en from "#lib/i18n/locales/en.json";
import Table from "./summit_log_table.svelte";

vi.mock("#lib/util/maplibre_worker.ts", () => ({}));
vi.mock("#lib/components/trail/map_with_elevation_maplibre.svelte", () => ({ default: () => ({}) }));
vi.mock("$env/dynamic/public", () => ({ env: {} }));
vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example", PUBLIC_MAP_MAX_POLYLINES: undefined, PUBLIC_PRIVATE_INSTANCE: undefined }));
vi.mock("svelte/transition", () => ({ fade: () => ({ duration: 0 }), fly: () => ({ duration: 0 }), slide: () => ({ duration: 0 }) }));
vi.mock("#lib/stores/theme_store.ts", async () => ({ theme: (await import("svelte/store")).writable("light") }));
beforeAll(() => { addMessages("en", en); init({ fallbackLocale: "en", initialLocale: "en" }); });

it("renders and edits the oldest entry when its photo expansion is absent", async () => {
    currentUser.set({ id: "user", actor: "actor" } as never);
    const empty = { ...new SummitLog("2026-10-01"), id: "oldest", author: "actor", photos: undefined } as unknown as SummitLog;
    const later = { ...new SummitLog("2026-10-02"), id: "later", author: "actor", photos: ["/api/v1/assets/photo/file"] } as SummitLog;
    const target = document.createElement("div"); document.body.append(target);
    const edit = vi.fn();
    let component: ReturnType<typeof mount> | undefined;
    try {
        component = mount(Table, { target, props: { summitLogs: [empty, later], handle: "alice", showPhotos: true, showMenu: true, onedit: edit } });
        flushSync();
        const menus = target.querySelectorAll<HTMLButtonElement>('button[aria-label="Open dropdown"]');
        expect(menus).toHaveLength(2);
        menus[0].click(); flushSync();
        const editButton = [...document.querySelectorAll<HTMLElement>("li.menu-item")].find(button => button.textContent?.trim() === en.edit);
        expect(editButton).toBeDefined();
        editButton!.click(); flushSync();
        expect(edit).toHaveBeenCalledWith(empty);
    } finally {
        if (component) await unmount(component);
        target.remove(); currentUser.set(null);
    }
});
