import type { RequestEvent } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { expect, it, vi } from 'vitest';
import type { Trail } from '#lib/models/trail.ts';
import { completeOwnerExportChildren } from './trail_export_response';

it('fills a saturated child expansion through every PocketBase response page', async () => {
    const children = Array.from({ length: 1001 }, (_, index) => ({
        id: String(index).padStart(15, '0'), trail: 'trail0000000001', name: `Waypoint ${index}`,
    }));
    const requests: URL[] = [];
    const backend = vi.fn(async (input: RequestInfo | URL) => {
        const url = new URL(String(input));
        requests.push(url);
        const page = Number(url.searchParams.get('page'));
        const perPage = Number(url.searchParams.get('perPage'));
        expect(url.pathname).toBe('/api/collections/waypoints/records');
        expect(url.searchParams.get('filter')).toBe('trail = "trail0000000001"');
        return Response.json({ page, perPage, totalItems: children.length, totalPages: Math.ceil(children.length / perPage), items: children.slice((page - 1) * perPage, page * perPage) });
    });
    const pb = new PocketBase('https://a.example');
    const event = { locals: { pb }, fetch: backend } as unknown as RequestEvent;
    const trail = { id: 'trail0000000001', expand: { waypoints_via_trail: children.slice(0, 1000), summit_logs_via_trail: [] } } as unknown as Trail;
    await completeOwnerExportChildren(event, [trail]);
    expect(trail.expand?.waypoints_via_trail).toEqual(children);
    expect(requests.map(url => url.searchParams.get('page'))).toEqual(['1', '2', '3']);
    expect(requests.every(url => url.searchParams.get('perPage') === '500')).toBe(true);
});
