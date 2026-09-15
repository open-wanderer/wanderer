import { SummitLog, type SummitLogFilter } from "$lib/models/summit_log";
import { APIError } from "$lib/util/api_util";
import { type AuthRecord, type ListResult } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";
import { isURL, objectToFormData } from "$lib/util/file_util";
import { assets_attach_to_target, assets_delete_removed, type AssetImportOmissionHandler, type CompletedAssetAttachments } from "./asset_store";
import { subcategories } from "./subcategory_store";
import { buildPocketBaseCategoryFilter } from "$lib/util/trail_filter_util";
import { nextDateValue } from "$lib/util/date_util";

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

export async function summit_logs_create(summitLog: SummitLog, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response> = fetch, user?: AuthRecord, onOmitted?: AssetImportOmissionHandler) {
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

    try {
        model.photos = await assets_attach_to_target({
            files: summitLog._photos,
            assetIds: summitLog._assetLinks,
            pluginLinks: summitLog._assetPluginLinks,
            target: {
                trail: model.trail ?? summitLog.trail,
                summit_log: model.id,
            },
            existingPhotos: model.photos,
            onOmitted,
            f,
        });
    } catch (cause) {
        const attachedPhotos: string[] = cause instanceof APIError ? cause.detail?.attachedPhotos ?? [] : [];
        model.photos = [...new Set([...(model.photos ?? []), ...attachedPhotos])];
        throw savedSummitLogFailure(cause, summitLog, model, remainingSummitLogAttachments(summitLog, cause), [
            ...new Set([...(summitLog.photos ?? []), ...model.photos]),
        ]);
    }

    return applySavedSummitLog(summitLog, model, {});
}

function summitLogGPXFile(summitLog: SummitLog): File | Blob | undefined {
    if (summitLog._gpx) {
        return summitLog._gpx;
    }
    if (summitLog.expand?.gpx_data) {
        return new Blob([summitLog.expand.gpx_data], { type: "text/xml" });
    }
}

export async function summit_logs_update(oldSummitLog: SummitLog, newSummitLog: SummitLog, onOmitted?: AssetImportOmissionHandler) {
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
    try {
        model.photos = await assets_attach_to_target({
            files: newSummitLog._photos,
            assetIds: newSummitLog._assetLinks,
            pluginLinks: newSummitLog._assetPluginLinks,
            target: {
                trail: model.trail ?? newSummitLog.trail,
                summit_log: model.id,
            },
            existingPhotos: newSummitLog.photos,
            onOmitted,
        });
    } catch (cause) {
        const attachedPhotos: string[] = cause instanceof APIError ? cause.detail?.attachedPhotos ?? [] : [];
        model.photos = [...new Set([...(oldSummitLog.photos ?? []), ...attachedPhotos])];
        throw savedSummitLogFailure(cause, newSummitLog, model, remainingSummitLogAttachments(newSummitLog, cause), [
            ...new Set([...(newSummitLog.photos ?? []), ...attachedPhotos]),
        ]);
    }
    const selectedPhotos = [...model.photos];
    try {
        await assets_delete_removed(oldSummitLog.photos, selectedPhotos, { summit_log: model.id });
    } catch (cause) {
        model.photos = [...new Set([...(oldSummitLog.photos ?? []), ...selectedPhotos])];
        throw savedSummitLogFailure(cause, newSummitLog, model, {}, selectedPhotos);
    }

    return applySavedSummitLog(newSummitLog, model, {});
}

type RemainingSummitLogAttachments = Pick<SummitLog, "_photos" | "_assetLinks" | "_assetPluginLinks">;

function remainingSummitLogAttachments(log: SummitLog, error: unknown): RemainingSummitLogAttachments {
    const completed: CompletedAssetAttachments | undefined = error instanceof APIError
        ? error.detail?.completedAttachments
        : undefined;
    return {
        _photos: completed?.files ? undefined : log._photos,
        _assetLinks: completed?.assetIds ? undefined : log._assetLinks,
        _assetPluginLinks: log._assetPluginLinks?.map((link) => ({
            ...link,
            assetIds: link.assetIds.filter((assetId) => !completed?.pluginLinks.some(
                (savedLink) => savedLink.pluginId === link.pluginId && savedLink.assetIds.includes(assetId),
            )),
        })).filter((link) => link.assetIds.length),
    };
}

function applySavedSummitLog(pending: SummitLog, saved: SummitLog, remaining: RemainingSummitLogAttachments, selectedPhotos = saved.photos): SummitLog {
    const persisted = {
        ...saved,
        photos: [...(saved.photos ?? [])],
        expand: { ...pending.expand, ...saved.expand },
    };
    delete persisted._photos;
    delete persisted._assetLinks;
    delete persisted._assetPluginLinks;
    delete persisted._gpx;
    Object.assign(pending, persisted, { photos: [...(selectedPhotos ?? [])] });
    for (const key of ["_photos", "_assetLinks", "_assetPluginLinks", "_gpx"] as const) {
        delete pending[key];
    }
    if (remaining._photos?.length) pending._photos = remaining._photos;
    if (remaining._assetLinks?.length) pending._assetLinks = remaining._assetLinks;
    if (remaining._assetPluginLinks?.length) pending._assetPluginLinks = remaining._assetPluginLinks;
    return persisted;
}

function savedSummitLogFailure(cause: unknown, pending: SummitLog, saved: SummitLog, remaining: RemainingSummitLogAttachments, selectedPhotos: string[]): APIError {
    const error = cause instanceof APIError ? cause : new APIError(502, "asset_import_failed", { cause });
    // Text, date and GPX have already been stored. Keep the entry and its ID;
    // only outstanding photo requests belong in the next save attempt.
    const persisted = applySavedSummitLog(pending, saved, remaining, selectedPhotos);
    error.detail = {
        ...error.detail,
        savedSummitLog: persisted,
        remainingAttachments: { _gpx: undefined, _photos: remaining._photos, _assetLinks: remaining._assetLinks, _assetPluginLinks: remaining._assetPluginLinks },
        remainingPhotos: [...selectedPhotos],
    };
    return error;
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
    const clauses: string[] = [];
    const categoryFilter = buildPocketBaseCategoryFilter(
        filter,
        get(subcategories),
        "trail",
    );
    if (categoryFilter) {
        clauses.push(categoryFilter);
    }

    if (filter.startDate) {
        clauses.push(`date>='${filter.startDate}'`);
    }

    if (filter.endDate) {
        clauses.push(`date<'${nextDateValue(filter.endDate)}'`);
    }

    if (filter.trail) {
        if (isURL(filter.trail)) {
            clauses.push(`(trail='${filter.trail}'||trail.iri='${filter.trail}'||trail='${filter.trail.substring(filter.trail.length - 15)}')`);
        } else {
            clauses.push(`trail='${filter.trail}'`);
        }
    }

    return clauses.join("&&");

}
