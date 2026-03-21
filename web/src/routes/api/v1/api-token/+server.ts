import type { APIToken } from '$lib/models/api_token';
import { create } from '$lib/util/api_util';
import { Collection, handleError, list } from '$lib/util/api_util';
import { json, type RequestEvent } from '@sveltejs/kit';
import { APITokenCreateSchema } from "$lib/models/api/api_token_schema";

export async function GET(event: RequestEvent) {
    try {
        const r = await list<APIToken>(event, Collection.api_tokens);

        return json(r)
    } catch (e: any) {
        return handleError(e);
    }
}


export async function PUT(event: RequestEvent) {
    try {
        const r = await create<APIToken>(event, APITokenCreateSchema, Collection.api_tokens)
        return json(r);
    } catch (e) {
        return handleError(e)
    }
}
