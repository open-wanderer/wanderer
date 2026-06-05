import type { PluginProvider } from "$lib/models/plugin_provider";
import type { PluginSystemPlugin } from "$lib/models/plugin_system";
import { APIError } from "$lib/util/api_util";
import { derived, writable, type Readable, type Writable } from "svelte/store";
import {
    pluginInstances,
    plugin_instances_index,
} from "./plugin_instance_store";

export const pluginProviders: Writable<PluginProvider[]> = writable([]);

// True when the user has at least one enabled plugin instance whose plugin
// advertises the trail send capability. Used to gate the trail "send to" action.
export const hasSendCapablePlugin: Readable<boolean> = derived(
    [pluginProviders, pluginInstances],
    ([$plugins, $instances]) => {
        const enabledProviders = new Set(
            $instances.filter((a) => a.enabled).map((a) => a.plugin_id),
        );
        return $plugins.some(
            (p) =>
                p.status === "available" &&
                (p.capabilities ?? []).includes("prepare_trail_send.v1") &&
                enabledProviders.has(p.id),
        );
    },
);

let pluginDataLoaded = false;

// Loads plugins and instances once per session so the derived gating
// stores are populated (e.g. for the trail "send to" action). Safe to call
// repeatedly; only the first call performs the requests.
export async function load_plugin_data_once() {
    if (pluginDataLoaded) {
        return;
    }
    pluginDataLoaded = true;
    try {
        await Promise.all([
            plugins_index(),
            plugin_instances_index(),
        ]);
    } catch (e) {
        pluginDataLoaded = false;
        console.error(e);
    }
}

export async function plugins_index(
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/plugins", {
        method: "GET",
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const data: { items: PluginSystemPlugin[] } = await r.json();
    const items = data.items.map(pluginSystemToPluginProvider);
    pluginProviders.set(items);

    return items;
}

function pluginSystemToPluginProvider(plugin: PluginSystemPlugin): PluginProvider {
    const contexts = Object.entries(
        (plugin.manifest.auth?.contexts ?? {}) as Record<string, any>,
    );
    const primaryAuth = contexts[0]?.[1] ?? {};
    const fields =
        primaryAuth.fields ??
        primaryAuth.secretFields ??
        (primaryAuth.secretField ? [primaryAuth.secretField] : []);

    return {
        id: plugin.id,
        name: plugin.name,
        description: plugin.description,
        icon: plugin.icon,
        iconDark: plugin.iconDark,
        version: plugin.version,
        auth: {
            type: primaryAuth.type ?? "none",
            fields,
            secretFields: primaryAuth.secretFields ?? fields,
            authorizationUrl: primaryAuth.authorizationUrl,
            tokenUrl: primaryAuth.tokenUrl,
            tokenRequestFormat: primaryAuth.tokenRequestFormat,
            scopes: primaryAuth.scopes,
            scopeSeparator: primaryAuth.scopeSeparator,
            authorizationParams: primaryAuth.authorizationParams,
            pkce: primaryAuth.pkce,
            tokenAuth: primaryAuth.tokenAuth,
        },
        configSchema: plugin.manifest.configSchema as PluginProvider["configSchema"],
        capabilities: plugin.capabilities,
        status: plugin.status,
        error: plugin.error,
    };
}
