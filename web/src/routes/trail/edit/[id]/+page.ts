import { Trail } from "$lib/models/trail";
import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { lists_index } from "$lib/stores/list_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { trails_show } from "$lib/stores/trail_store";
import { currentUser } from "$lib/stores/user_store";
import { designSelectableCategories } from "$lib/util/category_util";
import { error, type Load } from "@sveltejs/kit";
import { get } from "svelte/store";
import { locale } from "svelte-i18n";

export const load: Load = async ({ params, fetch, url }) => {
    const user = get(currentUser)

    if (!params.id) {
        return error(400, "Bad Request")
    }
    const categories = await categories_index(fetch)
    const categoryPreferences = await category_preferences_index(fetch)
    await subcategories_index(fetch)
    await subcategory_preferences_index(fetch)
    const lists = await lists_index({ q: "", author: user?.actor ?? "" }, 1, -1, fetch)

    let trail: Trail;
    let isDuplicateTrail = false;
    if (params.id === "new") {
        // duplicate trail
        if (url.searchParams.has("orig")) {
            isDuplicateTrail = true;
            const originalId = url.searchParams.get("orig")!;
            const originalTrail = await trails_show(originalId, undefined, undefined, true, fetch);
            trail = Trail.from(originalTrail)
        } else {
            const defaultCategory =
                designSelectableCategories(
                    categories,
                    categoryPreferences,
                    get(locale),
                )[0] ?? categories[0];

            trail = defaultCategory
                ? new Trail("", { category: defaultCategory })
                : new Trail("");
        }
    } else {
        trail = await trails_show(params.id, undefined, url.searchParams.get("share") ?? undefined, true, fetch);
    }

    return {
        trail: trail,
        isDuplicateTrail,
        lists: lists,
        categories: categories,
        categoryPreferences: categoryPreferences,
    }
};
