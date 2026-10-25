import GPX from "$lib/models/gpx/gpx";
import TrackSegment from "$lib/models/gpx/track-segment";
import GPXWaypoint from "$lib/models/gpx/waypoint";

// The route editor takes its anchors from the boundaries between track
// segments. When a drawn route is saved or exported, the point where two
// segments met is marked with this extension so the editor can split it again.
export const routeAnchorNamespace = "https://wanderer.to/xmlns/gpx/1";
const routeAnchorElement = "wanderer:anchor";

function markRouteAnchor(point: GPXWaypoint): boolean {
    const extensions = point.extensions as unknown;
    if (extensions === undefined || extensions === "") {
        (point as any).extensions = { [routeAnchorElement]: "" };
        return true;
    }
    if (typeof extensions === "object" && extensions !== null) {
        (extensions as Record<string, unknown>)[routeAnchorElement] = "";
        return true;
    }
    return false;
}

function unmarkRouteAnchor(point: GPXWaypoint): boolean {
    const extensions = point.extensions as unknown;
    if (typeof extensions !== "object" || extensions === null || !Object.prototype.hasOwnProperty.call(extensions, routeAnchorElement)) {
        return false;
    }
    delete (extensions as Record<string, unknown>)[routeAnchorElement];
    if (Object.keys(extensions).length === 0) {
        point.extensions = undefined;
    }
    return true;
}

export function joinContiguousTrackSegments(gpx: GPX) {
    let marked = false;

    for (const track of gpx.trk ?? []) {
        const joined: TrackSegment[] = [];

        for (const segment of track.trkseg ?? []) {
            const points = segment.trkpt ?? [];
            if (points.length === 0) {
                continue;
            }

            const current = joined.at(-1);
            const last = current?.trkpt?.at(-1);
            const first = points[0];
            const meets = last != null && Number(last.$.lat) === Number(first.$.lat) && Number(last.$.lon) === Number(first.$.lon);

            if (current?.trkpt && !current.extensions && !segment.extensions && meets && markRouteAnchor(last)) {
                current.trkpt.push(...points.slice(1));
                marked = true;
            } else {
                joined.push(segment);
            }
        }

        track.trkseg = joined;
    }

    if (marked) {
        (gpx.$ as Record<string, string>)["xmlns:wanderer"] = routeAnchorNamespace;
    }
}

export function splitTrackSegmentsAtAnchors(gpx: GPX) {
    for (const track of gpx.trk ?? []) {
        const split: TrackSegment[] = [];

        for (const segment of track.trkseg ?? []) {
            const points = segment.trkpt ?? [];
            let start = 0;

            for (let i = 0; i < points.length; i++) {
                if (!unmarkRouteAnchor(points[i]) || i === 0 || i === points.length - 1) {
                    continue;
                }
                const leg = new TrackSegment({ extensions: segment.extensions });
                leg.trkpt = points.slice(start, i + 1);
                split.push(leg);
                points[i] = new GPXWaypoint(points[i]);
                start = i;
            }

            if (start === 0) {
                split.push(segment);
            } else {
                const leg = new TrackSegment({ extensions: segment.extensions });
                leg.trkpt = points.slice(start);
                split.push(leg);
            }
        }

        track.trkseg = split;
    }

    delete (gpx.$ as Record<string, string>)["xmlns:wanderer"];
}

export function routeToStoredGpx(gpx: GPX): string {
    const stored = GPX.parse(gpx.toString());
    joinContiguousTrackSegments(stored);
    return stored.toString();
}
