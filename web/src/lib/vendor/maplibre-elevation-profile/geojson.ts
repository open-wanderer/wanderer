import type {
    Feature,
    FeatureCollection,
    GeoJsonObject,
    GeometryObject,
    LineString,
    MultiLineString,
    Position,
} from "geojson";

function extractLineStrings(
    geoJson: GeoJsonObject
): { lineStrings: Array<LineString | MultiLineString>; times: Date[] } {
    const lineStrings: Array<LineString | MultiLineString> = [];
    const times: Date[] = [];

    function extractFromGeometry(geometry: GeometryObject) {
        if (
            geometry.type === "LineString" ||
            geometry.type === "MultiLineString"
        ) {
            lineStrings.push(geometry as LineString | MultiLineString);
        }
    }

    function extractFromFeature(feature: Feature) {
        if (feature.geometry) {
            extractFromGeometry(feature.geometry);
        }
        if (feature.properties?.coordinateProperties?.times) {
            const coordinateTimes = feature.properties?.coordinateProperties?.times.map(
                (t: string) => new Date(t)
            );
            times.push(...coordinateTimes);
        }
    }

    function extractFromFeatureCollection(collection: FeatureCollection) {
        for (const feature of collection.features) {
            if (feature.type === "Feature") {
                extractFromFeature(feature);
            } else if (feature.type === "FeatureCollection") {
                extractFromFeatureCollection(
                    feature as unknown as FeatureCollection
                );
            }
        }
    }

    if (geoJson.type === "Feature") {
        extractFromFeature(geoJson as Feature);
    } else if (geoJson.type === "FeatureCollection") {
        extractFromFeatureCollection(geoJson as FeatureCollection);
    } else {
        extractFromGeometry(geoJson as GeometryObject);
    }

    return { lineStrings, times };
}

export function geoJsonObjectToPositionsAndTimes(
    geoJson: GeoJsonObject
): { positions: Position[]; times: Date[] } {
    const { lineStrings, times } = extractLineStrings(geoJson);
    const positionsGroups: Position[][] = [];

    for (let i = 0; i < lineStrings.length; i += 1) {
        const feature = lineStrings[i];
        if (feature.type === "LineString") {
            positionsGroups.push(feature.coordinates);
        } else if (feature.type === "MultiLineString") {
            positionsGroups.push(feature.coordinates.flat());
        }
    }

    return { positions: positionsGroups.flat(), times };
}
