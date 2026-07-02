import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
import { plugins_index } from "$lib/stores/plugin_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { categories_index } from "$lib/stores/category_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch }) => {
    const [pluginInstances, pluginProviders, categories, subcategories] = await Promise.all([
        plugin_instances_index(fetch),
        plugins_index(fetch),
        categories_index(fetch),
        subcategories_index(fetch),
        category_preferences_index(fetch),
        subcategory_preferences_index(fetch),
    ]);
    return { pluginInstances, pluginProviders, categories, subcategories };
};
