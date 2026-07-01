import type { Subcategory } from "$lib/models/subcategory";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";
import { writable, type Writable } from "svelte/store";

export const subcategories: Writable<Subcategory[]> = writable([]);

export async function subcategories_index(
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f(
        "/api/v1/subcategory?" +
            new URLSearchParams({
                perPage: "-1",
                expand: "category",
                sort: "category,name",
            }),
        {
            method: "GET",
        },
    );
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const response: ListResult<Subcategory> = await r.json();
    subcategories.set(response.items);

    return response.items as Subcategory[];
}
