import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
import { plugins_index } from "$lib/stores/plugin_store";
import { trails_show } from "$lib/stores/trail_store";
import { currentUser } from "$lib/stores/user_store";
import { APIError } from "$lib/util/api_util";
import { error, type Load, type NumericRange } from "@sveltejs/kit";
import { get } from "svelte/store";

export const load: Load = async ({ params, fetch, url }) => {
    try {
        const trail = await trails_show(params.id!, params.handle, url.searchParams.get("share") ?? undefined, true, fetch)
        const user = get(currentUser);
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
        const assetPluginIds = pluginInstances
            .filter(instance => instance.enabled && assetProviderIds.has(instance.plugin_id))
            .map(instance => instance.plugin_id);
        const assetPluginProviders = assetPluginIds
            .map(id => plugins.find(plugin => plugin.id === id))
            .filter(plugin => plugin !== undefined);

        return { trail, assetPluginIds, assetPluginProviders }
    } catch (e) {
        if (e instanceof APIError) {
            error(e.status as NumericRange<400, 599>, {
                message: e.status == 404 ? 'Not found' : e.message
            });
        }
        console.error(e);
    }

};