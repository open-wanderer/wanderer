import {
    Trail,
    defaultTrailDuplicateOptions,
    hasDuplicatePhotos,
    normalizeTrailDuplicateOptions,
    type TrailDuplicateOptions,
} from "$lib/models/trail";
import { categories_index } from "$lib/stores/category_store";
import { category_preferences_index } from "$lib/stores/category_preference_store";
import { lists_index } from "$lib/stores/list_store";
import { subcategory_preferences_index } from "$lib/stores/subcategory_preference_store";
import { subcategories_index } from "$lib/stores/subcategory_store";
import { trails_show } from "$lib/stores/trail_store";
import { currentUser } from "$lib/stores/user_store";
import { designSelectableCategories } from "$lib/util/category_util";
import { error, type Load } from "@sveltejs/kit";
import { get } from "svelte/store";
import { locale } from "svelte-i18n";

export const load: Load = async ({ params, fetch, url }) => {
    const user = get(currentUser)

    if (!params.id) {
        return error(400, "Bad Request")
    }
    const categories = await categories_index(fetch)
    const categoryPreferences = await category_preferences_index(fetch)
    await subcategories_index(fetch)
    await subcategory_preferences_index(fetch)
    const lists = await lists_index({ q: "", author: user?.actor ?? "" }, 1, -1, fetch)

    let trail: Trail;
    let duplicateSourceTrail: Trail | undefined;
    let duplicateOptions: TrailDuplicateOptions | undefined;
    if (params.id === "new") {
        // duplicate trail
        if (url.searchParams.has("orig")) {
            duplicateOptions = duplicateOptionsFromURL(url);
            const originalId = url.searchParams.get("orig")!;
            const originalTrail = await trails_show(
                originalId,
                undefined,
                undefined,
                true,
                fetch,
            );
            if (hasDuplicatePhotos(duplicateOptions)) {
                duplicateSourceTrail = originalTrail;
            }
            trail = Trail.from(originalTrail, user?.actor, duplicateOptions)
        } else {
            const defaultCategory =
                designSelectableCategories(
                    categories,
                    categoryPreferences,
                    get(locale),
                )[0] ?? categories[0];

            trail = defaultCategory
                ? new Trail("", { category: defaultCategory })
                : new Trail("");
        }
    } else {
        trail = await trails_show(
            params.id,
            undefined,
            url.searchParams.get("share") ?? undefined,
            true,
            fetch,
        );
    }

    return {
        trail: trail,
        duplicateSourceTrail,
        duplicateOptions,
        lists: lists,
        categories: categories,
        categoryPreferences: categoryPreferences,
    }
};

function duplicateOptionsFromURL(url: URL): TrailDuplicateOptions {
    return normalizeTrailDuplicateOptions({
        waypoints: copyOption(
            url,
            "copyWaypoints",
            defaultTrailDuplicateOptions.waypoints,
        ),
        summitLogs: copyOption(
            url,
            "copySummitLogs",
            defaultTrailDuplicateOptions.summitLogs,
        ),
        trailPhotos: copyOption(
            url,
            "copyTrailPhotos",
            defaultTrailDuplicateOptions.trailPhotos,
        ),
        waypointPhotos: copyOption(
            url,
            "copyWaypointPhotos",
            defaultTrailDuplicateOptions.waypointPhotos,
        ),
        summitLogPhotos: copyOption(
            url,
            "copySummitLogPhotos",
            defaultTrailDuplicateOptions.summitLogPhotos,
        ),
    });
}

function copyOption(url: URL, name: string, fallback: boolean) {
    const value = url.searchParams.get(name);
    if (value === null) {
        return fallback;
    }

    return value === "true";
}
