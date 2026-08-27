import type { FeatureCollection } from "geojson";
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

function distanceToExpected(
    coord: number[],
    expected: { lat: number; lon: number },
) {
    return Math.hypot(coord[0] - expected.lon, coord[1] - expected.lat);
}

function decodePolylineVariants(str: string) {
    return [5, 6].flatMap((precision) => {
        const coords = decodePolyline(str, precision);
        return [coords, coords.map((coord) => [coord[1], coord[0]])];
    });
}

export function polylineToGeoJSON(
    str: string,
    precision: number = 5,
    expected?: { lat: number; lon: number },
): FeatureCollection {
    let coords = decodePolyline(str, precision);

    if (expected && coords.length) {
        let best = coords;
        let bestDistance = distanceToExpected(coords[0], expected);
        for (const variant of decodePolylineVariants(str)) {
            if (!variant.length) {
                continue;
            }
            const distance = distanceToExpected(variant[0], expected);
            if (distance < bestDistance) {
                best = variant;
                bestDistance = distance;
            }
        }
        coords = best;
    }

    const geojson: FeatureCollection = {
        type: "FeatureCollection",
        features: [
            {
                properties: {},
                type: "Feature",
                geometry: {
                    type: "LineString",
                    coordinates: coords,
                },
            }
        ]

    };
    geojson.bbox = bbox(geojson)

    return geojson
};