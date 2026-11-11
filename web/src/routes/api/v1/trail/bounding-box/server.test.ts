import type { RequestEvent } from '@sveltejs/kit';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('#lib/util/activitypub_server_util.ts', () => ({
    getActorResponseForHandle: vi.fn(),
}));

vi.mock('#lib/server/category_preference_filter.ts', () => ({
    withTrailPreferenceMeiliFilter: vi.fn(async (_event: unknown, filter: unknown) => filter),
}));

vi.mock('#lib/util/secure_fetch_util.ts', () => ({
    fetchExternalJson: vi.fn(),
}));

import { getActorResponseForHandle } from '#lib/util/activitypub_server_util.ts';
import { fetchExternalJson } from '#lib/util/secure_fetch_util.ts';
import { GET } from './+server';

const mockedGetActor = vi.mocked(getActorResponseForHandle);
const mockedFetchJson = vi.mocked(fetchExternalJson);

function hit(field: string, value: number) {
    return { hits: [{ [field]: value }] };
}

type TestEvent = Omit<RequestEvent, 'fetch'> & { fetch: RequestEvent['fetch'] };

function buildEvent(options: { handle?: string; anonymous?: boolean } = {}): TestEvent {
    const url = new URL(
        `http://localhost/api/v1/trail/bounding-box${
            options.handle ? `?handle=${encodeURIComponent(options.handle)}` : ''
        }`,
    );

    const multiSearch = vi.fn(async () => ({
        results: [hit('min_lat', 1), hit('max_lat', 2), hit('min_lon', 3), hit('max_lon', 4)],
    }));

    return {
        url,
        locals: {
            pb: {
                authStore: { record: options.anonymous ? null : { id: 'me' } },
            },
            ms: { multiSearch },
        },
        fetch: vi.fn(),
    } as unknown as TestEvent;
}

function remoteActor() {
    return {
        id: 'remote-1',
        is_local: false,
        iri: 'https://remote.example/actor/1',
        preferred_username: 'bob',
    } as any;
}

describe('GET /api/v1/trail/bounding-box', () => {
    beforeEach(() => {
        mockedGetActor.mockReset();
    });

    it('answers an anonymous request without a handle with the world view', async () => {
        const event = buildEvent({ anonymous: true });

        const body = await (await GET(event)).json();

        expect(body.has_trails).toBe(false);
        expect((event.locals as any).ms.multiSearch).not.toHaveBeenCalled();
    });

    it('answers an anonymous request for a local profile, as another instance sends it', async () => {
        mockedGetActor.mockResolvedValue({
            actor: { id: 'actor-123', is_local: true, iri: 'http://localhost/actor/1', preferred_username: 'tester' } as any,
        });
        const event = buildEvent({ handle: 'tester', anonymous: true });

        const body = await (await GET(event)).json();

        expect(body).toEqual({ min_lat: 1, max_lat: 2, min_lon: 3, max_lon: 4, has_trails: true });
        const call = ((event.locals as any).ms.multiSearch as ReturnType<typeof vi.fn>).mock.calls[0][0];
        expect(call.queries[0].filter).toEqual(['author = actor-123']);
    });

    it('leaves the filter undefined when no handle is supplied', async () => {
        const event = buildEvent();

        await GET(event);

        const multiSearch = (event.locals as any).ms.multiSearch as ReturnType<typeof vi.fn>;
        const call = multiSearch.mock.calls[0][0];
        for (const query of call.queries) {
            expect(query.filter).toBeUndefined();
        }
    });

    it('scopes every one of the four multiSearch queries to a local actor', async () => {
        mockedGetActor.mockResolvedValue({
            actor: {
                id: 'actor-123',
                is_local: true,
                iri: 'http://localhost/actor/1',
                preferred_username: 'tester',
            } as any,
        });
        const event = buildEvent({ handle: 'tester' });

        await GET(event);

        const multiSearch = (event.locals as any).ms.multiSearch as ReturnType<typeof vi.fn>;
        const call = multiSearch.mock.calls[0][0];
        for (const query of call.queries) {
            expect(query.filter).toEqual(['author = actor-123']);
        }
    });

    it('proxies exactly once to a non-local actor origin and returns the proxied coordinates', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        const event = buildEvent({ handle: 'bob' });
        mockedFetchJson.mockResolvedValue({ status: 200, ok: true, json: { min_lat: 1, max_lat: 2, min_lon: 3, max_lon: 4, has_trails: true } });

        const response = await GET(event);
        const body = await response.json();

        expect(body).toEqual({ min_lat: 1, max_lat: 2, min_lon: 3, max_lon: 4, has_trails: true });
        expect(mockedFetchJson).toHaveBeenCalledTimes(1);
        const [calledUrl, calledOptions] = mockedFetchJson.mock.calls[0];
        expect(calledUrl).toContain('https://remote.example');
        expect(calledUrl).toContain('handle=bob');
        expect(calledOptions).toMatchObject({ deadlineMs: 4000, maxBytes: 16 * 1024 });
        expect(event.fetch).not.toHaveBeenCalled();
    });

    it('degrades to has_trails: false with HTTP 200 on a non-ok remote response', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        mockedFetchJson.mockResolvedValue({ status: 404, ok: false, json: {} });

        const response = await GET(buildEvent({ handle: 'bob' }));
        const body = await response.json();

        expect(response.status).toBe(200);
        expect(body.has_trails).toBe(false);
    });

    it('degrades to has_trails: false with HTTP 200 when the guard refuses the request', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        mockedFetchJson.mockRejectedValue(new Error('invalid_url'));

        const response = await GET(buildEvent({ handle: 'bob' }));
        const body = await response.json();

        expect(response.status).toBe(200);
        expect(body.has_trails).toBe(false);
    });

    it('degrades to has_trails: false with HTTP 200 on a malformed remote payload', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        mockedFetchJson.mockResolvedValue({ status: 200, ok: true, json: { min_lat: 1, max_lat: 'nope', min_lon: 3, max_lon: 4, has_trails: true } });

        const response = await GET(buildEvent({ handle: 'bob' }));
        const body = await response.json();

        expect(response.status).toBe(200);
        expect(body.has_trails).toBe(false);
    });
});
