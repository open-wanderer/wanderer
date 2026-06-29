import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
import { plugins_index } from "$lib/stores/plugin_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch, data, parent }) => {
    const parentData = await parent();
    const [
        categories,
        categoryPreferences,
        subcategories,
        subcategoryPreferences,
        pluginInstances,
        pluginProviders,
    ] = await Promise.all([
        categories_index(fetch),
        category_preferences_index(fetch),
        subcategories_index(fetch),
        subcategory_preferences_index(fetch),
        plugin_instances_index(fetch),
        plugins_index(fetch),
    ]);

    return {
        categories,
        categoryPreferences,
        subcategories,
        subcategoryPreferences,
        pluginInstances,
        pluginProviders,
        user: parentData.user,
        trailUsage: data?.trailUsage ?? { categories: {}, subcategories: {} },
    };
};
