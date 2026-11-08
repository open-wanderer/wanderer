import { flushSync, mount, tick, unmount } from "svelte";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { Trail } from "#lib/models/trail.ts";

const state = vi.hoisted(() => ({ markers: new Set<any>(), maps: [] as any[] }));

vi.mock("maplibre-gl", () => {
    const fallback = <T extends object>(target: T): T => new Proxy(target, {
        get(t, p, r) {
            if (p in t || typeof p === "symbol") return Reflect.get(t, p, r);
            return () => undefined;
        },
    });
    class Evented {
        handlers: Record<string, ((e: unknown) => void)[]> = {};
        on(event: string, a: unknown, b?: unknown) {
            const fn = (typeof a === "function" ? a : b) as (e: unknown) => void;
            if (typeof a !== "function") return this; // layer-scoped handlers are not needed
            (this.handlers[event] ??= []).push(fn);
            return this;
        }
        off() { return this; }
        once(event: string, fn: (e: unknown) => void) { return this.on(event, fn); }
        fire(event: string) { for (const fn of this.handlers[event] ?? []) fn({ target: this }); }
    }
    class FakeMap extends Evented {
        styleLayers = new globalThis.Map<string, unknown>();
        sources = new globalThis.Map<string, unknown>();
        style = {};
        constructor() { super(); const proxy = fallback(this); state.maps.push(proxy); return proxy; }
        loaded() { return true; }
        getCanvas() { return { style: {} }; }
        getStyle() { return { glyphs: "x", layers: [] }; }
        getLayer(id: string) { return this.styleLayers.get(id); }
        addLayer(layer: { id: string }) { this.styleLayers.set(layer.id, layer); return this; }
        removeLayer(id: string) { this.styleLayers.delete(id); return this; }
        getSource(id: string) { return this.sources.get(id); }
        addSource(id: string, source: object) { this.sources.set(id, { ...source, setData() {} }); return this; }
        removeSource(id: string) { this.sources.delete(id); return this; }
        getFilter() { return undefined; }
        getBounds() { return { getNorth: () => 1, getSouth: () => 0, getEast: () => 1, getWest: () => 0, toArray: () => [[0, 0], [1, 1]] }; }
        getCenter() { return { lat: 0, lng: 0 }; }
        getZoom() { return 10; }
        getTerrain() { return null; }
        isStyleLoaded() { return true; }
        getContainer() { return document.createElement("div"); }
    }
    class Marker {
        _element: HTMLElement;
        _lngLat: unknown;
        constructor(options: { element?: HTMLElement } = {}) {
            this._element = options.element ?? document.createElement("div");
            return fallback(this);
        }
        setLngLat(value: unknown) { this._lngLat = value; return this; }
        getLngLat() { return this._lngLat; }
        addTo() { state.markers.add(this); return this; }
        remove() { state.markers.delete(this); return this; }
        getElement() { return this._element; }
        setPopup() { return this; }
        setOpacity() { return this; }
        on() { return this; }
    }
    class Generic { constructor() { return fallback(this); } }
    class LngLatBounds { extend() { return this; } isEmpty() { return true; } toArray() { return [[0, 0], [1, 1]]; } }
    return {
        Map: FakeMap, Marker, Popup: Generic, NavigationControl: Generic, ScaleControl: Generic,
        GeolocateControl: Generic, TerrainControl: Generic, LngLatBounds, LngLat: Generic,
        setWorkerUrl() {}, default: {},
    };
});
vi.mock("#lib/util/maplibre_worker.ts", () => ({}));
vi.mock("$app/state", () => ({ page: { data: { settings: {}, subcategories: [] }, params: {}, url: new URL("https://a.example/") } }));
vi.mock("$app/env/public", () => ({ PUBLIC_POCKETBASE_URL: "https://a.example", PUBLIC_MAP_MAX_POLYLINES: undefined, PUBLIC_PRIVATE_INSTANCE: undefined }));
vi.mock("#lib/stores/theme_store.ts", async () => { const { writable } = await import("svelte/store"); return { theme: writable("light") }; });
vi.mock("#lib/vendor/maplibre-elevation-profile/elevationprofile-control.ts", () => ({ ElevationProfileControl: class {} }));
vi.mock("#lib/vendor/maplibre-style-switcher/style-switcher-control.ts", () => ({ StyleSwitcherControl: class {} }));

import Harness, { type MapMarkerLifecycleApi } from "./map_marker_lifecycle_harness.svelte";

const mounted: ReturnType<typeof mount>[] = [];
beforeEach(() => {
    state.markers.clear();
    state.maps.length = 0;
    localStorage.clear();
});
afterEach(async () => {
    await Promise.all(mounted.splice(0).map((component) => unmount(component)));
    document.body.replaceChildren();
});

const gpx = (lat: number, lon: number) => `<?xml version="1.0"?><gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg><trkpt lat="${lat}" lon="${lon}"></trkpt><trkpt lat="${lat + 0.01}" lon="${lon + 0.01}"></trkpt><trkpt lat="${lat + 0.02}" lon="${lon + 0.02}"></trkpt></trkseg></trk></gpx>`;
function trail(lat: number, lon: number) {
    return new Trail("Trail", { id: "trail0000000001", lat, lon, gpx_data: gpx(lat, lon) });
}

function startEndMarkers() {
    return [...state.markers]
        .filter((marker) => /fa-bullseye|fa-flag-checkered/.test(marker.getElement().innerHTML))
        .map((marker) => ({
            kind: /fa-bullseye/.test(marker.getElement().innerHTML) ? "start" : "end",
            position: marker.getLngLat(),
        }));
}
function pair(lat: number, lon: number) {
    return [
        { kind: "start", position: [lon, lat] },
        { kind: "end", position: [lon + 0.02, lat + 0.02] },
    ];
}
async function settle() {
    for (let i = 0; i < 5; i++) {
        flushSync();
        await tick();
        await new Promise((resolve) => setTimeout(resolve, 0));
    }
}
async function mountMap() {
    const api = {} as MapMarkerLifecycleApi;
    const target = document.createElement("div");
    document.body.append(target);
    mounted.push(mount(Harness, { target, props: { api, initialTrails: [trail(47, 8)] } }));
    await settle();
    state.maps.at(-1).fire("load");
    await settle();
    return api;
}

it("keeps only the current start and end markers through drawing, track changes and clearing", async () => {
    const api = await mountMap();
    expect(startEndMarkers()).toEqual(pair(47, 8));

    api.setDrawing(true);
    await settle();
    expect(startEndMarkers()).toEqual([]);

    api.setDrawing(false);
    await settle();
    expect(startEndMarkers()).toEqual(pair(47, 8));

    api.setTrails([trail(46, 9)]);
    await settle();
    expect(startEndMarkers()).toEqual(pair(46, 9));

    api.setTrails([]);
    await settle();
    expect(startEndMarkers()).toEqual([]);

    api.setTrails([trail(45, 10)]);
    await settle();
    expect(startEndMarkers()).toEqual(pair(45, 10));
});

it("hides the start and end markers while the track is hidden with the m key", async () => {
    await mountMap();
    expect(startEndMarkers()).toEqual(pair(47, 8));

    document.body.dispatchEvent(new KeyboardEvent("keydown", { key: "m", bubbles: true }));
    await settle();
    expect(startEndMarkers()).toEqual([]);

    document.body.dispatchEvent(new KeyboardEvent("keyup", { key: "m", bubbles: true }));
    await settle();
    expect(startEndMarkers()).toEqual(pair(47, 8));
});
