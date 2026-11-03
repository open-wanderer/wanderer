<script module lang="ts">
    export interface MapCamera { center: [number, number]; zoom: number; bearing: number; pitch: number }
</script>

<script lang="ts">
    import type { Snippet } from "svelte";
    import { onMount, tick } from "svelte";
    import { _ } from "svelte-i18n";
    import type { FeatureCollection } from "geojson";
    import * as M from "maplibre-gl";
    import type { Settings } from "#lib/models/settings.ts";
    import type { Trail, TrailBoundingBox, TrailFilter } from "#lib/models/trail.ts";
    import type { TrailSearchBounds } from "#lib/util/trail_search_util.ts";
    import { ResetViewControl } from "#lib/vendor/maplibre-reset-view/reset-view-control.ts";
    import SkeletonCard from "../base/skeleton_card.svelte";
    import SkeletonListItem from "../base/skeleton_list_item.svelte";
    import EmptyStateSearch from "../empty_states/empty_state_search.svelte";
    import MapWithElevationMaplibre from "./map_with_elevation_maplibre.svelte";
    import TrailResultLink from "./trail_result_link.svelte";
    import MergeDialog from "./trail_merge_dialog.svelte";
    import TrailCard from "./trail_card.svelte";
    import TrailDropdown from "./trail_dropdown.svelte";
    import TrailListItem from "./trail_list_item.svelte";

    interface Props {
        trails: Trail[];
        mapTrails: Trail[];
        clusters?: FeatureCollection;
        loading: boolean;
        busy?: boolean;
        totalHits?: number;
        display?: "cards" | "list";
        onupdate?: (updatedTrails?: Trail[]) => void;
        mapLoading?: boolean;
        initialBounds?: TrailSearchBounds;
        initialCenter?: { lat: number; lon: number };
        initialCamera?: MapCamera;
        boundingBox: TrailBoundingBox;
        settings?: Settings;
        near: TrailFilter["near"];
        onviewport: (bounds: TrailSearchBounds, zoom: number) => void;
        toolbar: Snippet;
        moreResults: Snippet;
    }
    let { trails, mapTrails, clusters, loading, busy = false, totalHits, display = "cards", onupdate, mapLoading = false,
        initialBounds, initialCenter, initialCamera, boundingBox, settings, near, onviewport, toolbar, moreResults }: Props = $props();
    const id = $props.id();
    // Space between the floating panels and the area that counts as visible.
    const insetGap = 16;
    let root: HTMLDivElement;
    let mapElement: HTMLDivElement;
    let panel: HTMLElement;
    let toolbarElement: HTMLDivElement;
    let map: M.Map | undefined = $state();
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let ready = false;
    let positioning = false;
    let defaultCamera: MapCamera | undefined;
    let locationRequest = 0;
    let pendingLocation: { lat: number; lon: number } | undefined;
    // Phones show either the map or the list, which then covers the map.
    let listShown = $state(false);
    // Wide screens can fold the list down to its header.
    let collapsed = $state(false);
    let selectedIds = $state(new Set<string>());
    let selection = $derived(new Set(trails.filter(trail => trail.id && selectedIds.has(trail.id))));
    let allSelected = $derived(trails.length > 0 && selection.size === trails.length);
    let resetViewControl: ResetViewControl | undefined = $state.raw();
    const resetViewAppearance = $derived(settings?.mapFocus === "location" && settings.location
        ? { label: $_("trail-map-reset-location"), icon: "fa-location-dot" }
        : { label: $_("trail-map-reset-view"), icon: "fa-arrows-to-circle" });

    // Keep the selection tied to loaded results, including fresh records after a refresh.
    $effect(() => {
        const visibleIds = new Set(trails.map(trail => trail.id));
        const kept = [...selectedIds].filter(id => visibleIds.has(id));
        if (kept.length !== selectedIds.size) selectedIds = new Set(kept);
    });

    $effect(() => {
        resetViewControl?.setAppearance(resetViewAppearance);
    });

    onMount(() => {
        try { collapsed = localStorage.getItem("mapResultsCollapsed") === "true"; } catch { /* Start expanded without storage. */ }
        const updateTop = () => {
            // A modal dialog fixes the body and moves its scroll offset into `top`.
            const scroll = document.body.style.position === "fixed" ? -parseFloat(document.body.style.top || "0") : window.scrollY;
            root.style.setProperty("--map-top", `${Math.round(root.getBoundingClientRect().top + scroll)}px`);
        };
        updateTop();
        window.addEventListener("resize", updateTop);
        // Phones start the list below the floating search bar.
        const toolbarSize = new ResizeObserver(() => root.style.setProperty("--toolbar-height", `${toolbarElement.offsetHeight}px`));
        toolbarSize.observe(toolbarElement);
        // Selection actions and their dialogs live in the list, so keep it open when a window narrows.
        const phone = window.matchMedia("(width < 768px)");
        const showSelection = () => { if (phone.matches && selection.size) listShown = true; };
        phone.addEventListener("change", showSelection);
        return () => {
            phone.removeEventListener("change", showSelection);
            ready = false;
            ++locationRequest;
            window.removeEventListener("resize", updateTop);
            toolbarSize.disconnect();
        };
    });

    function toggleSelection(trail?: Trail) {
        if (!trail) {
            selectedIds = allSelected ? new Set() : new Set(trails.flatMap(trail => trail.id ? [trail.id] : []));
            return;
        }
        if (!trail.id) return;
        const next = new Set(selectedIds);
        if (next.has(trail.id)) next.delete(trail.id);
        else next.add(trail.id);
        selectedIds = next;
    }
    async function clearSelection() {
        selectedIds = new Set();
        await tick();
        panel.querySelector<HTMLElement>('.trail-results a, .trail-results input[type="checkbox"]')?.focus();
    }
    function handleTrailClick(event: MouseEvent, trail: Trail) {
        if (!selection.size) return;
        event.preventDefault();
        toggleSelection(trail);
    }
    function hoverTrail(trail?: Trail) {
        if (trail) mapWithElevation?.highlightCluster(trail);
        else mapWithElevation?.unHighlightCluster();
    }
    function handleTrailsUpdated(updatedTrails?: Trail[]) {
        selectedIds = new Set();
        onupdate?.(updatedTrails);
    }
    function trailHref(trail: Trail) {
        return `/map/trail/@${trail.author}${trail.domain ? `@${trail.domain}` : ""}/${trail.id}`;
    }

    /** The parts of the map covered by the results panel and the search bar. */
    function insets() {
        const area = mapElement.getBoundingClientRect();
        const covered = (element: Element, edge: "right" | "bottom") => {
            const box = element.getBoundingClientRect();
            const overlaps = box.width && box.height && box.left < area.right && box.right > area.left && box.top < area.bottom && box.bottom > area.top;
            if (!overlaps) return 0;
            return edge === "right" ? box.right - area.left + insetGap : box.bottom - area.top + insetGap;
        };
        // On phones the list covers the whole map while it is shown; it never narrows the map area.
        const wide = window.matchMedia("(width >= 768px)").matches;
        // A collapsed list only covers the corner beside the search bar.
        return { left: wide && !collapsed ? covered(panel, "right") : 0, top: covered(toolbarElement, "bottom") };
    }
    function fitPadding(margin: number) {
        const { left, top } = insets();
        return { left: left + margin, top: top + margin, right: margin, bottom: margin };
    }

    /** The bounds of the map area that is not covered by the floating panels. */
    function visibleBounds(): TrailSearchBounds | undefined {
        if (!map) return;
        const { left, top } = insets();
        const { clientWidth: width, clientHeight: height } = mapElement;
        const corners = [[left, top], [width, top], [width, height], [left, height]]
            .map(([x, y]) => map!.unproject([x, y]));
        const longitudes = corners.map(corner => corner.lng);
        const latitudes = corners.map(corner => corner.lat);
        const west = Math.min(...longitudes);
        const east = Math.max(...longitudes);
        const normalize = (longitude: number) => ((((longitude + 180) % 360) + 360) % 360) - 180;
        const global = east - west >= 360;
        return {
            northEast: { lat: Math.min(90, Math.max(...latitudes)), lng: global ? 180 : normalize(east) },
            southWest: { lat: Math.max(-90, Math.min(...latitudes)), lng: global ? -180 : normalize(west) },
        };
    }
    function notifyViewport() {
        if (!ready || !map || positioning) return;
        const bounds = visibleBounds();
        if (bounds) onviewport(bounds, map.getZoom());
    }
    /** Move the camera without publishing intermediate areas, then publish the final one. */
    function position(move: () => void) {
        positioning = true;
        try { move(); }
        finally { positioning = false; }
        notifyViewport();
    }

    export function focusBounds(bounds: TrailSearchBounds) {
        if (!ready || !map) return false;
        // MapLibre needs an unwrapped east longitude for antimeridian views.
        const east = bounds.northEast.lng < bounds.southWest.lng ? bounds.northEast.lng + 360 : bounds.northEast.lng;
        position(() => map!.fitBounds([[bounds.southWest.lng, bounds.southWest.lat], [east, bounds.northEast.lat]],
            { animate: false, padding: fitPadding(0), bearing: map!.getBearing(), pitch: map!.getPitch() }));
        return true;
    }
    export function focusPlace(place: TrailFilter["near"]) {
        if (!ready || !map || !Number.isFinite(place.lat) || !Number.isFinite(place.lon)) return;
        listShown = false;
        position(() => map!.fitBounds(M.LngLatBounds.fromLngLat(new M.LngLat(place.lon!, place.lat!), place.radius),
            { padding: fitPadding(32), maxZoom: 14, animate: false }));
    }
    export function focusLocation(location: { lat: number; lon: number }) {
        if (!Number.isFinite(location.lat) || !Number.isFinite(location.lon)) return;
        ++locationRequest;
        listShown = false;
        if (!ready || !map) {
            pendingLocation = location;
            return;
        }
        position(() => centerOn(location, 14));
    }
    export function captureView(): MapCamera | undefined {
        if (!ready || !map) return;
        return { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
    }
    export function focusCamera(camera: MapCamera) {
        if (!ready || !map) return false;
        position(() => map!.jumpTo(camera));
        return true;
    }

    function centerOn(location: { lat: number; lon: number }, zoom: number) {
        map?.jumpTo({ center: [location.lon, location.lat], zoom });
    }

    function handleMapInit() {
        ready = true;
        if (map && !resetViewControl) {
            resetViewControl = new ResetViewControl(() => position(focusDefaultViewport), resetViewAppearance);
            map.addControl(resetViewControl, "top-right");
        }
        if (map) defaultCamera = { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
        position(() => {
            if (pendingLocation) centerOn(pendingLocation, 14);
            else if (initialCamera) map?.jumpTo(initialCamera);
            else if (initialBounds) {
                const east = initialBounds.northEast.lng < initialBounds.southWest.lng ? initialBounds.northEast.lng + 360 : initialBounds.northEast.lng;
                map?.fitBounds([[initialBounds.southWest.lng, initialBounds.southWest.lat], [east, initialBounds.northEast.lat]],
                    { animate: false, padding: fitPadding(0) });
            }
            else if (initialCenter) centerOn(initialCenter, 14);
            else if (Number.isFinite(near.lat) && Number.isFinite(near.lon)) {
                map?.fitBounds(M.LngLatBounds.fromLngLat(new M.LngLat(near.lon!, near.lat!), near.radius),
                    { padding: fitPadding(32), maxZoom: 14, animate: false });
            }
            else focusDefaultViewport();
            pendingLocation = undefined;
        });
    }

    function focusDefaultViewport() {
        // Ignore URL/snapshot bounds and place filters when explicitly returning
        // to the default view. Keep the search criteria themselves unchanged.
        map?.jumpTo({ bearing: 0, pitch: 0 });
        const request = ++locationRequest;
        const hasTrails = boundingBox.has_trails ?? (boundingBox.min_lon !== 0 || boundingBox.max_lon !== 0 || boundingBox.min_lat !== 0 || boundingBox.max_lat !== 0);
        if (settings?.mapFocus === "trails" && hasTrails) {
            if (boundingBox.min_lon === boundingBox.max_lon && boundingBox.min_lat === boundingBox.max_lat) {
                centerOn({ lat: boundingBox.min_lat, lon: boundingBox.min_lon }, 12);
            } else {
                map?.fitBounds([[boundingBox.min_lon, boundingBox.min_lat], [boundingBox.max_lon, boundingBox.max_lat]], { animate: false, padding: fitPadding(32) });
            }
        } else if (settings?.mapFocus === "location" && settings.location) {
            centerOn(settings.location, 12);
        } else {
            if (defaultCamera) map?.jumpTo(defaultCamera);
            navigator.geolocation?.getCurrentPosition(location => {
                if (!ready || !map || request !== locationRequest || map.isMoving()) return;
                position(() => centerOn({ lat: location.coords.latitude, lon: location.coords.longitude }, 12));
            }, () => { /* Keep the default camera when location is unavailable. */ });
        }
    }

    async function toggleCollapsed() {
        collapsed = !collapsed;
        try { localStorage.setItem("mapResultsCollapsed", String(collapsed)); } catch { /* Keep the choice for this visit. */ }
        await tick();
        // The search area grows or shrinks with the list.
        notifyViewport();
    }

    function handleMoveEnd() {
        // Any camera change, including the user's, supersedes a pending location request.
        if (!positioning) ++locationRequest;
        notifyViewport();
    }
</script>

{#snippet selectionTrigger({ toggleMenu, loading }: { toggleMenu: (event: MouseEvent) => void; loading: boolean })}
    <button type="button" class="btn-icon selection-menu" aria-label={$_("trail-results-selection-actions")}
        aria-busy={loading} disabled={loading} onclick={toggleMenu}>
        <i class={`fa ${loading ? "fa-spinner fa-spin" : "fa-ellipsis-vertical"}`} aria-hidden="true"></i>
    </button>
{/snippet}

<div bind:this={root} class="map-results" class:list-shown={listShown} class:collapsed>
    <div id="trail-map" class="map-canvas" bind:this={mapElement} aria-busy={mapLoading}>
        <MapWithElevationMaplibre onmoveend={handleMoveEnd} oninit={handleMapInit}
            trails={mapTrails} serverClusters={clusters} showElevation={false} showTerrain={true}
            activeTrail={-1} fitBounds="off" clusterTrails={true} scalePosition="bottom-right"
            bind:map bind:this={mapWithElevation} />
    </div>
    <div class="map-toolbar" bind:this={toolbarElement}>
        {@render toolbar()}
    </div>
    <section id={`${id}-trail-list`} class="results-panel" bind:this={panel} aria-labelledby={`${id}-results-title`}>
        <header class="result-header">
            <h2 id={`${id}-results-title`}>{$_("trail", { values: { n: 2 } })}</h2>
            {#if !loading && totalHits !== undefined}
                <span class="result-count">{totalHits >= 1000 ? "1000+" : totalHits}</span>
            {/if}
            <button type="button" class="btn-icon collapse-results" aria-controls={`${id}-trail-items`} aria-expanded={!collapsed}
                aria-label={$_(collapsed ? "trail-map-expand-list" : "trail-map-collapse-list")}
                title={$_(collapsed ? "trail-map-expand-list" : "trail-map-collapse-list")} onclick={toggleCollapsed}>
                <i class={`fa ${collapsed ? "fa-chevron-down" : "fa-chevron-up"}`} aria-hidden="true"></i>
            </button>
        </header>
        {#if selection.size > 0}
            <div class="selection-toolbar">
                <input type="checkbox" checked={allSelected} aria-label={$_("trail-results-select-loaded")}
                    class="w-4 h-4 accent-primary" onclick={() => toggleSelection()} />
                <span class="selection-count" aria-live="polite">{selection.size} {$_("selected")}</span>
                <TrailDropdown trails={selection} mode="multi-select" toggle={selectionTrigger} showMergeDialog={false}
                    onDelete={() => handleTrailsUpdated()} onShare={() => handleTrailsUpdated()}
                    onMerge={() => handleTrailsUpdated()} onUpdate={handleTrailsUpdated} />
                <button type="button" class="btn-icon" aria-label={$_("trail-results-clear-selection")} onclick={clearSelection}>
                    <i class="fa fa-xmark" aria-hidden="true"></i>
                </button>
            </div>
        {/if}
        <div id={`${id}-trail-items`} class="trail-results" aria-busy={loading || busy}>
            <div class="result-items" class:card-results={display === "cards"}>
                {#if loading}
                    {#each { length: 4 } as _}
                        {#if display === "list"}<div class="list-skeleton"><SkeletonListItem /></div>
                        {:else}<SkeletonCard />{/if}
                    {/each}
                {:else}
                    {#if !trails.length}<EmptyStateSearch />{/if}
                    {#each trails as trail (trail.id)}
                        <TrailResultLink {trail} list={display === "list"} selected={selection.has(trail)}
                            href={trailHref(trail)}
                            onclick={event => handleTrailClick(event, trail)} onselect={() => toggleSelection(trail)}
                            onhover={hovered => hoverTrail(hovered ? trail : undefined)}>
                            {#snippet children(hovered)}
                            {#if display === "list"}
                                <TrailListItem {trail} {hovered} selected={selection.has(trail)}
                                    showDescription={false} compact />
                            {:else}
                                <TrailCard {trail} fullWidth={true} {hovered} selected={selection.has(trail)} />
                            {/if}
                            {/snippet}
                        </TrailResultLink>
                    {/each}
                {/if}
            </div>
            {#if !loading}{@render moreResults()}{/if}
        </div>
    </section>
    <button type="button" class="map-view-toggle" aria-controls={`${id}-trail-list`} aria-expanded={listShown}
        onclick={() => listShown = !listShown}>
        <i class={`fa ${listShown ? "fa-map" : "fa-list-ul"}`} aria-hidden="true"></i>
        {listShown ? $_("map") : $_("list", { values: { n: 1 } })}
    </button>
</div>

<MergeDialog />

<style>
    /* MapLibre controls are 29 px wide with a 10 px margin; the search bar ends next to them. */
    .map-results { --panel-width: min(25rem, 40%); --panel-inset: 1rem; --controls-width: calc(39px + .75rem); height: max(30rem, calc(100dvh - var(--map-top, 0px))); position: relative; }
    .map-canvas { inset: 0; position: absolute; }
    .map-toolbar { left: calc(var(--panel-inset) * 2 + var(--panel-width)); min-width: 0; position: absolute; right: var(--controls-width); top: var(--panel-inset); z-index: 3; }
    .results-panel { background: rgba(var(--background)); border: 1px solid rgba(var(--input-border)); border-radius: .9rem; bottom: var(--panel-inset); display: flex; flex-direction: column; left: var(--panel-inset); min-height: 0; position: absolute; top: var(--panel-inset); width: var(--panel-width); z-index: 3; }
    /* Align the first control with the search bar and keep the rest clear of the results panel. */
    .map-canvas :global(.maplibregl-ctrl-top-right) { top: calc(var(--panel-inset) - 10px); }
    .map-canvas :global(.maplibregl-ctrl-top-left),
    .map-canvas :global(.maplibregl-ctrl-bottom-left) { left: calc(var(--panel-inset) + var(--panel-width)); }
    .result-header { align-items: center; border-bottom: 1px solid rgba(var(--input-border)); display: flex; flex-shrink: 0; gap: .5rem; padding: .5rem .5rem .5rem 1rem; }
    .collapse-results { align-items: center; display: flex; height: 2.25rem; justify-content: center; margin-left: auto; width: 2.25rem; }
    .collapsed .results-panel { bottom: auto; }
    .collapsed .result-header { border-bottom: 0; }
    .collapsed .trail-results, .collapsed .selection-toolbar:not(:has(dialog[open])) { display: none; }
    .result-header h2 { font-size: 1.125rem; font-weight: 600; }
    .result-count { color: rgba(var(--content), .6); font-size: .85rem; }
    .selection-toolbar { align-items: center; border-bottom: 1px solid rgba(var(--input-border)); display: flex; flex-shrink: 0; gap: .5rem; padding: .5rem 1rem; position: relative; z-index: 4; }
    .selection-count { flex: 1; font-size: .85rem; min-width: 0; }
    .selection-toolbar input, .selection-toolbar :global(.dropdown), .selection-toolbar :global(.btn-icon) { flex-shrink: 0; }
    .trail-results { display: flex; flex: 1; flex-direction: column; gap: 1rem; min-height: 0; overflow: auto; padding: .75rem; }
    .trail-results[aria-busy="true"] .result-items { opacity: .55; }
    .result-items { display: flex; flex-direction: column; gap: 1rem; min-width: 0; }
    .result-items > :global(.trail-result) { display: block; min-width: 0; }
    .results-panel :global(.trail-card), .results-panel :global(.skeleton-card) { min-width: 0; }
    .results-panel :global(.trail-card .tooltip)::before { right: 0; }
    .list-skeleton :global(.skeleton-list-item) { gap: .75rem; padding: .75rem; }
    .list-skeleton :global(.skeleton-list-item > div:first-child) { height: 4rem; width: 4rem; }
    .list-skeleton :global(.skeleton-list-item > div:last-child) { flex: 1; min-width: 0; }
    .map-view-toggle { display: none; }
    @media (width < 768px) {
        /* Phones: a full-screen map with the search bar on top; the list covers the map on demand. */
        .map-results { --panel-inset: .75rem; }
        .map-toolbar { left: var(--panel-inset); z-index: 4; }
        .results-panel { border: 0; border-radius: 0; inset: 0; padding-top: calc(var(--panel-inset) + var(--toolbar-height, 3.25rem)); width: auto; }
        /* An open selection dialog lives in the list; hiding it would close the dialog's focus. */
        .map-results:not(.list-shown) .results-panel:not(:has(dialog[open])) { display: none; }
        /* Phones switch between map and list instead. */
        .collapse-results { display: none; }
        .collapsed .results-panel { bottom: 0; }
        .collapsed .result-header { border-bottom: 1px solid rgba(var(--input-border)); }
        .collapsed .trail-results { display: flex; }
        .collapsed .selection-toolbar { display: flex; }
        .trail-results { padding-bottom: 5rem; }
        .map-canvas :global(.maplibregl-ctrl-top-left),
        .map-canvas :global(.maplibregl-ctrl-bottom-left) { left: 0; }
        .map-view-toggle { align-items: center; background: rgba(var(--content)); border-radius: 999px; bottom: calc(1.5rem + env(safe-area-inset-bottom, 0px)); color: rgba(var(--background)); cursor: pointer; display: inline-flex; font-weight: 600; gap: .5rem; left: 50%; min-height: 2.75rem; padding: .6rem 1.25rem; position: absolute; transform: translateX(-50%); z-index: 5; }
        .map-view-toggle:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: 3px; }
    }
</style>
