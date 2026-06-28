import type { UserCategoryPreference } from "$lib/models/category_preference";
import { APIError } from "$lib/util/api_util";
import { writable, type Writable } from "svelte/store";

export const categoryPreferences: Writable<UserCategoryPreference[]> = writable(
    [],
);

export async function category_preferences_index(
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/user-category-preference", {
        method: "GET",
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const response = (await r.json()) as UserCategoryPreference[];
    categoryPreferences.set(response);

    return response;
}

export async function category_preferences_save(
    preference: Pick<UserCategoryPreference, "category" | "visible">,
) {
    const r = await fetch("/api/v1/user-category-preference", {
        method: "PUT",
        body: JSON.stringify(preference),
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const response = (await r.json()) as UserCategoryPreference;
    categoryPreferences.update((preferences) => [
        ...preferences.filter((item) => item.id !== response.id),
        response,
    ]);

    return response;
}

export async function category_preferences_reorder(categories: string[]) {
    const r = await fetch("/api/v1/user-category-preference/reorder", {
        method: "POST",
        body: JSON.stringify({ categories }),
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return await r.json();
}
