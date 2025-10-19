import { Integration } from "$lib/models/integration";
import { APIError } from "$lib/util/api_util";
import { type ListResult } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";

export const integrations: Writable<Integration[]> = writable([])

export async function integrations_index(f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f('/api/v1/integration' + new URLSearchParams({
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedIntegrations: ListResult<Integration> = await r.json();

    integrations.set(fetchedIntegrations.items);

    return fetchedIntegrations.items;
}

export async function integrations_create(integration: Integration) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    integration.user = user.id;

    let r = await fetch('/api/v1/integration', {
        method: 'PUT',
        body: JSON.stringify(integration),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: Integration = await r.json();

    return model;
}

export async function uploadGpx(integrationName: string, file: File) {
    const formData = new FormData();
    formData.append('file', file);

    let r = await fetch(`/api/v1/integration/${integrationName}/upload`, {
        method: 'POST',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}

export async function integrations_update(integration: Integration, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {
    let r = await f('/api/v1/integration/' + integration.id, {
        method: 'POST',
        body: JSON.stringify(integration),
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: Integration = await r.json();

    return model;
}

export async function integrations_delete(integration: Integration) {
    const r = await fetch('/api/v1/integration/' + integration.id, {
        method: 'DELETE',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }
}