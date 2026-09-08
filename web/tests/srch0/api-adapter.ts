import { vi } from 'vitest';
import type { RequestEvent } from '@sveltejs/kit';
import { POST as proxy } from '../../src/routes/api/v1/search/[index]/+server';
import { POST as multi } from '../../src/routes/api/v1/search/multi/+server';
import { GET as actors } from '../../src/routes/api/v1/search/actor/+server';
import { POST as profileTrails } from '../../src/routes/api/v1/profile/[handle]/trails/+server';
import { POST as profileLists } from '../../src/routes/api/v1/profile/[handle]/lists/+server';
import { GET as recommend } from '../../src/routes/api/v1/trail/recommend/+server';
import { GET as bounds } from '../../src/routes/api/v1/trail/bounding-box/+server';
import { GET as filterValues } from '../../src/routes/api/v1/trail/filter/+server';
import { POST as cluster } from '../../src/routes/api/v1/search/trails/cluster/+server';
import { PUT as upload } from '../../src/routes/api/v1/trail/upload/+server';
import { apiEvent, routeError } from './api-context';
import { emptySearch } from './compiler-adapter';
import type { ApiAdapter, ApiInput, Context, Dataset, JsonRecord } from './types';

const external = vi.hoisted(() => ({ actor: {} as JsonRecord, upload: {} as JsonRecord }));
vi.mock('$lib/util/activitypub_server_util', () => ({ getActorResponseForHandle: async () => ({ actor: external.actor }) }));
// GPX-Decodierung liegt ausserhalb der Suchbeobachtung. PUT und seine private
// Duplikatprüfung laufen unverändert; nur die Parser-Ausgabe ist synthetisch.
vi.mock('$lib/util/gpx_util', () => ({
    fromFile: async () => ({ gpxData: '<gpx/>', gpxFile: new Blob(['<gpx/>']) }),
    gpx2trail: async () => ({ trail: structuredClone(external.upload) }),
}));

const handlers: Record<ApiAdapter, (event: RequestEvent) => Promise<Response>> = {
    'api-proxy': proxy, 'api-multi': multi, 'api-actor': actors,
    'api-profile-trails': profileTrails, 'api-profile-lists': profileLists,
    'api-recommendation': recommend, 'api-bounds': bounds, 'api-cluster': cluster,
    'api-filter-values': filterValues, 'api-upload': upload,
};

function uploadRequest(event: RequestEvent, input: ApiInput) {
    external.upload = input.trail ?? {};
    const form = new FormData();
    form.set('file', new File(['<gpx/>'], 'synthetic.gpx'));
    event.request = new Request('http://srch0.invalid', { method: 'PUT', body: form });
    event.fetch = async url => {
        const path = String(url);
        if (path.startsWith('/api/v1/geocoding/')) return Response.json({ features: [] });
        if (path.startsWith('/api/v1/trail/form')) return Response.json({ id: 'created-synthetic' });
        throw new Error(`Nicht deklarierter Upload-Aufruf: ${path}`);
    };
}

export async function observeAPI(context: Context, input: ApiInput, source: Dataset, engine?: unknown) {
    const requests: JsonRecord[] = [];
    const responses: unknown[] = input.search_ids
        ? [{ hits: input.search_ids.map(id => source.trails.find(trail => trail.id === id)) }]
        : [...(input.responses ?? [emptySearch])];
    const ms = engine ?? {
        index: (index: string) => ({ search: async (q: string, options: unknown) => {
            requests.push({ index, q, options });
            const response = responses.shift() as JsonRecord | undefined;
            return (response?.results as unknown[] | undefined)?.[0] ?? response ?? emptySearch;
        } }),
        multiSearch: async (body: JsonRecord) => { requests.push(body); return responses.shift() ?? { results: [{ hits: [] }] }; },
    };
    const event = apiEvent({
        principal: context.principal, preferences: context.preferences, ms,
        params: input.params, url: input.url, body: input.body,
        settings: input.settings, filterValues: input.filter_values,
        fetch: async (url, options) => { requests.push({ url: String(url), ...options }); return Response.json(input.remote_response); },
    });
    external.actor = input.actor ?? { id: 'alice', is_local: true };
    if (input.adapter === 'api-upload') uploadRequest(event, input);
    if (input.random !== undefined) vi.spyOn(Math, 'random').mockReturnValue(input.random);

    try {
        const response = await handlers[input.adapter](event);
        const body = await response.json() as JsonRecord & { detail?: { status?: number } };
        if (input.adapter === 'api-upload') return {
            engine_requests: requests, status: response.status,
            result: response.status >= 400
                ? { duplicate_id: body.id, duplicate_name: body.name, duplicate_domain: body.domain }
                : { id: body.id },
        };
        if (response.status >= 400) return {
            engine_requests: requests,
            diagnostics: { status: response.status, ...(body.detail?.status ? { cause_status: body.detail.status } : {}) },
        };
        return { engine_requests: requests, status: response.status, response: body };
    } catch (error) {
        return { engine_requests: requests, diagnostics: { status: routeError(error).status } };
    }
}
