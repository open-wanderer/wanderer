import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import GPX from "$lib/models/gpx/gpx";
import Track from "$lib/models/gpx/track";
import TrackSegment from "$lib/models/gpx/track-segment";
import Waypoint from "$lib/models/gpx/waypoint";
import type { RoutingCandidate, RoutingOptions } from "$lib/models/routing";
import { encodePolyline } from "$lib/util/polyline_util";
import {
    applyRoutingCandidate,
    calculateRoundTrip,
    clearRoutingCandidates,
    clearUndoRedoStack,
    deleteFromRoute,
    editRoute,
    insertIntoRoute,
    redo,
    revertRouteChange,
    reverseRoute,
    routingProvenanceMatchesOptions,
    routingStore,
    setRoute,
    undo,
} from "./routing_store.svelte";

afterEach(() => vi.unstubAllGlobals());

const options: RoutingOptions = {
    autoRouting: true,
    modeOfTransport: "pedestrian",
    category: "hiking",
    routingPluginId: "brouter",
    routingMode: "segment",
};

describe("round-trip request serialization", () => {
    it("unwraps reactive proxy preference and native-config maps", async () => {
        const request = vi.fn().mockResolvedValue(
            new Response(JSON.stringify({ candidates: [] }), {
                status: 200,
                headers: { "Content-Type": "application/json" },
            }),
        );
        vi.stubGlobal("fetch", request);
        const reactiveOptions: RoutingOptions = {
            ...options,
            nativeProfileKey: "pedestrian",
            preferences: new Proxy(
                { hillPreference: 0.6 },
                {},
            ),
            nativeConfig: new Proxy(
                { costing: new Proxy({ shortest: true }, {}) },
                {},
            ),
        };

        await calculateRoundTrip(
            { id: "start", lat: 47, lon: 8 },
            10_000,
            reactiveOptions,
        );

        const init = request.mock.calls[0][1] as RequestInit;
        const body = JSON.parse(String(init.body));
        expect(body.preferences).toEqual({ hillPreference: 0.6 });
        expect(body.profile.nativeConfig).toEqual({ costing: { shortest: true } });
    });
});

describe("routing provenance matching", () => {
    it("does not treat topology-only metadata as engine provenance", () => {
        expect(
            routingProvenanceMatchesOptions(
                { routeTopology: "closed_loop", roundTripRequestId: "rt_manual" },
                options,
            ),
        ).toBe(true);
    });

    it("uses the actual recalculated routing mode without a round-trip exception", () => {
        expect(
            routingProvenanceMatchesOptions(
                {
                    source: "round_trip",
                    routeTopology: "closed_loop",
                    pluginId: "brouter",
                    routingMode: "via",
                    category: "hiking",
                },
                options,
            ),
        ).toBe(false);
        expect(
            routingProvenanceMatchesOptions(
                {
                    routeTopology: "closed_loop",
                    pluginId: "brouter",
                    routingMode: "segment",
                    category: "hiking",
                },
                options,
            ),
        ).toBe(true);
    });

    it("accepts provenance from an applied alternative without switching the active engine", () => {
        expect(
            routingProvenanceMatchesOptions(
                {
                    source: "round_trip",
                    routeTopology: "closed_loop",
                    pluginId: "brouter",
                    category: "hiking",
                },
                options,
            ),
        ).toBe(true);
        expect(
            routingProvenanceMatchesOptions(
                {
                    source: "round_trip",
                    routeTopology: "closed_loop",
                    pluginId: "another-engine",
                    category: "hiking",
                },
                options,
            ),
        ).toBe(true);
    });
});

function waypoint(lat: number, lon: number, minute = 0) {
    return new Waypoint({
        $: { lat, lon },
        ele: 400 + minute,
        time: new Date(Date.UTC(2026, 0, 1, 12, minute)),
    });
}

function segment(points: [number, number][]) {
    return new TrackSegment({
        trkpt: points.map(([lat, lon], index) => waypoint(lat, lon, index)),
    });
}

function route(...segments: TrackSegment[]) {
    return new GPX({ trk: [new Track({ trkseg: segments })] });
}

function expectCompleteGPXClasses(gpx: GPX) {
    expect(gpx).toBeInstanceOf(GPX);
    expect(gpx.trk?.at(0)).toBeInstanceOf(Track);
    for (const trackSegment of gpx.trk?.at(0)?.trkseg ?? []) {
        expect(trackSegment).toBeInstanceOf(TrackSegment);
        for (const point of trackSegment.trkpt ?? []) {
            expect(point).toBeInstanceOf(Waypoint);
        }
    }
}

function expectCurrentFeatures() {
    expect(routingStore.route.features).toEqual(routingStore.route.getTotals());
}

const roundTripCandidate: RoutingCandidate = {
    id: "round-trip",
    compositionMode: "round_trip",
    summary: { distance: 2_000, duration: 1_200 },
    snappedAnchors: [
        { lat: 47, lon: 8 },
        { lat: 47.01, lon: 8.01 },
    ],
    segments: [
        {
            fromAnchor: 0,
            toAnchor: 1,
            geometry: {
                format: "encoded_polyline",
                precision: 6,
                coordinates: encodePolyline([
                    [47, 8],
                    [47.01, 8.01],
                ]),
            },
            distance: 2_000,
            duration: 1_200,
            provenance: {
                source: "round_trip",
                routeTopology: "closed_loop",
                pluginId: "brouter",
            },
        },
    ],
};

describe("route edit target state", () => {
    beforeEach(() => {
        routingStore.route = route(segment([[47, 8], [47.001, 8.001]]));
        routingStore.anchors = [];
        routingStore.segmentProvenance = [];
        routingStore.closedLoop = false;
        clearRoutingCandidates();
        clearUndoRedoStack();
    });

    it("adopts a supplied GPX directly and keeps changeset-based undo and redo", () => {
        const previous = routingStore.route;
        const next = route(segment([[46, 7], [46.01, 7.01]]));

        setRoute(next, true);

        expect(routingStore.route).toBe(next);
        expectCompleteGPXClasses(routingStore.route);
        expectCurrentFeatures();

        undo();
        expect(routingStore.route).not.toBe(previous);
        expect(routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)?.$.lat).toBe(47);
        expectCurrentFeatures();

        redo();
        expect(routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)?.$.lat).toBe(46);
        expectCurrentFeatures();
    });

    it("keeps inserted and edited route data as GPX model instances", async () => {
        const beforeInsert = routingStore.route;
        await insertIntoRoute([
            waypoint(47.002, 8.002),
            waypoint(47.003, 8.003, 1),
        ]);

        expect(routingStore.route).not.toBe(beforeInsert);
        expect(routingStore.route.trk?.at(0)?.trkseg).toHaveLength(2);
        expectCompleteGPXClasses(routingStore.route);
        expectCurrentFeatures();

        const beforeEdit = routingStore.route;
        await editRoute(0, [
            waypoint(48, 9),
            waypoint(48.01, 9.01, 1),
        ]);

        expect(routingStore.route).not.toBe(beforeEdit);
        expect(routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)?.$.lat).toBe(48);
        expectCompleteGPXClasses(routingStore.route);
        expectCurrentFeatures();

        revertRouteChange();
        expect(routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)?.$.lat).toBe(47);
        expectCurrentFeatures();
    });

    it("recomputes features on delete while adopting the new snapshot", () => {
        routingStore.route = route(
            segment([[47, 8], [47.01, 8.01]]),
            segment([[47.01, 8.01], [47.02, 8.02]]),
        );
        const previous = routingStore.route;

        deleteFromRoute(1);

        expect(routingStore.route).not.toBe(previous);
        expect(routingStore.route.trk?.at(0)?.trkseg).toHaveLength(1);
        expectCompleteGPXClasses(routingStore.route);
        expectCurrentFeatures();
    });

    it("adopts a fully reversed GPX snapshot with current features", () => {
        routingStore.route = route(
            segment([[47, 8], [47.01, 8.01]]),
            segment([[48, 9], [48.01, 9.01]]),
        );
        const previous = routingStore.route;

        reverseRoute();

        expect(routingStore.route).not.toBe(previous);
        expect(routingStore.route.trk?.at(0)?.trkseg?.at(0)?.trkpt?.at(0)?.$.lat).toBe(48.01);
        expectCompleteGPXClasses(routingStore.route);
        expectCurrentFeatures();
    });

    it("discards existing alternatives when a round trip is accepted as a replacement", () => {
        routingStore.routeCandidates = [
            roundTripCandidate,
            { ...roundTripCandidate, id: "alternative" },
        ];
        routingStore.routeCandidateOrigin = { route: routingStore.route };
        routingStore.selectedRouteCandidateId = null;
        routingStore.routeCandidateWarnings = ["warning"];

        applyRoutingCandidate(roundTripCandidate, true, false);

        expect(routingStore.routeCandidates).toEqual([]);
        expect(routingStore.routeCandidateOrigin).toBeUndefined();
        expect(routingStore.selectedRouteCandidateId).toBeUndefined();
        expect(routingStore.routeCandidateWarnings).toEqual([]);
        expect(routingStore.closedLoop).toBe(true);
        expectCompleteGPXClasses(routingStore.route);
        expectCurrentFeatures();
    });
});
