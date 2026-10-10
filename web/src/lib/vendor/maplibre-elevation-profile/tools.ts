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

// Returns the index of the track point a waypoint belongs to: the nearest one.
// Where the track covers the same ground twice (out-and-back, lollipop), several
// points are about as near, so pick between them by the photo's time if both it
// and the track have timestamps, else by the previously stored distance from start.
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

    const photoTime = times.length === positions.length ? waypoint._time?.getTime() : undefined;
    const storedDistance = waypoint.distance_from_start || undefined;
    if (nearest < 0 || (photoTime === undefined && storedDistance === undefined)) {
        return nearest;
    }

    const maxOffTrack = offTrack[nearest] + WAYPOINT_PASS_TOLERANCE_METERS;
    let best = nearest;
    let bestScore = Infinity;
    for (let i = 0; i < offTrack.length; i++) {
        if (offTrack[i] > maxOffTrack) {
            continue;
        }
        const score = photoTime !== undefined
            ? Math.abs(times[i].getTime() - photoTime)
            : Math.abs(cumulatedDistance[i] - storedDistance!);
        if (score < bestScore) {
            best = i;
            bestScore = score;
        }
    }
    return best;
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
