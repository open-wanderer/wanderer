import { get } from "svelte/store";
import { currentUser } from "./user_store";
import type { APIToken } from "$lib/models/api_token";
import { APIError } from "$lib/util/api_util";
import type { ListResult } from "pocketbase";

export async function api_tokens_index(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const r = await f(`/api/v1/api-token`, {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedTokens: ListResult<APIToken> = await r.json();

    return fetchedTokens;
}

export async function api_tokens_create(apiToken: APIToken) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    apiToken.user = user.id

    let r = await fetch('/api/v1/api-token', {
        method: 'PUT',
        body: JSON.stringify(apiToken),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: APIToken & { rawToken: string } = await r.json();

    return model;
}

export async function api_tokens_delete(token: APIToken) {
    const r = await fetch('/api/v1/api-token/' + token.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }


    return await r.json();

}