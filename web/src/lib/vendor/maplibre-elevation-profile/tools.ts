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
