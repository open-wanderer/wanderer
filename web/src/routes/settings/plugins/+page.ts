import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
import { plugins_index } from "$lib/stores/plugin_store";
import { categories_index } from "$lib/stores/category_store";
import { type Load } from "@sveltejs/kit";

export const load: Load = async ({ fetch }) => {
    const [pluginInstances, pluginProviders, categories] = await Promise.all([
        plugin_instances_index(fetch),
        plugins_index(fetch),
        categories_index(fetch),
    ]);
    return { pluginInstances, pluginProviders, categories };
};
