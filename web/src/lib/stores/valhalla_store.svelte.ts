import GPX from "$lib/models/gpx/gpx";
import Track from "$lib/models/gpx/track";
import TrackSegment from "$lib/models/gpx/track-segment";
import { haversineDistance } from "$lib/models/gpx/utils";
import Waypoint from "$lib/models/gpx/waypoint";
import { type RoutingOptions, type ValhallaAnchor, type ValhallaHeightResponse, type ValhallaRouteResponse } from "$lib/models/valhalla";
import { APIError } from "$lib/util/api_util";
import { decodePolyline, encodePolyline } from "$lib/util/polyline_util";
import { applyChangeset, diff, revertChangeset, type Changeset } from 'json-diff-ts';
import type { LngLat } from "maplibre-gl";
import { _ } from "svelte-i18n";
import { get } from "svelte/store";

const emtpyTrack = new Track({ trkseg: [] })

class ValhallaStore {
    route: GPX = $state(new GPX({ trk: [emtpyTrack] }));
    anchors: ValhallaAnchor[] = $state([]);
    undoStack: { delta: Changeset, reverseDelta: Changeset }[] = $state([]);
    redoStack: { delta: Changeset, reverseDelta: Changeset }[] = $state([]);
}

export const valhallaStore = new ValhallaStore();

export function clearRoute() {
    valhallaStore.route = new GPX({ trk: [emtpyTrack] });
}

export function clearAnchors() {
    for (const anchor of valhallaStore.anchors) {
        anchor.marker?.remove();
    }
    valhallaStore.anchors = [];
}

export function clearUndoRedoStack() {
    valhallaStore.undoStack = []
    valhallaStore.redoStack = []
}

function pushToUndoStack(delta: Changeset, reverseDelta: Changeset) {
    valhallaStore.undoStack.push({ delta, reverseDelta })
    valhallaStore.redoStack = []
}


export function setRoute(newRoute: GPX, undoable: boolean = false) {
    const delta = diff(valhallaStore.route, newRoute);
    const reverseDelta = diff(newRoute, valhallaStore.route);
    valhallaStore.route = applyChangeset(valhallaStore.route, delta);
    if (undoable) {
        pushToUndoStack(delta, reverseDelta)
    }

}

export async function calculateRouteBetween(startLat: number, startLon: number, endLat: number, endLon: number, options: RoutingOptions) {

    let shape;
    let duration: number;
    if (options.autoRouting) {
        let costingBody;
        switch (options.modeOfTransport) {
            case "bicycle":
                costingBody = { "costing": options.modeOfTransport, "costing_options": { [options.modeOfTransport]: options.bicycleOptions } }
                break;
            case "auto":
                costingBody = { "costing": options.modeOfTransport, "costing_options": { [options.modeOfTransport]: options.autoOptions } }
                break;
            case "pedestrian":
                costingBody = { "costing": options.modeOfTransport, "costing_options": { [options.modeOfTransport]: options.pedestrianOptions } }
                break;
        }
        const requestBody = {
            "directions_type": "none",
            "locations": [{ "lat": startLat, "lon": startLon }, { "lat": endLat, "lon": endLon }],
            ...costingBody
        }

        let r = await fetch("/api/v1/valhalla/route", { method: "POST", body: JSON.stringify(requestBody) })

        if (!r.ok) {
            const response = await r.json();
            throw new APIError(r.status, response.message, response.detail)
        }

        const routeResponse: ValhallaRouteResponse = await r.json();
        shape = routeResponse.trip.legs[0].shape
        duration = routeResponse.trip.summary.time
    } else {
        shape = encodePolyline([[startLat, startLon], [endLat, endLon]])
        duration = 0;
    }

    const r2 = await fetch("/api/v1/valhalla/height", { method: "POST", body: JSON.stringify({ encoded_polyline: shape }) })

    if (!r2.ok) {
        const response = await r2.json();
        throw new APIError(r2.status, response.message, response.detail)
    }

    const heightResponse: ValhallaHeightResponse = await r2.json()
    const points = decodePolyline(shape);
    const startTime = new Date().getTime();

    const waypoints = points.map((p, i) => new Waypoint({ $: { lat: p[1], lon: p[0] }, ele: heightResponse.height[i], time: new Date(startTime + (((duration * 1000) / points.length) * i)) }))

    return waypoints
}

export async function insertIntoRoute(waypoints: Waypoint[], index?: number) {
    const snapshot = new GPX({ ...valhallaStore.route })
    const segment = new TrackSegment({ trkpt: waypoints })

    if (index) {
        snapshot.trk?.at(0)?.trkseg?.splice(index, 0, segment);
    } else {
        snapshot.trk?.at(0)?.trkseg?.push(segment);
    }

    const delta = diff(valhallaStore.route, snapshot);
    const reverseDelta = diff(snapshot, valhallaStore.route);
    valhallaStore.route = applyChangeset(valhallaStore.route, delta);
    pushToUndoStack(delta, reverseDelta)

    valhallaStore.route.features = valhallaStore.route.getTotals();
}

export async function editRoute(index: number, waypoints: Waypoint[]) {
    const snapshot = new GPX({ ...valhallaStore.route })

    const segment = snapshot.trk?.at(0)?.trkseg?.at(index)
    if (segment) {
        segment.trkpt = waypoints
    }

    const delta = diff(valhallaStore.route, snapshot);
    const reverseDelta = diff(snapshot, valhallaStore.route)
    valhallaStore.route = applyChangeset(valhallaStore.route, delta);
    pushToUndoStack(delta, reverseDelta)



    valhallaStore.route.features = valhallaStore.route.getTotals();
}

export function deleteFromRoute(index: number) {
    const snapshot = new GPX({ ...valhallaStore.route })

    snapshot.trk?.at(0)?.trkseg?.splice(index, 1);
    snapshot.features = valhallaStore.route.getTotals();

    const delta = diff(valhallaStore.route, snapshot);
    const reverseDelta = diff(snapshot, valhallaStore.route)
    valhallaStore.route = applyChangeset(valhallaStore.route, delta);
    pushToUndoStack(delta, reverseDelta)
}

export function reverseRoute() {
    const snapshot = new GPX({ ...valhallaStore.route })
    for (const trk of snapshot.trk ?? []) {
        for (const seg of trk.trkseg ?? []) {
            seg.trkpt?.reverse()
        }
        trk.trkseg?.reverse()
    }
    snapshot.trk?.reverse()

    const delta = diff(valhallaStore.route, snapshot);
    const reverseDelta = diff(snapshot, valhallaStore.route);
    valhallaStore.route = applyChangeset(valhallaStore.route, delta);
    pushToUndoStack(delta, reverseDelta)

    valhallaStore.route.features = valhallaStore.route.getTotals();

    valhallaStore.anchors.reverse();

    valhallaStore.anchors.forEach((a, i) => {
        if (!a.marker) {
            return;
        }
        a.marker.getElement().textContent = "" + (i + 1);

        const anchorPopupHeading = a.marker
            .getPopup()
            ._content.getElementsByTagName("h5")[0];
        if (anchorPopupHeading) {
            anchorPopupHeading.textContent =
                get(_)("route-point") + " #" + (i + 1);
        }
    });
}

export function resetRoute() {
    const delta = diff(valhallaStore.route, new GPX({ trk: [new Track({ ...emtpyTrack })] }));
    const reverseDelta = diff(new GPX({ trk: [new Track({ ...emtpyTrack })] }), valhallaStore.route);
    valhallaStore.route = applyChangeset(valhallaStore.route, delta);
    pushToUndoStack(delta, reverseDelta)

    valhallaStore.anchors.forEach((a) => {
        if (!a.marker) {
            return;
        }

        a.marker.remove();
    })

    valhallaStore.anchors = []
}

export async function recalculateHeight() {
    await valhallaStore.route.correctElevation();
}

export async function splitSegment(index: number, pos: LngLat) {
    let seg = valhallaStore.route.trk?.at(0)?.trkseg?.at(index);
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

    editRoute(index, firstSegmentPoints)
    insertIntoRoute(secondSegmentPoints, index + 1)

}

export function normalizeRouteTime() {
    let currentTime = new Date();

    for (const seg of valhallaStore.route.trk?.at(0)?.trkseg ?? []) {

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
    const historyItem = valhallaStore.undoStack.pop()
    if (!historyItem) {
        return
    }
    valhallaStore.redoStack.push(historyItem)

    valhallaStore.route = applyChangeset(valhallaStore.route, historyItem.reverseDelta);
    valhallaStore.route.features = valhallaStore.route.getTotals();
}

export function redo() {
    const historyItem = valhallaStore.redoStack.pop()
    if (!historyItem) {
        return
    }
    valhallaStore.undoStack.push(historyItem)

    valhallaStore.route = applyChangeset(valhallaStore.route, historyItem.delta);
    valhallaStore.route.features = valhallaStore.route.getTotals();
}
