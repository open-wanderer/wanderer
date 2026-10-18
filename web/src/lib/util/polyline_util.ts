import type { GeoJSON } from "geojson";
import { bbox } from "./geojson_util";


function py2_round(value: number) {
    return Math.floor(Math.abs(value) + 0.5) * (value >= 0 ? 1 : -1);
}

function encode(current: number, previous: number, factor: number) {
    current = py2_round(current * factor);
    previous = py2_round(previous * factor);
    var coordinate = (current - previous) * 2;
    if (coordinate < 0) {
        coordinate = -coordinate - 1
    }
    var output = '';
    while (coordinate >= 0x20) {
        output += String.fromCharCode((0x20 | (coordinate & 0x1f)) + 63);
        coordinate /= 32;
    }
    output += String.fromCharCode((coordinate | 0) + 63);
    return output;
}


/**
 * Decode a Google/Valhalla encoded polyline string into coordinate pairs.
 *
 * Internally the algorithm accumulates latitude first and longitude second
 * (standard polyline encoding order), but each returned pair is ordered
 * `[lng, lat]` (GeoJSON / MapLibre convention, longitude first).
 *
 * @param str - Encoded polyline string
 * @param precision - Encoding precision exponent (default 6 for Valhalla)
 * @returns Array of `[lng, lat]` pairs
 */
export function decodePolyline(str: string, precision: number = 6) {
    var index = 0,
        lat = 0,
        lng = 0,
        coordinates = [],
        shift = 0,
        result = 0,
        byte = null,
        latitude_change,
        longitude_change,
        factor = Math.pow(10, precision);

    while (index < str.length) {
        byte = null;
        shift = 0;
        result = 0;

        do {
            byte = str.charCodeAt(index++) - 63;
            result |= (byte & 0x1f) << shift;
            shift += 5;
        } while (byte >= 0x20);

        latitude_change = ((result & 1) ? ~(result >> 1) : (result >> 1));

        shift = result = 0;

        do {
            byte = str.charCodeAt(index++) - 63;
            result |= (byte & 0x1f) << shift;
            shift += 5;
        } while (byte >= 0x20);

        longitude_change = ((result & 1) ? ~(result >> 1) : (result >> 1));

        lat += latitude_change;
        lng += longitude_change;

        coordinates.push([lng / factor, lat / factor]);
    }

    return coordinates;
};


/**
 * Encode an array of coordinate pairs into a Google/Valhalla encoded polyline string.
 *
 * IMPORTANT: Because `decodePolyline` returns `[lng, lat]` pairs, callers that
 * need encode/decode to be invertible MUST pass `[lat, lng]` pairs here
 * (i.e. the first element of each pair is treated as the first encoded value,
 * which the decoder interprets as latitude).
 *
 * @param coordinates - Array of `[lat, lng]` pairs (latitude-first for invertibility with decodePolyline)
 * @param precision - Encoding precision exponent (default 6 for Valhalla)
 */
export function encodePolyline(coordinates: number[][], precision: number = 6) {
    if (!coordinates.length) { return ''; }

    var factor = Math.pow(10, precision),
        output = encode(coordinates[0][0], 0, factor) + encode(coordinates[0][1], 0, factor);

    for (var i = 1; i < coordinates.length; i++) {
        var a = coordinates[i], b = coordinates[i - 1];
        output += encode(a[0], b[0], factor);
        output += encode(a[1], b[1], factor);
    }

    return output;
};

function flipped(coords: number[][]) {
    var flipped = [];
    for (var i = 0; i < coords.length; i++) {
        var coord = coords[i].slice();
        flipped.push([coord[1], coord[0]]);
    }
    return flipped;
}

export function polylineToGeoJSON(str: string, precision: number = 6) {
    var coords = decodePolyline(str, precision);
    const geojson = {
        type: "FeatureCollection",
        features: [
            {
                properties: {},
                type: "Feature",
                geometry: {
                    type: "LineString",
                    coordinates: flipped(coords),
                },
            }
        ]

    } as GeoJSON;
    geojson.bbox = bbox(geojson)

    return geojson
};