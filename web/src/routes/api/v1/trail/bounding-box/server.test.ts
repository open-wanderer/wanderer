import type { RequestEvent } from '@sveltejs/kit';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/util/activitypub_server_util', () => ({
    getActorResponseForHandle: vi.fn(),
}));

vi.mock('$lib/server/category_preference_filter', () => ({
    withTrailPreferenceMeiliFilter: vi.fn(async (_event: unknown, filter: unknown) => filter),
}));

import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
import { GET } from './+server';

const mockedGetActor = vi.mocked(getActorResponseForHandle);

function hit(field: string, value: number) {
    return { hits: [{ [field]: value }] };
}

function buildEvent(options: { handle?: string } = {}): RequestEvent {
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
                authStore: { record: { id: 'me' } },
            },
            ms: { multiSearch },
        },
        fetch: vi.fn(),
    } as unknown as RequestEvent;
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
        const fetchImpl = vi.fn(
            async () =>
                new Response(
                    JSON.stringify({ min_lat: 1, max_lat: 2, min_lon: 3, max_lon: 4, has_trails: true }),
                    { status: 200 },
                ),
        );
        event.fetch = fetchImpl as unknown as typeof event.fetch;

        const response = await GET(event);
        const body = await response.json();

        expect(body).toEqual({ min_lat: 1, max_lat: 2, min_lon: 3, max_lon: 4, has_trails: true });
        expect(fetchImpl).toHaveBeenCalledTimes(1);
        const [calledUrl, calledOptions] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
        expect(calledUrl).toContain('https://remote.example');
        expect(calledUrl).toContain('handle=bob');
        expect(calledOptions.signal).toBeInstanceOf(AbortSignal);
    });

    it('degrades to has_trails: false with HTTP 200 on a non-ok remote response', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        const event = buildEvent({ handle: 'bob' });
        event.fetch = vi.fn(async () => new Response('not found', { status: 404 }));

        const response = await GET(event);
        const body = await response.json();

        expect(response.status).toBe(200);
        expect(body.has_trails).toBe(false);
    });

    it('degrades to has_trails: false with HTTP 200 on a rejected fetch', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        const event = buildEvent({ handle: 'bob' });
        event.fetch = vi.fn(async () => {
            throw new Error('network down');
        });

        const response = await GET(event);
        const body = await response.json();

        expect(response.status).toBe(200);
        expect(body.has_trails).toBe(false);
    });

    it('degrades to has_trails: false with HTTP 200 on a malformed remote payload', async () => {
        mockedGetActor.mockResolvedValue({ actor: remoteActor() });
        const event = buildEvent({ handle: 'bob' });
        event.fetch = vi.fn(
            async () =>
                new Response(
                    JSON.stringify({ min_lat: 1, max_lat: 'nope', min_lon: 3, max_lon: 4, has_trails: true }),
                    { status: 200 },
                ),
        );

        const response = await GET(event);
        const body = await response.json();

        expect(response.status).toBe(200);
        expect(body.has_trails).toBe(false);
    });
});
