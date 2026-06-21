import type { PluginInstance } from "$lib/models/plugin_instance";
import { APIError } from "$lib/util/api_util";
import type { ListResult } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";

export const pluginInstances: Writable<PluginInstance[]> = writable([]);

export async function plugin_instances_index(
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-instance?perPage=-1", {
        method: "GET",
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    const fetchedInstances: ListResult<PluginInstance> = await r.json();
    pluginInstances.set(fetchedInstances.items);

    return fetchedInstances.items;
}

export async function plugin_instances_create(
    instance: Partial<PluginInstance>,
) {
    const user = get(currentUser);
    if (!user) {
        throw Error("Unauthenticated");
    }

    const r = await fetch("/api/v1/plugin-instance", {
        method: "PUT",
        body: JSON.stringify({
            ...instance,
            user: user.id,
        }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as PluginInstance;
}

export async function plugin_instances_update(
    instance: PluginInstance,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    if (!instance.id) {
        throw Error("Plugin instance has no id");
    }

    const r = await f("/api/v1/plugin-instance/" + instance.id, {
        method: "POST",
        body: JSON.stringify(instance),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as PluginInstance;
}

export async function plugin_oauth_start(
    data: {
        pluginId: string;
        instanceId?: string;
        authContext?: string;
        redirectUri: string;
    },
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/oauth/start", {
        method: "POST",
        body: JSON.stringify(data),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as { url: string; state: string; instanceId: string };
}

export async function plugin_auth_validate(
    data: {
        pluginId: string;
        instanceId?: string;
        authContext?: string;
        auth: Record<string, string>;
    },
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/auth/validate", {
        method: "POST",
        body: JSON.stringify(data),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as { ok: boolean; authContext?: string };
}

export async function plugin_oauth_callback(
    data: { instanceId: string; code: string; state: string },
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/oauth/callback", {
        method: "POST",
        body: JSON.stringify(data),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as { ok: boolean };
}

export async function plugin_oauth_revoke(
    instanceId: string,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/oauth/revoke", {
        method: "POST",
        body: JSON.stringify({ instanceId }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as { ok: boolean };
}

export async function plugin_category_remap_preview(
    instanceId: string,
    config?: Record<string, unknown>,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/category-remap/preview", {
        method: "POST",
        body: JSON.stringify({ instanceId, config }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as { count: number; backfilledSinceMapping?: number };
}

export async function plugin_category_remap_apply(
    instanceId: string,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/category-remap/apply", {
        method: "POST",
        body: JSON.stringify({ instanceId }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as { count: number; remapped?: number };
}
