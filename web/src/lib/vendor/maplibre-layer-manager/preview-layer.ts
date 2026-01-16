import type { MapMouseEvent, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";
import * as M from "maplibre-gl";

export class PreviewLayer implements BaseLayer {

    private map: M.Map;

    spec: StyleSpecification;
    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }> = {
        "preview": {
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        },
        "preview-start-points": {
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        }
    };

    constructor(map: M.Map, geojson: GeoJSON.FeatureCollection, listeners?: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }>) {

        this.map = map;
        this.listeners = {
            "preview": { ...this.listeners["preview"], ...listeners?.["preview"] },
            "preview-start-points": { ...this.listeners["preview-start-points"], ...listeners?.["preview-start-points"] }
        }

        const startPoints: GeoJSON.FeatureCollection = {
            type: "FeatureCollection",
            features: geojson.features.map((f, i) => ({
                type: "Feature",
                properties: {
                    ...f.properties,
                    id: i
                },
                geometry: {
                    type: "Point",
                    coordinates: (f.geometry as any).coordinates[0]
                }
            }))
        };
        this.spec = {
            version: 8,
            name: "preview",
            sources: {
                "preview": {
                    type: "geojson",
                    data: geojson,
                },
                "preview-start-points": {
                    type: "geojson",
                    data: startPoints,
                }
            },
            layers: [
                {
                    id: "preview",
                    type: "line",
                    source: "preview",
                    minzoom: 10,
                    paint: {
                        "line-color": ["get", "color"],
                        "line-width": 5,
                    },
                },
                {
                    id: "preview-start-points",
                    type: "circle",
                    source: "preview-start-points",
                    minzoom: 10,
                    paint: {
                        "circle-color": "#242734",
                        "circle-radius": 6,
                        "circle-stroke-width": 2,
                        "circle-stroke-color": "#fff",
                    },
                },
                {
                    id: "preview-direction-carets",
                    type: "symbol",
                    source: "preview",
                    minzoom: 10,
                    layout: {
                        "symbol-placement": "line",
                        "symbol-spacing": [
                            "interpolate",
                            ["exponential", 1.5],
                            ["zoom"],
                            0,
                            80,
                            18,
                            200,
                        ],
                        "icon-image": "direction-caret",
                        "icon-size": [
                            "interpolate",
                            ["exponential", 1.5],
                            ["zoom"],
                            0,
                            0.5,
                            18,
                            0.8,
                        ],
                    },
                }
            ]

        };
    }
}