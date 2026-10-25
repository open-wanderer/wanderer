import { PUBLIC_POCKETBASE_URL } from '$app/env/public';
import { version } from "$app/env";

import { handleError } from '#lib/util/api_util.ts';
import type { RequestEvent } from '@sveltejs/kit';

export async function GET(event: RequestEvent) {
    try {
        const upstream = `${PUBLIC_POCKETBASE_URL}/.well-known/nodeinfo/2.1`;
        const response = await event.fetch(upstream);
        const payload = await response.json();
        payload.software.version = version;
        return new Response(JSON.stringify(payload), {
            status: response.status,
            headers: { 'Content-Type': 'application/json' },
        });
    } catch (err) {
        return handleError(err);
    }
}
