import { flushSync, mount, unmount } from "svelte";
import { addMessages, init } from "svelte-i18n";
import { afterEach, beforeAll, expect, it, vi } from "vitest";
import { Trail } from "#lib/models/trail.ts";
import TrailPage from "./view/[handle]/[id]/+page.svelte";
import MapPage from "../map/trail/[handle]/[id]/+page.svelte";

vi.mock("$app/state", () => ({ page: { params: { handle: "@walker" }, url: new URL("https://a.example/trail/view/shared") } }));
vi.mock("#lib/stores/trail_store.ts", () => ({ trail: {} }));
vi.mock("#lib/components/trail/trail_info_panel.svelte", () => ({ default: () => ({}) }));
vi.mock("#lib/components/trail/map_with_elevation_maplibre.svelte", () => ({ default: () => ({}) }));
vi.mock("maplibre-gl", () => ({}));
vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example" }));
const mounted: ReturnType<typeof mount>[] = [];
beforeAll(() => { addMessages("en", { trail: "Trail", map: "Map" }); init({ fallbackLocale: "en", initialLocale: "en" }); });
afterEach(async () => { for (const component of mounted.splice(0)) await unmount(component); document.body.replaceChildren(); });

it.each(["trail", "map"])("renders %s OpenGraph when photos are absent", (kind) => {
    const trail = new Trail("Shared hike");
    delete trail.photos;
    const target = document.createElement("div"); document.body.append(target);
    const Component = kind === "trail" ? TrailPage : MapPage;
    mounted.push(mount(Component, { target, props: { data: { trail } as never } }));
    flushSync();
    expect(document.head.querySelectorAll('meta[property="og:image"]')).toHaveLength(0);
});

it.each(["trail", "map"])("excludes shared SVG paths from %s OpenGraph", (kind) => {
    const trail = new Trail("Shared hike", { photos: [
        "/api/v1/files/assets/preview/wanderer_route_placeholder_abcdefghij.svg?share=token",
        "/api/v1/files/assets/photo/summit.jpg?share=token",
    ] });
    const target = document.createElement("div"); document.body.append(target);
    const Component = kind === "trail" ? TrailPage : MapPage;
    mounted.push(mount(Component, { target, props: { data: { trail } as never } }));
    flushSync();
    expect([...document.head.querySelectorAll('meta[property="og:image"]')].map((meta) => meta.getAttribute("content")))
        .toEqual(["https://a.example/api/v1/files/assets/photo/summit.jpg?share=token"]);
});
