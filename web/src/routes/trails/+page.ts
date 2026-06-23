import type { TrailFilter } from "$lib/models/trail";
import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { trails_get_filter_values } from "$lib/stores/trail_store";
import { normalizeCategoryName } from "$lib/util/category_util";
import type { Load } from "@sveltejs/kit";

export const load: Load = async ({ url, fetch }) => {
    const filterValues = await trails_get_filter_values(fetch);

    const filter: TrailFilter = {
        q: "",
        category: [],
        subcategory: [],
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
        elevationLossLimit: filterValues.max_elevation_loss,
        sort: "created",
        sortOrder: "+",
    };
    const categories = await categories_index(fetch)
    const subcategories = await subcategories_index(fetch)

    const paramAuthor = url.searchParams.get("author");
    if (paramAuthor) {
        filter.author = paramAuthor;
        filter.shared = undefined;
    }

    const paramCategory = url.searchParams.get("category");
    if (paramCategory) {
        const normalizedCategory = normalizeCategoryName(paramCategory);
        const category = categories.find(
            (item) =>
                item.id === paramCategory ||
                normalizeCategoryName(item.name) === normalizedCategory,
        );
        if (category) {
            filter.category.push(category.id);
        }
    }

    const paramSubcategory = url.searchParams.get("subcategory");
    if (paramSubcategory) {
        const normalizedSubcategory = normalizeCategoryName(paramSubcategory);
        const subcategory = subcategories.find(
            (item) =>
                item.id === paramSubcategory ||
                normalizeCategoryName(item.name) === normalizedSubcategory,
        );
        if (subcategory) {
            filter.subcategory.push(subcategory.id);
        }
    }

    await category_preferences_index(fetch)
    await subcategory_preferences_index(fetch)

    return {
        categories,
        filter: filter
    };
};
