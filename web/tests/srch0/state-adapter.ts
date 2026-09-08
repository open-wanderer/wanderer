import type { LoadEvent, ServerLoadEvent } from '@sveltejs/kit';
import type { TrailFilter } from '$lib/models/trail';
import { sanitizeTrailFilter } from '$lib/util/trail_filter_util';
import { load as listLoad } from '../../src/routes/trails/+page';
import { load as mapLoad } from '../../src/routes/map/+page';
import type { Context, Dataset, FilterInput, StateInput } from './types';

export async function loadFilter(context: Context, input: FilterInput, source: Dataset): Promise<TrailFilter> {
    const responses: Record<string, unknown> = {
        '/api/v1/trail/filter': input.limits ?? { max_distance: 20000, max_elevation_gain: 4000, max_elevation_loss: 4000 },
        '/api/v1/category': { items: source.categories },
        '/api/v1/subcategory': { items: source.subcategories },
        '/api/v1/user-category-preference': [],
        '/api/v1/user-subcategory-preference': [],
        '/api/v1/trail/bounding-box': { min_lat: 0, max_lat: 0, min_lon: 0, max_lon: 0, has_trails: false },
    };
    const fetcher: typeof fetch = async url => {
        const path = new URL(String(url), 'http://srch0.invalid').pathname;
        if (!(path in responses)) throw new Error(`Nicht deklarierter Loader-Aufruf: ${path}`);
        return Response.json(responses[path]);
    };
    const loader = context.surface === 'map' ? mapLoad : listLoad;
    // Der Test stellt nur die beiden vom produktiven Loader gelesenen Felder bereit.
    const event = { url: new URL(input.url ?? '/trails', 'http://srch0.invalid'), fetch: fetcher } as LoadEvent & ServerLoadEvent;
    const loaded = await loader(event) as { filter: TrailFilter };
    return { ...loaded.filter, ...input.filter } as TrailFilter;
}

export async function observeState(context: Context, input: StateInput, source: Dataset) {
    const filter = await loadFilter(context, input, source);
    return { legacy_state: input.adapter === 'sanitize' ? sanitizeTrailFilter(input.candidate, filter) : filter };
}
