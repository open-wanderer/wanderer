import type { UserSubcategoryPreference } from "$lib/models/subcategory_preference";
import { APIError } from "$lib/util/api_util";
import { writable, type Writable } from "svelte/store";

export const subcategoryPreferences: Writable<UserSubcategoryPreference[]> =
    writable([]);

export async function subcategory_preferences_index(
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/user-subcategory-preference", {
        method: "GET",
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const response = (await r.json()) as UserSubcategoryPreference[];
    subcategoryPreferences.set(response);

    return response;
}

export async function subcategory_preferences_save(
    preference: Pick<UserSubcategoryPreference, "subcategory" | "visible">,
) {
    const r = await fetch("/api/v1/user-subcategory-preference", {
        method: "PUT",
        body: JSON.stringify(preference),
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const response = (await r.json()) as UserSubcategoryPreference;
    subcategoryPreferences.update((preferences) => [
        ...preferences.filter((item) => item.id !== response.id),
        response,
    ]);

    return response;
}

export async function subcategory_preferences_reorder(
    category: string,
    subcategories: string[],
) {
    const r = await fetch("/api/v1/user-subcategory-preference/reorder", {
        method: "POST",
        body: JSON.stringify({ category, subcategories }),
    });
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return await r.json();
}
