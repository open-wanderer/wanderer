// Teardown-only helpers for cleaning up test data via the SvelteKit /api/v1/* proxy.
// These functions accept a Playwright APIRequestContext (the `request` fixture) which
// automatically forwards the stored session cookie from playwright/.auth/user.json.
//
// NOTE: deleteAllTrails only removes trails visible to the authenticated test user
// (scoped by PocketBase collection rules — desired behavior for isolated teardown).
// Not safe for concurrent use across parallel workers; relies on fullyParallel: false.

import { type APIRequestContext } from '@playwright/test';

export async function deleteTrail(request: APIRequestContext, id: string): Promise<void> {
    await request.delete(`/api/v1/trail/${id}`);
    // DELETE /api/v1/trail/:id returns { acknowledged: boolean } — no body parsing needed
}

export async function deleteList(request: APIRequestContext, id: string): Promise<void> {
    await request.delete(`/api/v1/list/${id}`);
}

export async function deleteComment(request: APIRequestContext, id: string): Promise<void> {
    await request.delete(`/api/v1/comment/${id}`);
}

export async function deleteAllTrails(request: APIRequestContext): Promise<void> {
    const response = await request.get('/api/v1/trail', {
        params: { perPage: '-1' }
    });
    const data = await response.json();
    for (const trail of data.items ?? []) {
        await deleteTrail(request, trail.id);
    }
}
