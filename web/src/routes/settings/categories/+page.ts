import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch }) => {
    const [
        categories,
        categoryPreferences,
        subcategories,
        subcategoryPreferences,
    ] = await Promise.all([
        categories_index(fetch),
        category_preferences_index(fetch),
        subcategories_index(fetch),
        subcategory_preferences_index(fetch),
    ]);

    return {
        categories,
        categoryPreferences,
        subcategories,
        subcategoryPreferences,
    };
};
