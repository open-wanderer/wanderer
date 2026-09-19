import GPX from "$lib/models/gpx/gpx";
import Track from "$lib/models/gpx/track";
import TrackSegment from "$lib/models/gpx/track-segment";
import { haversineDistance } from "$lib/models/gpx/utils";
import Waypoint from "$lib/models/gpx/waypoint";
import {
    type RoutingEffectiveControls,
    type RoutingEngine,
    type RoutingNativeControls,
    type RoutingOptions,
    type RoutingAnchor,
    type RoutingElevationResponse,
    type RoutingProfile,
    type RoutingProfileMapping,
    type RoutingRouteResponse,
    type RoutingRouteResult,
    type RoutingSettings,
    type RoutingCandidate,
    type RoutingSegment,
    type RoutingSegmentProvenance,
    type RoutingEngineError,
    type RoutingManeuverResponse,
} from "$lib/models/routing";
import { APIError } from "$lib/util/api_util";
import { cloneRoutingControlValues } from "$lib/util/routing_control_util";
import { decodePolyline, encodePolyline } from "$lib/util/polyline_util";
import { renderRoutingAnchorMarker, routingAnchorTitle } from "$lib/util/routing_anchor_util";
import {
    isPersistedClosedLoop,
    roundTripLoopProvenanceForSegments,
    roundTripPreferencesForEngine,
    roundTripProfileForEngine,
} from "$lib/util/routing_round_trip_util";
import { applyChangeset, diff, revertChangeset, type Changeset } from 'json-diff-ts';
import type { LngLat } from "maplibre-gl";
import { _ } from "svelte-i18n";
import { get } from "svelte/store";

const emtpyTrack = new Track({ trkseg: [] })

class RoutingStore {
    route: GPX = $state(new GPX({ trk: [emtpyTrack] }));
    anchors: RoutingAnchor[] = $state([]);
    closedLoop = $state(false);
    segmentProvenance: (RoutingSegmentProvenance | null)[] = $state([]);
    routeCandidates: RoutingCandidate[] = $state([]);
    selectedRouteCandidateId: string | null | undefined = $state();
    routeCandidateOrigin: {
        route: GPX;
    } | undefined = $state();
    routeCandidateErrors: RoutingEngineError[] = $state([]);
    routeCandidateWarnings: string[] = $state([]);
    routeCandidatesLoading = $state(false);
    routeCandidatesStale = $state(false);
    routeCandidateRequestedCount = $state(0);
    undoStack: { delta: Changeset, reverseDelta: Changeset, anchorsBefore?: RoutingAnchor[], anchorsAfter?: RoutingAnchor[], provenanceBefore?: (RoutingSegmentProvenance | null)[], provenanceAfter?: (RoutingSegmentProvenance | null)[] }[] = $state([]);
    redoStack: { delta: Changeset, reverseDelta: Changeset, anchorsBefore?: RoutingAnchor[], anchorsAfter?: RoutingAnchor[], provenanceBefore?: (RoutingSegmentProvenance | null)[], provenanceAfter?: (RoutingSegmentProvenance | null)[] }[] = $state([]);
}

export const routingStore = new RoutingStore();

async function routingApi<T>(
    path: string,
    init?: RequestInit,
    request: typeof fetch = fetch,
): Promise<T> {
    const r = await request(path, init);

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response)
    }

    return await r.json() as T;
}

export async function routingEngines(): Promise<RoutingEngine[]> {
    const response = await routingApi<{ engines: RoutingEngine[] }>("/api/v1/routing/engines");
    return response.engines;
}

export function routingManeuvers(
    trailId: string,
    options: { language?: string; share?: string } = {},
): Promise<RoutingManeuverResponse> {
    return routingApi<RoutingManeuverResponse>("/api/v1/routing/maneuvers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ trailId, ...options }),
    });
}

export function checkRoutingPlugin(pluginId: string, config?: Record<string, unknown>): Promise<{ ok: boolean }> {
    return routingApi<{ ok: boolean }>("/api/v1/routing/check", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pluginId, config }),
    });
}

export function routingSettings(request: typeof fetch = fetch): Promise<RoutingSettings> {
    return routingApi<RoutingSettings>("/api/v1/routing/settings", undefined, request);
}

export function updateRoutingSettings(settings: Partial<RoutingSettings>): Promise<RoutingSettings> {
    return routingApi<RoutingSettings>("/api/v1/routing/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(settings),
    });
}

export function adminRoutingSettings(): Promise<RoutingSettings> {
    return routingApi<RoutingSettings>("/api/v1/routing/admin/settings");
}

export function updateAdminRoutingSettings(settings: Partial<RoutingSettings>): Promise<RoutingSettings> {
    return routingApi<RoutingSettings>("/api/v1/routing/admin/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(settings),
    });
}

export async function routingProfileMappings(): Promise<RoutingProfileMapping[]> {
    const response = await routingApi<{ items: RoutingProfileMapping[] }>("/api/v1/routing/mappings");
    return response.items;
}

export function createRoutingProfileMapping(mapping: Partial<RoutingProfileMapping>): Promise<RoutingProfileMapping> {
    return routingApi<RoutingProfileMapping>("/api/v1/routing/mappings", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(mapping),
    });
}

export function updateRoutingProfileMapping(mapping: Partial<RoutingProfileMapping> & { id: string }): Promise<RoutingProfileMapping> {
    return routingApi<RoutingProfileMapping>(`/api/v1/routing/mappings/${mapping.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(mapping),
    });
}

export async function routingProfiles(): Promise<RoutingProfile[]> {
    const response = await routingApi<{ profiles: RoutingProfile[] }>("/api/v1/routing/profiles");
    return response.profiles;
}

export function createRoutingProfile(profile: Partial<RoutingProfile>): Promise<RoutingProfile> {
    return routingApi<RoutingProfile>("/api/v1/routing/profiles", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(profile),
    });
}

export function updateRoutingProfile(profile: Partial<RoutingProfile> & { id: string }): Promise<RoutingProfile> {
    return routingApi<RoutingProfile>(`/api/v1/routing/profiles/${profile.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(profile),
    });
}

export async function deleteRoutingProfile(id: string): Promise<void> {
    await routingApi(`/api/v1/routing/profiles/${id}`, {
        method: "DELETE",
    });
}

export function routingEffectiveControls(input: {
    category: string
    subcategory?: string
    routing?: {
        mode?: string
        engines?: { pluginId: string; instanceId?: string }[]
    }
}): Promise<RoutingEffectiveControls> {
    return routingApi<RoutingEffectiveControls>("/api/v1/routing/effective-controls", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
    });
}

export function routingNativeControls(input: {
    pluginId: string
    instanceId?: string
    profileId?: string
    nativeProfileKey?: string
    nativeConfig?: Record<string, unknown>
}): Promise<RoutingNativeControls> {
    return routingApi<RoutingNativeControls>("/api/v1/routing/native-controls", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
    });
}

export function clearRoute() {
    routingStore.route = new GPX({ trk: [emtpyTrack] });
    routingStore.segmentProvenance = [];
    routingStore.closedLoop = false;
    clearRoutingCandidates();
}

export function clearAnchors() {
    for (const anchor of routingStore.anchors) {
        anchor.marker?.remove();
    }
    routingStore.anchors = [];
}

export function clearUndoRedoStack() {
    routingStore.undoStack = []
    routingStore.redoStack = []
}

function pushToUndoStack(delta: Changeset, reverseDelta: Changeset, provenanceBefore = routingStore.segmentProvenance, provenanceAfter = routingStore.segmentProvenance) {
    routingStore.undoStack.push({
        delta,
        reverseDelta,
        provenanceBefore: cloneProvenance(provenanceBefore),
        provenanceAfter: cloneProvenance(provenanceAfter),
    })
    routingStore.redoStack = []
}


export function setRoute(
    newRoute: GPX,
    undoable: boolean = false,
    provenance: (RoutingSegmentProvenance | null)[] = [],
    preserveRoutingCandidates = false,
) {
    const provenanceBefore = cloneProvenance(routingStore.segmentProvenance);
    const delta = diff(routingStore.route, newRoute);
    const reverseDelta = diff(newRoute, routingStore.route);
    routingStore.route = newRoute;
    const segmentCount = newRoute.trk?.at(0)?.trkseg?.length ?? 0;
    routingStore.segmentProvenance = cloneProvenance(
        Array.from(
            { length: segmentCount },
            (_, index) => provenance[index] ?? null,
        ),
    );
    routingStore.closedLoop = isPersistedClosedLoop(
        routingStore.segmentProvenance,
        segmentCount,
    );
    if (!preserveRoutingCandidates) {
        clearRoutingCandidates();
    }
    if (undoable) {
        pushToUndoStack(delta, reverseDelta, provenanceBefore, routingStore.segmentProvenance)
    }

}

export async function calculateRouteBetween(startLat: number, startLon: number, endLat: number, endLon: number, options: RoutingOptions): Promise<RoutingRouteResult> {

    let shape;
    let duration: number;
    let snappedAnchors;
    let heights: number[] | undefined;
    let provenance: RoutingSegmentProvenance | undefined;
    if (options.autoRouting) {
        const pluginId = options.routingPluginId;
        if (!pluginId) {
            throw new APIError(400, "No routing provider configured")
        }
        const requestBody = routingRequestBody(
            [{ lat: startLat, lon: startLon }, { lat: endLat, lon: endLon }],
            options,
            false,
        );

        let r = await fetch("/api/v1/routing/route", { method: "POST", body: JSON.stringify(requestBody) })

        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response)
        }

        const routeResponse: RoutingRouteResponse = await r.json();
        routingStore.routeCandidateErrors = routeResponse.engineErrors ?? [];
        routingStore.routeCandidateWarnings = routeResponse.warnings ?? [];
        const candidate = routeResponse.candidates?.[0];
        const segment = candidate?.segments?.[0];
        if (!segment) {
            throw new APIError(502, "No route candidate returned")
        }
        shape = segment.geometry.coordinates
        duration = segment.duration
        snappedAnchors = candidate.snappedAnchors
        heights = candidate.elevation?.heights
        provenance = segment.provenance
    } else {
        shape = encodePolyline([[startLat, startLon], [endLat, endLon]])
        duration = 0;
    }

    const points = decodePolyline(shape);

    if ((!heights || heights.length !== points.length) && options.routingElevationPluginId) {
        const r2 = await fetch("/api/v1/routing/elevation", { method: "POST", body: JSON.stringify({ pluginId: options.routingElevationPluginId, instanceId: options.routingElevationInstanceId, encodedPolyline: shape }) })

        if (!r2.ok) {
            const response = await r2.json();
            throw new APIError(r2.status, response.message, response)
        }

        const heightResponse: RoutingElevationResponse = await r2.json()
        heights = heightResponse.heights
    }
    const startTime = new Date().getTime();

    const waypoints = points.map((p, i) => new Waypoint({ $: { lat: p[1], lon: p[0] }, ele: heights?.[i], time: new Date(startTime + (((duration * 1000) / points.length) * i)) }))

    return { waypoints, snappedAnchors, provenance }
}

function routingMode(modeOfTransport: RoutingOptions["modeOfTransport"]) {
    switch (modeOfTransport) {
        case "pedestrian":
            return "foot";
        case "bicycle":
            return "bike";
        case "auto":
            return "motor";
    }
}

function routingNativeConfig(options: RoutingOptions) {
    return cloneRoutingControlValues(options.nativeConfig);
}

function routingPreferences(options: RoutingOptions) {
    return cloneRoutingControlValues(options.preferences);
}

export function routingProvenanceMatchesOptions(
    provenance: RoutingSegmentProvenance,
    options: RoutingOptions,
) {
    // Topology-only records deliberately make no claim about which engine
    // produced their geometry and therefore cannot conflict with an engine.
    if (!provenance.pluginId) {
        return true;
    }
    if (
        provenance.pluginId === options.routingPluginId &&
        options.routingInstanceId &&
        provenance.instanceId &&
        provenance.instanceId !== options.routingInstanceId
    ) {
        return false;
    }
    if (
        (provenance.category ?? "") !== (options.category ?? "") ||
        (provenance.subcategory ?? "") !== (options.subcategory ?? "") ||
        (provenance.routingMode !== undefined &&
            provenance.routingMode !== (options.routingMode ?? "segment"))
    ) {
        return false;
    }
    if (
        options.nativeProfileKey &&
        provenance.profileKey &&
        provenance.profileKey !== options.nativeProfileKey
    ) {
        return false;
    }
    const currentProfileRevision = options.profileRevisions?.[provenance.pluginId];
    if (
        provenance.profileRevision &&
        currentProfileRevision &&
        provenance.profileRevision !== currentProfileRevision
    ) {
        return false;
    }
    const requestedPreferences = provenance.requestedPreferences;
    const requestedNativeConfig = provenance.requestedNativeConfig;
    return (
        (requestedPreferences === undefined ||
            routingMapContains(requestedPreferences, routingPreferences(options))) &&
        (provenance.pluginId !== options.routingPluginId ||
            requestedNativeConfig === undefined ||
            routingMapContains(requestedNativeConfig, routingNativeConfig(options)))
    );
}

function routingMapContains(
    actual: Record<string, unknown> | undefined,
    expected: Record<string, unknown> | undefined,
) {
    for (const [key, value] of Object.entries(expected ?? {})) {
        if (value === undefined) continue;
        if (JSON.stringify(actual?.[key]) !== JSON.stringify(value)) {
            return false;
        }
    }
    return true;
}

function routingRequestBody(
    anchors: { lat: number; lon: number }[],
    options: RoutingOptions,
    requestVariants: boolean,
    referenceRoute?: GPX,
) {
    const pluginId = options.routingPluginId ?? "";
    const profileKey = options.nativeProfileKey ?? "";
    return {
        pluginId,
        instanceId: options.routingInstanceId,
        engineMode: requestVariants ? (options.engineMode ?? "parallel") : "single",
        desiredVariants: requestVariants ? (options.desiredVariants ?? 3) : 1,
        requestVariants,
        referenceGeometry:
            requestVariants && referenceRoute
                ? routingReferenceGeometry(referenceRoute)
                : undefined,
        routingMode: options.routingModeExplicit
            ? (options.routingMode ?? "segment")
            : undefined,
        anchors,
        mode: routingMode(options.modeOfTransport),
        profile: {
            pluginId,
            key: profileKey,
            kind: profileKey ? "builtin" : undefined,
            nativeConfig: routingNativeConfig(options),
        },
        preferences: routingPreferences(options),
        category: options.category,
        subcategory: options.subcategory,
        options: {
            alternatives: 1,
            includeElevation: requestVariants,
        },
    };
}

function routingReferenceGeometry(route: GPX) {
    const coordinates = route
        .flatten()
        .map((point) => [point.$.lat, point.$.lon])
        .filter(
            (point): point is [number, number] =>
                Number.isFinite(point[0]) && Number.isFinite(point[1]),
        );
    if (coordinates.length < 2) return undefined;
    return {
        format: "encoded_polyline",
        precision: 6,
        coordinates: encodePolyline(coordinates),
    };
}

export async function calculateRouteForAnchors(
    anchors: RoutingAnchor[],
    options: RoutingOptions,
    requestVariants = false,
    referenceRoute?: GPX,
): Promise<RoutingRouteResponse> {
    if (anchors.length < 2) {
        throw new APIError(400, "At least two routing anchors are required");
    }
    const response = await routingApi<RoutingRouteResponse>("/api/v1/routing/route", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(
            routingRequestBody(
                anchors.map(({ lat, lon }) => ({ lat, lon })),
                options,
                requestVariants,
                referenceRoute,
            ),
        ),
    });
    routingStore.routeCandidateErrors = response.engineErrors ?? [];
    routingStore.routeCandidateWarnings = response.warnings ?? [];
    return response;
}

export async function calculateRoundTrip(
    start: RoutingAnchor,
    targetDistance: number,
    options: RoutingOptions,
    direction?: number,
    seed?: string,
    providerChanged = false,
): Promise<RoutingRouteResponse> {
    const base = routingRequestBody(
        [{ lat: start.lat, lon: start.lon }],
        options,
        false,
    );
    const response = await routingApi<RoutingRouteResponse>("/api/v1/routing/round-trip", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            pluginId: base.pluginId,
            instanceId: options.routingInstanceId,
            start: { lat: start.lat, lon: start.lon },
            targetDistance,
            direction,
            seed,
            mode: base.mode,
            profile: roundTripProfileForEngine(base.profile, providerChanged),
            preferences: roundTripPreferencesForEngine(
                base.preferences,
                providerChanged,
            ),
            category: base.category,
            subcategory: base.subcategory,
            options: { includeElevation: true },
        }),
    });
    routingStore.routeCandidateErrors = response.engineErrors ?? [];
    routingStore.routeCandidateWarnings = response.warnings ?? [];
    return response;
}

export function prepareRoutingProfile(
    options: RoutingOptions,
    requestVariants = false,
): Promise<{
    prepared: number;
    engineErrors?: RoutingEngineError[];
}> {
    return routingApi("/api/v1/routing/profile-prepare", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(routingRequestBody([], options, requestVariants)),
    });
}

export async function calculateRouteVariants(
    anchors: RoutingAnchor[],
    options: RoutingOptions,
): Promise<RoutingRouteResponse> {
    if (!routingStore.routeCandidateOrigin || routingStore.routeCandidatesStale) {
        routingStore.routeCandidateOrigin = {
            route: new GPX({ ...routingStore.route }),
        };
        routingStore.selectedRouteCandidateId = null;
    }
    routingStore.routeCandidatesLoading = true;
    routingStore.routeCandidateRequestedCount = options.desiredVariants ?? 1;
    routingStore.routeCandidateErrors = [];
    routingStore.routeCandidateWarnings = [];
    try {
        const response = await calculateRouteForAnchors(
            anchors,
            options,
            true,
            routingStore.routeCandidateOrigin.route,
        );
        routingStore.routeCandidates = response.candidates ?? [];
        if (
            typeof routingStore.selectedRouteCandidateId === "string" &&
            !routingStore.routeCandidates.some(
                (candidate) => candidate.id === routingStore.selectedRouteCandidateId,
            )
        ) {
            routingStore.selectedRouteCandidateId = undefined;
        }
        routingStore.routeCandidateErrors = response.engineErrors ?? [];
        routingStore.routeCandidateWarnings = response.warnings ?? [];
        routingStore.routeCandidatesStale = false;
        return response;
    } finally {
        routingStore.routeCandidatesLoading = false;
    }
}

export function clearRoutingCandidates() {
    routingStore.routeCandidates = [];
    routingStore.selectedRouteCandidateId = undefined;
    routingStore.routeCandidateOrigin = undefined;
    routingStore.routeCandidateErrors = [];
    routingStore.routeCandidateWarnings = [];
    routingStore.routeCandidatesStale = false;
    routingStore.routeCandidateRequestedCount = 0;
}

export function markRoutingCandidatesStale() {
    if (!routingStore.routeCandidateOrigin && !routingStore.routeCandidates.length) {
        return;
    }
    routingStore.routeCandidatesStale = true;
    routingStore.selectedRouteCandidateId = undefined;
}

export function applyRoutingCandidate(
    candidate: RoutingCandidate,
    undoable = true,
    preserveRoutingCandidates = true,
) {
    const route = routingCandidateToGPX(candidate);
    routingStore.selectedRouteCandidateId = candidate.id;
    setRoute(
        route,
        undoable,
        candidate.segments.map((segment) => segment.provenance ?? null),
        preserveRoutingCandidates,
    );
    routingStore.closedLoop = candidate.compositionMode === "round_trip";
    return candidate.snappedAnchors;
}

export function routingCandidateToGPX(candidate: RoutingCandidate) {
    const heightSlices = routingCandidateHeightSlices(candidate);
    let segmentStart = new Date().getTime();
    const segments = candidate.segments.map((segment, index) => {
        const waypoints = routingSegmentWaypoints(segment, heightSlices[index], segmentStart);
        segmentStart += segment.duration * 1000;
        return new TrackSegment({ trkpt: waypoints });
    });
    return new GPX({ trk: [new Track({ trkseg: segments })] });
}

function routingSegmentWaypoints(segment: RoutingSegment, heights: number[] | undefined, startTime: number) {
    const points = decodePolyline(segment.geometry.coordinates);
    return points.map(
        (point, index) =>
            new Waypoint({
                $: { lat: point[1], lon: point[0] },
                ele: heights?.[index],
                time: new Date(
                    startTime + ((segment.duration * 1000) / Math.max(points.length - 1, 1)) * index,
                ),
            }),
    );
}

function routingCandidateHeightSlices(candidate: RoutingCandidate): (number[] | undefined)[] {
    const counts = candidate.segments.map(
        (segment) => decodePolyline(segment.geometry.coordinates).length,
    );
    const expected = counts.reduce((total, count, index) => total + count - (index > 0 ? 1 : 0), 0);
    const heights = candidate.elevation?.heights;
    if (!heights || heights.length !== expected) {
        return counts.map(() => undefined);
    }
    let cursor = 0;
    return counts.map((count, index) => {
        const start = index === 0 ? cursor : cursor - 1;
        const slice = heights.slice(start, start + count);
        cursor = start + count;
        return slice;
    });
}

function cloneProvenance(values: (RoutingSegmentProvenance | null)[]) {
    return $state.snapshot(values);
}

function applyRouteChangeset(changeset: Changeset) {
    return applyChangeset(new GPX({ ...routingStore.route }), changeset);
}

export async function insertIntoRoute(waypoints: Waypoint[], index?: number, provenance?: RoutingSegmentProvenance) {
    const provenanceBefore = cloneProvenance(routingStore.segmentProvenance);
    const snapshot = new GPX({ ...routingStore.route })
    const segment = new TrackSegment({ trkpt: waypoints })
    const insertionIndex = index ?? snapshot.trk?.at(0)?.trkseg?.length ?? 0;

    if (index !== undefined) {
        snapshot.trk?.at(0)?.trkseg?.splice(index, 0, segment);
    } else {
        snapshot.trk?.at(0)?.trkseg?.push(segment);
    }
    snapshot.features = snapshot.getTotals();

    const delta = diff(routingStore.route, snapshot);
    const reverseDelta = diff(snapshot, routingStore.route);
    routingStore.route = snapshot;
    routingStore.segmentProvenance.splice(insertionIndex, 0, provenance ?? null);
    clearRoutingCandidates();
    pushToUndoStack(delta, reverseDelta, provenanceBefore, routingStore.segmentProvenance)
}

export async function editRoute(
    index: number,
    waypoints: Waypoint[],
    provenance?: RoutingSegmentProvenance,
    preserveRoutingCandidates = false,
) {
    const provenanceBefore = cloneProvenance(routingStore.segmentProvenance);
    const snapshot = new GPX({ ...routingStore.route })

    const segment = snapshot.trk?.at(0)?.trkseg?.at(index)
    if (segment) {
        segment.trkpt = waypoints
    }
    snapshot.features = snapshot.getTotals();

    const delta = diff(routingStore.route, snapshot);
    const reverseDelta = diff(snapshot, routingStore.route)
    routingStore.route = snapshot;
    routingStore.segmentProvenance[index] = provenance ?? null;
    if (!preserveRoutingCandidates) {
        clearRoutingCandidates();
    }
    pushToUndoStack(delta, reverseDelta, provenanceBefore, routingStore.segmentProvenance)
}

export function deleteFromRoute(index: number) {
    const provenanceBefore = cloneProvenance(routingStore.segmentProvenance);
    const snapshot = new GPX({ ...routingStore.route })

    snapshot.trk?.at(0)?.trkseg?.splice(index, 1);
    snapshot.features = snapshot.getTotals();

    const delta = diff(routingStore.route, snapshot);
    const reverseDelta = diff(snapshot, routingStore.route)
    routingStore.route = snapshot;
    routingStore.segmentProvenance.splice(index, 1);
    clearRoutingCandidates();
    pushToUndoStack(delta, reverseDelta, provenanceBefore, routingStore.segmentProvenance)
}

export function reverseRoute() {
    const provenanceBefore = cloneProvenance(routingStore.segmentProvenance);
    const snapshot = new GPX({ ...routingStore.route })
    for (const trk of snapshot.trk ?? []) {
        for (const seg of trk.trkseg ?? []) {
            seg.trkpt?.reverse()
        }
        trk.trkseg?.reverse()
    }
    snapshot.trk?.reverse()
    snapshot.features = snapshot.getTotals();

    const delta = diff(routingStore.route, snapshot);
    const reverseDelta = diff(snapshot, routingStore.route);
    routingStore.route = snapshot;
    const segmentCount = snapshot.trk?.at(0)?.trkseg?.length ?? 0;
    const loopMetadata =
        routingStore.closedLoop && isPersistedClosedLoop(provenanceBefore, segmentCount)
            ? provenanceBefore.find((entry) => entry?.routeTopology === "closed_loop")
            : undefined;
    if (loopMetadata) {
        // Reversing invalidates provider/profile provenance because the route
        // was not calculated in that direction. Retain only loop topology.
        routingStore.segmentProvenance = roundTripLoopProvenanceForSegments(
            loopMetadata,
            Array.from({ length: segmentCount }),
            loopMetadata.roundTripActualMeters ?? snapshot.features.distance,
        );
    } else {
        // A reversed provider route was not calculated with the active profile
        // in that direction, so its provenance deliberately becomes unknown.
        routingStore.segmentProvenance = (snapshot.trk?.at(0)?.trkseg ?? []).map(() => null);
    }
    clearRoutingCandidates();
    pushToUndoStack(delta, reverseDelta, provenanceBefore, routingStore.segmentProvenance)

    if (routingStore.closedLoop && routingStore.anchors.length > 1) {
        const [start, ...synthetic] = routingStore.anchors;
        routingStore.anchors = [start, ...synthetic.reverse()];
    } else {
        routingStore.anchors.reverse();
    }

    routingStore.anchors.forEach((a, i) => {
        if (!a.marker) {
            return;
        }
        renderRoutingAnchorMarker(
            a.marker.getElement(),
            i,
            routingStore.anchors.length,
            routingStore.closedLoop,
        );

        const anchorPopupHeading = a.marker
            .getPopup()
            ._content.getElementsByTagName("h5")[0];
        if (anchorPopupHeading) {
            anchorPopupHeading.textContent = routingAnchorTitle(
                i,
                routingStore.anchors.length,
                get(_),
                routingStore.closedLoop,
            );
        }
    });
}

export function resetRoute() {
    const provenanceBefore = cloneProvenance(routingStore.segmentProvenance);
    const delta = diff(routingStore.route, new GPX({ trk: [new Track({ ...emtpyTrack })] }));
    const reverseDelta = diff(new GPX({ trk: [new Track({ ...emtpyTrack })] }), routingStore.route);
    routingStore.route = applyRouteChangeset(delta);
    routingStore.segmentProvenance = [];
    routingStore.closedLoop = false;
    clearRoutingCandidates();
    pushToUndoStack(delta, reverseDelta, provenanceBefore, routingStore.segmentProvenance)

    routingStore.anchors.forEach((a) => {
        if (!a.marker) {
            return;
        }

        a.marker.remove();
    })

    routingStore.anchors = []
}

export async function recalculateHeight() {
    await routingStore.route.correctElevation();
}

export async function splitSegment(index: number, pos: LngLat) {
    let seg = routingStore.route.trk?.at(0)?.trkseg?.at(index);
    if (!seg || !seg.trkpt) {
        return;
    }
    const points = seg.trkpt;

    let bestSplitIndex: number = 0
    let minDistance = Infinity
    for (let i = 1; i < points.length; i++) {
        const pt = points[i]
        const dist = haversineDistance(pt.$.lat!, pt.$.lon!, pos.lat, pos.lng);

        if (dist < minDistance) {
            bestSplitIndex = i
            minDistance = dist
        }
    }

    const intersectionPoint = new Waypoint({ ...points[bestSplitIndex], $: { lat: pos.lat, lon: pos.lng } });
    const firstSegmentPoints = [...points.slice(0, bestSplitIndex), intersectionPoint];
    const secondSegmentPoints = [intersectionPoint, ...points.slice(bestSplitIndex)];

    const provenance = routingStore.segmentProvenance[index] ?? undefined;
    await editRoute(index, firstSegmentPoints, provenance ?? undefined)
    await insertIntoRoute(secondSegmentPoints, index + 1, provenance ?? undefined)

}

export function normalizeRouteTime() {
    let currentTime = new Date();

    for (const seg of routingStore.route.trk?.at(0)?.trkseg ?? []) {

        if (!seg.trkpt?.length) {
            continue
        }
        const baseTime = seg.trkpt[0].time?.getTime() ?? 0;
        for (let i = 0; i < seg.trkpt.length; i++) {
            const wp = seg.trkpt![i];
            const offset = (wp.time?.getTime() ?? 0) - baseTime;
            const adjustedTime = new Date(currentTime.getTime() + offset);

            wp.time = adjustedTime
        }
        currentTime = new Date(seg.trkpt[seg.trkpt.length - 1].time!.getTime());
    }
}

export function undo() {
    const historyItem = routingStore.undoStack.pop()
    if (!historyItem) {
        return undefined
    }
    routingStore.redoStack.push(historyItem)
    routingStore.selectedRouteCandidateId = undefined;

    routingStore.route = applyRouteChangeset(historyItem.reverseDelta);
    routingStore.segmentProvenance = cloneProvenance(historyItem.provenanceBefore ?? []);
    routingStore.route.features = routingStore.route.getTotals();
    return historyItem;
}

export function revertRouteChange() {
    const historyItem = routingStore.undoStack.pop();
    if (!historyItem) return;
    routingStore.selectedRouteCandidateId = undefined;
    routingStore.route = applyRouteChangeset(historyItem.reverseDelta);
    routingStore.segmentProvenance = cloneProvenance(historyItem.provenanceBefore ?? []);
    routingStore.route.features = routingStore.route.getTotals();
}

export function redo() {
    const historyItem = routingStore.redoStack.pop()
    if (!historyItem) {
        return undefined
    }
    routingStore.undoStack.push(historyItem)
    routingStore.selectedRouteCandidateId = undefined;

    routingStore.route = applyRouteChangeset(historyItem.delta);
    routingStore.segmentProvenance = cloneProvenance(historyItem.provenanceAfter ?? []);
    routingStore.route.features = routingStore.route.getTotals();
    return historyItem;
}
