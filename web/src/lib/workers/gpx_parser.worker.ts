import './worker_polyfill';
import GPX from '$lib/models/gpx/gpx';

export type GpxWorkerRequest = {
    id: string;
    gpxData: string;
    includeRoute?: boolean;
    includeWaypoints?: boolean;
};

export type GpxWorkerResponse = {
    id: string;
    geojson?: GeoJSON.FeatureCollection;
    error?: string;
};

self.onmessage = (event: MessageEvent<GpxWorkerRequest>) => {
    const { id, gpxData, includeRoute, includeWaypoints } = event.data;
    try {
        const gpx = GPX.parse(gpxData);
        const geojson = gpx.toGeoJSON(includeRoute, includeWaypoints);
        self.postMessage({ id, geojson } as GpxWorkerResponse);
    } catch (err: any) {
        self.postMessage({ id, error: err?.message || 'Failed to parse GPX' } as GpxWorkerResponse);
    }
};
