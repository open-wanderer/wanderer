import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
import { plugins_index } from "$lib/stores/plugin_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { categories_index } from "$lib/stores/category_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { routingSettings } from "$lib/stores/routing_store.svelte";
import type { RoutingSettings } from "$lib/models/routing";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch }) => {
    // Refreshing the provider cache can provision default instances for newly
    // installed plugins, so it must finish before instances are listed.
    const pluginProviders = await plugins_index(fetch);
    const [pluginInstances, categories, subcategories, currentRoutingSettings] = await Promise.all([
        plugin_instances_index(fetch),
        categories_index(fetch),
        subcategories_index(fetch),
        routingSettings(fetch).catch(() => ({} as RoutingSettings)),
        category_preferences_index(fetch),
        subcategory_preferences_index(fetch),
    ]);
    return { pluginInstances, pluginProviders, categories, subcategories, currentRoutingSettings };
};
