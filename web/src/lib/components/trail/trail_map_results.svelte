<script module lang="ts">
    export interface MapCamera { center: [number, number]; zoom: number; bearing: number; pitch: number }
</script>

<script lang="ts">
    import type { Snippet } from "svelte";
    import { flushSync, onMount, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";
    import type { FeatureCollection } from "geojson";
    import * as M from "maplibre-gl";
    import type { Settings } from "#lib/models/settings.ts";
    import type { Trail, TrailBoundingBox, TrailFilter } from "#lib/models/trail.ts";
    import type { TrailSearchBounds } from "#lib/util/trail_search_util.ts";
    import { isMapCanvasResize, resizeMapCanvas, paintMapCanvas } from "#lib/util/map_resize.ts";
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
        trails: Trail[];
        mapTrails: Trail[];
        clusters?: FeatureCollection;
        loading: boolean;
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
        // A completed window series is already debounced by the results component.
        onviewport: (bounds: TrailSearchBounds, zoom: number, entering?: boolean, settled?: boolean) => void;
        onzoom?: (zoom: number, bounds?: TrailSearchBounds, settled?: boolean) => void;
        onwindowresize?: () => void;
        moreResults: Snippet;
    }
    let { trails, mapTrails, clusters, loading, display = "cards", tableHeader = [], filter, onsort, onupdate, active = true, mapLoading = false, initialBounds, initialCenter,
        boundingBox, settings, near, onviewport, onzoom, onwindowresize, moreResults, initialCamera }: Props = $props();
    const id = $props.id();
    let root: HTMLDivElement;
    let mapElement: HTMLDivElement;
    let map: M.Map | undefined = $state();
    let mapWithElevation: MapWithElevationMaplibre | undefined = $state();
    let ready = false;
    let initialized = false;
    let showMap = $state(true);
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
    let lastCamera: (MapCamera & { width: number; height: number }) | undefined;
    let resizeTimer: ReturnType<typeof setTimeout> | undefined;
    let hiddenArea: TrailSearchBounds | undefined;
    let cameraPositioned = false;
    const gesturePointers = new Map<number, string>();
    let deferredResize: { preserveSearch: boolean; settled?: boolean } | undefined;
    let deferredSettle: { area?: TrailSearchBounds; preserveCamera: boolean; publish: boolean } | undefined;
    let resizingMap = false;
    let initializingViewport = false;
    let defaultCamera: MapCamera | undefined;
    let resizing = $state(false);
    let canvasSize = { width: 0, height: 0 };
    let resumeRevision = 0;
    let layoutRevision = 0;
    let locationRequest = 0;
    let pendingLocation: { lat: number; lon: number } | undefined;
    let resetViewControl: ResetViewControl | undefined = $state.raw();
    const resetViewAppearance = $derived(settings?.mapFocus === "location" && settings.location
        ? { label: $_("trail-map-reset-location"), icon: "fa-location-dot" }
        : { label: $_("trail-map-reset-view"), icon: "fa-arrows-to-circle" });
    let savedListScroll: number | undefined;
    function rememberListPosition() {
        const list = root?.querySelector<HTMLElement>('.trail-results');
        if (list?.getClientRects().length) savedListScroll = list.scrollTop;
    }
    function restoreListPosition() {
        const list = root?.querySelector<HTMLElement>('.trail-results');
        if (savedListScroll !== undefined && list?.getClientRects().length) list.scrollTop = savedListScroll;
    }
    let controlsSize: ResizeObserver | undefined;
    const layoutStorageKey = "mapResultsLayout";
    const paneGap = 16;
    const minListWidth = 280;
    const minMapWidth = 320;
    let preferredWidth = $state(400);
    type LayoutMode = "split" | "map" | "results";
    let layoutMode = $state<LayoutMode>("split");
    let resultsCollapsed = $derived(layoutMode === "map");
    let resultsExpanded = $derived(layoutMode === "results");
    let wideViewport = $state(true);
    let miniMap = $derived(resultsExpanded && wideViewport);
    let containerWidth = $state(0);
    let containerHeight = $state(0);
    let maxListWidth = $derived(Math.max(minListWidth, containerWidth - (minMapWidth + paneGap)));
    let minimumListWidth = $derived(Math.min(minListWidth, maxListWidth));
    let listWidth = $derived(resultsCollapsed ? 0 : Math.max(minimumListWidth, Math.min(preferredWidth, maxListWidth)));
    let dragging = $state(false);
    let drag: { pointer: number; pointerType: string; startX: number; width: number; preferred: number; mode: LayoutMode; target: HTMLElement } | undefined;
    type OverlaySize = { width: number; height: number };
    type OverlayPosition = { right: number; bottom: number };
    const minOverlayWidth = 256;
    const overlayBottomInset = 24;
    let minOverlayHeight = $state(160);
    let pendingOverlayMinimum: number | undefined;
    let overlaySize = $state<OverlaySize>();
    let overlayPosition = $state<OverlayPosition>();
    let overlayHidden = $state(false);
    let maxOverlayWidth = $derived(Math.max(minOverlayWidth, Math.min(800, containerWidth - 64)));
    let maxOverlayHeight = $derived(Math.max(minOverlayHeight, containerHeight - 112));
    let overlayWidth = $derived(Math.round(Math.max(minOverlayWidth, Math.min(overlaySize?.width ?? Math.min(384, containerWidth * .3), maxOverlayWidth))));
    let overlayHeight = $derived(Math.round(Math.max(minOverlayHeight, Math.min(overlaySize?.height ?? Math.min(240, containerHeight * .3), maxOverlayHeight))));
    let maxOverlayRight = $derived(Math.max(0, containerWidth - overlayWidth));
    let maxOverlayBottom = $derived(Math.max(overlayBottomInset, containerHeight - overlayHeight));
    let overlayRight = $derived(Math.round(Math.max(0, Math.min(overlayPosition?.right ?? 0, maxOverlayRight))));
    let overlayBottom = $derived(Math.round(Math.max(overlayBottomInset, Math.min(overlayPosition?.bottom ?? overlayBottomInset, maxOverlayBottom))));
    let overlayDragging = $state(false);
    let overlayDrag: { pointer: number; pointerType: string; x: number; y: number; size: OverlaySize; preferred?: OverlaySize; area?: TrailSearchBounds; target: HTMLElement } | undefined;
    let overlayMove = $state.raw<{ pointer: number; pointerType: string; x: number; y: number; position: OverlayPosition; preferred?: OverlayPosition; target: HTMLElement }>();

    $effect(() => {
        const listHidden = miniMap || !wideViewport || resultsCollapsed;
        const overlayGone = !miniMap || overlayHidden;
        untrack(() => {
            if (listHidden && drag) void finishResize();
            if (overlayGone && overlayDrag) void finishOverlayResize();
            if (overlayGone && overlayMove) void finishOverlayMove();
        });
    });

    $effect(() => {
        resetViewControl?.setAppearance(resetViewAppearance);
    });

    function resizeLayout() {
        if (!active || !root?.clientWidth) return;
        containerWidth = root.clientWidth;
        wideViewport = window.matchMedia("(width >= 768px)").matches;
        const viewport = window.visualViewport;
        const bottom = (viewport?.offsetTop ?? 0) + (viewport?.height ?? window.innerHeight);
        const lockedScroll = document.body.style.position === "fixed" ? -parseFloat(document.body.style.top || "0") : window.scrollY;
        const documentTop = root.getBoundingClientRect().top + lockedScroll;
        root.style.height = `${Math.max(280, bottom - documentTop - 12)}px`;
        containerHeight = root.clientHeight;
    }

    function resizeMap() {
        if (map) resizeMapCanvas(map);
        if (map && mapElement) canvasSize = { width: mapElement.clientWidth, height: mapElement.clientHeight };
        const previous = resizingMap;
        resizingMap = true;
        try {
            if (ready && hiddenArea && mapVisible()) {
                const area = hiddenArea;
                hiddenArea = undefined;
                focusBounds(area);
            }
        } finally { resizingMap = previous; }
    }

    function mapVisible() { return showMap || wideViewport; }
    function gestureActive() { return Boolean(drag || overlayDrag || overlayMove) || Boolean(ready && map?.isMoving()); }

    function captureCamera() {
        return ready && cameraPositioned && map ? { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch(), ...canvasSize } : undefined;
    }

    function restoreLayoutCamera() {
        if (!map || !lastCamera) return false;
        if (!lastCamera.bearing && !lastCamera.pitch) return false;
        const ratio = lastCamera.width && lastCamera.height
            ? Math.min(canvasSize.width / lastCamera.width, canvasSize.height / lastCamera.height) : 1;
        const { center, zoom, bearing, pitch } = lastCamera;
        map.jumpTo({ center, zoom: zoom + Math.log2(ratio || 1), bearing, pitch });
        return true;
    }

    function resizeToArea(area: TrailSearchBounds | undefined) {
        resizeMap();
        if (!map || !ready) return;
        // A rotated/pitched bounds hull cannot reconstruct its original camera.
        // Retain the camera frame across layouts so repeated changes cannot drift.
        if (!restoreLayoutCamera()) {
            if (area) focusBounds(area);
            lastCamera = captureCamera();
        }
        paintMapCanvas(map);
        onzoom?.(map.getZoom(), renderedBounds());
    }

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

    function resizeAutomatically(preserveSearch = false, forcePublish = false, settled = false) {
        if (!active || !mapElement.clientWidth || !mapElement.clientHeight || !mapVisible()) return;
        if (!forcePublish && canvasSize.width === mapElement.clientWidth && canvasSize.height === mapElement.clientHeight) return;
        // Painting is independent of search publication. Public resize() stops
        // MapLibre's drag/wheel handlers and also unlocks GeolocateControl.
        resizeMap();
        if (drag || overlayDrag || overlayMove || resizing) return;
        if (gestureActive()) {
            deferredResize = { preserveSearch: preserveSearch && (deferredResize?.preserveSearch ?? true),
                settled: settled || Boolean(deferredResize?.settled) };
            return;
        }
        if (!ready) return;
        if (preserveSearch || document.querySelector("dialog:modal") || document.body.style.position === "fixed") renderViewport(settled);
        else notifyViewport(false, settled);
    }
    function renderViewport(settled = false) {
        if (ready && map && mapVisible()) onzoom?.(map.getZoom(), renderedBounds(), settled);
    }
    function finishDeferredResize() {
        if (gestureActive()) return;
        if (deferredSettle) {
            const pending = deferredSettle;
            deferredSettle = undefined;
            void settleLayout(pending?.area, pending?.preserveCamera ?? true, pending?.publish);
        } else if (deferredResize && !resizing && resizeTimer === undefined) {
            const pending = deferredResize;
            deferredResize = undefined;
            resizeAutomatically(pending.preserveSearch, true, pending.settled);
        }
    }
    function handleMoveEnd() {
        if (map && isMapCanvasResize(map)) return;
        if (resizingMap || initializingViewport || !ready) return;
        cameraPositioned = !hiddenArea;
        if (deferredResize) deferredResize.preserveSearch = false;
        if (deferredSettle || (resizing && (drag || overlayDrag || overlayMove))) {
            deferredSettle = { preserveCamera: true, publish: true };
        }
        // A camera stopped while the narrow map is being hidden is not a
        // published camera. Restore the list's bounds rather than snapshot it.
        if (!mapVisible()) cameraPositioned = false;
        if (!resizing) ++locationRequest;
        if (!deferredResize && !deferredSettle && resizeTimer === undefined) notifyViewport();
        requestAnimationFrame(finishDeferredResize);
    }

    // Keep the MapLibre instance and tiles across display changes.
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
            if (!mapVisible() && !sameArea) { showMap = true; await tick(); }
            if (!active || !ready || revision !== resumeRevision) return;
            if (!mapVisible()) {
                resizing = false;
                if (lastViewport) onviewport(lastViewport, lastCamera?.zoom ?? map?.getZoom() ?? 11, true);
                return;
            }
            resizeMap();
            if (!initialized || !sameArea) { resizing = false; initializeViewport(); return; }
            if (lastCamera && !restoreLayoutCamera()) {
                const { center, zoom, bearing, pitch } = lastCamera;
                map?.jumpTo({ center, zoom, bearing, pitch });
            }
            resizing = false;
            notifyViewport(true);
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
            layoutMode = stored?.expanded === true ? "results" : stored?.collapsed === true ? "map" : "split";
            if (typeof stored?.overlay?.width === "number" && Number.isFinite(stored.overlay.width)
                && typeof stored.overlay.height === "number" && Number.isFinite(stored.overlay.height)) {
                overlaySize = { width: stored.overlay.width, height: stored.overlay.height };
            }
            if (typeof stored?.overlay?.right === "number" && Number.isFinite(stored.overlay.right)
                && typeof stored.overlay.bottom === "number" && Number.isFinite(stored.overlay.bottom)) {
                overlayPosition = { right: stored.overlay.right, bottom: stored.overlay.bottom };
            }
            overlayHidden = stored?.overlay?.hidden === true;
        } catch { /* Use the default split when storage is unavailable. */ }
        let lastWindowSize = { width: innerWidth, height: innerHeight };
        const publishResize = () => {
            resizeTimer = undefined;
            if (drag || overlayDrag || overlayMove || resizing) {
                resizeTimer = setTimeout(publishResize, 200);
                return;
            }
            // A visualViewport resize can already have painted this size.
            // The completed window series also owns any camera motion completed during it.
            deferredResize = undefined;
            // This series already waited 200 ms; the page must search immediately.
            resizeAutomatically(false, true, true);
        };
        const resize = () => {
            const changed = lastWindowSize.width !== innerWidth || lastWindowSize.height !== innerHeight;
            lastWindowSize = { width: innerWidth, height: innerHeight };
            flushSync(resizeLayout);
            if (mapVisible()) resizeMap();
            else if (map?.isMoving()) cameraPositioned = false;
            if (changed) {
                if (active && mapVisible()) onwindowresize?.();
                clearTimeout(resizeTimer);
                resizeTimer = setTimeout(publishResize, 200);
            }
        };
        const resizeHeader = () => {
            flushSync(resizeLayout);
            resizeAutomatically(true);
        };
        resizeLayout();
        const beginGesture = (event: PointerEvent) => {
            if (event.button === 0 && event.target instanceof HTMLCanvasElement) gesturePointers.set(event.pointerId, event.pointerType);
        };
        const releaseHandles = (event?: PointerEvent) => {
            void finishResize(event);
            void finishOverlayResize(event);
            void finishOverlayMove(event);
        };
        const releaseGesture = (mouseOnly = false) => {
            const held = [...gesturePointers.values()].some(type => !mouseOnly || type === 'mouse');
            if (held) {
                gesturePointers.clear();
                // A held pointer alone does not own a camera animation.
                const gesture = [map?.dragPan, map?.touchZoomRotate, map?.dragRotate, map?.touchPitch].some(handler => handler?.isActive());
                if (map && gesture) {
                    const moving = map.isMoving();
                    let ended = false;
                    const onEnd = () => { ended = true; };
                    map.on('moveend', onEnd);
                    try { map.stop(); }
                    finally { map.off('moveend', onEnd); }
                    if (moving && !ended) map.fire('moveend');
                }
            }
            if (!mouseOnly || drag?.pointerType === 'mouse') void finishResize();
            if (!mouseOnly || overlayDrag?.pointerType === 'mouse') void finishOverlayResize();
            if (!mouseOnly || overlayMove?.pointerType === 'mouse') void finishOverlayMove();
            requestAnimationFrame(finishDeferredResize);
        };
        const blur = () => releaseGesture();
        const contextMenu = (event: MouseEvent) => { if (!event.defaultPrevented) releaseGesture(true); };
        const mouseMove = (event: MouseEvent) => { if (event.buttons === 0) releaseGesture(true); };
        const visibility = () => { if (document.hidden) releaseGesture(); };
        const endGesture = (event: PointerEvent) => {
            gesturePointers.delete(event.pointerId);
            releaseHandles(event);
            requestAnimationFrame(finishDeferredResize);
        };
        mapElement.addEventListener('pointerdown', beginGesture, true);
        document.addEventListener('lostpointercapture', releaseHandles);
        document.addEventListener('visibilitychange', visibility);
        window.addEventListener('contextmenu', contextMenu);
        window.addEventListener('mousemove', mouseMove);
        window.addEventListener('blur', blur);
        window.addEventListener('pointerup', endGesture);
        window.addEventListener('pointercancel', endGesture);
        window.addEventListener("resize", resize);
        window.visualViewport?.addEventListener("resize", resizeHeader);
        const observer = new ResizeObserver(() => resizeAutomatically());
        observer.observe(mapElement);
        const rootSize = new ResizeObserver(() => { containerWidth = root.clientWidth; containerHeight = root.clientHeight; });
        rootSize.observe(root);
        const searchSize = new ResizeObserver(resizeHeader);
        const search = root.closest("main")?.querySelector("[data-search-root]");
        if (search) searchSize.observe(search);
        return () => {
            ready = false;
            ++resumeRevision;
            ++layoutRevision;
            ++locationRequest;
            clearTimeout(resizeTimer);
            observer.disconnect();
            rootSize.disconnect();
            searchSize.disconnect();
            controlsSize?.disconnect();
            mapElement.removeEventListener('pointerdown', beginGesture, true);
            document.removeEventListener('lostpointercapture', releaseHandles);
            document.removeEventListener('visibilitychange', visibility);
            window.removeEventListener('contextmenu', contextMenu);
            window.removeEventListener('mousemove', mouseMove);
            window.removeEventListener('blur', blur);
            window.removeEventListener('pointerup', endGesture);
            window.removeEventListener('pointercancel', endGesture);
            window.removeEventListener("resize", resize);
            window.visualViewport?.removeEventListener("resize", resizeHeader);
        };
    });

    function notifyViewport(entering = false, settled = false) {
        if (!active || !ready || !map || resizing) return;
        const hidden = (!mapVisible()) || !mapElement.clientWidth || !mapElement.clientHeight;
        if (hidden && !entering) return;
        if (!hidden || !lastViewport) lastViewport = renderedBounds();
        lastCamera = captureCamera();
        if (!lastViewport) return;
        onviewport(lastViewport, map.getZoom(), entering, settled);
    }

    export function focusBounds(bounds: TrailSearchBounds) {
        if (!ready || !map) return false;
        if (!mapVisible()) {
            lastViewport = bounds;
            hiddenArea = bounds;
            cameraPositioned = false;
            if (!resizing) onviewport(bounds, map.getZoom());
            return true;
        }
        // MapLibre needs an unwrapped east longitude for antimeridian views.
        const east = bounds.northEast.lng < bounds.southWest.lng ? bounds.northEast.lng + 360 : bounds.northEast.lng;
        cameraPositioned = true;
        hiddenArea = undefined;
        map.fitBounds([[bounds.southWest.lng, bounds.southWest.lat], [east, bounds.northEast.lat]], { animate: false, bearing: map.getBearing(), pitch: map.getPitch() });
        notifyViewport();
        return true;
    }

    export function focusPlace(place: TrailFilter["near"], reveal = true) {
        if (!ready || !map || !Number.isFinite(place.lat) || !Number.isFinite(place.lon)) return;
        if (reveal && !mapVisible()) {
            showMap = true;
            void tick().then(() => { resizeMap(); focusPlace(place); });
            return;
        }
        map.fitBounds(M.LngLatBounds.fromLngLat(new M.LngLat(place.lon!, place.lat!), place.radius), {
            padding: 32, maxZoom: 14, animate: false,
        });
        notifyViewport();
    }

    export async function focusLocation(location: { lat: number; lon: number }) {
        if (!Number.isFinite(location.lat) || !Number.isFinite(location.lon)) return;
        ++locationRequest;
        showMap = true;
        if (!ready || !map) {
            pendingLocation = location;
            return;
        }
        resizing = true;
        await tick();
        resizeMap();
        map.jumpTo({ center: [location.lon, location.lat], zoom: 14 });
        paintMapCanvas(map);
        resizing = false;
        notifyViewport();
    }

    function handleMapInit() {
        ready = true;
        const previous = resizing;
        resizing = true;
        resizeMap();
        resizing = previous;
        if (map && !resetViewControl) {
            resetViewControl = new ResetViewControl(resetViewport, resetViewAppearance);
            map.addControl(resetViewControl, "top-right");
            controlsSize = new ResizeObserver(updateOverlayMinimum);
            for (const corner of ["top-left", "top-right", "bottom-left", "bottom-right"]) {
                const controls = mapElement.querySelector(`.maplibregl-ctrl-${corner}`);
                if (controls) controlsSize.observe(controls, { box: "border-box" });
            }
            updateOverlayMinimum();
        }
        if (map) defaultCamera = { center: map.getCenter().toArray(), zoom: map.getZoom(), bearing: map.getBearing(), pitch: map.getPitch() };
        if (active) initializeViewport();
    }

    function updateOverlayMinimum() {
        const height = (corner: string) => mapElement.querySelector<HTMLElement>(`.maplibregl-ctrl-${corner}`)?.offsetHeight ?? 0;
        // Corner stacks include their margins. Leave a gap between top and
        // bottom controls (including attribution), plus the map's borders.
        const minimum = Math.max(160, Math.ceil(Math.max(
            height("top-left") + height("bottom-left"),
            height("top-right") + height("bottom-right"),
        ) + 14));
        if (overlayMove || overlayDrag) pendingOverlayMinimum = minimum;
        else minOverlayHeight = minimum;
    }

    function applyOverlayMinimum() {
        if (pendingOverlayMinimum === undefined) return false;
        const changed = minOverlayHeight !== pendingOverlayMinimum;
        minOverlayHeight = pendingOverlayMinimum;
        pendingOverlayMinimum = undefined;
        return changed;
    }

    function initializeViewport() {
        initialized = true;
        initializingViewport = true;
        resizing = true;
        if (pendingLocation) {
            map?.jumpTo({ center: [pendingLocation.lon, pendingLocation.lat], zoom: 14 });
            pendingLocation = undefined;
        }
        else if (initialCamera) map?.jumpTo(initialCamera);
        else if (initialBounds) focusBounds(initialBounds);
        else if (initialCenter) map?.jumpTo({ center: [initialCenter.lon, initialCenter.lat], zoom: 14 });
        else if (Number.isFinite(near.lat) && Number.isFinite(near.lon)) focusPlace(near, false);
        else focusDefaultViewport();
        if (!hiddenArea) cameraPositioned = true;
        initializingViewport = false;
        resizing = false;
        notifyViewport(true);
    }

    export function captureView(): MapCamera | undefined {
        const camera = captureCamera();
        if (!camera) return;
        return { center: camera.center, zoom: camera.zoom, bearing: camera.bearing, pitch: camera.pitch };
    }
    export function focusCamera(camera: MapCamera) {
        if (!ready || !map) return false;
        resizing = true;
        resizeMap();
        map.jumpTo(camera);
        hiddenArea = undefined;
        cameraPositioned = true;
        resizing = false;
        notifyViewport(true);
        return true;
    }

    function resetViewport() {
        if (!ready || !map) return;
        resizing = true;
        focusDefaultViewport();
        resizing = false;
        notifyViewport();
    }

    function focusDefaultViewport() {
        // Ignore URL/snapshot bounds and place filters when explicitly returning
        // to the default view. Keep the search criteria themselves unchanged.
        map?.jumpTo({ bearing: 0, pitch: 0 });
        const request = ++locationRequest;
        const hasTrails = boundingBox.has_trails ?? (boundingBox.min_lon !== 0 || boundingBox.max_lon !== 0 || boundingBox.min_lat !== 0 || boundingBox.max_lat !== 0);
        if (settings?.mapFocus === "trails" && hasTrails) {
            if (boundingBox.min_lon === boundingBox.max_lon && boundingBox.min_lat === boundingBox.max_lat) {
                map?.jumpTo({ center: [boundingBox.min_lon, boundingBox.min_lat], zoom: 12 });
            } else {
                map?.fitBounds([[boundingBox.min_lon, boundingBox.min_lat], [boundingBox.max_lon, boundingBox.max_lat]], { animate: false, padding: 32 });
            }
        } else if (settings?.mapFocus === "location" && settings.location) {
            map?.jumpTo({ center: [settings.location.lon, settings.location.lat], zoom: 12 });
        } else {
            if (defaultCamera) map?.jumpTo(defaultCamera);
            navigator.geolocation?.getCurrentPosition(position => {
                if (!ready || !map || request !== locationRequest || map.isMoving()) return;
                const previous = resizing;
                resizing = true;
                map.jumpTo({ center: [position.coords.longitude, position.coords.latitude], zoom: 12 });
                resizing = previous;
                lastViewport = undefined;
                notifyViewport(true);
            }, () => { /* Keep the default camera when location is unavailable. */ });
        }
    }

    function persistLayout() {
        try {
            const overlay = overlaySize || overlayPosition || overlayHidden
                ? { ...overlaySize, ...overlayPosition, hidden: overlayHidden || undefined } : undefined;
            localStorage.setItem(layoutStorageKey, JSON.stringify({ width: preferredWidth, collapsed: layoutMode === "map", expanded: resultsExpanded, overlay }));
        } catch { /* The splitter still works when browser storage is disabled. */ }
    }
    function beginResize(event: PointerEvent) {
        if (event.button !== 0) return;
        const target = event.currentTarget as HTMLElement;
        event.preventDefault();
        target.focus();
        drag = { pointer: event.pointerId, pointerType: event.pointerType, startX: event.clientX, width: listWidth,
            preferred: preferredWidth, mode: layoutMode, target };
        ++layoutRevision;
        target.setPointerCapture(event.pointerId);
        resizing = true;
        dragging = true;
    }
    function moveResize(event: PointerEvent) {
        if (!drag || event.pointerId !== drag.pointer) return;
        layoutMode = "split";
        preferredWidth = Math.max(minimumListWidth, Math.min(maxListWidth, drag.width + event.clientX - drag.startX));
    }
    async function finishResize(event?: PointerEvent, cancel = false) {
        if (!drag || (event && event.pointerId !== drag.pointer)) return;
        const current = drag;
        drag = undefined;
        if (cancel) {
            preferredWidth = current.preferred;
            layoutMode = current.mode;
        }
        if (current.target.hasPointerCapture(current.pointer)) current.target.releasePointerCapture(current.pointer);
        dragging = false;
        if (!cancel) persistLayout();
        await settleLayout(undefined, true, true);
    }
    async function toggleResults() {
        await changeLayout(resultsCollapsed ? "split" : "map");
        root.querySelector<HTMLButtonElement>(".collapse-results")?.focus();
    }
    function setOverlaySize(width: number, height: number) {
        overlaySize = {
            width: Math.round(Math.max(minOverlayWidth, Math.min(maxOverlayWidth, width))),
            height: Math.round(Math.max(minOverlayHeight, Math.min(maxOverlayHeight, height))),
        };
    }
    function beginOverlayResize(event: PointerEvent) {
        if (event.button !== 0 || !miniMap) return;
        event.preventDefault();
        event.stopPropagation();
        const target = event.currentTarget as HTMLElement;
        target.focus();
        overlayDrag = { pointer: event.pointerId, pointerType: event.pointerType, x: event.clientX, y: event.clientY,
            size: { width: overlayWidth, height: overlayHeight }, preferred: overlaySize, area: lastViewport, target };
        ++layoutRevision;
        resizing = true;
        overlayDragging = true;
        target.setPointerCapture(event.pointerId);
    }
    function moveOverlayResize(event: PointerEvent) {
        if (!overlayDrag || overlayDrag.pointer !== event.pointerId) return;
        setOverlaySize(overlayDrag.size.width + overlayDrag.x - event.clientX,
            overlayDrag.size.height + overlayDrag.y - event.clientY);
    }
    async function finishOverlayResize(event?: PointerEvent, cancel = false) {
        if (!overlayDrag || (event && overlayDrag.pointer !== event.pointerId)) return;
        const current = overlayDrag;
        overlayDrag = undefined;
        applyOverlayMinimum();
        if (cancel) overlaySize = current.preferred;
        if (current.target.hasPointerCapture(current.pointer)) current.target.releasePointerCapture(current.pointer);
        overlayDragging = false;
        if (!cancel) persistLayout();
        await settleLayout(current.area);
    }
    async function resizeOverlayKey(event: KeyboardEvent) {
        if (event.key === "Escape" && overlayDrag) {
            event.preventDefault();
            void finishOverlayResize(undefined, true);
            return;
        }
        const step = event.shiftKey ? 64 : 16;
        let width = overlayWidth;
        let height = overlayHeight;
        if (event.key === "ArrowLeft") width += step;
        else if (event.key === "ArrowRight") width -= step;
        else if (event.key === "ArrowUp") height += step;
        else if (event.key === "ArrowDown") height -= step;
        else if (event.key === "Home") { width = minOverlayWidth; height = minOverlayHeight; }
        else if (event.key === "End") { width = maxOverlayWidth; height = maxOverlayHeight; }
        else return;
        event.preventDefault();
        resizing = true;
        setOverlaySize(width, height);
        persistLayout();
        await settleLayout(lastViewport);
    }
    async function toggleExpanded() {
        await changeLayout(resultsExpanded ? "split" : "results");
        root.querySelector<HTMLButtonElement>(".expand-results")?.focus();
    }
    async function setOverlayHidden(hidden: boolean) {
        overlayHidden = hidden;
        persistLayout();
        if (hidden) await tick();
        else {
            resizing = true;
            await settleLayout(lastViewport);
        }
        root.querySelector<HTMLButtonElement>(hidden ? ".overlay-reopen" : ".overlay-close")?.focus();
    }
    function setOverlayPosition(right: number, bottom: number) {
        overlayPosition = {
            right: Math.round(Math.max(0, Math.min(maxOverlayRight, right))),
            bottom: Math.round(Math.max(overlayBottomInset, Math.min(maxOverlayBottom, bottom))),
        };
    }
    function beginOverlayMove(event: PointerEvent) {
        if (event.button !== 0 || !miniMap || overlayHidden) return;
        event.preventDefault();
        event.stopPropagation();
        const target = event.currentTarget as HTMLElement;
        target.focus();
        overlayMove = { pointer: event.pointerId, pointerType: event.pointerType, x: event.clientX, y: event.clientY,
            position: { right: overlayRight, bottom: overlayBottom }, preferred: overlayPosition, target };
        ++layoutRevision;
        resizing = true;
        target.setPointerCapture(event.pointerId);
    }
    function moveOverlay(event: PointerEvent) {
        if (!overlayMove || overlayMove.pointer !== event.pointerId) return;
        setOverlayPosition(overlayMove.position.right + overlayMove.x - event.clientX,
            overlayMove.position.bottom + overlayMove.y - event.clientY);
    }
    async function finishOverlayMove(event?: PointerEvent, cancel = false) {
        if (!overlayMove || (event && overlayMove.pointer !== event.pointerId)) return;
        const current = overlayMove;
        overlayMove = undefined;
        const resized = applyOverlayMinimum();
        if (cancel) overlayPosition = current.preferred;
        if (current.target.hasPointerCapture(current.pointer)) current.target.releasePointerCapture(current.pointer);
        if (!cancel) persistLayout();
        await settleLayout(resized ? lastViewport : undefined, !resized);
    }
    function moveOverlayKey(event: KeyboardEvent) {
        if (event.key === "Escape" && overlayMove) {
            event.preventDefault();
            event.stopPropagation();
            finishOverlayMove(undefined, true);
            return;
        }
        const step = event.shiftKey ? 64 : 16;
        let right = overlayRight;
        let bottom = overlayBottom;
        if (event.key === "ArrowLeft") right += step;
        else if (event.key === "ArrowRight") right -= step;
        else if (event.key === "ArrowUp") bottom += step;
        else if (event.key === "ArrowDown") bottom -= step;
        else if (event.key === "Home") { right = 0; bottom = overlayBottomInset; }
        else if (event.key === "End") { right = maxOverlayRight; bottom = maxOverlayBottom; }
        else return;
        event.preventDefault();
        setOverlayPosition(right, bottom);
        persistLayout();
    }
    async function changeLayout(mode: LayoutMode) {
        const area = lastViewport;
        resizing = true;
        layoutMode = mode;
        persistLayout();
        await settleLayout(area);
    }
    export async function refreshLayout() {
        if (!ready) return;
        resizing = true;
        await settleLayout(lastViewport);
    }
    async function settleLayout(area: TrailSearchBounds | undefined, preserveCamera = false, publish = false) {
        if (gestureActive()) {
            deferredSettle = { area: map?.isMoving() ? undefined : area ?? deferredSettle?.area,
                preserveCamera, publish: publish || Boolean(deferredSettle?.publish) };
            return false;
        }
        const revision = ++layoutRevision;
        await tick();
        if (revision !== layoutRevision || !root.isConnected) return false;
        resizeLayout();
        if (mapVisible()) resizeMap();
        await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
        if (revision !== layoutRevision || !root.isConnected) return false;
        if (gestureActive()) {
            deferredSettle = { area: undefined, preserveCamera, publish: publish || Boolean(deferredSettle?.publish) };
            return false;
        }
        if (deferredSettle?.publish) deferredResize = { preserveSearch: false };
        publish ||= deferredResize?.preserveSearch === false || resizeTimer !== undefined;
        const requestedArea = publish ? undefined : area;
        clearTimeout(resizeTimer);
        resizeTimer = undefined;
        resizing = true;
        if (map && mapVisible()) {
            if (requestedArea) resizeToArea(requestedArea);
            else resizeMap();
        }
        deferredResize = undefined;
        deferredSettle = undefined;
        resizing = Boolean(drag || overlayDrag || overlayMove);
        const locked = document.querySelector('dialog:modal') || document.body.style.position === 'fixed';
        if (publish && !locked) notifyViewport();
        else if (preserveCamera) renderViewport();
        return true;
    }
    async function resizeKey(event: KeyboardEvent) {
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
        resizing = true;
        layoutMode = "split";
        preferredWidth = Math.max(minimumListWidth, Math.min(maxListWidth, width));
        persistLayout();
        await settleLayout(undefined, true, true);
    }

    async function toggleMap() {
        const previous = lastViewport;
        resizing = true;
        showMap = !showMap;
        if (!showMap && map?.isMoving()) {
            map.stop();
            cameraPositioned = false;
        }
        await tick();
        if (showMap) {
            resizeToArea(previous);
        }
        resizing = false;
        if (showMap) notifyViewport();
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

{#snippet expandResultsButton()}
    <button type="button" class="btn-icon pane-button expand-results" class:map-overlay-button={miniMap} onclick={toggleExpanded} aria-pressed={resultsExpanded}
        aria-controls={`${id}-trail-list trail-map`} aria-label={$_(resultsExpanded ? "trail-map-restore-split" : "trail-map-expand-results")}
        title={$_(resultsExpanded ? "trail-map-restore-split" : "trail-map-expand-results")}>
        <i class="fa fa-expand" aria-hidden="true"></i>
    </button>
{/snippet}

<div bind:this={root} class="map-results" data-layout-ready={!resizing} class:show-map={showMap} class:results-collapsed={resultsCollapsed} class:results-expanded={miniMap} class:dragging class:resizing-overlay={overlayDragging} class:moving-overlay={Boolean(overlayMove)}
    style:--results-width={`${listWidth}px`} style:--overlay-width={`${overlayWidth}px`} style:--overlay-height={`${overlayHeight}px`}
    style:--overlay-min-height={`${minOverlayHeight}px`} style:--overlay-right={`${overlayRight}px`} style:--overlay-bottom={`${overlayBottom}px`} style:--pane-gap={`${paneGap}px`}>
    <button type="button" class="map-view-toggle" onclick={toggleMap}
        aria-label={$_(showMap ? "trail-map-show-results" : "trail-map-show-map")} aria-pressed={!showMap}>
        <i class={`fa ${showMap ? "fa-list-ul" : "fa-map"}`} aria-hidden="true"></i>
        {$_(showMap ? "trail-map-show-results" : "trail-map-show-map")}
    </button>
    <div id={`${id}-trail-list`} class="trail-list">
        {#if !miniMap && !resultsCollapsed}
            <div class="result-header">
                <span>{$_("trail", { values: { n: 2 } })}</span>
                <div class="pane-actions">{@render expandResultsButton()}{@render toggleResultsButton()}</div>
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
        {#if !miniMap}
            <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions (A focusable separator implements the WAI-ARIA window splitter pattern.) -->
            <div class="list-resize" role="separator" tabindex="0" aria-orientation="vertical" aria-controls={`${id}-trail-list`}
                aria-label={$_("trail-map-resize-results")} aria-valuemin={Math.round(minimumListWidth)} aria-valuemax={Math.round(maxListWidth)} aria-valuenow={Math.round(listWidth)}
                oncontextmenu={event => event.preventDefault()} onpointerdown={beginResize} onpointermove={moveResize} onpointerup={finishResize}
                onpointercancel={event => finishResize(event, true)} onlostpointercapture={event => finishResize(event, true)}
                onkeydown={resizeKey}></div>
        {/if}
    </div>
    {#if resultsCollapsed}
        <div class="collapsed-actions">{@render toggleResultsButton()}</div>
    {/if}
    <div class="map-pane" class:overview-hidden={miniMap && overlayHidden}>
        <div id="trail-map" bind:this={mapElement} aria-busy={mapLoading} aria-hidden={miniMap && overlayHidden} inert={miniMap && overlayHidden}>
            <!-- This component owns resizing; MapLibre's delayed resize observer
                 would otherwise publish layout changes as map movements. -->
            <MapWithElevationMaplibre onmoveend={handleMoveEnd} oninit={handleMapInit}
                mapOptions={{ trackResize: false }}
                trails={mapTrails} serverClusters={clusters} showElevation={false} showTerrain={true}
                showInfoPopup={!miniMap} showClusterPopups={!miniMap} activeTrail={-1} fitBounds="off" clusterTrails={true}
                bind:map bind:this={mapWithElevation} />
            {#if miniMap && !overlayHidden}
                <button type="button" class="overlay-resize" aria-label={$_("trail-map-resize-overlay")}
                    aria-describedby={`${id}-overlay-help`} title={$_("trail-map-resize-overlay")}
                    oncontextmenu={event => event.preventDefault()} onpointerdown={beginOverlayResize} onpointermove={moveOverlayResize} onpointerup={finishOverlayResize}
                    onpointercancel={event => finishOverlayResize(event, true)} onlostpointercapture={event => finishOverlayResize(event, true)}
                    onkeydown={resizeOverlayKey}></button>
                <span id={`${id}-overlay-help`} class="sr-only">{$_("trail-map-resize-overlay-help")}</span>
            {/if}
        </div>
        {#if miniMap && !overlayHidden}
            <div class="overview-actions">
                {@render expandResultsButton()}
                <button type="button" class="btn-icon pane-button map-overlay-button overlay-move" aria-controls="trail-map"
                    aria-label={$_("trail-map-move-overlay")} title={$_("trail-map-move-overlay")} aria-describedby={`${id}-move-overlay-help`}
                    oncontextmenu={event => event.preventDefault()} onpointerdown={beginOverlayMove} onpointermove={moveOverlay} onpointerup={finishOverlayMove}
                    onpointercancel={event => finishOverlayMove(event, true)} onlostpointercapture={event => finishOverlayMove(event, true)}
                    onkeydown={moveOverlayKey}>
                    <i class="fa fa-up-down-left-right" aria-hidden="true"></i>
                </button>
                <button type="button" class="btn-icon pane-button map-overlay-button overlay-close" aria-controls="trail-map"
                    aria-label={$_("trail-map-hide-overlay")} title={$_("trail-map-hide-overlay")} onclick={() => setOverlayHidden(true)}>
                    <i class="fa fa-xmark" aria-hidden="true"></i>
                </button>
                <span id={`${id}-move-overlay-help`} class="sr-only">{$_("trail-map-move-overlay-help")}</span>
            </div>
        {/if}
    </div>
    {#if miniMap && overlayHidden}
        <button type="button" class="btn-icon pane-button map-overlay-button overlay-reopen" aria-controls="trail-map"
            aria-label={$_("trail-map-show-overlay")} title={$_("trail-map-show-overlay")} onclick={() => setOverlayHidden(false)}>
            <i class="fa fa-map" aria-hidden="true"></i>
        </button>
    {/if}
</div>

<MergeDialog />

<style>
    .map-results { display: grid; gap: var(--pane-gap); grid-template-columns: var(--results-width) minmax(0, 1fr); height: calc(100dvh - 225px); min-height: 280px; position: relative; }
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
    /* Keep release events inside MapLibre when the surrounding layout grows
       during a drag, before its deferred canvas resize. */
    .map-pane :global(.maplibregl-canvas-container) { position: absolute; inset: 0; }
    #trail-map { border: 1px solid rgba(var(--input-border)); border-radius: .9rem; height: 100%; min-height: 0; overflow: hidden; }
    .overview-hidden { pointer-events: none; }
    .overview-hidden #trail-map { opacity: 0; visibility: hidden; }
    .map-view-toggle { display: none; }
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
    .overview-actions { display: flex; gap: .25rem; position: absolute; right: .75rem; top: -1rem; z-index: 3; }
    .overview-actions .pane-button { height: 2rem; width: 2rem; }
    .overview-actions .pane-button i { font-size: 1.125rem; }
    .overlay-reopen { bottom: 1.5rem; height: 4rem; position: fixed; right: 1.5rem; width: 4rem; z-index: 3; }
    .overlay-reopen i { font-size: 2rem; }
    .overlay-move { cursor: grab; touch-action: none; }
    .moving-overlay, .moving-overlay .overlay-move { cursor: grabbing; user-select: none; }
    .dragging { cursor: col-resize; user-select: none; }
    .resizing-overlay { cursor: nwse-resize; user-select: none; }
    .overlay-resize { background: transparent; border: 0; border-radius: .5rem 0 0 0; cursor: nwse-resize; display: none; height: 1.5rem; left: 0; position: absolute; top: 0; touch-action: none; width: 1.5rem; z-index: 3; }
    .overlay-resize:focus-visible { outline: 2px solid rgba(var(--input-border-focus)); outline-offset: -3px; }
    @media (width >= 768px) {
        .results-collapsed .trail-list, .results-collapsed .selection-toolbar { display: contents; }
        .results-collapsed .trail-results, .results-collapsed .list-resize,
        .results-collapsed .selection-toolbar > :global(:not(dialog)) { display: none; }
        .results-expanded { grid-template-columns: minmax(0, 1fr); min-height: calc(var(--overlay-min-height) + 112px); }
        .results-expanded .trail-list { background: transparent; border: 0; }
        .results-expanded .trail-results { padding: 0 .25rem calc(var(--overlay-height) + 4.5rem) 0; }
        .results-expanded .map-pane { bottom: var(--overlay-bottom); height: var(--overlay-height); position: absolute; right: var(--overlay-right); width: var(--overlay-width); z-index: 2; }
        .results-expanded #trail-map { box-shadow: 0 3px 16px rgb(0 0 0 / .25); }
        .results-expanded #trail-map :global(.maplibregl-ctrl-top-right) { padding-top: .75rem; }
        .results-expanded .overlay-resize { display: flex; }
    }
    @media (width < 768px) {
        .result-header, .list-resize, .collapsed-actions { display: none; }
        .map-results { display: flex; flex-direction: column; gap: .5rem; }
        .map-view-toggle { align-items: center; align-self: flex-start; background: rgba(var(--input-background)); border: 1px solid rgba(var(--input-border)); border-radius: .5rem; cursor: pointer; display: inline-flex; font-size: .8rem; gap: .5rem; padding: .5rem .8rem; }
        .trail-list, .map-pane { flex: 1; min-height: 0; }
        .map-results:not(.show-map) .map-pane { display: none; }
        .map-results.show-map .trail-list,
        .map-results.show-map .selection-toolbar { display: contents; }
        .map-results.show-map .trail-results,
        .map-results.show-map .list-resize,
        .map-results.show-map .selection-toolbar > :global(:not(dialog)) { display: none; }
    }
</style>
