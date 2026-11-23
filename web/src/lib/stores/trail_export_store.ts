import PocketBase from "pocketbase";
import type { Trail } from "#lib/models/trail.ts";
import { APIError } from "#lib/util/api_util.ts";

// Keep the account export scoped to its owner and hold only one response page.
export async function* ownedTrailExportPages(actor: string, photos: boolean, request: typeof fetch = fetch): AsyncGenerator<Trail[]> {
    const filter = new PocketBase().filter("author = {:actor}", { actor });
    const expand = ["author", "category", "subcategory", "subcategory.category", "waypoints_via_trail", "summit_logs_via_trail", "tags"];
    for (let page = 1; ; page++) {
        const response = await request("/api/v1/trail?" + new URLSearchParams({ export: "owned", page: String(page), perPage: "100", sort: "id", filter, expand: expand.join(","), photos: String(photos) }));
        const body = await response.json();
        if (!response.ok) throw new APIError(response.status, body.message, body.detail);
        yield body.items;
        if (body.items.length < 100 || (body.totalPages && page >= body.totalPages)) return;
    }
}
