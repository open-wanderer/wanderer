import { trails_get_bounding_box } from "#lib/stores/trail_store.ts";
import { loadTrailSearch } from "#lib/util/trail_search_load.ts";
import type { Load } from "@sveltejs/kit";

export const load: Load = async ({ url, fetch }) => {
    const [search, boundingBox] = await Promise.all([
        loadTrailSearch(url, fetch),
        trails_get_bounding_box(fetch),
    ]);
    return { ...search, boundingBox, filter: { ...search.filter, sort: "created", sortOrder: "-" } };
};
