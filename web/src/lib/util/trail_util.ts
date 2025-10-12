import { Trail } from "$lib/models/trail";
import type { Threshold } from "$lib/models/difficulty_algorithms";

const diffDifficult = "difficult";
const diffModerate = "moderate";
const diffEasy = "easy";

export function getTrailDifficulty(t: Trail, thresholds: Threshold[]) : "easy" | "moderate" | "difficult" | undefined {
    if (!t.distance || !t.elevation_gain) return undefined;

    if (doGetTrailDifficulty(t, thresholds, diffDifficult) == diffDifficult)
        return diffDifficult;
    if (doGetTrailDifficulty(t, thresholds, diffModerate) == diffModerate)
        return diffModerate;
    
    return diffEasy;
}

function doGetTrailDifficulty(t: Trail, thresholds: Threshold[], difficulty: "moderate" | "difficult") : "moderate" | "difficult" | undefined {
    if (!t.distance || !t.elevation_gain) return undefined;

    let threshDuration = thresholds.find((thresh) => thresh.difficulty == difficulty && thresh.type == "duration");
    if (t.duration && threshDuration && t.duration > threshDuration.limit)
            return difficulty;
    
    let threshDistance = thresholds.find((thresh) => thresh.difficulty == difficulty && thresh.type == "distance");
    if (threshDistance && t.distance > threshDistance.limit)
        return difficulty;
    
    let threshElevation = thresholds.find((thresh) => thresh.difficulty == difficulty && thresh.type == "elevation");
    if (threshElevation && t.elevation_gain > threshElevation.limit)
        return difficulty;

    return undefined;
}