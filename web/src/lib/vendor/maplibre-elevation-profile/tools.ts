import { haversineDistance } from "$lib/models/gpx/utils";
import type {
    Position
} from "geojson";


export function haversineCumulatedDistanceWgs84(path: Position[]): number[] {
    if (path.length < 2) {
        return [];
    }

    let totalDistance = 0;
    const distances: number[] = [0];    

    for (let i = 0; i < path.length - 1; i++) {
        const [lon1, lat1] = path[i];
        const [lon2, lat2] = path[i + 1];

        totalDistance += haversineDistance(lat1, lon1, lat2, lon2);
        distances.push(totalDistance);
    }    

    return distances; // Array of cumulative distances in meters
}

// Track points this much farther from a waypoint than the nearest one count as
// equally near, about the error of a phone GPS fix.
const WAYPOINT_PASS_TOLERANCE_METERS = 25;

// A stored distance from start is only trusted when it is the distance of one of
// the candidate points. Values in other units (legacy imperial waypoints stored
// miles × 1000) or from before the track was edited match none of them.
const STORED_DISTANCE_MATCH_METERS = 1;

// PocketBase stores an unset number as 0, so 0 cannot mean "at the first track
// point". Waypoints there store this instead, and a stored 0 is no hint.
export const MIN_STORED_DISTANCE_METERS = 0.01;

// Returns the index of the track point a waypoint belongs to: the nearest one.
// Where the track covers the same ground twice (out-and-back, lollipop), several
// passes are about as near, so pick the pass by the photo's time if the track
// has real timestamps that cover it, else by the stored distance from start,
// and then the nearest point of that pass. A hint never moves a waypoint along
// a single pass.
export function waypointTrackIndex(
    positions: Position[],
    times: Date[],
    cumulatedDistance: number[],
    waypoint: { lat: number, lon: number, _time?: Date, distance_from_start?: number },
): number {
    const offTrack = positions.map(([lon, lat]) => haversineDistance(lat, lon, waypoint.lat, waypoint.lon));

    let nearest = -1;
    for (let i = 0; i < offTrack.length; i++) {
        if (nearest < 0 || offTrack[i] < offTrack[nearest]) {
            nearest = i;
        }
    }
    if (nearest < 0) {
        return nearest;
    }

    // Consecutive candidate points belong to the same pass.
    const maxOffTrack = offTrack[nearest] + WAYPOINT_PASS_TOLERANCE_METERS;
    const passes: number[][] = [];
    for (let i = 0; i < offTrack.length; i++) {
        if (offTrack[i] > maxOffTrack) {
            continue;
        }
        const pass = passes.at(-1);
        if (pass && pass.at(-1) === i - 1) {
            pass.push(i);
        } else {
            passes.push([i]);
        }
    }
    if (passes.length < 2) {
        return nearest;
    }

    const score = passScore(positions, times, cumulatedDistance, waypoint);
    if (!score) {
        return nearest;
    }

    let bestPass: number[] | undefined;
    let bestScore = Infinity;
    for (const pass of passes) {
        for (const i of pass) {
            const pointScore = score(i);
            if (pointScore < bestScore) {
                bestPass = pass;
                bestScore = pointScore;
            }
        }
    }
    if (!bestPass) {
        return nearest;
    }
    return bestPass.reduce((a, b) => (offTrack[b] < offTrack[a] ? b : a));
}

// Returns how far a track point is from the waypoint's hint (lower is closer),
// or undefined when the waypoint has no usable hint.
function passScore(
    positions: Position[],
    times: Date[],
    cumulatedDistance: number[],
    waypoint: { _time?: Date, distance_from_start?: number },
): ((index: number) => number) | undefined {
    const photoTime = waypoint._time?.getTime();
    if (photoTime !== undefined && Number.isFinite(photoTime) && times.length === positions.length) {
        const trackTimes = times.map((t) => t.getTime());
        let start = Infinity;
        let end = -Infinity;
        for (const t of trackTimes) {
            start = Math.min(start, t);
            end = Math.max(end, t);
        }
        // A photo outside the track's time range was not taken while recording
        // it, e.g. on a drawn route whose times are synthetic or were shifted
        // to the time of an edit, so its time says nothing about the pass.
        // NaN for a missing time fails these comparisons as well.
        if (photoTime >= start && photoTime <= end) {
            return (i) => Math.abs(trackTimes[i] - photoTime);
        }
    }

    const storedDistance = waypoint.distance_from_start;
    if (storedDistance && Number.isFinite(storedDistance)) {
        return (i) => {
            const mismatch = Math.abs(cumulatedDistance[i] - storedDistance);
            return mismatch <= STORED_DISTANCE_MATCH_METERS ? mismatch : Infinity;
        };
    }
    return undefined;
}

export function smoothElevations(positions: Position[], windowSize: number): Position[] {
    // Ensure windowSize is valid (at least 1)
    if (windowSize < 1) {
        console.warn("Window size must be at least 1.");
        return positions;
    }

    const len = positions.length;
    if (len === 0) {
        return positions;
    }

    const half = Math.floor(windowSize / 2);
    const result: Position[] = new Array(len);

    for (let i = 0; i < len; i++) {
        const start = Math.max(0, i - half);
        const end = Math.min(len, i + half + 1);

        let weightedSum = 0;
        let weightTotal = 0;
        let weight = 1;

        for (let j = start; j < end; j++) {
            weightedSum += (positions[j][2] ?? 0) * weight;
            weightTotal += weight;
            weight++;
        }

        result[i] = [positions[i][0], positions[i][1], weightedSum / weightTotal] as Position;
    }

    return result;
}
