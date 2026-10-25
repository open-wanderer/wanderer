import { json, type RequestEvent } from '@sveltejs/kit';
// @ts-ignore
import { env } from '$env/dynamic/private';

export async function GET(_event: RequestEvent) {
    return json({
        links: [{
            rel: 'http://nodeinfo.diaspora.software/ns/schema/2.1',
            href: `${env.ORIGIN}/.well-known/nodeinfo/2.1`,
        }],
    });
}
