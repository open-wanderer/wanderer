<script lang="ts">
    import type { Snippet } from "svelte";
    import { onMount, tick } from "svelte";
    import { _ } from "svelte-i18n";
    import type { FeatureCollection } from "geojson";
    import * as M from "maplibre-gl";
    import type { Settings } from "$lib/models/settings";
    import type { Trail, TrailBoundingBox, TrailFilter } from "$lib/models/trail";
    import type { TrailSearchBounds } from "$lib/util/trail_search_util";
    import SkeletonCard from "../base/skeleton_card.svelte";
    import EmptyStateSearch from "../empty_states/empty_state_search.svelte";
    import MapWithElevationMaplibre from "./map_with_elevation_maplibre.svelte";
    import TrailCard from "./trail_card.svelte";

    interface Props {
        trails: Trail[];
        mapTrails: Trail[];
        clusters?: FeatureCollection;
        loading: boolean;
        active?: boolean;
        mapLoading?: boolean;
        initialBounds?: TrailSearchBounds;
        initialCenter?: { lat: number; lon: number };
        boundingBox: TrailBoundingBox;
        settings?: Settings;
        near: TrailFilter["near"];
        onviewport: (bounds: TrailSearchBounds, zoom: number, entering?: boolean) => void;
        moreResults: Snippet;
    }
    let { trails, mapTrails, clusters, loading, active = true, mapLoading = false, initialBounds, initialCenter,
        boundingBox, settings, near, onviewport, moreResults }: Props = $props();
    let root: HTMLDivElement;
    let mapElement: HTMLDivElement;
    let map: M.Map | undefined = $state();
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let ready = false;
    let initialized = false;
    let showMap = $state(true);
    let lastViewport: TrailSearchBounds | undefined;
    let lastCamera: { center: [number, number]; zoom: number; bearing: number; pitch: number } | undefined;
    let defaultCamera: typeof lastCamera;
    let resizing = false;
    let resumeRevision = 0;

    function resizeLayout() {
        if (!active || !root?.clientWidth) return;
        const viewport = window.visualViewport;
        const bottom = (viewport?.offsetTop ?? 0) + (viewport?.height ?? window.innerHeight);
        root.style.height = `${Math.max(280, bottom - root.getBoundingClientRect().top - 12)}px`;
    }

    // Keep the MapLibre instance and tiles across display changes. Restore its
    // camera after hidden-container or window resizes, before reporting bounds.
    $effect(() => {
        const visible = active;
        const revision = ++resumeRevision;
        if (!visible || !ready) return;
        resizing = true;
        void tick().then(async () => {
            if (!active || !ready || revision !== resumeRevision) return;
            resizeLayout();
            const sameArea = initialBounds && lastViewport && [
                initialBounds.northEast.lat - lastViewport.northEast.lat,
                initialBounds.northEast.lng - lastViewport.northEast.lng,
                initialBounds.southWest.lat - lastViewport.southWest.lat,
                initialBounds.southWest.lng - lastViewport.southWest.lng,
            ].every(difference => Math.abs(difference) < 1e-7);
            if (!showMap && !sameArea) { showMap = true; await tick(); }
            if (!active || !ready || revision !== resumeRevision) return;
            if (!showMap) {
                resizing = false;
                if (lastViewport) onviewport(lastViewport, lastCamera?.zoom ?? map?.getZoom() ?? 11, true);
                return;
            }
            map?.resize();
            if (!initialized || !sameArea) { resizing = false; initializeViewport(); return; }
            if (lastCamera) map?.jumpTo(lastCamera);
            resizing = false;
            notifyViewport(true);
        });
    });

    onMount(() => {
        const resize = resizeLayout;
        resize();
        window.addEventListener("resize", resize);
        window.visualViewport?.addEventListener("resize", resize);
        const observer = new ResizeObserver(() => {
            if (active && mapElement.clientWidth > 0 && mapElement.clientHeight > 0) map?.resize();
        });
        observer.observe(mapElement);
        const searchSize = new ResizeObserver(resizeLayout);
        const search = root.closest("main")?.querySelector("[data-search-root]");
        if (search) searchSize.observe(search);
        return () => {
            ready = false;
            ++resumeRevision;
            observer.disconnect();
            searchSize.disconnect();
            window.removeEventListener("resize", resize);
            window.visualViewport?.removeEventListener("resize", resize);
        };
    });

    function notifyViewport(entering = false) {
        if (!active || !ready || !map || !showMap || resizing || !mapElement.clientWidth || !mapElement.clientHeight) return;
        const bounds = map.getBounds();
        const normalize = (longitude: number) => ((((longitude + 180) % 360) + 360) % 360) - 180;
        const global = bounds.getEast() - bounds.getWest() >= 360;
        lastViewport = {
            northEast: { lat: bounds.getNorth(), lng: global ? 180 : normalize(bounds.getEast()) },
            southWest: { lat: bounds.getSouth(), lng: global ? -180 : normalize(bounds.getWest()) },
        };
        lastCamera = { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
        onviewport(lastViewport, map.getZoom(), entering);
    }

    export function focusBounds(bounds: TrailSearchBounds) {
        if (!ready || !map) return;
        if (!showMap) { lastViewport = bounds; onviewport(bounds, map.getZoom()); return; }
        // MapLibre needs an unwrapped east longitude for antimeridian views.
        const east = bounds.northEast.lng < bounds.southWest.lng ? bounds.northEast.lng + 360 : bounds.northEast.lng;
        map.fitBounds([[bounds.southWest.lng, bounds.southWest.lat], [east, bounds.northEast.lat]], { animate: false });
        notifyViewport();
    }

    export function focusPlace(place: TrailFilter["near"]) {
        if (!ready || !map || !Number.isFinite(place.lat) || !Number.isFinite(place.lon)) return;
        if (!showMap) {
            showMap = true;
            void tick().then(() => { map?.resize(); focusPlace(place); });
            return;
        }
        map.fitBounds(M.LngLatBounds.fromLngLat(new M.LngLat(place.lon!, place.lat!), place.radius), {
            padding: 32, maxZoom: 14, animate: false,
        });
        notifyViewport();
    }

    function handleMapInit() {
        ready = true;
        if (map) defaultCamera = { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
        if (active) initializeViewport();
    }

    function initializeViewport() {
        initialized = true;
        resizing = true;
        if (initialBounds) focusBounds(initialBounds);
        else if (initialCenter) map?.jumpTo({ center: [initialCenter.lon, initialCenter.lat], zoom: 14 });
        else if (Number.isFinite(near.lat) && Number.isFinite(near.lon)) focusPlace(near);
        else if (settings?.mapFocus === "location" && settings.location) {
            map?.jumpTo({ center: [settings.location.lon, settings.location.lat], zoom: 12 });
        } else if (boundingBox.has_trails ?? (boundingBox.min_lon !== 0 || boundingBox.max_lon !== 0 || boundingBox.min_lat !== 0 || boundingBox.max_lat !== 0)) {
            if (boundingBox.min_lon === boundingBox.max_lon && boundingBox.min_lat === boundingBox.max_lat) {
                map?.jumpTo({ center: [boundingBox.min_lon, boundingBox.min_lat], zoom: 12 });
            } else {
                map?.fitBounds([[boundingBox.min_lon, boundingBox.min_lat], [boundingBox.max_lon, boundingBox.max_lat]], { animate: false, padding: 32 });
            }
        } else if (defaultCamera) map?.jumpTo(defaultCamera);
        resizing = false;
        notifyViewport(true);
    }

    async function toggleMap() {
        const previous = lastViewport;
        resizing = true;
        showMap = !showMap;
        await tick();
        if (showMap) {
            map?.resize();
            if (previous) focusBounds(previous);
        }
        resizing = false;
        if (showMap) notifyViewport();
    }
</script>

<div bind:this={root} class="map-results" class:show-map={showMap}>
    <button type="button" class="map-view-toggle" onclick={toggleMap}
        aria-label={$_(showMap ? "trail-map-show-results" : "trail-map-show-map")} aria-pressed={!showMap}>
        <i class={`fa ${showMap ? "fa-list-ul" : "fa-map"}`} aria-hidden="true"></i>
        {$_(showMap ? "trail-map-show-results" : "trail-map-show-map")}
    </button>
    <div id="trail-list">
        {#if loading}
            {#each { length: 4 } as _}<SkeletonCard />{/each}
        {:else}
            {#if !trails.length}<EmptyStateSearch />{/if}
            {#each trails as trail (trail.id)}
                <a href="/map/trail/@{trail.author}{trail.domain ? `@${trail.domain}` : ''}/{trail.id}">
                    <TrailCard {trail} fullWidth={true} hovered={false} selected={false}
                        onmouseenter={() => mapWithElevation?.highlightCluster(trail)} onmouseleave={() => mapWithElevation?.unHighlightCluster()} />
                </a>
            {/each}
            {@render moreResults()}
        {/if}
    </div>
    <div id="trail-map" bind:this={mapElement} aria-busy={mapLoading}>
        <MapWithElevationMaplibre onmoveend={() => notifyViewport()} oninit={handleMapInit}
            trails={mapTrails} serverClusters={clusters} showElevation={false} showTerrain={true}
            showInfoPopup={true} activeTrail={-1} fitBounds="off" clusterTrails={true}
            bind:map bind:this={mapWithElevation} />
    </div>
</div>

<style>
    .map-results { display: grid; gap: 1rem; grid-template-columns: clamp(360px, 32vw, 440px) minmax(0, 1fr); height: calc(100dvh - 225px); min-height: 280px; position: relative; }
    #trail-list { display: flex; flex-direction: column; gap: 1rem; min-width: 0; overflow-y: auto; padding-right: .25rem; }
    #trail-list > a { display: block; }
    #trail-map { border: 1px solid rgba(var(--input-border)); border-radius: .9rem; min-height: 0; overflow: hidden; }
    .map-view-toggle { display: none; }
    @media (max-width: 767px) {
        .map-results { display: flex; flex-direction: column; gap: .5rem; }
        .map-view-toggle { align-items: center; align-self: flex-start; background: rgba(var(--input-background)); border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; display: inline-flex; font-size: .8rem; gap: .5rem; padding: .5rem .8rem; }
        #trail-list, #trail-map { flex: 1; min-height: 0; }
        .show-map #trail-list, .map-results:not(.show-map) #trail-map { display: none; }
    }
</style>
