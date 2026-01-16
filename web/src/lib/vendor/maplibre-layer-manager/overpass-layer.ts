/**
 * Component adapted from: https://github.com/gpxstudio/gpx.studio/blob/main/website/src/lib/components/layer-control/OverpassLayer.ts
 * Original author: @vcoppe
 * License: MIT
 */

import { createOverpassPopup } from "$lib/util/maplibre_util";
import * as M from "maplibre-gl";
import { type LngLatBounds, type MapMouseEvent, type StyleSpecification } from "maplibre-gl";
import { pois, type BaseLayer, type MapState } from "./layers";
import type { OverpassResponse } from "./types";
import { env } from '$env/dynamic/public'

export class OverpassLayer implements BaseLayer {
    private overpassApiURL: string = (env.PUBLIC_OVERPASS_API_URL && env.PUBLIC_OVERPASS_API_URL.length > 0 ? env.PUBLIC_OVERPASS_API_URL : "https://overpass-api.de") + "/api/interpreter";

    data: GeoJSON.FeatureCollection = ({ type: 'FeatureCollection', features: [] });

    private minZoom = 12;

    private cachedQueries: { x: string, y: string, query: string }[] = [];
    private cachedData: { query: string, id: number, feature: GeoJSON.Feature }[] = []
    private tileSize = 0.1;


    spec: StyleSpecification = {
        version: 8,
        name: "overpass",
        sources: {
            overpass: {
                type: 'geojson',
                data: this.data,
            }
        },
        layers: [
            {
                id: 'overpass',
                type: 'symbol',
                source: 'overpass',
                layout: {
                    'icon-image': ['get', 'icon'],
                    'icon-size': 0.25,
                    'icon-padding': 0,
                    'icon-allow-overlap': ['step', ['zoom'], false, 14, true],
                },
            }
        ],
    };

    listeners = {
        "overpass": {
            onMouseDown: (e: MapMouseEvent) => {
                this.openPopup(e)
            },
            onEnter: (e: MapMouseEvent) => {
                this.openPopup(e);
            },
        }
    }

    private popup: M.Popup;
    private map: M.Map;
    private currentPopupCoordinates: GeoJSON.Position | null = null

    constructor(map: M.Map) {
        this.map = map;
        this.popup = new M.Popup()
            .setMaxWidth("420px")
    }

    private openPopup(e: MapMouseEvent) {
        const features = (e as any).features as GeoJSON.Feature[];
        const point = features[0].geometry as GeoJSON.Point;
        const content = createOverpassPopup(features[0], point.coordinates);

        this.currentPopupCoordinates = point.coordinates;
        this.popup
            .setLngLat(point.coordinates as M.LngLatLike)
            .setDOMContent(content)
            .addTo(this.map);

        this.map.on("mousemove", this.distanceNotifierBinded)
    }

    distanceNotifierBinded = this.distanceNotifier.bind(this);
    private distanceNotifier(e: MapMouseEvent) {
        if (this.currentPopupCoordinates === null) {
            return
        }

        if (this.map.project(this.currentPopupCoordinates as M.LngLatLike).dist(this.map.project(e.lngLat)) > 60) {
            this.popup.remove();
            this.map.off("mousemove", this.distanceNotifier)
            this.currentPopupCoordinates = null;
        }
    }

    async updateLayerIfNeeded(state: MapState, bounds: LngLatBounds) {

        let activeQueries: string[] = this.getActiveQueries(state);

        if (this.map.getZoom() >= this.minZoom) {

            await this.fetchMissingTilesForBounds(bounds, activeQueries);
        }


        const filter: M.FilterSpecification = ['in', 'query', ...this.getActiveQueries(state)];

        this.map.setFilter('overpass', filter);

        return filter
    }

    private async fetchMissingTilesForBounds(bounds: M.LngLatBounds, activeQueries: string[]) {
        const result: [string, LngLatBounds][] = [];

        const south = bounds.getSouth();
        const north = bounds.getNorth();
        const west = bounds.getWest();
        const east = bounds.getEast();

        for (let lat = south; lat < north; lat += this.tileSize) {
            for (let lng = west; lng < east; lng += this.tileSize) {
                const x = Math.floor(lat / this.tileSize) * this.tileSize;
                const y = Math.floor(lng / this.tileSize) * this.tileSize;
                const cachedQueriesAtPosition = this.cachedQueries.filter(q => q.x == x.toFixed(4) && q.y == y.toFixed(4)) ?? [];
                const missingQueries = activeQueries.filter(
                    (query) =>
                        !cachedQueriesAtPosition.some(
                            (querytile) =>
                                querytile.query === query
                        )
                );
                if (missingQueries.length > 0) {
                    const tileBounds = this.getBoundsForTile(x, y)
                    await this.fetchTile(x, y, missingQueries, tileBounds)
                }

            }
        }

        return result;
    }

    private async fetchTile(x: number, y: number, activeQueries: string[], bounds: LngLatBounds) {
        const q = this.getOverpassQuery(activeQueries, bounds)
        if (!q.length) {
            return;
        }
        const r = await fetch(`${this.overpassApiURL}?data=${q}`)
        const response: OverpassResponse = await r.json();

        this.cacheData(x, y, response, activeQueries)
        this.loadIcons(activeQueries)

    }

    private cacheData(x: number, y: number, data: OverpassResponse, activeQueries: string[]) {
        if (data.elements === undefined) {
            return;
        }

        this.cachedQueries = this.cachedQueries.concat(activeQueries.map((query) => ({ x: x.toFixed(4), y: y.toFixed(4), query })));

        for (let element of data.elements) {
            for (let query of activeQueries) {
                if (this.belongsToQuery(element, query)) {
                    this.cachedData.push({
                        query,
                        id: element.id,
                        feature: {
                            type: 'Feature',
                            geometry: {
                                type: 'Point',
                                coordinates: element.center
                                    ? [element.center.lon, element.center.lat]
                                    : [element.lon, element.lat],
                            },
                            properties: {
                                id: element.id,
                                lat: element.center ? element.center.lat : element.lat,
                                lon: element.center ? element.center.lon : element.lon,
                                query: query,
                                icon: `overpass-${query}`,
                                tags: element.tags,
                                type: element.type,
                            },
                        },
                    });
                }
            }
        }
        this.data.features = this.cachedData.map(d => d.feature);
    }

    private getActiveQueries(state: MapState) {
        const activeQueries: string[] = []
        for (const category of Object.keys(state.pois)) {
            for (const [name, active] of Object.entries(state.pois[category])) {
                if (active) {
                    activeQueries.push(name)
                }
            }
        }

        return activeQueries
    }

    private belongsToQuery(element: OverpassResponse["elements"][number], query: string) {
        if (Array.isArray(pois[query].tags)) {
            return pois[query].tags.some((tags) => this.belongsToQueryItem(element, tags));
        } else {
            return this.belongsToQueryItem(element, pois[query].tags);
        }
    }

    private belongsToQueryItem(element: any, tags: Record<string, string | boolean | string[]>) {
        return Object.entries(tags).every(([tag, value]) =>
            Array.isArray(value) ? value.includes(element.tags[tag]) : element.tags[tag] === value
        );
    }

    private getOverpassQuery(activeQueries: string[], bounds: LngLatBounds) {
        return `[bbox:${bounds.getSouth()},${bounds.getWest()},${bounds.getNorth()},${bounds.getEast()}][out:json];(${this.getQueries(activeQueries)});out center;`;

    }

    private getQueries(queries: string[]) {
        return queries.map((query) => this.getQuery(query)).join('');
    }

    private getQuery(query: string) {
        if (Array.isArray(pois[query].tags)) {
            return pois[query].tags.map((tags) => this.getQueryItem(tags)).join('');
        } else {
            return this.getQueryItem(pois[query].tags);
        }
    }

    private getQueryItem(tags: Record<string, string | boolean | string[]>) {
        let arrayEntry = Object.entries(tags).find(([_, value]) => Array.isArray(value));
        if (arrayEntry !== undefined) {
            return (arrayEntry[1] as string[])
                .map(
                    (val) =>
                        `nwr${Object.entries(tags)
                            .map(([tag, value]) => `[${tag}=${tag === arrayEntry[0] ? val : value}]`)
                            .join('')};`
                )
                .join('');
        } else {
            return `nwr${Object.entries(tags)
                .map(([tag, value]) => `[${tag}=${value}]`)
                .join('')};`;
        }
    }

    private loadIcons(activeQueries: string[]) {
        activeQueries.forEach((q) => {
            if (!this.map.hasImage(`overpass-${q}`)) {
                let icon = new Image(100, 100);
                icon.onload = () => {
                    if (!this.map.hasImage(`overpass-${q}`)) {
                        this.map.addImage(`overpass-${q}`, icon);
                    }
                };

                const svg = `
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40">
                    <circle cx="20" cy="20" r="20" fill="${pois[q].icon.bg}" />
                    <g transform="translate(8 8)">
                    ${transformFontAwesomeIcon(pois[q].icon.svg)}
                    </g>
                </svg>
                `

                icon.src =
                    'data:image/svg+xml,' +
                    encodeURIComponent(svg);
            }
        });
    }

    private getBoundsForTile(lat: number, lng: number): LngLatBounds {
        return new M.LngLatBounds(
            [lng, lat],
            [lng + this.tileSize, lat + this.tileSize]
        );
    }

    private showDebugTiles(tiles: [string, LngLatBounds][]) {
        const features = tiles.map(([_, bounds]) => boundsToPolygonFeature(bounds));

        const debugSource = this.map.getSource('debug') as M.GeoJSONSource;

        if (debugSource) {
            debugSource.setData({
                type: 'FeatureCollection',
                features,
            });
        }
    }

}


function boundsToPolygonFeature(bounds: M.LngLatBounds): GeoJSON.Feature<GeoJSON.Polygon> {
    const [[west, south], [east, north]] = bounds.toArray();
    const coordinates = [[
        [west, south],
        [east, south],
        [east, north],
        [west, north],
        [west, south],
    ]];

    return {
        type: 'Feature',
        geometry: {
            type: 'Polygon',
            coordinates: coordinates,
        },
        properties: {},
    };
}

function transformFontAwesomeIcon(svgText: string): string {
    const viewBoxMatch = svgText.match(/viewBox="0 0 (\d+) (\d+)"/);
    if (!viewBoxMatch) throw new Error("SVG viewBox not found");

    const originalWidth = parseFloat(viewBoxMatch[1]);
    const originalHeight = parseFloat(viewBoxMatch[2]);

    const maxDim = Math.max(originalWidth, originalHeight);
    const scale = 24 / maxDim;

    const scaledWidth = originalWidth * scale;
    const scaledHeight = originalHeight * scale;

    const dx = (24 - scaledWidth) / 2;
    const dy = (24 - scaledHeight) / 2;

    let innerContent = svgText
        .replace(/<svg[^>]*>/, '')
        .replace(/<\/svg>/, '')
        .trim();

    innerContent = innerContent.replace(
        /<path/g,
        '<path fill="white"'
    );

    return `
    <g transform="translate(${dx.toFixed(2)} ${dy.toFixed(2)}) scale(${scale.toFixed(6)})">
      ${innerContent}
    </g>
  `;
}

