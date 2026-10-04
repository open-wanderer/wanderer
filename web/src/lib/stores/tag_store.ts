import { type Tag } from "$lib/models/tag";
import { APIError } from "$lib/util/api_util";
import PocketBase, { type ListResult } from "pocketbase";
import { writable, type Writable } from "svelte/store";

let tags: Writable<Tag[]> = writable([]);
const filterBuilder = new PocketBase();


export async function tags_index(name: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/tag?' + new URLSearchParams({
        filter: filterBuilder.filter("name ~ {:name}", { name }),
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<Tag> = await r.json();

    tags.set(response.items);

    return response;

}

export async function tags_create(tag: Tag) {
    let r = await fetch('/api/v1/tag', {
        method: 'PUT',
        body: JSON.stringify(tag),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}
