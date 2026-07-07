import type { SummitLogFilter } from "$lib/models/summit_log";
import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { profile_stats_index } from "$lib/stores/profile_store";
import { error, type Load } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, parent }) => {

    if (!params.handle) {
        error(404, "Not found")
    }

    const date = new Date()
    date.setUTCHours(6)
    const y = date.getFullYear()
    const m = date.getMonth();
    const firstDay = new Date(y, m, 2);
    const lastDay = new Date(y, m + 1, 1);

    await categories_index(fetch)
    await category_preferences_index(fetch)

    const filter: SummitLogFilter = {
        startDate: firstDay.toISOString().slice(0, 10),
        endDate: lastDay.toISOString().slice(0, 10),
        category: []
    }
    try {
        const logs = await profile_stats_index(params.handle, filter, fetch);
        return { filter, logs }

    } catch (e) {
        return { logs: [], filter }
    }
};
