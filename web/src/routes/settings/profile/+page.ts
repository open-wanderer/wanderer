import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch }) => {
    const categories = await categories_index(fetch);
    const categoryPreferences = await category_preferences_index(fetch);

    return { categories, categoryPreferences };
};
