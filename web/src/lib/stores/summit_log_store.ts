import { SummitLog, type SummitLogFilter } from "#lib/models/summit_log.ts";
import { APIError } from "#lib/util/api_util.ts";
import { type AuthRecord, type ListResult } from "pocketbase";
import { get, writable, type Writable } from "svelte/store";
import { currentUser } from "./user_store";
import { isURL, objectToFormData } from "#lib/util/file_util.ts";
import { assetIdFromPhotoURL, replaceCopiedAssets, shownAssetLinks } from "#lib/util/asset_link_util.ts";
import { assets_attach_to_target, assets_delete_removed, type AssetRemovalDeferral, type CompletedAssetAttachments } from "./asset_store";
import { subcategories } from "./subcategory_store";
import { buildPocketBaseCategoryFilter } from "#lib/util/trail_filter_util.ts";
import { nextDateValue } from "#lib/util/date_util.ts";
import { federationSaveBatchID, withFederationSaveBatch } from "#lib/util/federation_save_util.ts";

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

    const formData = objectToFormData(summitLog, ["expand", "photos", "_photos", "_gpx", "_assetLinks"])

    const gpx = summitLogGPXFile(summitLog);
    if (gpx) {
        formData.append("gpx", gpx)
    }


    const batchId = federationSaveBatchID();
    let r = await f('/api/v1/summit-log/form?' + new URLSearchParams({
        expand: "author,summit_log_assets_via_summit_log.asset",
        federationBatch: batchId,
    }), {
        method: 'PUT',
        body: formData,
    })

    if (!r.ok) {
        const response = await r.json();
        const failure = new APIError(r.status, response.message, response.detail);

        throw failure;
    }

    let model: SummitLog = await r.json();

    return withFederationSaveBatch("summit-log", model.id, () => attachCreatedSummitLogPhotos(summitLog, model, f), f, batchId, true);
}

async function attachCreatedSummitLogPhotos(summitLog: SummitLog, model: SummitLog, f: (url: RequestInfo | URL, config?: RequestInit) => Promise<Response>) {
    try {
        model.photos = await assets_attach_to_target({
            files: summitLog._photos,
            assetIds: shownAssetLinks(summitLog._assetLinks, summitLog.photos),
            onLinked: (assets) => replaceCopiedAssets(summitLog, assets),
            target: {
                trail: model.trail ?? summitLog.trail,
                summit_log: model.id,
            },
            existingPhotos: model.photos,
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

export async function summit_logs_update(oldSummitLog: SummitLog, newSummitLog: SummitLog, deferRemovedPhotos?: AssetRemovalDeferral, deferComplete?: (complete: () => Promise<void>) => void) {
    const user = get(currentUser)
    if (!user) {
        throw Error("Unauthenticated")
    }

    return withFederationSaveBatch("summit-log", newSummitLog.id, () => updateSummitLogWithAssets(oldSummitLog, newSummitLog, user, deferRemovedPhotos), undefined, undefined, false, deferComplete);
}

async function updateSummitLogWithAssets(oldSummitLog: SummitLog, newSummitLog: SummitLog, user: NonNullable<AuthRecord>, deferRemovedPhotos?: AssetRemovalDeferral) {
    const previousPhotos = [...(oldSummitLog.photos ?? [])];
    newSummitLog.author = user.actor

    const formData = objectToFormData(newSummitLog, ["expand", "gpx", "photos", "_photos", "_gpx", "_assetLinks"])

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
            assetIds: shownAssetLinks(newSummitLog._assetLinks, newSummitLog.photos),
            onLinked: (assets) => replaceCopiedAssets(newSummitLog, assets),
            target: {
                trail: model.trail ?? newSummitLog.trail,
                summit_log: model.id,
            },
            existingPhotos: newSummitLog.photos ?? oldSummitLog.photos,
        });
    } catch (cause) {
        const attachedPhotos: string[] = cause instanceof APIError ? cause.detail?.attachedPhotos ?? [] : [];
        model.photos = [...new Set([...(oldSummitLog.photos ?? []), ...savedAttachedPhotos(attachedPhotos, newSummitLog, cause)])];
        throw savedSummitLogFailure(cause, newSummitLog, model, remainingSummitLogAttachments(newSummitLog, cause), [
            ...new Set([...(newSummitLog.photos ?? []), ...attachedPhotos]),
        ]);
    }
    const selectedPhotos = [...model.photos];
    const removePhotos = async () => {
        try {
            await assets_delete_removed(previousPhotos, selectedPhotos, { summit_log: model.id });
        } catch (cause) {
            model.photos = [...new Set([...previousPhotos, ...selectedPhotos])];
            throw savedSummitLogFailure(cause, newSummitLog, model, {}, selectedPhotos);
        }

    };
    if (deferRemovedPhotos) deferRemovedPhotos(removePhotos);
    else await removePhotos();

    return applySavedSummitLog(newSummitLog, model, {});
}

// The attached photos start with those the form already had. The ones linked
// by ID are saved only once linking completed: listed earlier, dropping one
// before the retry would send a removal, and that deletes a photo linked by ID
// without other links.
function savedAttachedPhotos(attachedPhotos: string[], log: SummitLog, error: unknown): string[] {
    const linked = error instanceof APIError && error.detail?.completedAttachments?.assetIds;
    const unlinked = new Set(linked ? [] : log._assetLinks);
    return attachedPhotos.filter((photo) => !unlinked.has(assetIdFromPhotoURL(photo) ?? ""));
}

type RemainingSummitLogAttachments = Pick<SummitLog, "_photos" | "_assetLinks">;

function remainingSummitLogAttachments(log: SummitLog, error: unknown): RemainingSummitLogAttachments {
    const completed: CompletedAssetAttachments | undefined = error instanceof APIError
        ? error.detail?.completedAttachments
        : undefined;
    return {
        _photos: completed?.files ? undefined : log._photos?.slice(completed?.uploadedFiles ?? 0),
        _assetLinks: completed?.assetIds ? undefined : log._assetLinks,
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
    delete persisted._gpx;
    Object.assign(pending, persisted, { photos: [...(selectedPhotos ?? [])] });
    for (const key of ["_photos", "_assetLinks", "_gpx"] as const) {
        delete pending[key];
    }
    if (remaining._photos?.length) pending._photos = remaining._photos;
    if (remaining._assetLinks?.length) pending._assetLinks = remaining._assetLinks;
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
        remainingAttachments: { _gpx: undefined, _photos: remaining._photos, _assetLinks: remaining._assetLinks },
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
