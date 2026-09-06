import { describe, expect, it } from "vitest";
import { encodePolyline, polylineToGeoJSON } from "./polyline_util";

describe("polylineToGeoJSON", () => {
    it("decodes backend polylines (lat/lon pairs, precision 5) into lon/lat coordinates", () => {
        // Mirrors db/util/polyline.go: coordinates are encoded as [lat, lon].
        const encoded = encodePolyline(
            [
                [47.37174, 8.54226],
                [47.37693, 8.53839],
            ],
            5,
        );

        const geojson = polylineToGeoJSON(encoded);
        const line = geojson.features[0].geometry;

        expect(line.type).toBe("LineString");
        expect(line.type === "LineString" && line.coordinates).toEqual([
            [8.54226, 47.37174],
            [8.53839, 47.37693],
        ]);
        expect(geojson.bbox).toEqual([8.53839, 47.37174, 8.54226, 47.37693]);
    });
});
