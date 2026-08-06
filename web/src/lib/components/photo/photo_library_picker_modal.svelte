<script lang="ts">
    import "maplibre-gl/dist/maplibre-gl.css";

    import type { Feature, FeatureCollection, Geometry, Point, Position } from "geojson";
    import * as M from "maplibre-gl";
    import Supercluster from "supercluster";
    import { onDestroy, tick, untrack } from "svelte";
    import { _ } from "svelte-i18n";

    import GPX from "$lib/models/gpx/gpx";
    import type { PhotoLibraryCandidate } from "$lib/models/photo_library";
    import type { PluginProvider } from "$lib/models/plugin_provider";
    import { baseMapStyles } from "$lib/vendor/maplibre-layer-manager/layers";
    import { markerElement, syncMarkerHighlightClass } from "$lib/util/maplibre_util";
    import { polylineToGeoJSON } from "$lib/util/polyline_util";
    import DoubleSlider from "../base/double_slider.svelte";
    import Modal from "../base/modal.svelte";

    interface LibraryOutput {
        candidates?: PhotoLibraryCandidate[];
        existingExternalRefs?: { provider: string; id: string }[];
        hasMore?: boolean;
        cursorId?: string;
        restartRequired?: boolean;
        error?: { message?: string };
    }

    interface PluginCandidateBatch {
        candidates: PhotoLibraryCandidate[];
        restartRequired: boolean;
    }

    interface PluginPaginationState {
        cursorId?: string;
        hasMore: boolean;
        loading: boolean;
    }

    interface PhotoCluster {
        key: string;
        lat: number;
        lon: number;
        candidates: PhotoLibraryCandidate[];
    }

    interface PhotoPointProperties {
        candidateKey: string;
    }

    type PhotoPoint = Feature<Point, PhotoPointProperties>;

    const WANDERER_PROVIDER_ID = "wanderer";
    const WANDERER_LOGO = "/favicon.png";
    const DAY_MS = 24 * 60 * 60 * 1000;
    const DATE_RANGE_YEAR_STEPS = [1, 5, 10];
    const DATE_RANGE_EXPAND_THRESHOLD = 1 / 3;
    const DEFAULT_LOOKBACK_YEARS = 1;
    const WANDERER_LIBRARY_PER_PAGE = 100;

    interface Props {
        id?: string;
        trailId?: string;
        waypointId?: string;
        summitLogId?: string;
        lat?: number;
        lon?: number;
        trailData?: string;
        trailPolyline?: string;
        assetPluginIds?: string[];
        assetPluginProviders?: PluginProvider[];
        doubleRadius?: boolean;
        title?: string;
        actionLabel?: string;
        onselect?: (candidates: PhotoLibraryCandidate[]) => void | Promise<void>;
    }

    const TRAIL_LINE_COLOR = "#3549bb";

    let {
        id = "photo-library-picker-modal",
        trailId,
        waypointId,
        summitLogId,
        lat,
        lon,
        trailData,
        trailPolyline,
        assetPluginIds,
        assetPluginProviders = [],
        doubleRadius = false,
        title,
        actionLabel,
        onselect,
    }: Props = $props();

    let modal: Modal;
    let mapContainer: HTMLDivElement = $state()!;
    let map: M.Map | undefined;
    let mapReady = $state(false);
    let markers: M.Marker[] = [];
    let popup: M.Popup | undefined;

    let loading = $state(false);
    let loadingMore = $state(false);
    let saving = $state(false);
    let error = $state("");
    let fatalError = $state(false);
    let candidates = $state<PhotoLibraryCandidate[]>([]);
    let selectedKeys = $state(new Set<string>());
    let providerFilter = $state("all");
    let activeClusterKey = $state<string | undefined>();
    let highlightedClusterKey = $state<string | undefined>();
    let previewIndex = $state(0);
    let existingWandererExternalKeys = $state(new Set<string>());
    let mapZoom = $state(0);
    let mapBounds = $state<M.LngLatBounds | undefined>();
    let dateRange = $state(dateSliderDefaults());
    let wandererPage = $state(1);
    let wandererHasMore = $state(false);
    let pluginPagination = $state<Record<string, PluginPaginationState>>({});

    onDestroy(() => {
        destroyMap();
    });

    const activeAssetPluginIds = $derived(
        assetPluginProviders.length > 0 ? Array.from(new Set((assetPluginIds ?? []).filter(Boolean))) : [],
    );
    const hasAssetPlugin = $derived(activeAssetPluginIds.length > 0);
    const providerCandidates = $derived(filterByProvider(candidates));
    const clusters = $derived(mapReady ? buildClusters(providerCandidates, mapZoom, mapBounds) : []);
    const mapCandidates = $derived(filterByMapBounds(providerCandidates));
    const visibleCandidates = $derived(activeClusterKey ? filterByActiveCluster(providerCandidates) : mapCandidates);
    const selectedCandidates = $derived(candidates.filter((candidate) => selectedKeys.has(candidateKey(candidate))));
    const providerOptions = $derived(buildProviderOptions(candidates));
    const showWandererLoadMore = $derived(wandererHasMore && (providerFilter === "all" || providerFilter === WANDERER_PROVIDER_ID));
    const pluginLoadMoreIds = $derived(
        activeAssetPluginIds.filter(
            (pluginId) => pluginPagination[pluginId]?.hasMore && (providerFilter === "all" || providerFilter === pluginId),
        ),
    );
    const showLoadMore = $derived(showWandererLoadMore || pluginLoadMoreIds.length > 0);
    const dateRangeLabel = $derived(formatDateRangeLabel(dateRange.start, dateRange.end, dateRange.min, dateRange.max));
    const previewCandidate = $derived(visibleCandidates[Math.min(previewIndex, Math.max(visibleCandidates.length - 1, 0))]);
    const mapFiltered = $derived(
        Boolean(mapBounds) &&
        mapCandidates.length < providerCandidates.filter(hasMarkerCoordinates).length,
    );

    $effect(() => {
        if (providerFilter !== "all" && !providerOptions.some((option) => option.id === providerFilter)) {
            providerFilter = "all";
        }
    });

    $effect(() => {
        if (activeClusterKey && !clusters.some((cluster) => cluster.key === activeClusterKey)) {
            activeClusterKey = undefined;
        }
    });

    $effect(() => {
        if (previewIndex >= visibleCandidates.length) {
            previewIndex = Math.max(visibleCandidates.length - 1, 0);
        }
    });

    $effect(() => {
        const ready = mapReady;
        const syncKey = mapSyncKey();
        if (ready && syncKey) {
            untrack(() => syncMap());
        }
    });

    $effect(() => {
        const ready = mapReady;
        const currentClusters = clusters;
        if (ready) {
            untrack(() => syncPhotoMarkers(currentClusters));
        }
    });

    $effect(() => {
        highlightedClusterKey;
        activeClusterKey;
        syncMarkerStyles();
    });

    $effect(() => {
        const next = dateSliderDefaults();
        if (next.resetKey !== dateRange.resetKey) {
            dateRange = next;
        }
    });

    export async function openModal() {
        modal.openModal();
        destroyMap();
        await tick();
        await loadCandidates();
        await tick();
        ensureMap();
        map?.resize();
        syncMap();
    }

    function requestBody(extra: Record<string, unknown> = {}) {
        return {
            ...(trailId ? { trailId } : {}),
            ...(waypointId ? { waypointId } : {}),
            ...(summitLogId ? { summitLogId } : {}),
            ...(hasCoordinate(lat) && hasCoordinate(lon) ? { lat, lon } : {}),
            ...(!trailId && trailData ? { trailData } : {}),
            ...(trailPolyline ? { trailPolyline } : {}),
            ...selectedTimeWindow(),
            doubleRadius,
            ...extra,
        };
    }

    async function loadCandidates() {
        loading = true;
        error = "";
        fatalError = false;
        candidates = [];
        selectedKeys = new Set();
        existingWandererExternalKeys = new Set();
        wandererPage = 1;
        wandererHasMore = false;
        pluginPagination = {};
        activeClusterKey = undefined;
        highlightedClusterKey = undefined;
        mapZoom = 0;
        mapBounds = undefined;
        previewIndex = 0;
        try {
            const loaders: Promise<PhotoLibraryCandidate[]>[] = [
                loadWandererCandidates(1),
                ...activeAssetPluginIds.map(async (id) => (await loadPluginCandidates(id)).candidates),
            ];

            const settled = await Promise.allSettled(loaders);
            const fulfilled = settled.filter(
                (item): item is PromiseFulfilledResult<PhotoLibraryCandidate[]> => item.status === "fulfilled",
            );
            const result = fulfilled.flatMap((item) => item.value);
            const failures = settled.filter((item): item is PromiseRejectedResult => item.status === "rejected");

            if (!fulfilled.length && failures.length) {
                throw failures[0].reason;
            }
            candidates = uniqueCandidates(result);
            if (failures.length) {
                error = failures
                    .map((failure) => failure.reason?.message ?? String(failure.reason))
                    .filter(Boolean)
                    .join("; ");
            }
        } catch (e: any) {
            console.error(e);
            fatalError = true;
            error = e?.message ?? "Unable to load photos";
        } finally {
            loading = false;
        }
    }

    async function loadPluginCandidates(pluginId: string, cursorId?: string): Promise<PluginCandidateBatch> {
        const pluginPath = `/api/v1/plugins/assets/${encodeURIComponent(pluginId)}`;
        const r = await fetch(`${pluginPath}/candidates`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(requestBody(cursorId ? { cursorId } : {})),
        });
        if (!r.ok) {
            const response = await r.json().catch(() => ({}));
            throw new Error(response.message ?? "Unable to load plugin photos");
        }
        const output: LibraryOutput = await r.json();
        if (output.error) {
            throw new Error(output.error.message ?? "Unable to load plugin photos");
        }
        if (output.restartRequired) {
            pluginPagination[pluginId] = { hasMore: false, loading: false };
            return { candidates: [], restartRequired: true };
        }
        if (output.hasMore === true && !output.cursorId) {
            throw new Error("Plugin candidate response is missing its cursor");
        }
        pluginPagination[pluginId] = {
            cursorId: output.cursorId,
            hasMore: output.hasMore === true && Boolean(output.cursorId),
            loading: false,
        };
        const pluginCandidates: PhotoLibraryCandidate[] = (output.candidates ?? []).map((candidate) => ({
            ...candidate,
            source: "plugin" as const,
            pluginId,
            providerId: candidate.providerId ?? pluginId,
            externalProvider: candidate.externalProvider ?? candidate.providerId ?? pluginId,
            externalId: candidate.externalId ?? candidate.assetId,
            thumbnailUrl: candidate.thumbnailUrl ?? pluginThumbnailUrl({
                ...candidate,
                pluginId,
                providerId: candidate.providerId ?? pluginId,
            }),
        }));
        return { candidates: pluginCandidates, restartRequired: false };
    }

    async function loadMorePluginCandidates(pluginId: string) {
        const pagination = pluginPagination[pluginId];
        if (!pagination?.hasMore || !pagination.cursorId || pagination.loading) {
            return;
        }
        pagination.loading = true;
        error = "";
        try {
            let batch = await loadPluginCandidates(pluginId, pagination.cursorId);
            if (batch.restartRequired) {
                candidates = candidates.filter((candidate) => candidate.pluginId !== pluginId);
                batch = await loadPluginCandidates(pluginId);
            }
            candidates = uniqueCandidates([...candidates, ...batch.candidates]);
            await tick();
            syncMap();
        } catch (e: any) {
            console.error(e);
            error = e?.message ?? "Unable to load plugin photos";
            pluginPagination[pluginId] = { ...pagination, loading: false };
        }
    }

    async function loadWandererCandidates(page: number): Promise<PhotoLibraryCandidate[]> {
        const r = await fetch("/api/v1/assets/library", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(requestBody({
                page,
                perPage: WANDERER_LIBRARY_PER_PAGE,
            })),
        });
        if (!r.ok) {
            const response = await r.json().catch(() => ({}));
            throw new Error(response.message ?? "Unable to load wanderer photos");
        }
        const output: LibraryOutput = await r.json();
        const refs = output.existingExternalRefs ?? [];
        existingWandererExternalKeys = new Set(refs.map((ref) => externalRefKey(ref.provider, ref.id)));
        wandererPage = page;
        wandererHasMore = output.hasMore === true;
        return (output.candidates ?? []).map((candidate) => ({
            ...candidate,
            source: "wanderer",
            providerId: candidate.providerId ?? "wanderer",
        }));
    }

    async function loadMoreWandererCandidates() {
        if (!wandererHasMore || loadingMore) {
            return;
        }
        loadingMore = true;
        error = "";
        try {
            const next = await loadWandererCandidates(wandererPage + 1);
            candidates = uniqueCandidates([...candidates, ...next]);
            await tick();
            syncMap();
        } catch (e: any) {
            console.error(e);
            error = e?.message ?? "Unable to load wanderer photos";
        } finally {
            loadingMore = false;
        }
    }

    function uniqueCandidates(items: PhotoLibraryCandidate[]) {
        const byIdentity = new Map<string, PhotoLibraryCandidate>();
        for (const candidate of items.filter((item) => !isKnownWandererPluginDuplicate(item))) {
            const key = candidateIdentityKey(candidate);
            const existing = byIdentity.get(key);
            if (!existing || preferCandidate(candidate, existing)) {
                byIdentity.set(key, candidate);
            }
        }
        return Array.from(byIdentity.values()).sort((a, b) => {
            if ((a.distanceFromStart ?? 0) !== (b.distanceFromStart ?? 0)) {
                return (a.distanceFromStart ?? 0) - (b.distanceFromStart ?? 0);
            }
            if ((a.distance ?? 0) !== (b.distance ?? 0)) {
                return (a.distance ?? 0) - (b.distance ?? 0);
            }
            return Date.parse(b.takenAt || "0") - Date.parse(a.takenAt || "0");
        });
    }

    function candidateIdentityKey(candidate: PhotoLibraryCandidate) {
        const externalProvider = candidate.externalProvider ?? (candidate.source === "plugin" ? providerId(candidate) : undefined);
        const externalId = candidate.externalId ?? (candidate.source === "plugin" ? candidate.assetId : undefined);
        if (externalProvider && externalId) {
            return `external:${externalProvider}:${externalId}`;
        }
        return `asset:${candidateKey(candidate)}`;
    }

    function preferCandidate(candidate: PhotoLibraryCandidate, existing: PhotoLibraryCandidate) {
        if (candidate.source === "wanderer" && existing.source !== "wanderer") {
            return true;
        }
        if (candidate.source !== "wanderer" && existing.source === "wanderer") {
            return false;
        }
        return Boolean(candidate.thumbnailUrl) && !existing.thumbnailUrl;
    }

    function isKnownWandererPluginDuplicate(candidate: PhotoLibraryCandidate) {
        if (candidate.source !== "plugin") {
            return false;
        }
        const externalProvider = candidate.externalProvider ?? providerId(candidate);
        const externalId = candidate.externalId ?? candidate.assetId;
        return existingWandererExternalKeys.has(externalRefKey(externalProvider, externalId));
    }

    function externalRefKey(provider: string, id: string) {
        return `${provider}:${id}`;
    }

    function filterByProvider(items: PhotoLibraryCandidate[]) {
        if (providerFilter === "all") {
            return items;
        }
        return items.filter((candidate) => displayProviderId(candidate) === providerFilter);
    }

    function filterByActiveCluster(items: PhotoLibraryCandidate[]) {
        if (!activeClusterKey) {
            return items;
        }
        const cluster = clusterForKey(activeClusterKey);
        if (!cluster) {
            return items;
        }
        const visibleKeys = new Set(cluster.candidates.map(candidateKey));
        return items.filter((candidate) => visibleKeys.has(candidateKey(candidate)));
    }

    function filterByMapBounds(items: PhotoLibraryCandidate[]) {
        if (!mapBounds) {
            return items;
        }
        return items.filter((candidate) => {
            const position = markerPosition(candidate);
            return position ? mapBounds?.contains([position.lon, position.lat]) : false;
        });
    }

    function buildClusters(items: PhotoLibraryCandidate[], zoom: number, bounds?: M.LngLatBounds) {
        const positionedCandidates = items.filter(hasMarkerCoordinates);
        const candidatesByKey = new Map(positionedCandidates.map((candidate) => [candidateKey(candidate), candidate]));
        const features: PhotoPoint[] = positionedCandidates.map((candidate) => {
            const position = markerPosition(candidate)!;
            return {
                type: "Feature",
                properties: {
                    candidateKey: candidateKey(candidate),
                },
                geometry: {
                    type: "Point",
                    coordinates: [position.lon, position.lat],
                },
            };
        });

        const index = new Supercluster<PhotoPointProperties>({
            radius: 42,
            maxZoom: 22,
            minPoints: 2,
        });
        index.load(features);

        return index.getClusters(superclusterBounds(bounds), Math.floor(zoom)).map((feature) => {
            const [lon, lat] = feature.geometry.coordinates;
            const properties = feature.properties as PhotoPointProperties & {
                cluster?: boolean;
                cluster_id?: number;
            };

            if (properties.cluster && properties.cluster_id !== undefined) {
                const candidates = index
                    .getLeaves(properties.cluster_id, positionedCandidates.length)
                    .map((leaf) => candidatesByKey.get(leaf.properties.candidateKey))
                    .filter(isPhotoLibraryCandidate);
                return {
                    key: `cluster:${properties.cluster_id}`,
                    lat,
                    lon,
                    candidates,
                };
            }

            const candidate = candidatesByKey.get(properties.candidateKey);
            if (!candidate) {
                return undefined;
            }
            return {
                key: `asset:${candidateKey(candidate)}`,
                lat,
                lon,
                candidates: [candidate],
            };
        }).filter(isPhotoCluster).sort((a, b) => {
            const distance = (a.candidates[0]?.distanceFromStart ?? 0) - (b.candidates[0]?.distanceFromStart ?? 0);
            if (distance !== 0) {
                return distance;
            }
            return a.key.localeCompare(b.key);
        });
    }

    function superclusterBounds(bounds?: M.LngLatBounds): [number, number, number, number] {
        if (!bounds) {
            return [-180, -85, 180, 85];
        }
        return [
            Math.max(-180, bounds.getWest()),
            Math.max(-85, bounds.getSouth()),
            Math.min(180, bounds.getEast()),
            Math.min(85, bounds.getNorth()),
        ];
    }

    function clusterForKey(key: string) {
        return clusters.find((cluster) => cluster.key === key);
    }

    function clusterKeyForCandidate(candidate: PhotoLibraryCandidate) {
        const key = candidateKey(candidate);
        return clusters.find((cluster) => cluster.candidates.some((item) => candidateKey(item) === key))?.key;
    }

    function isPhotoCluster(cluster: PhotoCluster | undefined): cluster is PhotoCluster {
        return Boolean(cluster);
    }

    function isPhotoLibraryCandidate(candidate: PhotoLibraryCandidate | undefined): candidate is PhotoLibraryCandidate {
        return Boolean(candidate);
    }

    function extendBoundsWithCandidates(bounds: M.LngLatBounds, items: PhotoLibraryCandidate[]) {
        let hasBounds = false;
        for (const candidate of items) {
            const position = markerPosition(candidate);
            if (!position) {
                continue;
            }
            bounds.extend([position.lon, position.lat]);
            hasBounds = true;
        }
        return hasBounds;
    }

    function buildProviderOptions(items: PhotoLibraryCandidate[]) {
        const ids = Array.from(new Set(items.map(displayProviderId))).filter(Boolean);
        return ids.map((id) => ({
            id,
            label: providerLabel(id),
            icon: providerIcon(id),
        }));
    }

    function providerId(candidate: PhotoLibraryCandidate) {
        return candidate.providerId ?? candidate.pluginId ?? WANDERER_PROVIDER_ID;
    }

    function displayProviderId(candidate: PhotoLibraryCandidate) {
        return candidate.source === "wanderer" ? WANDERER_PROVIDER_ID : providerId(candidate);
    }

    function providerLabel(id: string) {
        if (id === WANDERER_PROVIDER_ID) {
            return "wanderer";
        }
        const provider = assetPluginProviders.find((item) => item.id === id);
        return provider?.displayName ?? provider?.name ?? id;
    }

    function providerIcon(id: string) {
        if (id === WANDERER_PROVIDER_ID) {
            return WANDERER_LOGO;
        }
        const provider = assetPluginProviders.find((item) => item.id === id);
        return provider?.icon ?? provider?.iconDark;
    }

    function markerLat(candidate: PhotoLibraryCandidate) {
        return candidate.lat;
    }

    function markerLon(candidate: PhotoLibraryCandidate) {
        return candidate.lon;
    }

    function markerPosition(candidate: PhotoLibraryCandidate) {
        const lat = markerLat(candidate);
        const lon = markerLon(candidate);
        if (!hasCoordinate(lat) || !hasCoordinate(lon)) {
            return undefined;
        }
        return { lat, lon };
    }

    function hasMarkerCoordinates(candidate: PhotoLibraryCandidate) {
        return markerPosition(candidate) !== undefined;
    }

    function candidateKey(candidate: PhotoLibraryCandidate) {
        return `${candidate.source ?? "plugin"}:${providerId(candidate)}:${candidate.assetId}`;
    }

    function hasCoordinate(value: unknown): value is number {
        return typeof value === "number" && Number.isFinite(value);
    }

    function pluginThumbnailUrl(candidate: PhotoLibraryCandidate) {
        const pluginId = candidate.pluginId ?? candidate.providerId;
        if (!pluginId) {
            return "";
        }
        const plugin = encodeURIComponent(pluginId);
        return `/api/v1/plugins/assets/${plugin}/thumbnail/${encodeURIComponent(candidate.assetId)}`;
    }

    function thumbnailUrl(candidate: PhotoLibraryCandidate) {
        return candidate.thumbnailUrl ?? pluginThumbnailUrl(candidate);
    }

    function toggleCandidate(candidate: PhotoLibraryCandidate) {
        const next = new Set(selectedKeys);
        const key = candidateKey(candidate);
        if (next.has(key)) {
            next.delete(key);
        } else {
            next.add(key);
        }
        selectedKeys = next;
    }

    function selectVisible() {
        const next = new Set(selectedKeys);
        const allVisibleSelected = visibleCandidates.length > 0 && visibleCandidates.every((candidate) => next.has(candidateKey(candidate)));
        for (const candidate of visibleCandidates) {
            const key = candidateKey(candidate);
            if (allVisibleSelected) {
                next.delete(key);
            } else {
                next.add(key);
            }
        }
        selectedKeys = next;
    }

    function isSelected(candidate: PhotoLibraryCandidate) {
        return selectedKeys.has(candidateKey(candidate));
    }

    function setPreviewCandidate(candidate: PhotoLibraryCandidate) {
        const index = visibleCandidates.findIndex((item) => candidateKey(item) === candidateKey(candidate));
        if (index >= 0) {
            previewIndex = index;
        }
    }

    function nextPreview(offset: number) {
        if (!visibleCandidates.length) {
            return;
        }
        previewIndex = (previewIndex + offset + visibleCandidates.length) % visibleCandidates.length;
    }

    function clearActiveCluster() {
        activeClusterKey = undefined;
        highlightedClusterKey = undefined;
        closePopup();
    }

    function resetPhotoScope() {
        clearActiveCluster();
        fitMapToContent();
    }

    async function handleDateRangeChange([start, end]: [number, number]) {
        const nextStart = clampDateSliderValue(start);
        const nextEnd = clampDateSliderValue(end);
        const expanded = expandDateRangeIfNeeded(nextStart, nextEnd);
        if (!expanded) {
            dateRange = {
                ...dateRange,
                start: nextStart,
                end: nextEnd,
            };
        }
        destroyMap();
        await loadCandidates();
        await tick();
        ensureMap();
        map?.resize();
        syncMap();
    }

    function selectedTimeWindow() {
        const start = clampDateSliderValue(dateRange.start);
        const end = clampDateSliderValue(dateRange.end);
        return {
            ...(start > dateRange.min ? { takenAfter: new Date(start).toISOString() } : {}),
            ...(end < dateRange.max ? { takenBefore: endOfLocalDay(new Date(end)).toISOString() } : {}),
        };
    }

    function dateSliderDefaults(rangeYears?: number) {
        const now = new Date();
        const max = startOfLocalDay(now).getTime();
        const trailWindow = trailTimeWindow();
        const hasTrailWindow = Boolean(trailWindow.start || trailWindow.end);
        const years = rangeYears ?? initialDateRangeYears(now, trailWindow);
        const min = dateRangeMinForYears(now, years);
        let start: number;
        let end: number;
        if (hasTrailWindow) {
            start = trailWindow.start ? clamp(startOfLocalDay(trailWindow.start).getTime(), min, max) : min;
            end = trailWindow.end ? clamp(startOfLocalDay(trailWindow.end).getTime(), min, max) : max;
        } else {
            // No GPX timestamps: default the selection to the past year instead
            // of the full range so the request carries a bounded takenAfter and
            // the plugin does not scan the entire library.
            end = max;
            start = clamp(dateRangeMinForYears(now, DEFAULT_LOOKBACK_YEARS), min, max);
        }
        if (start > end) {
            [start, end] = [end, start];
        }
        return {
            years,
            min,
            max,
            start,
            end,
            key: dateSliderKey(min, max),
            resetKey: dateSliderResetKey(max, trailWindow),
        };
    }

    function initialDateRangeYears(now: Date, trailWindow: { start?: Date; end?: Date }) {
        const trailStart = trailWindow.start ? startOfLocalDay(trailWindow.start).getTime() : undefined;
        if (trailStart === undefined) {
            // Keep the range wider than the default lookback so the start handle
            // sits off the left edge and takenAfter is actually sent.
            return nextDateRangeYearStep(DEFAULT_LOOKBACK_YEARS) ?? lastDateRangeYearStep();
        }
        return DATE_RANGE_YEAR_STEPS.find((years) => trailStart >= dateRangeMinForYears(now, years)) ?? lastDateRangeYearStep();
    }

    function dateRangeMinForYears(now: Date, years: number) {
        const minDate = new Date(now);
        minDate.setFullYear(minDate.getFullYear() - years);
        return startOfLocalDay(minDate).getTime();
    }

    function expandDateRangeIfNeeded(start: number, end: number) {
        if (start <= dateRange.min) {
            return false;
        }
        if (start > dateRange.min + (dateRange.max - dateRange.min) * DATE_RANGE_EXPAND_THRESHOLD) {
            return false;
        }
        const nextYears = nextDateRangeYearStep(dateRange.years);
        if (!nextYears) {
            return false;
        }
        const nextRange = dateSliderBounds(nextYears);
        dateRange = {
            ...dateRange,
            years: nextYears,
            min: nextRange.min,
            max: nextRange.max,
            start: clamp(start, nextRange.min, nextRange.max),
            end: clamp(end, nextRange.min, nextRange.max),
            key: dateSliderKey(nextRange.min, nextRange.max),
        };
        return true;
    }

    function dateSliderBounds(years: number) {
        const now = new Date();
        return {
            min: dateRangeMinForYears(now, years),
            max: startOfLocalDay(now).getTime(),
        };
    }

    function nextDateRangeYearStep(current: number) {
        return DATE_RANGE_YEAR_STEPS.find((years) => years > current);
    }

    function lastDateRangeYearStep() {
        return DATE_RANGE_YEAR_STEPS[DATE_RANGE_YEAR_STEPS.length - 1] ?? 10;
    }

    function dateSliderKey(min: number, max: number) {
        return `${min}:${max}`;
    }

    function dateSliderResetKey(max: number, trailWindow: { start?: Date; end?: Date }) {
        return `${max}:${trailWindow.start?.getTime() ?? ""}:${trailWindow.end?.getTime() ?? ""}`;
    }

    function trailTimeWindow() {
        if (!trailData) {
            return {};
        }
        try {
            const gpx = GPX.parse(trailData);
            let start: Date | undefined;
            let end: Date | undefined;
            for (const track of gpx.trk ?? []) {
                for (const segment of track.trkseg ?? []) {
                    for (const point of segment.trkpt ?? []) {
                        const timestamp = validDate(point.time);
                        if (!timestamp) {
                            continue;
                        }
                        if (!start || timestamp < start) {
                            start = timestamp;
                        }
                        if (!end || timestamp > end) {
                            end = timestamp;
                        }
                    }
                }
            }
            return { start, end };
        } catch (e) {
            console.warn("Unable to parse picker trail timestamps", e);
            return {};
        }
    }

    function validDate(value?: Date) {
        if (!value || Number.isNaN(value.getTime())) {
            return undefined;
        }
        return value;
    }

    function clampDateSliderValue(value: number) {
        return clamp(Math.round(value), dateRange.min, dateRange.max);
    }

    function startOfLocalDay(date: Date) {
        return new Date(date.getFullYear(), date.getMonth(), date.getDate());
    }

    function endOfLocalDay(date: Date) {
        return new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1, 0, 0, 0, -1);
    }

    function clamp(value: number, min: number, max: number) {
        return Math.max(min, Math.min(max, value));
    }

    function formatDateRangeLabel(start: number, end: number, min: number, max: number) {
        const hasStart = start > min;
        const hasEnd = end < max;
        if (hasStart && hasEnd) {
            return `${formatDateValue(start)} - ${formatDateValue(end)}`;
        }
        if (hasStart) {
            return $_("date-range-from", { values: { date: formatDateValue(start) } });
        }
        if (hasEnd) {
            return $_("date-range-until", { values: { date: formatDateValue(end) } });
        }
        return "";
    }

    function formatDateValue(value: number) {
        return new Date(value).toLocaleDateString();
    }

    async function confirmSelection(closeModal: () => void) {
        if (!selectedCandidates.length || saving) {
            return;
        }
        saving = true;
        try {
            await onselect?.(selectedCandidates);
            closeModal();
        } catch (e: any) {
            console.error(e);
            error = e?.message ?? "Unable to import photos";
        } finally {
            saving = false;
        }
    }

    function ensureMap() {
        if (map || !mapContainer || !mapContainer.isConnected) {
            return;
        }
        map = new M.Map({
            container: mapContainer,
            style: baseMapStyles.OpenFreeMap,
            center: hasCoordinate(lon) && hasCoordinate(lat) ? [lon, lat] : [8.3, 46.8],
            zoom: hasCoordinate(lon) && hasCoordinate(lat) ? 12 : 5,
            attributionControl: false,
        });
        map.addControl(new M.NavigationControl({ showCompass: false }), "top-right");
        map.on("load", () => {
            mapReady = true;
            syncMap();
        });
        map.on("zoomend", () => syncMapViewport());
        map.on("moveend", () => syncMapViewport());
        map.on("click", () => clearActiveCluster());
    }

    function destroyMap() {
        closePopup();
        for (const marker of markers) {
            marker.remove();
        }
        markers = [];
        map?.remove();
        map = undefined;
        mapReady = false;
        mapZoom = 0;
        mapBounds = undefined;
    }

    function syncMap() {
        if (!map || !mapReady) {
            return;
        }
        clearMapLayers();
        closePopup();

        const trailGeoJSON = resolveTrailGeoJSON();
        if (trailGeoJSON?.features?.length) {
            addTrailLayer(trailGeoJSON);
        }

        fitMapToBounds(contentBounds(trailGeoJSON));
        updateMapViewport();
    }

    function syncPhotoMarkers(currentClusters: PhotoCluster[]) {
        if (!map || !mapReady) {
            return;
        }
        clearPhotoMarkers();
        closePopup();
        for (const cluster of currentClusters) {
            const element = createMarkerElement(cluster);
            const marker = new M.Marker({ element, anchor: "center" })
                .setLngLat([cluster.lon, cluster.lat])
                .addTo(map);
            markers.push(marker);
        }
        syncMarkerStyles();
    }

    function syncMapViewport() {
        updateMapViewport();
    }

    function updateMapViewport() {
        if (!map || !mapReady) {
            mapZoom = 0;
            mapBounds = undefined;
            return;
        }
        mapZoom = map.getZoom();
        mapBounds = map.getBounds();
    }

    function fitMapToContent() {
        if (!map || !mapReady) {
            return;
        }
        fitMapToBounds(contentBounds());
        syncMapViewport();
    }

    function contentBounds(trailGeoJSON = resolveTrailGeoJSON()) {
        const bounds = new M.LngLatBounds();
        let hasBounds = false;
        if (trailGeoJSON?.features?.length) {
            hasBounds = extendBoundsWithGeoJSON(bounds, trailGeoJSON) || hasBounds;
        }
        hasBounds = extendBoundsWithCandidates(bounds, providerCandidates) || hasBounds;
        return hasBounds ? bounds : undefined;
    }

    function fitMapToBounds(bounds?: M.LngLatBounds) {
        if (!map || !bounds) {
            return;
        }
        map.fitBounds(bounds, {
            padding: 40,
            maxZoom: 15,
            animate: false,
        });
    }

    function mapSyncKey() {
        return [
            trailData ?? "",
            trailPolyline ?? "",
            providerCandidates
                .map((candidate) => {
                    const position = markerPosition(candidate);
                    return `${candidateKey(candidate)}:${position?.lat ?? ""}:${position?.lon ?? ""}`;
                })
                .join("|"),
        ].join("::");
    }

    function clearMapLayers() {
        if (!map) {
            return;
        }
        if (map.getLayer("photo-picker-trail")) {
            map.removeLayer("photo-picker-trail");
        }
        if (map.getSource("photo-picker-trail")) {
            map.removeSource("photo-picker-trail");
        }
    }

    function clearPhotoMarkers() {
        for (const marker of markers) {
            marker.remove();
        }
        markers = [];
    }

    function addTrailLayer(geojson: FeatureCollection) {
        if (!map) {
            return;
        }
        map.addSource("photo-picker-trail", {
            type: "geojson",
            data: geojson,
        });
        map.addLayer({
            id: "photo-picker-trail",
            type: "line",
            source: "photo-picker-trail",
            paint: {
                "line-color": TRAIL_LINE_COLOR,
                "line-width": 5,
            },
        });
    }

    function resolveTrailGeoJSON(): FeatureCollection | undefined {
        if (trailData) {
            try {
                return GPX.parse(trailData).toGeoJSON();
            } catch (e) {
                console.warn("Unable to parse picker trail data", e);
            }
        }
        if (trailPolyline) {
            return polylineToGeoJSON(trailPolyline) as FeatureCollection;
        }
    }

    function createMarkerElement(cluster: PhotoCluster) {
        const element = document.createElement("button");
        element.type = "button";
        element.className = "photo-library-marker";
        element.dataset.cluster = cluster.key;
        element.textContent = String(cluster.candidates.length);
        element.addEventListener("mouseenter", () => {
            highlightedClusterKey = cluster.key;
            openPopup(cluster);
        });
        element.addEventListener("mouseleave", () => {
            highlightedClusterKey = undefined;
            closePopup();
        });
        element.addEventListener("click", (event) => {
            event.stopPropagation();
            activeClusterKey = activeClusterKey === cluster.key ? undefined : cluster.key;
            previewIndex = 0;
        });
        return element;
    }

    function syncMarkerStyles() {
        const markerElements = markers.map(markerElement);
        syncMarkerHighlightClass(markerElements, markerElementForCluster(activeClusterKey), "is-active");
        syncMarkerHighlightClass(markerElements, markerElementForCluster(highlightedClusterKey), "is-highlighted");
    }

    function markerElementForCluster(clusterKey?: string) {
        if (!clusterKey) {
            return undefined;
        }
        return markers
            .map(markerElement)
            .find((element) => element?.dataset.cluster === clusterKey);
    }

    function openPopup(cluster: PhotoCluster) {
        if (!map) {
            return;
        }
        closePopup();
        const photos = cluster.candidates.slice(0, 4);
        const overflowCount = Math.max(0, cluster.candidates.length - photos.length);
        popup = new M.Popup({
            closeButton: false,
            closeOnClick: false,
            offset: 18,
            className: "photo-library-popup",
        })
            .setLngLat([cluster.lon, cluster.lat])
            .setHTML(`
                <div class="photo-library-popup-grid ${photos.length === 1 ? "is-single" : ""}">
                    ${photos.map((candidate, index) => `
                        <div class="photo-library-popup-thumb">
                            <img src="${escapeAttribute(thumbnailUrl(candidate))}" alt="">
                            ${overflowCount > 0 && index === photos.length - 1 ? `<span class="photo-library-popup-more">+${overflowCount}</span>` : ""}
                        </div>
                    `).join("")}
                </div>
            `)
            .addTo(map);
    }

    function closePopup() {
        popup?.remove();
        popup = undefined;
    }

    function escapeAttribute(value: string) {
        return value.replaceAll("&", "&amp;").replaceAll("\"", "&quot;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
    }

    function extendBoundsWithGeoJSON(bounds: M.LngLatBounds, geojson: FeatureCollection) {
        let hasBounds = false;
        for (const feature of geojson.features) {
            hasBounds = extendBoundsWithGeometry(bounds, feature.geometry) || hasBounds;
        }
        return hasBounds;
    }

    function extendBoundsWithGeometry(bounds: M.LngLatBounds, geometry: Geometry | null): boolean {
        if (!geometry) {
            return false;
        }
        let hasBounds = false;
        if (geometry.type === "GeometryCollection") {
            for (const child of geometry.geometries) {
                hasBounds = extendBoundsWithGeometry(bounds, child) || hasBounds;
            }
            return hasBounds;
        }
        walkCoordinates(geometry.coordinates as Position | Position[] | Position[][] | Position[][][], (position) => {
            const [lng, lat] = position;
            if (Number.isFinite(lng) && Number.isFinite(lat)) {
                bounds.extend([lng, lat]);
                hasBounds = true;
            }
        });
        return hasBounds;
    }

    function walkCoordinates(
        coordinates: Position | Position[] | Position[][] | Position[][][],
        visit: (position: Position) => void,
    ) {
        if (!Array.isArray(coordinates)) {
            return;
        }
        if (typeof coordinates[0] === "number") {
            visit(coordinates as Position);
            return;
        }
        for (const child of coordinates as Position[] | Position[][] | Position[][][]) {
            walkCoordinates(child, visit);
        }
    }
</script>

<Modal {id} size="w-[min(96vw,76rem)] max-w-[96vw]" title={title ?? $_("photo-library")} bind:this={modal}>
    {#snippet content()}
        <div class="h-[72vh] min-h-[34rem] max-h-[46rem] w-full overflow-x-auto">
            {#if loading}
                <div class="flex min-h-80 items-center justify-center text-sm text-gray-500">
                    {$_("loading")}...
                </div>
            {:else if fatalError}
                <div class="flex min-h-80 items-center justify-center text-sm text-red-500">
                    {error}
                </div>
            {:else}
                <div class="grid h-full min-w-[42rem] grid-cols-[minmax(16rem,22rem)_minmax(22rem,1fr)] gap-4">
                    <section class="flex min-h-0 flex-col gap-3">
                        {#if error}
                            <div class="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-500">
                                {error}
                            </div>
                        {/if}
                        {#if hasAssetPlugin && providerOptions.length > 1}
                            <div class="flex flex-wrap gap-2">
                                <button
                                    type="button"
                                    class="rounded border border-input-border px-3 py-1.5 text-sm {providerFilter === 'all' ? 'bg-primary text-white' : 'bg-background hover:bg-menu-item-background-hover'}"
                                    onclick={() => (providerFilter = "all")}
                                >
                                    {$_("all-photos")}
                                </button>
                                {#each providerOptions as provider (provider.id)}
                                    <button
                                        type="button"
                                        class="flex items-center gap-2 rounded border border-input-border px-3 py-1.5 text-sm {providerFilter === provider.id ? 'bg-primary text-white' : 'bg-background hover:bg-menu-item-background-hover'}"
                                        onclick={() => (providerFilter = provider.id)}
                                    >
                                        {#if provider.icon}
                                            <img
                                                src={provider.icon}
                                                alt=""
                                                class="rounded-sm {provider.id === WANDERER_PROVIDER_ID ? 'h-3 w-3' : 'h-4 w-4'}"
                                            />
                                        {:else if provider.id === "wanderer"}
                                            <i class="fa fa-route text-xs"></i>
                                        {/if}
                                        <span>{provider.label}</span>
                                    </button>
                                {/each}
                            </div>
                        {/if}
                        <div>
                            <div class="flex items-center justify-between gap-3">
                                <p class="text-sm font-medium">{$_("photo-time-scope")}</p>
                                {#if dateRangeLabel}
                                    <span class="text-sm text-gray-500">{dateRangeLabel}</span>
                                {/if}
                            </div>
                            <div class="px-4">
                                {#key dateRange.key}
                                    <DoubleSlider
                                        minValue={dateRange.min}
                                        maxValue={dateRange.max}
                                        step={DAY_MS}
                                        bind:currentMin={dateRange.start}
                                        bind:currentMax={dateRange.end}
                                        onset={handleDateRangeChange}
                                    ></DoubleSlider>
                                {/key}
                            </div>
                        </div>
                        <div class="flex items-center justify-between gap-3 text-sm text-gray-500">
                            <span>{visibleCandidates.length} / {providerCandidates.length}</span>
                            <div class="flex items-center gap-2">
                                {#if activeClusterKey || mapFiltered}
                                    <button type="button" class="btn-link text-sm" onclick={resetPhotoScope}>
                                        {$_("reset")}
                                    </button>
                                {/if}
                                <button type="button" class="btn-link text-sm" onclick={selectVisible} disabled={!visibleCandidates.length}>
                                    {$_("select-visible")}
                                </button>
                            </div>
                        </div>
                        {#if visibleCandidates.length}
                            <div class="min-h-0 flex-1 overflow-y-auto rounded-lg border border-input-border">
                                {#each visibleCandidates as candidate (candidateKey(candidate))}
                                    {@const currentProviderId = displayProviderId(candidate)}
                                    {@const currentProviderIcon = providerIcon(currentProviderId)}
                                    <div
                                        role="listitem"
                                        class="flex w-full items-center gap-3 border-b border-separator px-3 py-2 text-left last:border-b-0 hover:bg-menu-item-background-hover {previewCandidate && candidateKey(previewCandidate) === candidateKey(candidate) ? 'bg-menu-item-background-hover' : ''}"
                                        onmouseenter={() => (highlightedClusterKey = clusterKeyForCandidate(candidate))}
                                        onmouseleave={() => (highlightedClusterKey = undefined)}
                                    >
                                        <input
                                            type="checkbox"
                                            class="h-4 w-4 shrink-0 rounded border-input-border bg-input-background accent-primary"
                                            checked={isSelected(candidate)}
                                            onchange={() => toggleCandidate(candidate)}
                                            aria-label={candidate.originalFileName}
                                        />
                                        <button
                                            type="button"
                                            class="flex min-w-0 flex-1 items-center gap-3 text-left"
                                            onclick={() => setPreviewCandidate(candidate)}
                                        >
                                            <img src={thumbnailUrl(candidate)} alt="" class="h-12 w-12 shrink-0 rounded object-cover" loading="lazy" />
                                            <div class="min-w-0 flex-1">
                                                <div class="truncate text-sm font-medium">{candidate.originalFileName}</div>
                                                <div class="flex flex-wrap items-center gap-x-2 text-xs text-gray-500">
                                                    <span
                                                        class="provider-icon {currentProviderId === WANDERER_PROVIDER_ID ? 'is-wanderer' : ''}"
                                                        title={providerLabel(currentProviderId)}
                                                    >
                                                        {#if currentProviderIcon}
                                                            <img
                                                                src={currentProviderIcon}
                                                                alt={providerLabel(currentProviderId)}
                                                                class={currentProviderId === WANDERER_PROVIDER_ID
                                                                    ? "h-3.5 w-3.5 rounded-sm object-contain"
                                                                    : "h-4 w-4 rounded-sm object-contain"}
                                                            />
                                                        {:else if currentProviderId === "wanderer"}
                                                            <i class="fa fa-route"></i>
                                                        {:else}
                                                            <i class="fa fa-images"></i>
                                                        {/if}
                                                    </span>
                                                    {#if candidate.takenAt}
                                                        <span>{new Date(candidate.takenAt).toLocaleDateString()}</span>
                                                    {/if}
                                                    {#if candidate.distance}
                                                        <span>{Math.round(candidate.distance)} m</span>
                                                    {/if}
                                                </div>
                                            </div>
                                        </button>
                                    </div>
                                {/each}
                                {#if showLoadMore}
                                    <div class="flex flex-wrap justify-center gap-3 border-t border-separator p-3 text-center">
                                        {#if showWandererLoadMore}
                                            <button type="button" class="btn-link text-sm" onclick={loadMoreWandererCandidates} disabled={loadingMore}>
                                                {providerLabel(WANDERER_PROVIDER_ID)}: {loadingMore ? $_("loading") : $_("more")}
                                            </button>
                                        {/if}
                                        {#each pluginLoadMoreIds as pluginId}
                                            <button
                                                type="button"
                                                class="btn-link text-sm"
                                                onclick={() => loadMorePluginCandidates(pluginId)}
                                                disabled={pluginPagination[pluginId]?.loading}
                                            >
                                                {providerLabel(pluginId)}: {pluginPagination[pluginId]?.loading ? $_("loading") : $_("more")}
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        {:else if showLoadMore}
                            <div class="flex min-h-0 flex-1 flex-wrap items-center justify-center gap-3 rounded-lg border border-dashed border-input-border text-sm text-gray-500">
                                {#if showWandererLoadMore}
                                    <button type="button" class="btn-link text-sm" onclick={loadMoreWandererCandidates} disabled={loadingMore}>
                                        {providerLabel(WANDERER_PROVIDER_ID)}: {loadingMore ? $_("loading") : $_("more")}
                                    </button>
                                {/if}
                                {#each pluginLoadMoreIds as pluginId}
                                    <button
                                        type="button"
                                        class="btn-link text-sm"
                                        onclick={() => loadMorePluginCandidates(pluginId)}
                                        disabled={pluginPagination[pluginId]?.loading}
                                    >
                                        {providerLabel(pluginId)}: {pluginPagination[pluginId]?.loading ? $_("loading") : $_("more")}
                                    </button>
                                {/each}
                            </div>
                        {:else}
                            <div class="flex min-h-0 flex-1 items-center justify-center rounded-lg border border-dashed border-input-border text-sm text-gray-500">
                                {$_("no-results")}
                            </div>
                        {/if}
                    </section>

                    <section class="grid min-h-0 grid-rows-[minmax(14rem,0.9fr)_minmax(18rem,1.4fr)] gap-4">
                        <div bind:this={mapContainer} class="min-h-0 overflow-hidden rounded-lg border border-input-border"></div>
                        <div class="min-h-0 overflow-hidden rounded-lg border border-input-border">
                            {#if previewCandidate}
                                <div class="relative h-full w-full bg-black">
                                    <img src={thumbnailUrl(previewCandidate)} alt="" class="block h-full w-full object-contain" />
                                    {#if visibleCandidates.length > 1}
                                        <button
                                            type="button"
                                            class="absolute left-3 top-1/2 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white"
                                            onclick={() => nextPreview(-1)}
                                            aria-label="Previous photo"
                                        >
                                            <i class="fa fa-chevron-left"></i>
                                        </button>
                                        <button
                                            type="button"
                                            class="absolute right-3 top-1/2 -translate-y-1/2 rounded-full bg-black/60 p-2 text-white"
                                            onclick={() => nextPreview(1)}
                                            aria-label="Next photo"
                                        >
                                            <i class="fa fa-chevron-right"></i>
                                        </button>
                                    {/if}
                                    <div class="absolute bottom-0 left-0 right-0 flex items-center justify-between gap-3 bg-black/65 px-3 py-2 text-xs text-white">
                                        <span class="truncate">{previewCandidate.originalFileName}</span>
                                        <span class="shrink-0">{previewIndex + 1} / {visibleCandidates.length}</span>
                                    </div>
                                </div>
                            {:else}
                                <div class="flex h-full items-center justify-center text-sm text-gray-500">
                                    {$_("no-results")}
                                </div>
                            {/if}
                        </div>
                    </section>
                </div>
            {/if}
        </div>
    {/snippet}
    {#snippet footer({ closeModal })}
        <div class="flex flex-wrap items-center justify-between gap-3">
            <span class="text-sm text-gray-500">{selectedCandidates.length} {$_("selected")}</span>
            <div class="flex items-center gap-3">
                <button class="btn-secondary" type="button" onclick={closeModal}>{$_("cancel")}</button>
                <button
                    class="btn-primary"
                    type="button"
                    disabled={!selectedCandidates.length || saving}
                    onclick={() => confirmSelection(closeModal)}
                >
                    {saving ? $_("loading") : (actionLabel ?? $_("add-photos"))}
                </button>
            </div>
        </div>
    {/snippet}
</Modal>

<style lang="postcss">
    @reference "tailwindcss";
    @reference "../../../css/app.css";

    :global(.photo-library-marker) {
        @apply flex h-7 min-w-7 items-center justify-center rounded-full border-2 border-white bg-primary px-2 text-xs font-semibold text-white shadow-lg transition-colors;
    }

    :global(.photo-library-marker.is-highlighted),
    :global(.photo-library-marker.is-active) {
        border-color: rgb(255 255 255);
        box-shadow:
            0 0 0 4px rgba(var(--primary), 0.35),
            0 0 0 8px rgba(var(--primary), 0.16);
        z-index: 1;
    }

    :global(.photo-library-popup .maplibregl-popup-content) {
        @apply rounded-lg bg-background p-2 shadow-lg;
    }

    :global(.photo-library-popup-grid) {
        @apply grid grid-cols-2 gap-1;
    }

    :global(.photo-library-popup-grid.is-single) {
        @apply grid-cols-1;
    }

    :global(.photo-library-popup-thumb) {
        @apply relative h-14 w-14 overflow-hidden rounded;
    }

    :global(.photo-library-popup-thumb img) {
        @apply h-full w-full object-cover;
    }

    :global(.photo-library-popup-more) {
        @apply absolute inset-0 flex items-center justify-center text-xs font-semibold text-white;
        background: rgba(0, 0, 0, 0.65);
    }

    .provider-icon {
        @apply inline-flex h-6 w-6 shrink-0 items-center justify-center overflow-hidden rounded border border-input-border bg-background text-[0.7rem] text-gray-500;
    }

    .provider-icon img {
        @apply h-4 w-4 rounded-sm object-contain;
    }

    .provider-icon.is-wanderer img {
        @apply h-3.5 w-3.5;
    }
</style>
