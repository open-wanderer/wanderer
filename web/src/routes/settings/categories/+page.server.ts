import { Collection } from "$lib/util/api_util";
import type { ServerLoad } from "@sveltejs/kit";

type TrailCategoryUsageRecord = {
    category?: string;
    subcategory?: string;
};

function increment(counts: Record<string, number>, id?: string) {
    if (!id) {
        return;
    }

    counts[id] = (counts[id] ?? 0) + 1;
}

export const load: ServerLoad = async ({ locals }) => {
    const trailUsage = {
        categories: {} as Record<string, number>,
        subcategories: {} as Record<string, number>,
    };

    if (!locals.user?.actor) {
        return { trailUsage };
    }

    const trails = await locals.pb
        .collection(Collection.trails)
        .getFullList<TrailCategoryUsageRecord>({
            filter: locals.pb.filter("author = {:author}", {
                author: locals.user.actor,
            }),
            fields: "category,subcategory",
            requestKey: null,
        });

    for (const trail of trails) {
        increment(trailUsage.categories, trail.category);
        increment(trailUsage.subcategories, trail.subcategory);
    }

    return { trailUsage };
};
