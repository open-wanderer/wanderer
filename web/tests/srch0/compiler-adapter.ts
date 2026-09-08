import { vi } from 'vitest';
import type { LngLat } from 'maplibre-gl';
import type { TrailSearchResult } from '$lib/models/trail';
import { searchResultToTrailList, trails_search_filter, trails_search_bounding_box } from '$lib/stores/trail_store';
import { searchTrails, searchMulti } from '$lib/stores/search_store';
import { lists_search_filter } from '$lib/stores/list_store';
import { loadFilter } from './state-adapter';
import type { CompilerInput, Context, Dataset, DTOInput, JsonRecord, RecordedRequest } from './types';

export const emptySearch = { hits: [], page: 1, hitsPerPage: 25, totalPages: 0, totalHits: 0 };

export async function observeDTO(input: DTOInput, source: Dataset) {
    const hits = input.hits ?? input.ids.map(id => {
        const sourceHit = source.trails.find(trail => trail.id === id);
        if (!sourceHit) throw new Error(`Unbekannte synthetische Trail-ID: ${id}`);
        const hit: JsonRecord = { ...sourceHit, ...input.hit_overrides?.[id] };
        for (const field of input.omit_fields ?? []) delete hit[field];
        // Diese Fälle prüfen ausdrücklich unvollständige und historische DTOs.
        return hit as unknown as TrailSearchResult;
    });
    const trails = await searchResultToTrailList(hits);
    return { dto: input.fields ? trails.map(trail => Object.fromEntries(input.fields!.map(field => [field, trail[field]]))) : trails };
}

export async function observeCompiler(context: Context, input: CompilerInput, source: Dataset) {
    const requests: RecordedRequest[] = [];
    const fetcher: typeof fetch = async (url, init) => {
        requests.push({ url: String(url), body: JSON.parse(String(init?.body)) });
        if (String(url).endsWith('/cluster')) return Response.json(input.adapter === 'map-search' && input.cluster_response || { features: [], totalHits: 0 });
        if (String(url).endsWith('/multi')) return Response.json({ results: [{ hits: [] }, { hits: [] }] });
        return Response.json(emptySearch);
    };
    vi.stubGlobal('fetch', fetcher);

    switch (input.adapter) {
        case 'list-search':
            await trails_search_filter(await loadFilter(context, input, source), input.page ?? 1, input.per_page ?? 25, fetcher);
            break;
        case 'map-search':
            await trails_search_bounding_box(input.north_east as LngLat, input.south_west as LngLat,
                await loadFilter(context, input, source), input.page ?? 1, input.zoom ?? 11, input.per_page ?? 50, input.load_map_data ?? true);
            break;
        case 'generic-search': await searchTrails(input.q, input.options); break;
        case 'global-multi': await searchMulti(input.options); break;
        case 'list-index-search': await lists_search_filter(input.filter, input.page ?? 1, input.per_page ?? 30, fetcher); break;
    }
    return { engine_requests: requests };
}
