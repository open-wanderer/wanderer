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
    };

    constructor(map: M.Map, geojson: GeoJSON.FeatureCollection, options?: { listeners?: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }> }) {

        this.map = map;
        const listeners = options?.listeners;
        this.listeners = {
            "preview": { ...this.listeners["preview"], ...listeners?.["preview"] },
        }

        this.spec = {
            version: 8,
            name: "preview",
            glyphs: "https://tiles.openfreemap.org/fonts/{fontstack}/{range}.pbf",
            sources: {
                "preview": {
                    type: "geojson",
                    data: geojson,
                },
            },
            layers: [
                {
                    id: "preview",
                    type: "line",
                    source: "preview",
                    paint: {
                        "line-color": ["get", "color"],
                        "line-width": 5,
                    },
                },
                {
                    id: "preview-direction-carets",
                    type: "symbol",
                    source: "preview",
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