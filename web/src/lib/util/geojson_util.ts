import type { BBox, Feature, FeatureCollection, GeoJSON, GeoJsonObject, GeometryCollection, Position } from "geojson";

export function bbox(
    geojson: GeoJSON,
): BBox {
    if (geojson.bbox != null) {
        return geojson.bbox;
    }
    const result: BBox = [Infinity, Infinity, -Infinity, -Infinity];
    let startPoint: Position, endPoint: Position;
    coordEach(geojson, (coord) => {
        startPoint ??= [coord[0], coord[1]];
        endPoint = coord;
        if (result[0] > coord[0]) {
            result[0] = coord[0];
        }
        if (result[1] > coord[1]) {
            result[1] = coord[1];
        }
        if (result[2] < coord[0]) {
            result[2] = coord[0];
        }
        if (result[3] < coord[1]) {
            result[3] = coord[1];
        }
    });

    return result;
}

export function findStartAndEndPoints(geojson: GeoJsonObject): Position[] {
    const startEndPoints: Position[] = [];

    // Check if it's a FeatureCollection
    if ((geojson as any).features) {
        (geojson as any).features.forEach((feature: any) => {
            const geometry = feature.geometry;
            extractStartAndEndPointsFromGeometry(geometry, startEndPoints);
        });
    } else if ((geojson as any).geometry) {
        // Single Feature
        const geometry = (geojson as any).geometry;
        extractStartAndEndPointsFromGeometry(geometry, startEndPoints);
    } else {
        console.warn(
            "Unsupported GeoJSON type. Expected FeatureCollection or Feature."
        );
    }

    return startEndPoints;
}

export function splitMultiLineStringToLineStrings(geojson: GeoJsonObject): FeatureCollection {
    const features: Feature[] = [];
    (geojson as any).features.forEach((feature: any) => {
        if (feature.geometry.type === "MultiLineString") {

            feature.geometry.coordinates.forEach((lineString: any, lineIndex: number) => {
                features.push({
                    type: "Feature",
                    geometry: {
                        type: "LineString",
                        coordinates: lineString,
                    },
                    properties: {
                        ...feature.properties,
                        coordinateProperties: {
                            ...feature.properties.coordinateProperties,
                            times: feature.properties.coordinateProperties?.times?.[lineIndex]
                        },
                        featureId: features.length,
                        segmentId: lineIndex,
                    },
                });
            });
        } else if (feature.geometry.type === "LineString") {
            features.push({
                ...feature,
                properties: {
                    ...feature.properties,
                    featureId: features.length,
                    segmentId: 0
                },
            });
        }
    });

    return {
        type: "FeatureCollection",
        features: features,
    };
}

function extractStartAndEndPointsFromGeometry(geometry: any, startEndPoints: Position[]) {
    if (geometry.type === "LineString") {
        const coords = geometry.coordinates as number[][];
        const start: [number, number] = [coords[0][0], coords[0][1]]; // First point
        const end: [number, number] = [
            coords[coords.length - 1][0],
            coords[coords.length - 1][1],
        ]; // Last point
        startEndPoints.push(start, end);
    } else if (geometry.type === "MultiLineString") {
        const coords = geometry.coordinates as number[][][];
        const start: [number, number] = [
            coords[0][0][0],
            coords[0][0][1],
        ]; // First point of the first line
        const lastLine = coords[coords.length - 1];
        const end: [number, number] = [
            lastLine[lastLine.length - 1][0],
            lastLine[lastLine.length - 1][1],
        ]; // Last point of the last line
        startEndPoints.push(start, end);
    } else if (geometry.type === "Point") {
        const coords = geometry.coordinates as number[];
        startEndPoints.push(coords as [number, number], coords as [number, number]);
    } else if (geometry.type === "MultiPoint") {
        const coords = geometry.coordinates as number[][];
        const start: [number, number] = [coords[0][0], coords[0][1]]; // First point
        const end: [number, number] = [
            coords[coords.length - 1][0],
            coords[coords.length - 1][1],
        ]; // Last point
        startEndPoints.push(start, end);
    } else if (geometry.type === "Polygon") {
        const coords = geometry.coordinates as number[][][];
        const start: [number, number] = [
            coords[0][0][0],
            coords[0][0][1],
        ]; // First point of the first ring
        const lastRing = coords[coords.length - 1];
        const end: [number, number] = [
            lastRing[lastRing.length - 1][0],
            lastRing[lastRing.length - 1][1],
        ]; // Last point of the last ring
        startEndPoints.push(start, end);
    } else if (geometry.type === "MultiPolygon") {
        const coords = geometry.coordinates as number[][][][];
        const firstPolygon = coords[0];
        const start: [number, number] = [
            firstPolygon[0][0][0],
            firstPolygon[0][0][1],
        ]; // First point of the first ring of the first polygon
        const lastPolygon = coords[coords.length - 1];
        const lastRing = lastPolygon[lastPolygon.length - 1];
        const end: [number, number] = [
            lastRing[lastRing.length - 1][0],
            lastRing[lastRing.length - 1][1],
        ]; // Last point of the last ring of the last polygon
        startEndPoints.push(start, end);
    } else {
        console.warn(
            `Geometry type ${geometry.type} is not supported for start/end point extraction.`
        );
    }
}

function coordEach(geojson: GeoJSON, callback: (
    currentCoord: number[],
    coordIndex: number,
    featureIndex: number,
    multiFeatureIndex: number,
    geometryIndex: number
) => void | false,
    excludeWrapCoord?: boolean,) {
    // Handles null Geometry -- Skips this GeoJSON
    if (geojson === null) return;
    var j,
        k,
        l,
        geometry,
        stopG,
        coords,
        geometryMaybeCollection,
        wrapShrink = 0,
        coordIndex = 0,
        isGeometryCollection,
        type = geojson.type,
        isFeatureCollection = type === "FeatureCollection",
        isFeature = type === "Feature",
        stop = isFeatureCollection ? (geojson as FeatureCollection).features.length : 1;

    for (var featureIndex = 0; featureIndex < stop; featureIndex++) {
        geometryMaybeCollection = isFeatureCollection
            ? (geojson as FeatureCollection).features[featureIndex].geometry
            : isFeature
                ? (geojson as Feature).geometry
                : geojson;
        isGeometryCollection = geometryMaybeCollection
            ? geometryMaybeCollection.type === "GeometryCollection"
            : false;
        stopG = isGeometryCollection
            ? (geometryMaybeCollection as GeometryCollection).geometries.length
            : 1;

        for (var geomIndex = 0; geomIndex < stopG; geomIndex++) {
            var multiFeatureIndex = 0;
            var geometryIndex = 0;
            geometry = isGeometryCollection
                ? (geometryMaybeCollection as GeometryCollection).geometries[geomIndex]
                : geometryMaybeCollection;

            // Handles null Geometry -- Skips this geometry
            if (geometry === null) continue;
            coords = (geometry as any).coordinates;
            var geomType = geometry.type;

            wrapShrink =
                excludeWrapCoord &&
                    (geomType === "Polygon" || geomType === "MultiPolygon")
                    ? 1
                    : 0;

            switch (geomType) {
                case null:
                    break;
                case "Point":
                    if (
                        callback(
                            coords,
                            coordIndex,
                            featureIndex,
                            multiFeatureIndex,
                            geometryIndex
                        ) === false
                    )
                        return false;
                    coordIndex++;
                    multiFeatureIndex++;
                    break;
                case "LineString":
                case "MultiPoint":
                    for (j = 0; j < coords.length; j++) {
                        if (
                            callback(
                                coords[j],
                                coordIndex,
                                featureIndex,
                                multiFeatureIndex,
                                geometryIndex
                            ) === false
                        )
                            return false;
                        coordIndex++;
                        if (geomType === "MultiPoint") multiFeatureIndex++;
                    }
                    if (geomType === "LineString") multiFeatureIndex++;
                    break;
                case "Polygon":
                case "MultiLineString":
                    for (j = 0; j < coords.length; j++) {
                        for (k = 0; k < coords[j].length - wrapShrink; k++) {
                            if (
                                callback(
                                    coords[j][k],
                                    coordIndex,
                                    featureIndex,
                                    multiFeatureIndex,
                                    geometryIndex
                                ) === false
                            )
                                return false;
                            coordIndex++;
                        }
                        if (geomType === "MultiLineString") multiFeatureIndex++;
                        if (geomType === "Polygon") geometryIndex++;
                    }
                    if (geomType === "Polygon") multiFeatureIndex++;
                    break;
                case "MultiPolygon":
                    for (j = 0; j < coords.length; j++) {
                        geometryIndex = 0;
                        for (k = 0; k < coords[j].length; k++) {
                            for (l = 0; l < coords[j][k].length - wrapShrink; l++) {
                                if (
                                    callback(
                                        coords[j][k][l],
                                        coordIndex,
                                        featureIndex,
                                        multiFeatureIndex,
                                        geometryIndex
                                    ) === false
                                )
                                    return false;
                                coordIndex++;
                            }
                            geometryIndex++;
                        }
                        multiFeatureIndex++;
                    }
                    break;
                case "GeometryCollection":
                    for (j = 0; j < (geometry as GeometryCollection).geometries.length; j++)
                        if (
                            coordEach((geometry as GeometryCollection).geometries[j], callback, excludeWrapCoord) ===
                            false
                        )
                            return false;
                    break;
                default:
                    throw new Error("Unknown Geometry Type");
            }
        }
    }
}