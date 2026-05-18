import type { MapMouseEvent, SourceSpecification, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";
import * as M from "maplibre-gl";

export class ClusterLayer implements BaseLayer {

    spec: StyleSpecification;

    private map: M.Map;


    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }> = {
        "clusters": {
            onMouseUp: this.zoomOnCluster.bind(this),
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        },
        "unclustered-point": {
            onMouseUp: this.zoomOnUnclusteredPoint.bind(this),
            onEnter: () => this.map!.getCanvas().style.cursor = "pointer",
            onLeave: () => this.map!.getCanvas().style.cursor = ""
        }
    };

    constructor(map: M.Map, geojson: GeoJSON.FeatureCollection, listeners?: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }>) {
        this.map = map;
        this.listeners = {
            "clusters": { ...this.listeners["clusters"], ...listeners?.["clusters"] },
            "unclustered-point": { ...this.listeners["unclustered-point"], ...listeners?.["unclustered-point"] }
        }

        this.spec = {
            version: 8,
            name: "clusters",
            glyphs: "https://tiles.openfreemap.org/fonts/{fontstack}/{range}.pbf",
            sources: {
                "cluster-trails": {
                    type: "geojson",
                    data: geojson,
                }
            },
            layers: [
                {
                    id: "clusters",
                    type: "circle",
                    source: "cluster-trails",
                    paint: {
                        "circle-color": "#242734",
                        "circle-radius": [
                            "step",
                            ["get", "point_count"],
                            12,
                            5,
                            15,
                            10,
                            18,
                            50,
                            22,
                            100,
                            25,
                            500,
                            30,
                        ],
                        "circle-stroke-width": 2,
                        "circle-stroke-color": "#fff",
                    },
                },
                {
                    id: "cluster-count",
                    type: "symbol",
                    source: "cluster-trails",
                    layout: {
                        "text-field": ["get", "point_count_abbreviated"],
                        "text-font": ["Noto Sans Regular"],
                        "text-size": 11,
                        "text-allow-overlap": true,
                        "text-ignore-placement": true,
                    },
                    paint: {
                        "text-color": "#fff",
                    },
                }
            ]

        };
    }

    private async zoomOnCluster(e: MapMouseEvent) {
        const features = this.map.queryRenderedFeatures(e.point, {
            layers: ["clusters"],
        });
        const currentZoom = this.map.getZoom();
        this.map.flyTo({
            center: (features[0].geometry as any).coordinates,
            zoom: currentZoom + 2,
            maxDuration: 3000
        });
    }

    private zoomOnUnclusteredPoint(e: MapMouseEvent) {
        const coordinates = (e as any).features[0].geometry.coordinates.slice();

        this.map.flyTo({
            center: coordinates,
            zoom: 12,
            maxDuration: 3000
        });
    }
}