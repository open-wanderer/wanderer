import type { FeatureCollection, Position } from "geojson";
import sax from "sax";

// Errors the DOM reader stops at too: the file is damaged, and what was read of
// it cannot be trusted. It gets past everything else, such as a bare ampersand
// in a name, and so does this.
const FATAL_ERRORS = ["Unclosed root tag", "Unexpected end", "Unexpected close tag", "Unmatched closing tag"];

type Lap = { name?: string, tracks: Position[][] };
type Trackpoint = { lat?: number, lon?: number, ele?: number, time?: string };

/**
 * Reads a TCX on the server. A DOM of a long recording does not fit into the
 * heap, while this keeps only what the GPX is built from. The result matches
 * what the vendored tcx() gives fromGeoJSON. onPoint gets the running number of
 * points, and can throw to stop reading.
 */
export function tcxToGeoJSON(xml: string, onPoint: (count: number) => void): FeatureCollection {
    const parser = sax.parser(true);
    const laps: Lap[] = [];
    const courses: Lap[] = [];

    let lap: Lap | null = null;
    let lapTag = "";
    let track: Position[] | null = null;
    let point: Trackpoint | null = null;
    let text = "";
    let points = 0;
    let hasRoot = false;

    parser.onerror = (error) => {
        if (FATAL_ERRORS.some((message) => error.message.startsWith(message))) {
            throw error;
        }
        parser.resume();
    };
    parser.onopentag = ({ name }) => {
        hasRoot = true;
        text = "";
        if (lap === null && (name === "Lap" || name === "Courses")) {
            lap = { tracks: [] };
            lapTag = name;
            (name === "Lap" ? laps : courses).push(lap);
        } else if (lap !== null && name === "Track") {
            track = [];
        } else if (track !== null && name === "Trackpoint") {
            point = {};
        }
    };
    parser.ontext = (value) => { text += value; };
    parser.oncdata = (value) => { text += value; };
    parser.onclosetag = (name) => {
        if (point !== null && track !== null) {
            if (name === "LatitudeDegrees") point.lat ??= parseFloat(text);
            else if (name === "LongitudeDegrees") point.lon ??= parseFloat(text);
            else if (name === "AltitudeMeters") point.ele ??= parseFloat(text);
            else if (name === "Time") point.time ??= text;
            else if (name === "Trackpoint") {
                const position = toPosition(point);
                if (position) {
                    track.push(position);
                    onPoint(++points);
                }
                point = null;
            }
        } else if (track !== null && lap !== null && name === "Track") {
            if (track.length >= 2) {
                lap.tracks.push(track);
            }
            track = null;
        } else if (lap !== null && name === "Name") {
            lap.name ??= text;
        } else if (name === lapTag) {
            lap = null;
        }
        text = "";
    };

    parser.write(xml).close();
    if (!hasRoot) {
        throw new Error("Not an XML document");
    }

    return {
        type: "FeatureCollection",
        features: [...laps, ...courses].filter(({ tracks }) => tracks.length > 0).map(({ name, tracks }) => ({
            type: "Feature",
            properties: name ? { name } : {},
            geometry: tracks.length === 1
                ? { type: "LineString", coordinates: tracks[0] }
                : { type: "MultiLineString", coordinates: tracks },
        })),
    };
}

// As in tcx(), a point has a time only if it has an altitude element.
function toPosition({ lat, lon, ele, time }: Trackpoint): Position | null {
    if (lat === undefined || lon === undefined || isNaN(lat) || isNaN(lon)) {
        return null;
    }
    if (ele === undefined) {
        return [lon, lat];
    }
    return [lon, lat, isNaN(ele) ? null : ele, time ?? ""] as unknown as Position;
}
