import type { RequestEvent } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { describe, expect, it, vi } from 'vitest';
import { POST } from './+server';

function requestEvent(data: unknown, response = Response.json({ items: [] })) {
    const pb = new PocketBase('https://pocketbase.test');
    pb.authStore.save('synthetic-owner-token');
    const fetch = vi.fn(async (_url: RequestInfo | URL, _options?: RequestInit) => response);
    return {
        event: {
            url: new URL('https://wanderer.test/api/v1/tag/lookup'),
            request: new Request('https://wanderer.test/api/v1/tag/lookup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            }),
            locals: { pb },
            fetch,
        } as unknown as RequestEvent,
        fetch,
    };
}

describe('exact tag lookup proxy', () => {
    it.each([
        'Berg Tour',
        'trailing\\',
        'internal\\backslash',
        `O'Brien "quoted" %_ || name != ''`,
        '  e\u0301 日本語 👩‍👩‍👧‍👦  ',
        '',
        '   ',
        '🌍'.repeat(5000),
    ])('sends a literal name in the request body through the actual SDK', async name => {
        const match = { id: 'existingtag0001', name };
        const { event, fetch } = requestEvent({ name, id: 'ignoredid000001', filter: 'ignored' }, Response.json({ items: [match] }));

        const response = await POST(event);

        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({ items: [match] });
        expect(fetch).toHaveBeenCalledOnce();
        const [url, options] = fetch.mock.calls[0];
        expect(String(url)).toBe('https://pocketbase.test/tags/lookup');
        expect(options?.method).toBe('POST');
        expect(JSON.parse(String(options?.body))).toEqual({ name });
        expect(new Headers(options?.headers).get('Authorization')).toBe('synthetic-owner-token');
        expect(new Headers(options?.headers).get('Content-Type')).toBe('application/json');
    });

    it('returns a lookup miss without creating a tag', async () => {
        const { event, fetch } = requestEvent({ name: 'New name' });
        const response = await POST(event);
        expect(response.status).toBe(200);
        expect(await response.json()).toEqual({ items: [] });
        expect(fetch).toHaveBeenCalledOnce();
    });

    it.each([{}, { name: null }, { name: 123 }, { name: 'Berg\tTour' }, { name: 'nul\u0000' }, { name: '🌍'.repeat(5001) }])('rejects invalid requests before contacting the backend', async data => {
        const { event, fetch } = requestEvent(data);
        const response = await POST(event);
        expect(response.status).toBe(400);
        expect(fetch).not.toHaveBeenCalled();
    });

    it('preserves backend access errors', async () => {
        const { event, fetch } = requestEvent({ name: 'Berg Tour' }, Response.json({ message: 'Denied' }, { status: 403 }));
        const response = await POST(event);
        expect(response.status).toBe(403);
        expect(await response.json()).toMatchObject({ message: 'Denied' });
        expect(fetch).toHaveBeenCalledOnce();
    });
});
