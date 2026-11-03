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
                    filter: ["all", ["!=", ["get", "is_large"], true], [">", ["get", "point_count"], 1]],
                    paint: {
                        "circle-color": "#242734",
                        "circle-radius": [
                            "step",
                            ["get", "point_count"],
                            10,
                            5,
                            12,
                            10,
                            15,
                            50,
                            18,
                            100,
                            22,
                            500,
                            25,
                        ],
                        "circle-stroke-width": 2,
                        "circle-stroke-color": "#fff",
                    },
                },
                {
                    // Category marker images are registered by the map before it adds this layer.
                    id: "unclustered-point",
                    type: "symbol",
                    source: "cluster-trails",
                    filter: ["all", ["!=", ["get", "is_large"], true], ["==", ["get", "point_count"], 1]],
                    layout: {
                        "icon-image": ["get", "marker_icon"],
                        "icon-allow-overlap": true,
                        "icon-ignore-placement": true,
                    },
                },
                {
                    id: "cluster-count",
                    type: "symbol",
                    source: "cluster-trails",
                    filter: ["all", ["!=", ["get", "is_large"], true], [">", ["get", "point_count"], 1]],
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
        const feature = features[0];
        if (!feature) {
            return;
        }

        const currentZoom = this.map.getZoom();
        this.map.flyTo({
            center: (feature.geometry as any).coordinates,
            zoom: currentZoom + 2,
            maxDuration: 3000
        });
    }

    private zoomOnUnclusteredPoint(e: MapMouseEvent) {
        // Without hover, a tap opens the trail's popup instead.
        if (!window.matchMedia("(hover: hover)").matches) {
            return;
        }
        const feature = (e as any).features?.[0];
        if (!feature) {
            return;
        }

        const coordinates = feature.geometry.coordinates.slice();

        this.map.flyTo({
            center: coordinates,
            zoom: 12,
            maxDuration: 3000
        });
    }
}
