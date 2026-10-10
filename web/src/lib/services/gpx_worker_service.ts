import type { FeatureCollection } from 'geojson';
import type { GpxWorkerRequest, GpxWorkerResponse } from '$lib/workers/gpx_parser.worker';

export interface ParseGpxOptions {
    includeRoute?: boolean;
    includeWaypoints?: boolean;
}

export class GpxWorkerService {
    private static readonly MAX_CACHE_SIZE = 50;

    private worker: Worker | null = null;
    private workerFailed = false;
    private pending = new Map<string, { resolve: (fc: FeatureCollection) => void; reject: (err: Error) => void }>();
    private inFlight = new Map<string, Promise<FeatureCollection>>();
    private cache = new Map<string, FeatureCollection>();

    private hasWorkerEnvironment(): boolean {
        return typeof window !== 'undefined' && typeof Worker !== 'undefined';
    }

    private getWorker(): Worker | null {
        if (this.workerFailed || !this.hasWorkerEnvironment()) {
            return null;
        }

        if (!this.worker) {
            try {
                this.worker = new Worker(
                    new URL('../workers/gpx_parser.worker.ts', import.meta.url),
                    { type: 'module' }
                );

                this.worker.onmessage = (event: MessageEvent<GpxWorkerResponse>) => {
                    const { id, geojson, error } = event.data;
                    const pendingRequest = this.pending.get(id);
                    if (!pendingRequest) {
                        return;
                    }
                    this.pending.delete(id);

                    if (error) {
                        pendingRequest.reject(new Error(error));
                    } else if (geojson) {
                        pendingRequest.resolve(geojson);
                    } else {
                        pendingRequest.reject(new Error('Empty worker response'));
                    }
                };

                this.worker.onerror = (err) => {
                    console.error('GPX Worker error:', err);
                    this.handleWorkerFailure(new Error('GPX Worker script error or failed to load'));
                };

                this.worker.onmessageerror = (err) => {
                    console.error('GPX Worker message serialization error:', err);
                    this.handleWorkerFailure(new Error('GPX Worker message serialization failed'));
                };
            } catch (err) {
                console.warn('Failed to initialize GPX Worker, using fallback', err);
                this.workerFailed = true;
                return null;
            }
        }

        return this.worker;
    }

    private handleWorkerFailure(error: Error): void {
        this.workerFailed = true;
        if (this.worker) {
            try {
                this.worker.terminate();
            } catch {
                // ignore
            }
            this.worker = null;
        }

        const currentPending = Array.from(this.pending.values());
        this.pending.clear();
        for (const req of currentPending) {
            req.reject(error);
        }
    }

    private parseViaWorker(key: string, gpxData: string, options?: ParseGpxOptions): Promise<FeatureCollection> {
        const worker = this.getWorker();
        if (!worker) {
            return Promise.reject(new Error('Worker unavailable'));
        }

        return new Promise<FeatureCollection>((resolve, reject) => {
            this.pending.set(key, { resolve, reject });
            worker.postMessage({
                id: key,
                gpxData,
                includeRoute: options?.includeRoute,
                includeWaypoints: options?.includeWaypoints
            } as GpxWorkerRequest);
        });
    }

    private async parseFallback(gpxData: string, options?: ParseGpxOptions): Promise<FeatureCollection> {
        const { default: GPX } = await import('$lib/models/gpx/gpx');
        const gpx = GPX.parse(gpxData);
        return gpx.toGeoJSON(options?.includeRoute, options?.includeWaypoints);
    }

    public parseGpxToGeoJSON(
        id: string,
        gpxData: string,
        options?: ParseGpxOptions
    ): Promise<FeatureCollection> {
        const cached = this.getCached(id);
        if (cached) {
            return Promise.resolve(cached);
        }

        const existing = this.inFlight.get(id);
        if (existing) {
            return existing;
        }

        const promise = (
            this.workerFailed || !this.hasWorkerEnvironment()
                ? this.parseFallback(gpxData, options)
                : this.parseViaWorker(id, gpxData, options).catch((err) => {
                      console.warn('GPX Worker error, falling back to main thread', err);
                      return this.parseFallback(gpxData, options);
                  })
        )
            .then((geojson) => {
                this.setCache(id, geojson);
                return geojson;
            })
            .finally(() => {
                this.inFlight.delete(id);
            });

        this.inFlight.set(id, promise);
        return promise;
    }

    public getCached(id: string): FeatureCollection | undefined {
        const val = this.cache.get(id);
        if (val !== undefined) {
            this.cache.delete(id);
            this.cache.set(id, val);
        }
        return val;
    }

    public setCache(id: string, geojson: FeatureCollection): void {
        this.cache.delete(id);
        this.cache.set(id, geojson);
        if (this.cache.size > GpxWorkerService.MAX_CACHE_SIZE) {
            const oldestKey = this.cache.keys().next().value;
            if (oldestKey !== undefined) {
                this.cache.delete(oldestKey);
            }
        }
    }

    public clearCache(id?: string): void {
        if (id) {
            this.cache.delete(id);
        } else {
            this.cache.clear();
        }
    }
}

export const gpxWorkerService = new GpxWorkerService();
