import { json, type RequestEvent } from '@sveltejs/kit';
import { ORIGIN } from '$app/env/private';

export async function GET(_event: RequestEvent) {
    return json({
        links: [{
            rel: 'http://nodeinfo.diaspora.software/ns/schema/2.1',
            href: `${ORIGIN}/.well-known/nodeinfo/2.1`,
        }],
    });
}
