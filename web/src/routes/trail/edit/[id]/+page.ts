import { Trail } from "$lib/models/trail";
import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { lists_index } from "$lib/stores/list_store";
import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
import { plugins_index } from "$lib/stores/plugin_store";
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
    const subcategories = await subcategories_index(fetch)
    const subcategoryPreferences = await subcategory_preferences_index(fetch)
    const lists = await lists_index({ q: "", author: user?.actor ?? "" }, 1, -1, fetch)

    let trail: Trail;
    if (params.id === "new") {
        // duplicate trail
        if (url.searchParams.has("orig")) {
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

    const [pluginInstances, plugins] = user
        ? await Promise.all([
              plugin_instances_index(fetch).catch(() => []),
              plugins_index(fetch).catch(() => []),
          ])
        : [[], []];
    const assetProviderIds = new Set(
        plugins
            .filter(plugin => plugin.type === "assets")
            .map(plugin => plugin.id),
    );
    const assetPlugins = pluginInstances
        .filter(instance => instance.enabled && assetProviderIds.has(instance.plugin_id))
        .map(instance => instance.plugin_id);
    const assetPluginProviders = assetPlugins
        .map(id => plugins.find(plugin => plugin.id === id))
        .filter(plugin => plugin !== undefined);
    const assetPluginActive = assetPlugins.length > 0;

    return {
        trail,
        lists,
        categories,
        categoryPreferences,
        subcategories,
        subcategoryPreferences,
        assetPlugins,
        assetPluginProviders,
        assetPluginActive,
    }
};
