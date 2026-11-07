import type { FeatureCollection, Position } from "geojson";
import sax from "sax";

// A file that ends early or has mismatched tags is refused, as by the DOM reader:
// recovering from those would silently drop points. Like that reader, this gets
// past other damage, such as a bare ampersand in a name.
const FATAL_ERRORS = ["Unclosed root tag", "Unexpected end", "Unexpected close tag", "Unmatched closing tag"];

// sax reports some errors once per character, so a file of them must not get
// to be read to its end. No real file comes close to either limit.
const MAX_RECOVERED_ERRORS = 10_000;
const MAX_DEPTH = 1_000;

// sax only checks the length of its buffers between writes.
const CHUNK_LENGTH = 64 * 1024;
const BUFFER_OVERRUN = "Max buffer length exceeded: ";

const END_OF_DOCUMENT = Symbol("end of document");

const POINT_FIELDS = new Set(["LatitudeDegrees", "LongitudeDegrees", "AltitudeMeters", "Time"]);

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
    let reading: string | null = null;
    let text = "";
    let points = 0;
    let hasRoot = false;
    let depth = 0;
    let recoveredErrors = 0;

    parser.onerror = (error) => {
        if (FATAL_ERRORS.some((message) => error.message.startsWith(message))) {
            throw error;
        }
        if (error.message.startsWith(BUFFER_OVERRUN)) {
            // Comments, attributes and the like, which are not read. Text
            // and CDATA never get here.
            const buffer = error.message.slice(BUFFER_OVERRUN.length).split("\n")[0];
            (parser as unknown as Record<string, string>)[buffer] = "";
        } else if (++recoveredErrors > MAX_RECOVERED_ERRORS) {
            throw new Error("Too many errors in TCX file");
        }
        parser.resume();
    };
    // Attributes are not read, and sax would keep those of a tag in one object.
    parser.onattribute = () => {
        parser.tag.attributes = {};
    };
    parser.onopentag = ({ name }) => {
        hasRoot = true;
        if (++depth > MAX_DEPTH) {
            throw new Error("TCX file is nested too deeply");
        }
        if (reading === null && (point !== null ? POINT_FIELDS.has(name) : lap !== null && name === "Name")) {
            reading = name;
            text = "";
        }
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
    parser.ontext = (value) => {
        if (reading !== null) text += value;
    };
    parser.oncdata = (value) => {
        if (reading !== null) text += value;
    };
    parser.onclosetag = (name) => {
        if (name === reading) {
            reading = null;
            if (point === null) {
                if (lap !== null) lap.name ??= text;
            } else if (name === "LatitudeDegrees") point.lat ??= parseFloat(text);
            else if (name === "LongitudeDegrees") point.lon ??= parseFloat(text);
            else if (name === "AltitudeMeters") point.ele ??= parseFloat(text);
            else point.time ??= text;
        } else if (point !== null && track !== null && name === "Trackpoint") {
            const position = toPosition(point);
            if (position) {
                track.push(position);
                onPoint(++points);
            }
            point = null;
        } else if (point === null && track !== null && lap !== null && name === "Track") {
            if (track.length >= 2) {
                lap.tracks.push(track);
            }
            track = null;
        } else if (point === null && name === lapTag) {
            lap = null;
        }
        if (--depth === 0) {
            // What follows the root can only produce errors.
            throw END_OF_DOCUMENT;
        }
    };

    try {
        for (let offset = 0; offset < xml.length; offset += CHUNK_LENGTH) {
            parser.write(xml.slice(offset, offset + CHUNK_LENGTH));
        }
        parser.close();
    } catch (error) {
        if (error !== END_OF_DOCUMENT) {
            throw error;
        }
    }
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
