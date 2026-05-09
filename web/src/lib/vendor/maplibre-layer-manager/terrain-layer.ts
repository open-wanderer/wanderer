import type { StyleSpecification } from "maplibre-gl";
import type { BaseLayer } from "./layers";

export class TerrainLayer implements BaseLayer {
    spec: StyleSpecification;

    constructor(terrainURL: string, hillshadingURL?: string) {
        const hillshadingEnabled = Boolean(hillshadingURL);

        this.spec = {
            version: 8,
            name: "terrain",
            sources: {
                terrain: {
                    type: "raster-dem",
                    url: terrainURL,
                },
                ...(hillshadingEnabled ? {
                    hillshading: {
                        type: "raster-dem",
                        url: hillshadingURL
                    }
                } : {})

            },
            layers: hillshadingEnabled ? [{
                id: "hillshading",
                source: "terrain",
                type: "hillshade",
                layout: {
                    visibility: "none",
                },
            }] : []

        }

    }
}
