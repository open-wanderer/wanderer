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

// Keeps the shared instance list in step with a saved instance so derived
// gates (trail send, track resync) react without a page reload.
function rememberPluginInstance(saved: PluginInstance) {
    if (!saved?.id) {
        return;
    }
    pluginInstances.update((items) => {
        const index = items.findIndex((item) => item.id === saved.id);
        if (index === -1) {
            return [...items, saved];
        }
        const next = [...items];
        next[index] = saved;
        return next;
    });
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

    const created = (await r.json()) as PluginInstance;
    rememberPluginInstance(created);
    return created;
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

    const updated = (await r.json()) as PluginInstance;
    rememberPluginInstance(updated);
    return updated;
}

export type TrackResyncKind = "planned" | "completed";

export type TrackResyncPreview = {
    available: boolean;
    reason?: string;
    provider?: string;
    externalId?: string;
    // The reference does not record the item kind; the user states it.
    kindRequired?: boolean;
    suggestedKind?: TrackResyncKind;
    // The provider asked to wait this long after an earlier failure.
    retryAfterSeconds?: number;
    // The reference predates merge tracking; a merge could have moved it
    // here, so the user should check provider and item.
    originUnverified?: boolean;
};

export type TrackResyncResult = {
    trailId: string;
    provider: string;
    externalId: string;
    kind: string;
    // Set when the track was stored but a follow-up such as the search
    // index update failed.
    warning?: string;
    // The trail fields the resync changed, to update a local copy in place.
    track: {
        gpx: string;
        distance: number;
        elevation_gain: number;
        elevation_loss: number;
        duration: number;
        lat: number;
        lon: number;
        polyline: string;
        min_lat: number;
        max_lat: number;
        min_lon: number;
        max_lon: number;
        bounding_box_diagonal: number;
        updated: string;
    };
};

export type TrackResyncPreviewIdentity = Pick<
    TrackResyncPreview,
    "provider" | "externalId"
>;

// Tells whether the track of an imported trail can be fetched again from its plugin.
export async function plugin_track_resync_preview(
    trailId: string,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    const r = await f("/api/v1/plugin-system/track-resync/preview", {
        method: "POST",
        body: JSON.stringify({ trailId }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as TrackResyncPreview;
}

// Fetches the track again from the plugin and replaces it, immediately.
export async function plugin_track_resync(
    trailId: string,
    expected: TrackResyncPreviewIdentity,
    kind?: TrackResyncKind,
    f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch,
) {
    if (!expected.provider || !expected.externalId) {
        throw new Error("Track resync preview has no provider identity");
    }
    const r = await f("/api/v1/plugin-system/track-resync", {
        method: "POST",
        body: JSON.stringify({
            trailId,
            expectedProvider: expected.provider,
            expectedExternalId: expected.externalId,
            ...(kind ? { kind } : {}),
        }),
    });

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail);
    }

    return (await r.json()) as TrackResyncResult;
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
