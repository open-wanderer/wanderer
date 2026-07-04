import { SummitLog, type SummitLogFilter } from "$lib/models/summit_log";
import { APIError } from "$lib/util/api_util";
import { type AuthRecord, type ListResult } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";
import { isURL, objectToFormData } from "$lib/util/file_util";
import { asset_photo_url, assets_create, assets_delete_removed, assets_import_plugin_links, assets_link } from "./asset_store";

export const summitLog: Writable<SummitLog> = writable(new SummitLog(new Date().toISOString().substring(0, 10)));
export const summitLogs: Writable<SummitLog[]> = writable([]);

export async function summit_logs_index(filter?: SummitLogFilter, handle?: string, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch) {

    const r = await f('/api/v1/summit-log?' + new URLSearchParams({
        ...(filter ? { filter: buildFilterText(filter) } : {}),
        perPage: "-1",
        expand: "trail.category,trail.subcategory,trail.subcategory.category,author,summit_log_assets_via_summit_log.asset",
        sort: "+date",
        ...(handle ? { handle } : {})
    }), {
        method: 'GET',
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const fetchedSummitLogs: ListResult<SummitLog> = await r.json();

    summitLogs.set(fetchedSummitLogs.items);

    return fetchedSummitLogs;
}

export async function summit_logs_create(summitLog: SummitLog, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord) {
    user ??= get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    summitLog.author = user.actor

    const formData = objectToFormData(summitLog, ["expand", "photos", "_photos", "_gpx", "_assetLinks", "_assetPluginLinks"])

    const gpx = summitLogGPXFile(summitLog);
    if (gpx) {
        formData.append("gpx", gpx)
    }


    let r = await f('/api/v1/summit-log/form?' + new URLSearchParams({
        expand: "author,summit_log_assets_via_summit_log.asset"
    }), {
        method: 'PUT',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    let model: SummitLog = await r.json();

    if (summitLog._photos?.length) {
        const assets = await assets_create(summitLog._photos, {
            summit_log: model.id,
        }, f);
        model.photos = [...(model.photos ?? []), ...assets.map(asset_photo_url)];
    }
    if (summitLog._assetLinks?.length) {
        const assets = await assets_link(summitLog._assetLinks, {
            summit_log: model.id,
        }, f);
        model.photos = [...(model.photos ?? []), ...assets.map(asset_photo_url)];
    }
    if (summitLog._assetPluginLinks?.length) {
        const assets = await assets_import_plugin_links(summitLog._assetPluginLinks, {
            trail: model.trail,
            summit_log: model.id,
        }, f);
        model.photos = [...(model.photos ?? []), ...assets.map(asset_photo_url)];
    }

    return model;
}

function summitLogGPXFile(summitLog: SummitLog): File | Blob | undefined {
    if (summitLog._gpx) {
        return summitLog._gpx;
    }
    if (summitLog.expand?.gpx_data) {
        return new Blob([summitLog.expand.gpx_data], { type: "text/xml" });
    }
}

export async function summit_logs_update(oldSummitLog: SummitLog, newSummitLog: SummitLog) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    newSummitLog.author = user.actor

    const formData = objectToFormData(newSummitLog, ["expand", "gpx", "photos", "_photos", "_gpx", "_assetLinks", "_assetPluginLinks"])

    if (newSummitLog._gpx) {
        formData.append("gpx", newSummitLog._gpx);
    } else if (newSummitLog.gpx === "") {
        formData.append("gpx", "");
    }

    let r = await fetch('/api/v1/summit-log/form/' + newSummitLog.id + '?' + new URLSearchParams({
        expand: "author,summit_log_assets_via_summit_log.asset"
    }), {
        method: 'POST',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    const model: SummitLog = await r.json();
    await assets_delete_removed(oldSummitLog.photos, newSummitLog.photos, {
        summit_log: model.id,
    });
    if (newSummitLog._photos?.length) {
        const assets = await assets_create(newSummitLog._photos, {
            summit_log: model.id,
        });
        model.photos = uniquePhotoURLs([...(newSummitLog.photos ?? []), ...assets.map(asset_photo_url)]);
    }
    if (newSummitLog._assetLinks?.length) {
        const assets = await assets_link(newSummitLog._assetLinks, {
            summit_log: model.id,
        });
        model.photos = uniquePhotoURLs([...(model.photos ?? newSummitLog.photos ?? []), ...assets.map(asset_photo_url)]);
    }
    if (newSummitLog._assetPluginLinks?.length) {
        const assets = await assets_import_plugin_links(newSummitLog._assetPluginLinks, {
            trail: model.trail,
            summit_log: model.id,
        });
        model.photos = uniquePhotoURLs([...(model.photos ?? newSummitLog.photos ?? []), ...assets.map(asset_photo_url)]);
    }

    return model;
}

function uniquePhotoURLs(photos: string[]): string[] {
    return [...new Set(photos)];
}

export async function summit_logs_delete(summitLog: SummitLog) {
    const r = await fetch('/api/v1/summit-log/' + summitLog.id, {
        method: 'DELETE',
    })
    if (!r.ok) {
        const response = await r.json();
        throw new APIError(r.status, response.message, response.detail)
    }

    return await r.json();

}

export function buildFilterText(filter: SummitLogFilter,): string {
    let filterText: string = "";

    if (filter.category.length > 0) {
        filterText += `trail.category!=null&&'${filter.category.join(",")}'~trail.category`;
    }

    if (filter.startDate) {
        filterText += `${filter.category.length ? '&&' : ''}date>='${filter.startDate}'`
    }

    if (filter.endDate) {
        filterText += `${filter.category.length || filter.startDate ? '&&' : ''}date<='${filter.endDate}'`
    }

    if (filter.trail) {
        if (filter.category.length || filter.startDate || filter.endDate) {
            filterText += "&&"
        }
        if (isURL(filter.trail)) {
            filterText += `trail='${filter.trail}'||trail.iri='${filter.trail}'||trail='${filter.trail.substring(filter.trail.length - 15)}'`;
        } else {
            filterText += `trail='${filter.trail}'`
        }
    }

    return filterText;

}
