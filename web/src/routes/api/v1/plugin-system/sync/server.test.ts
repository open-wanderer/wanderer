import type { RequestEvent } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { describe, expect, it, vi } from 'vitest';
import { POST } from './+server';

function requestEvent(response: Response) {
    const pb = new PocketBase('https://pocketbase.test');
    pb.authStore.save('synthetic-owner-token');
    const fetch = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response);
    return {
        event: {
            url: new URL('https://wanderer.test/api/v1/plugin-system/sync'),
            request: new Request('https://wanderer.test/api/v1/plugin-system/sync', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ instanceId: 'instance0000001' }),
            }),
            locals: { pb },
            fetch,
        } as unknown as RequestEvent,
        fetch,
    };
}

describe('plugin sync proxy', () => {
    it('starts the sync with the user token and answers 202', async () => {
        const { event, fetch } = requestEvent(Response.json({ started: true }, { status: 202 }));

        const response = await POST(event);

        expect(response.status).toBe(202);
        expect(await response.json()).toEqual({ started: true });
        const [url, options] = fetch.mock.calls[0];
        expect(String(url)).toBe('https://pocketbase.test/plugins/sync');
        expect(JSON.parse(String(options?.body))).toEqual({ instanceId: 'instance0000001' });
        expect(new Headers(options?.headers).get('Authorization')).toBe('synthetic-owner-token');
    });

    // The proxy does not pass on the Retry-After header, so the settings page
    // reads the wait from the body.
    it('passes on the wait of a refused sync in the body', async () => {
        const { event } = requestEvent(
            Response.json(
                { status: 429, message: 'Plugin sync was started too recently.', data: { retryAfter: 112 } },
                { status: 429, headers: { 'Retry-After': '112' } },
            ),
        );

        const response = await POST(event);

        expect(response.status).toBe(429);
        expect((await response.json()).data).toEqual({ retryAfter: 112 });
    });
});
