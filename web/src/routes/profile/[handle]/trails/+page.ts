import type { TrailFilter } from "$lib/models/trail";
import { profile_trails_index } from "$lib/stores/profile_store";
import { categories_index } from "$lib/stores/category_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { error, type Load } from "@sveltejs/kit";

export const load: Load = async ({ params, fetch, parent }) => {
    if (!params.handle) {
        error(404, "Not found")
    }
    const { actor } = await parent()

    const filter: TrailFilter = {
        q: "",
        category: [],
        subcategory: [],
        tags: [],
        difficulty: [0, 1, 2],
        author: actor.id,
        public: true,
        shared: true,
        liked: false,
        private: true,
        near: {
            radius: 2000,
        },
        distanceMin: 0,
        distanceMax: 0,
        distanceLimit: 0,
        elevationGainMin: 0,
        elevationGainMax: 0,
        elevationGainLimit: 0,
        elevationLossMin: 0,
        elevationLossMax: 0,
        elevationLossLimit: 0,
        sort: "created",
        sortOrder: "+",
    };
    try {
        await categories_index(fetch)
        await subcategories_index(fetch)
        await subcategory_preferences_index(fetch)
        const trails = await profile_trails_index(params.handle, filter, 1, 12, fetch)
        return { trails, filter }
    } catch (e) {
        return { trails: { hits: {}, items: [], page: 1, totalPages: 0 }, filter }
    }
};
