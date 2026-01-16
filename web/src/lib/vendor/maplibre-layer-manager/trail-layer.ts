import type { MapMouseEvent, Marker, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";

export class TrailLayer implements BaseLayer {

    spec: StyleSpecification;
    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }>
    markers: Record<string, Marker> = {};

    constructor(id: string, geojson: GeoJSON.FeatureCollection, color: string, listerners?: { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }) {
        this.spec = {
            version: 8,
            name: id,
            sources: {
                [id]: {
                    type: "geojson",
                    data: geojson,
                }
            },
            layers: [{
                id: id,
                type: "line",
                source: id,
                paint: {
                    "line-color": color,
                    "line-width": 5,
                },
            }]

        };

        this.listeners = { [id]: listerners ?? {} }
    }
}