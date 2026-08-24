import type { FilterSpecification, MapMouseEvent, Marker, StyleSpecification } from "maplibre-gl";
import * as M from "maplibre-gl";
import type { BaseLayer } from "./layers";

export class TrailLayer implements BaseLayer {

    spec: StyleSpecification;
    listeners: Record<string, { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; }>
    markers: Record<string, Marker> = {};

    constructor(id: string, geojson: GeoJSON.FeatureCollection, color: string, options?: {
        listeners?: { onMouseUp?: (e: MapMouseEvent) => void; onMouseDown?: (e: MapMouseEvent) => void; onEnter?: (e: MapMouseEvent) => void; onLeave?: (e: MapMouseEvent) => void; onMouseMove?: (e: MapMouseEvent) => void; };
        lineWidth?: number;
        lineOpacity?: number;
    }) {
        const layer: M.LineLayerSpecification = {
            id: id,
            type: "line",
            source: id,
            paint: {
                "line-color": color,
                "line-width": options?.lineWidth ?? 5,
                "line-opacity": options?.lineOpacity ?? 1,
            },
        };
        
        this.spec = {
            version: 8,
            name: id,
            sources: {
                [id]: {
                    type: "geojson",
                    data: geojson,
                }
            },
            layers: [layer]

        };

        this.listeners = { [id]: options?.listeners ?? {} }
    }
}
