import type { Category } from "$lib/models/category";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";
import { writable, type Writable } from "svelte/store";

export const categories: Writable<Category[]> = writable([])

export async function categories_index(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    const r = await f('/api/v1/category?perPage=-1&sort=name', {
        method: 'GET',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const response: ListResult<Category> = await r.json();

    categories.set(response.items);

    return response.items as Category[];
}
