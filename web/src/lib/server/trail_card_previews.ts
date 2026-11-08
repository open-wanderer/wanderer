import type { RequestEvent } from "@sveltejs/kit";
import type { Trail } from "#lib/models/trail.ts";

export function withoutListPhotoExpands(params: URLSearchParams): URLSearchParams {
    const copy = new URLSearchParams(params);
    copy.set("expand", (copy.get("expand") ?? "").split(",").filter(path => !path.split(".").some(field => field.includes("_assets_via_") || field.startsWith("assets_via_"))).join(","));
    return copy;
}

export async function completeTrailCardPreviews(event: RequestEvent, trails: Trail[]): Promise<void> {
    const ids = [...new Set(trails.map(trail => trail.id).filter((id): id is string => Boolean(id)))];
    const previews: Record<string, { photos: string[]; thumbnail: number }> = {};
    for (let offset = 0; offset < ids.length; offset += 100) {
        const result = await event.locals.pb.send("/assets/trail-previews", {
            method: "GET", query: { ids: ids.slice(offset, offset + 100).join(","), share: event.url.searchParams.get("share") ?? "" },
            signal: event.request.signal,
        }) as { previews: typeof previews };
        Object.assign(previews, result.previews);
    }
    for (const trail of trails) {
        const preview = trail.id ? previews[trail.id] : undefined;
        if (preview) { trail.photos = preview.photos; trail.thumbnail = preview.thumbnail; }
    }
}
