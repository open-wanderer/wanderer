import type { SourceSpecification, StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";

export class CaretLayer implements BaseLayer {

    spec: StyleSpecification;

    constructor(source: SourceSpecification) {
        this.spec = {
            version: 8,
            name: "direction-carets",
            sources: {
                caret: source
            },
            layers: [{
                id: "direction-carets",
                type: "symbol",
                source: "caret",
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
            }]

        };
    }
}