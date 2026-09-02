import type { SummitLogFilter } from "$lib/models/summit_log";
import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { profile_stats_index } from "$lib/stores/profile_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { monthDateRange } from "$lib/util/date_util";
import { error, type Load } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, parent }) => {

    if (!params.handle) {
        error(404, "Not found")
    }

    const currentMonth = monthDateRange(new Date());

    await Promise.all([
        categories_index(fetch),
        category_preferences_index(fetch),
        subcategories_index(fetch),
        subcategory_preferences_index(fetch),
    ]);

    const filter: SummitLogFilter = {
        startDate: currentMonth.start,
        endDate: currentMonth.end,
        category: [],
        subcategory: [],
    }
    try {
        const activities = await profile_stats_index(params.handle, filter, fetch);
        return { filter, activities }

    } catch (e) {
        return { activities: [], filter }
    }
};
