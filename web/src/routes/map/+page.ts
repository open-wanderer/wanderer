import type { TrailFilter } from "$lib/models/trail";
import { categories_index } from "$lib/stores/category_store";
import { trails_get_bounding_box, trails_get_filter_values } from "$lib/stores/trail_store";
import type { ServerLoad } from "@sveltejs/kit";

export const load: ServerLoad = async ({ params, locals, fetch }) => {
    const boundingBox = await trails_get_bounding_box(fetch);
    const filterValues = await trails_get_filter_values(fetch);

    const filter: TrailFilter = {
        q: "",
        category: [],
        tags: [],
        difficulty: [0, 1, 2],
        author: "",
        public: true,
        shared: true,
        liked: false,
        private: true,
        near: {
            radius: 2000,
        },
        distanceMin: 0,
        distanceMax: filterValues.max_distance,
        distanceLimit: filterValues.max_distance,
        elevationGainMin: 0,
        elevationGainMax: filterValues.max_elevation_gain,
        elevationGainLimit: filterValues.max_elevation_gain,
        elevationLossMin: 0,
        elevationLossMax: filterValues.max_elevation_loss,
        elevationLossLimit: filterValues.max_elevation_gain,
        sort: "created",
        sortOrder: "-",
    };

    await categories_index(fetch)

    return { filter: filter, boundingBox: boundingBox }
};