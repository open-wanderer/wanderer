<script module lang="ts">
    export interface MapCamera { center: [number, number]; zoom: number; bearing: number; pitch: number }
</script>

<script lang="ts">
    import type { Snippet } from "svelte";
    import { onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import type { FeatureCollection } from "geojson";
    import * as M from "maplibre-gl";
    import type { Settings } from "#lib/models/settings.ts";
    import type { Trail, TrailBoundingBox, TrailFilter } from "#lib/models/trail.ts";
    import type { TrailSearchBounds } from "#lib/util/trail_search_util.ts";
    import { ResetViewControl } from "#lib/vendor/maplibre-reset-view/reset-view-control.ts";
    import SkeletonCard from "../base/skeleton_card.svelte";
    import SkeletonListItem from "../base/skeleton_list_item.svelte";
    import type { SelectItem } from "../base/select.svelte";
    import EmptyStateSearch from "../empty_states/empty_state_search.svelte";
    import MapWithElevationMaplibre from "./map_with_elevation_maplibre.svelte";
    import TrailResultLink from "./trail_result_link.svelte";
    import MergeDialog from "./trail_merge_dialog.svelte";
    import TrailCard from "./trail_card.svelte";
    import TrailDropdown from "./trail_dropdown.svelte";
    import TrailListItem from "./trail_list_item.svelte";
    import TrailTable from "./trail_table.svelte";

    interface Props {
        expanded?: boolean;
        onenlarge?: () => void;
        trails: Trail[];
        mapTrails: Trail[];
        clusters?: FeatureCollection;
        loading: boolean;
        showMap?: boolean;
        display?: "cards" | "list" | "table";
        tableHeader?: SelectItem[];
        filter?: TrailFilter;
        onsort?: (sort: string) => void;
        onupdate?: (updatedTrails?: Trail[]) => void;
        active?: boolean;
        mapLoading?: boolean;
        initialBounds?: TrailSearchBounds;
        initialCenter?: { lat: number; lon: number };
        initialCamera?: MapCamera;
        boundingBox: TrailBoundingBox;
        settings?: Settings;
        near: TrailFilter["near"];
        onviewport: (bounds: TrailSearchBounds, zoom: number) => void;
        moreResults: Snippet;
    }
    let { expanded = false, onenlarge, trails, mapTrails, clusters, loading, showMap = $bindable(true), display = "cards", tableHeader = [], filter, onsort, onupdate, active = true, mapLoading = false, initialBounds, initialCenter,
        boundingBox, settings, near, onviewport, moreResults, initialCamera }: Props = $props();
    const id = $props.id();
    let root: HTMLDivElement;
    let mapElement: HTMLDivElement;
    let map: M.Map | undefined = $state();
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let ready = false;
    let initialized = false;
    let selectedIds = $state(new Set<string>());
    let selection = $derived(new Set(trails.filter(trail => trail.id && selectedIds.has(trail.id))));
    let allSelected = $derived(trails.length > 0 && selection.size === trails.length);

    // Keep selection tied to loaded results, including fresh records after a
    // refresh. Appending more results and switching views retain the selection.
    $effect(() => {
        const visibleIds = new Set(trails.map(trail => trail.id));
        untrack(() => {
            if ([...selectedIds].some(id => !visibleIds.has(id))) {
                selectedIds = new Set([...selectedIds].filter(id => visibleIds.has(id)));
            }
        });
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
        root.querySelector<HTMLElement>('.trail-results a, .trail-results input[type="checkbox"]')?.focus();
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
    let lastViewport: TrailSearchBounds | undefined;
    let hiddenArea: TrailSearchBounds | undefined;
    let cameraPositioned = false;
    let initializingViewport = false;
    let defaultCamera: MapCamera | undefined;
    let resumeRevision = 0;
    let locationRequest = 0;
    let pendingLocation: { lat: number; lon: number } | undefined;
    let enlargeControl: ResetViewControl | undefined = $state.raw();
    const enlargeAppearance = $derived({ label: $_(expanded ? "trail-map-shrink" : "trail-map-enlarge"), icon: expanded ? "fa-minimize" : "fa-maximize" });
    let resetViewControl: ResetViewControl | undefined = $state.raw();
    const resetViewAppearance = $derived(settings?.mapFocus === "location" && settings.location
        ? { label: $_("trail-map-reset-location"), icon: "fa-location-dot" }
        : { label: $_("trail-map-reset-view"), icon: "fa-arrows-to-eye" });
    let savedListScroll: number | undefined;
    function rememberListPosition() {
        const list = root?.querySelector<HTMLElement>('.trail-results');
        if (list?.getClientRects().length) savedListScroll = list.scrollTop;
    }
    function restoreListPosition() {
        const list = root?.querySelector<HTMLElement>('.trail-results');
        if (savedListScroll !== undefined && list?.getClientRects().length) list.scrollTop = savedListScroll;
    }
    const layoutStorageKey = "mapResultsLayout";
    const paneGap = 16;
    const insetGap = 16;
    const minListWidth = 280;
    const minMapWidth = 320;
    let preferredWidth = $state(400);
    type LayoutMode = "split" | "map";
    let layoutMode = $state<LayoutMode>("split");
    let resultsCollapsed = $derived(layoutMode === "map");
    let wideViewport = $state(true);
    let containerWidth = $state(0);
    let maxListWidth = $derived(Math.max(minListWidth, containerWidth - (minMapWidth + paneGap)));
    let minimumListWidth = $derived(Math.min(minListWidth, maxListWidth));
    let listWidth = $derived(resultsCollapsed ? 0 : Math.max(minimumListWidth, Math.min(preferredWidth, maxListWidth)));
    let dragging = $state(false);
    let drag: { pointer: number; pointerType: string; startX: number; width: number; preferred: number; mode: LayoutMode; target: HTMLElement } | undefined;

    $effect(() => {
        const listHidden = !wideViewport || resultsCollapsed;
        untrack(() => {
            if (listHidden && drag) void finishResize();
        });
    });

    $effect(() => {
        resetViewControl?.setAppearance(resetViewAppearance);
    });

    $effect(() => { enlargeControl?.setAppearance(enlargeAppearance); });
    $effect(() => {
        if (expanded) root?.style.removeProperty("height");
        else void tick().then(resizeLayout);
    });
    $effect(() => {
        if (expanded || !root) return;
        const searchSize = new ResizeObserver(resizeLayout);
        const search = root.closest("main")?.querySelector("[data-search-root]");
        if (search) searchSize.observe(search);
        return () => searchSize.disconnect();
    });

    function resizeLayout() {
        if (!active || !root?.clientWidth) return;
        containerWidth = root.clientWidth;
        wideViewport = window.matchMedia("(width >= 768px)").matches;
        if (expanded) return;
        const viewport = window.visualViewport;
        const bottom = (viewport?.offsetTop ?? 0) + (viewport?.height ?? window.innerHeight);
        const lockedScroll = document.body.style.position === "fixed" ? -parseFloat(document.body.style.top || "0") : window.scrollY;
        const documentTop = root.getBoundingClientRect().top + lockedScroll;
        root.style.height = `${Math.max(280, bottom - documentTop - 12)}px`;
    }

    function mapVisible() { return showMap || wideViewport; }

    function renderedBounds(): TrailSearchBounds | undefined {
        if (!map) return;
        const bounds = map.getBounds();
        const normalize = (longitude: number) => ((((longitude + 180) % 360) + 360) % 360) - 180;
        const global = bounds.getEast() - bounds.getWest() >= 360;
        return {
            northEast: { lat: bounds.getNorth(), lng: global ? 180 : normalize(bounds.getEast()) },
            southWest: { lat: bounds.getSouth(), lng: global ? -180 : normalize(bounds.getWest()) },
        };
    }

    /** The parts of the map covered by the results panel and the search bar. */
    function insets() {
        if (!expanded) return { left: 0, top: 0 };
        const area = mapElement.getBoundingClientRect();
        const panel = root.querySelector(".trail-list")!;
        const toolbarElement = root.closest("main")?.querySelector(".map-search-toolbar");
        const covered = (element: Element | null | undefined, edge: "right" | "bottom") => {
            if (!element) return 0;
            const box = element.getBoundingClientRect();
            const overlaps = box.width && box.height && box.left < area.right && box.right > area.left && box.top < area.bottom && box.bottom > area.top;
            if (!overlaps) return 0;
            return edge === "right" ? box.right - area.left + insetGap : box.bottom - area.top + insetGap;
        };
        // On phones the list covers the whole map while it is shown; it never narrows the map area.
        const wide = window.matchMedia("(width >= 768px)").matches;
        // A collapsed list only covers the corner beside the search bar.
        return { left: wide && !resultsCollapsed ? covered(panel, "right") : 0, top: covered(toolbarElement, "bottom") };
    }
    function fitPadding(margin: number) {
        const { left, top } = insets();
        return { left: left + margin, top: top + margin, right: margin, bottom: margin };
    }

    /** The bounds of the map area that is not covered by the floating panels. */
    function visibleBounds(): TrailSearchBounds | undefined {
        if (!map) return;
        const { left, top } = insets();
        if (!left && !top) return renderedBounds();
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

    function handleMoveEnd() {
        if (!ready || initializingViewport || !mapVisible()) return;
        if (hiddenArea) { focusBounds(hiddenArea); return; }
        cameraPositioned = true;
        notifyViewport();
    }

    // Reuse the map when switching result displays; MapLibre observes its size.
    $effect(() => {
        const visible = active;
        const revision = ++resumeRevision;
        if (!visible || !ready) return;
        void tick().then(() => {
            if (!active || !ready || revision !== resumeRevision) return;
            resizeLayout();
            if (!initialized) initializeViewport();
            else if (initialBounds && JSON.stringify(initialBounds) !== JSON.stringify(lastViewport)) focusBounds(initialBounds);
            else notifyViewport();
        });
    });

    $effect.pre(() => {
        // Capture before a visibility change; hidden lists report zero in Firefox.
        active; resultsCollapsed; showMap; wideViewport;
        untrack(rememberListPosition);
        void tick().then(restoreListPosition);
    });

    onMount(() => {
        try {
            const stored = JSON.parse(localStorage.getItem(layoutStorageKey) ?? "null");
            if (typeof stored?.width === "number" && Number.isFinite(stored.width)) preferredWidth = Math.max(minListWidth, stored.width);
            // Older expanded/overlay fields are ignored; only the split width and collapse state remain.
            layoutMode = stored?.collapsed === true ? "map" : "split";
        } catch { /* Use the default split when storage is unavailable. */ }
        resizeLayout();
        const release = (event?: PointerEvent) => finishResize(event);
        const blur = () => finishResize();
        window.addEventListener("resize", resizeLayout);
        window.visualViewport?.addEventListener("resize", resizeLayout);
        window.addEventListener("blur", blur);
        window.addEventListener("pointerup", release);
        document.addEventListener("lostpointercapture", release);
        const rootSize = new ResizeObserver(() => { containerWidth = root.clientWidth; });
        rootSize.observe(root);
        return () => {
            ready = false;
            ++resumeRevision;
            ++locationRequest;
            rootSize.disconnect();
            window.removeEventListener("resize", resizeLayout);
            window.visualViewport?.removeEventListener("resize", resizeLayout);
            window.removeEventListener("blur", blur);
            window.removeEventListener("pointerup", release);
            document.removeEventListener("lostpointercapture", release);
        };
    });

    function notifyViewport() {
        if (!active || !ready || !map) return;
        if (mapVisible() || !lastViewport) lastViewport = visibleBounds();
        if (lastViewport) onviewport(lastViewport, map.getZoom());
    }

    export function focusBounds(bounds: TrailSearchBounds) {
        if (!ready || !map) return false;
        if (!mapVisible()) {
            lastViewport = hiddenArea = bounds;
            cameraPositioned = false;
            if (!initializingViewport) onviewport(bounds, map.getZoom());
            return true;
        }
        const east = bounds.northEast.lng < bounds.southWest.lng ? bounds.northEast.lng + 360 : bounds.northEast.lng;
        hiddenArea = undefined;
        cameraPositioned = true;
        map.fitBounds([[bounds.southWest.lng, bounds.southWest.lat], [east, bounds.northEast.lat]], { animate: false, ...(expanded ? { padding: fitPadding(0) } : {}), bearing: map.getBearing(), pitch: map.getPitch() });
        return true;
    }

    export function focusPlace(place: TrailFilter["near"], reveal = true) {
        if (!ready || !map || !Number.isFinite(place.lat) || !Number.isFinite(place.lon)) return;
        if (reveal && !mapVisible()) {
            map.once("resize", () => focusPlace(place, false));
            showMap = true;
            return;
        }
        map.fitBounds(M.LngLatBounds.fromLngLat(new M.LngLat(place.lon!, place.lat!), place.radius), {
            padding: expanded ? fitPadding(32) : 32, maxZoom: 14, animate: false,
        });
    }

    export async function focusLocation(location: { lat: number; lon: number }) {
        if (!Number.isFinite(location.lat) || !Number.isFinite(location.lon)) return;
        ++locationRequest;
        showMap = true;
        if (!ready || !map) { pendingLocation = location; return; }
        await tick();
        centerOn(location, 14);
    }

    function handleMapInit() {
        ready = true;
        if (map && !resetViewControl) {
            resetViewControl = new ResetViewControl(resetViewport, resetViewAppearance);
            map.addControl(resetViewControl, "top-right");
        }
        if (map && onenlarge && !enlargeControl) {
            enlargeControl = new ResetViewControl(() => onenlarge?.(), enlargeAppearance, "enlarge-map");
            map.addControl(enlargeControl, "top-right");
        }
        if (map) defaultCamera = { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
        if (active) initializeViewport();
    }

    function initializeViewport() {
        initialized = true;
        initializingViewport = true;
        if (pendingLocation) {
            centerOn(pendingLocation, 14);
            pendingLocation = undefined;
        }
        else if (initialCamera) map?.jumpTo(initialCamera);
        else if (initialBounds) focusBounds(initialBounds);
        else if (initialCenter) centerOn(initialCenter, 14);
        else if (Number.isFinite(near.lat) && Number.isFinite(near.lon)) focusPlace(near, false);
        else focusDefaultViewport();
        if (!hiddenArea) cameraPositioned = true;
        initializingViewport = false;
        notifyViewport();
    }

    export function captureView(): MapCamera | undefined {
        if (!ready || !cameraPositioned || !map) return;
        return { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
    }
    export function focusCamera(camera: MapCamera) {
        if (!ready || !map) return false;
        hiddenArea = undefined;
        cameraPositioned = true;
        map.jumpTo(camera);
        return true;
    }

    function resetViewport() {
        if (!ready || !map) return;
        initializingViewport = true;
        focusDefaultViewport();
        initializingViewport = false;
        notifyViewport();
    }

    function centerOn(location: { lat: number; lon: number }, zoom: number) {
        const center: [number, number] = [location.lon, location.lat];
        if (!expanded) { map?.jumpTo({ center, zoom }); return; }
        const { left, top } = insets();
        map?.easeTo({ center, zoom, offset: [left / 2, top / 2], duration: 0 });
    }

    function focusDefaultViewport() {
        // Ignore URL/snapshot bounds and place filters when explicitly returning
        // to the default view. Keep the search criteria themselves unchanged.
        map?.jumpTo({ bearing: 0, pitch: 0 });
        const request = ++locationRequest;
        const hasTrails = boundingBox.has_trails ?? (boundingBox.min_lon !== 0 || boundingBox.max_lon !== 0 || boundingBox.min_lat !== 0 || boundingBox.max_lat !== 0);
        if (settings?.mapFocus === "trails" && hasTrails) {
            if (boundingBox.min_lon === boundingBox.max_lon && boundingBox.min_lat === boundingBox.max_lat) {
                centerOn({ lon: boundingBox.min_lon, lat: boundingBox.min_lat }, 12);
            } else {
                map?.fitBounds([[boundingBox.min_lon, boundingBox.min_lat], [boundingBox.max_lon, boundingBox.max_lat]], { animate: false, padding: expanded ? fitPadding(32) : 32 });
            }
        } else if (settings?.mapFocus === "location" && settings.location) {
            centerOn(settings.location, 12);
        } else {
            if (defaultCamera) map?.jumpTo(defaultCamera);
            const camera = () => map && JSON.stringify([map.getCenter().toArray(), map.getZoom(), map.getBearing(), map.getPitch()]);
            const requestedCamera = camera();
            navigator.geolocation?.getCurrentPosition(position => {
                if (!ready || !map || request !== locationRequest || map.isMoving() || camera() !== requestedCamera) return;
                centerOn({ lon: position.coords.longitude, lat: position.coords.latitude }, 12);
                lastViewport = visibleBounds();
                notifyViewport();
            }, () => { /* Keep the default camera when location is unavailable. */ });
        }
    }

    function persistLayout() {
        try {
            localStorage.setItem(layoutStorageKey, JSON.stringify({ width: preferredWidth, collapsed: layoutMode === "map" }));
        } catch { /* The splitter still works when browser storage is disabled. */ }
    }
    function beginResize(event: PointerEvent) {
        if (event.button !== 0) return;
        const target = event.currentTarget as HTMLElement;
        event.preventDefault();
        target.focus();
        drag = { pointer: event.pointerId, pointerType: event.pointerType, startX: event.clientX, width: listWidth,
            preferred: preferredWidth, mode: layoutMode, target };
        target.setPointerCapture(event.pointerId);
        dragging = true;
    }
    function moveResize(event: PointerEvent) {
        if (!drag || event.pointerId !== drag.pointer) return;
        layoutMode = "split";
        preferredWidth = Math.max(minimumListWidth, Math.min(maxListWidth, drag.width + event.clientX - drag.startX));
    }
    function finishResize(event?: PointerEvent, cancel = false) {
        if (!drag || (event && event.pointerId !== drag.pointer)) return;
        const current = drag;
        drag = undefined;
        if (cancel) {
            preferredWidth = current.preferred;
            layoutMode = current.mode;
        }
        if (current.target.hasPointerCapture(current.pointer)) current.target.releasePointerCapture(current.pointer);
        dragging = false;
        if (!cancel) {
            persistLayout();
            if (expanded && listWidth !== current.width) void tick().then(notifyViewport);
        }
    }
    async function toggleResults() {
        layoutMode = resultsCollapsed ? "split" : "map";
        persistLayout();
        await tick();
        root.querySelector<HTMLButtonElement>(".collapse-results")?.focus();
        // The search area grows or shrinks with the list.
        if (expanded) notifyViewport();
    }
    function resizeKey(event: KeyboardEvent) {
        if (event.key === "Escape" && drag) {
            event.preventDefault();
            void finishResize(undefined, true);
            return;
        }
        if (event.key === "Enter") { event.preventDefault(); void toggleResults(); return; }
        const step = event.shiftKey ? 64 : 16;
        const width = event.key === "ArrowLeft" ? listWidth - step
            : event.key === "ArrowRight" ? listWidth + step
            : event.key === "Home" ? minimumListWidth
            : event.key === "End" ? maxListWidth : undefined;
        if (width === undefined) return;
        event.preventDefault();
        layoutMode = "split";
        preferredWidth = Math.max(minimumListWidth, Math.min(maxListWidth, width));
        persistLayout();
        if (expanded) void tick().then(notifyViewport);
    }

</script>

{#snippet selectionTrigger({ toggleMenu, loading }: { toggleMenu: (event: MouseEvent) => void; loading: boolean })}
    <button type="button" class="btn-icon selection-menu" aria-label={$_("trail-results-selection-actions")}
        aria-busy={loading} disabled={loading} onclick={toggleMenu}>
        <i class={`fa ${loading ? "fa-spinner fa-spin" : "fa-ellipsis-vertical"}`} aria-hidden="true"></i>
    </button>
{/snippet}

{#snippet toggleResultsButton()}
    <button type="button" class="btn-icon pane-button collapse-results" class:map-overlay-button={resultsCollapsed} onclick={toggleResults} aria-expanded={!resultsCollapsed}
        aria-controls={`${id}-trail-list`} aria-label={$_(resultsCollapsed ? "trail-map-restore-results" : "trail-map-hide-results")}
        title={$_(resultsCollapsed ? "trail-map-restore-results" : "trail-map-hide-results")}>
        <i class={`fa ${resultsCollapsed ? "fa-list-ul" : "fa-xmark"}`} aria-hidden="true"></i>
    </button>
{/snippet}

<div bind:this={root} class="map-results" class:expanded class:show-map={showMap} class:results-collapsed={resultsCollapsed} class:dragging
    style:--results-width={`${listWidth}px`} style:--pane-gap={`${paneGap}px`}>
    <div id={`${id}-trail-list`} class="trail-list">
        {#if !resultsCollapsed}
            <div class="result-header">
                <span>{$_("trail", { values: { n: 2 } })}</span>
                <div class="pane-actions">{@render toggleResultsButton()}</div>
            </div>
        {/if}
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
        <div class="trail-results" onscroll={rememberListPosition}>
            {#if display === "table"}
                <TrailTable trails={loading ? null : trails} selection={!selection.size ? undefined : selection} onTrailSelect={toggleSelection}
                    {tableHeader} {filter} {onsort} items={4}
                    trailHref={trail => `/map/trail/@${trail.author}${trail.domain ? `@${trail.domain}` : ''}/${trail.id}`}
                    onhover={hoverTrail} />
                {#if !loading && !trails.length}<EmptyStateSearch />{/if}
            {:else}
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
                                href={`/map/trail/@${trail.author}${trail.domain ? `@${trail.domain}` : ''}/${trail.id}`}
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
            {/if}
            {#if !loading}{@render moreResults()}{/if}
        </div>
        <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions (A focusable separator implements the WAI-ARIA window splitter pattern.) -->
        <div class="list-resize" role="separator" tabindex="0" aria-orientation="vertical" aria-controls={`${id}-trail-list`}
            aria-label={$_("trail-map-resize-results")} aria-valuemin={Math.round(minimumListWidth)} aria-valuemax={Math.round(maxListWidth)} aria-valuenow={Math.round(listWidth)}
            oncontextmenu={event => event.preventDefault()} onpointerdown={beginResize} onpointermove={moveResize} onpointerup={finishResize}
            onpointercancel={event => finishResize(event, true)} onlostpointercapture={event => finishResize(event, true)}
            onkeydown={resizeKey}></div>
    </div>
    {#if resultsCollapsed}
        <div class="collapsed-actions">{@render toggleResultsButton()}</div>
    {/if}
    <div class="map-pane">
        <div id="trail-map" bind:this={mapElement} aria-busy={mapLoading}>
            <MapWithElevationMaplibre onmoveend={handleMoveEnd} oninit={handleMapInit}
                scalePosition={wideViewport && !expanded ? "top-left" : "bottom-right"}
                trails={mapTrails} serverClusters={clusters} showElevation={false} showTerrain={true}
                showInfoPopup={true} showClusterPopups={true} activeTrail={-1} fitBounds="off" clusterTrails={true}
                bind:map bind:this={mapWithElevation} />
        </div>
    </div>
</div>

<MergeDialog />

<style>
    .map-results { display: grid; gap: var(--pane-gap); grid-template-columns: var(--results-width) minmax(0, 1fr); height: calc(100dvh - 225px); min-height: 280px; position: relative; }
    .map-results.expanded { display: block; height: 100%; min-height: 0; pointer-events: none; position: static; }
    .expanded .trail-list { pointer-events: auto; }
    .expanded .map-pane { display: block; inset: 0; pointer-events: auto; position: absolute; z-index: 0; }
    .expanded #trail-map { border: 0; border-radius: 0; }
    .expanded .collapsed-actions { left: auto; pointer-events: auto; position: relative; top: auto; width: max-content; }
    .trail-list { background: rgba(var(--background)); border: 1px solid rgba(var(--input-border)); border-radius: .9rem; box-sizing: border-box; display: flex; flex-direction: column; min-height: 0; min-width: 0; position: relative; }
    .result-header { align-items: center; background: rgba(var(--input-background)); border-bottom: 1px solid rgba(var(--input-border)); border-radius: .9rem .9rem 0 0; display: flex; flex-shrink: 0; font-size: .9rem; font-weight: 600; gap: .5rem; justify-content: space-between; padding: .25rem .5rem .25rem .75rem; }
    .selection-toolbar { align-items: center; border-bottom: 1px solid rgba(var(--input-border)); display: flex; flex-shrink: 0; gap: .5rem; padding: .5rem .75rem; position: relative; z-index: 4; }
    .selection-count { flex: 1; font-size: .85rem; min-width: 0; }
    .selection-toolbar input { flex-shrink: 0; }
    .selection-toolbar :global(.dropdown) { flex-shrink: 0; }
    .selection-toolbar :global(.btn-icon) { flex-shrink: 0; }
    .trail-results { border-radius: 0 0 .9rem .9rem; display: flex; flex: 1; flex-direction: column; gap: 1rem; min-height: 0; min-width: 0; overflow: auto; padding: .5rem; }
    .result-items { display: flex; flex-direction: column; gap: 1rem; min-width: 0; }
    .result-items > :global(.trail-result) { display: block; min-width: 0; }
    .card-results { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 18rem), 1fr)); align-items: start; }
    .card-results > :global(.trail-result), .card-results > :global(.skeleton-card) { max-width: 36rem; width: 100%; }
    .trail-list :global(.table-container) { flex-shrink: 0; }
    .trail-list :global(.trail-card), .trail-list :global(.skeleton-card) { min-width: 0; }
    .trail-list :global(.trail-card .tooltip)::before { right: 0; }
    .list-skeleton :global(.skeleton-list-item) { gap: .75rem; padding: .75rem; }
    .list-skeleton :global(.skeleton-list-item > div:first-child) { height: 4rem; width: 4rem; }
    .list-skeleton :global(.skeleton-list-item > div:last-child) { flex: 1; min-width: 0; }
    .map-pane { min-height: 0; min-width: 0; position: relative; }
    #trail-map { border: 1px solid rgba(var(--input-border)); border-radius: .9rem; height: 100%; min-height: 0; overflow: hidden; }
    .results-collapsed { grid-template-columns: minmax(0, 1fr); }
    .list-resize { background: transparent; bottom: 0; cursor: col-resize; position: absolute; right: calc(-1px - var(--pane-gap)); top: 0; touch-action: none; width: var(--pane-gap); z-index: 3; }
    .list-resize::before { background: rgba(var(--content), .35); border-radius: 999px; content: ""; height: 6rem; left: 50%; position: absolute; top: 50%; transform: translate(-50%, -50%); width: 4px; }
    .list-resize:hover::before, .list-resize:focus-visible::before, .dragging .list-resize::before { background: rgba(var(--input-border-focus)); }
    .list-resize:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: -2px; }
    .pane-actions { display: flex; gap: .25rem; }
    .pane-button { align-items: center; color: rgba(var(--content)); cursor: pointer; display: flex; height: 2.75rem; justify-content: center; width: 2.75rem; }
    .pane-button i { font-size: .85rem; }
    .map-overlay-button { filter: drop-shadow(0 2px 5px rgb(0 0 0 / .3)); }
    .map-overlay-button { background-color: rgba(var(--menu-background)); }
    @media (hover: hover) { .map-overlay-button:hover { background-color: rgba(var(--secondary-hover)); } }
    .collapsed-actions { left: 1rem; position: absolute; top: 3rem; z-index: 3; }
    .dragging { cursor: col-resize; user-select: none; }
    @media (width >= 768px) {
        .map-results.expanded #trail-map :global(.maplibregl-ctrl-bottom-right) { max-width: 20rem; }
        .expanded .trail-list { background: rgba(var(--menu-background), .96); box-shadow: 0 3px 16px rgb(0 0 0 / .2); height: 100%; width: var(--results-width); z-index: 1; }
        .results-collapsed .trail-list, .results-collapsed .selection-toolbar { display: contents; }
        .results-collapsed .trail-results, .results-collapsed .list-resize,
        .results-collapsed .selection-toolbar > :global(:not(dialog)) { display: none; }
    }
    .expanded #trail-map :global(.maplibregl-ctrl-bottom-right) { align-items: flex-end; box-sizing: border-box; display: flex; gap: .5rem; max-width: 100%; padding: 0 .5rem .5rem; }
    .expanded #trail-map :global(.maplibregl-ctrl-bottom-right > .maplibregl-ctrl) { clear: none; margin: 0; }
    .expanded #trail-map :global(.maplibregl-ctrl-attrib) { flex-shrink: 1; min-width: 0; order: 0; overflow-wrap: anywhere; }
    .expanded #trail-map :global(.maplibregl-ctrl-scale) { flex-shrink: 0; order: 1; }
    @media (width < 768px) {
        #trail-map :global(.maplibregl-ctrl-bottom-right) { align-items: flex-end; box-sizing: border-box; display: flex; gap: .5rem; max-width: 100%; padding: 0 .5rem .5rem; }
        #trail-map :global(.maplibregl-ctrl-bottom-right > .maplibregl-ctrl) { clear: none; margin: 0; }
        #trail-map :global(.maplibregl-ctrl-attrib) { flex-shrink: 1; min-width: 0; order: 0; overflow-wrap: anywhere; }
        #trail-map :global(.maplibregl-ctrl-scale) { flex-shrink: 0; order: 1; }
        .result-header, .list-resize, .collapsed-actions { display: none; }
        .map-results { display: flex; flex-direction: column; gap: .5rem; }
        .trail-list, .map-pane { flex: 1; min-height: 0; }
        .map-results:not(.show-map) .map-pane { display: none; }
        .map-results.show-map .trail-list,
        .map-results.show-map .selection-toolbar { display: contents; }
        .map-results.show-map .trail-results,
        .map-results.show-map .list-resize,
        .map-results.show-map .selection-toolbar > :global(:not(dialog)) { display: none; }
    }
</style>
