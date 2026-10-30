import { loadTrailSearch } from "#lib/util/trail_search_load.ts";
import type { Load } from "@sveltejs/kit";

export const load: Load = ({ url, fetch }) => loadTrailSearch(url, fetch);
