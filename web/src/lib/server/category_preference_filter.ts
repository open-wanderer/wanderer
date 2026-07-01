import type { UserCategoryPreference } from "$lib/models/category_preference";
import type { UserSubcategoryPreference } from "$lib/models/subcategory_preference";
import { Collection } from "$lib/util/api_util";
import type { RequestEvent } from "@sveltejs/kit";

type MeiliFilter = string | string[] | undefined;

type TrailPreferenceCache = {
    categories?: Promise<UserCategoryPreference[]>;
    subcategories?: Promise<UserSubcategoryPreference[]>;
};

const preferenceCache = new WeakMap<RequestEvent, TrailPreferenceCache>();

function quotedList(ids: string[]) {
    return `[${ids.map((id) => `'${id}'`).join(", ")}]`;
}

function meiliFilterParts(filter: unknown): string[] {
    if (!filter) {
        return [];
    }
    if (typeof filter === "string") {
        return filter.trim() ? [filter] : [];
    }
    if (Array.isArray(filter)) {
        return filter.flatMap((item) => meiliFilterParts(item));
    }

    return [];
}

function isIdOnlyDetailQuery(parts: string[]) {
    return parts.length > 0 && parts.every((part) => /\bid\s+IN\b/i.test(part));
}

async function userCategoryPreferences(event: RequestEvent) {
    if (!event.locals.user) {
        return [];
    }

    const cache = cachedTrailPreferences(event);
    cache.categories ??= event.locals.pb
        .collection(Collection.user_category_preferences)
        .getFullList<UserCategoryPreference>({
            filter: event.locals.pb.filter("user = {:user}", {
                user: event.locals.user.id,
            }),
            requestKey: null,
        });

    return cache.categories;
}

async function userSubcategoryPreferences(event: RequestEvent) {
    if (!event.locals.user) {
        return [];
    }

    const cache = cachedTrailPreferences(event);
    cache.subcategories ??= event.locals.pb
        .collection(Collection.user_subcategory_preferences)
        .getFullList<UserSubcategoryPreference>({
            filter: event.locals.pb.filter("user = {:user}", {
                user: event.locals.user.id,
            }),
            requestKey: null,
        });

    return cache.subcategories;
}

function cachedTrailPreferences(event: RequestEvent) {
    let cache = preferenceCache.get(event);
    if (!cache) {
        cache = {};
        preferenceCache.set(event, cache);
    }

    return cache;
}

async function trailPreferenceExclusions(event: RequestEvent) {
    const [preferences, subcategoryPreferences] = await Promise.all([
        userCategoryPreferences(event),
        userSubcategoryPreferences(event),
    ]);

    return {
        hiddenCategoryIds: preferences
            .filter((preference) => preference.visible === false)
            .map((preference) => preference.category),
        hiddenSubcategoryIds: subcategoryPreferences
            .filter((preference) => preference.visible === false)
            .map((preference) => preference.subcategory),
    };
}

export async function withTrailPreferenceMeiliFilter(
    event: RequestEvent,
    filter: MeiliFilter,
): Promise<MeiliFilter> {
    const parts = meiliFilterParts(filter);
    if (!event.locals.user || isIdOnlyDetailQuery(parts)) {
        return filter;
    }

    const { hiddenCategoryIds, hiddenSubcategoryIds } =
        await trailPreferenceExclusions(event);

    const preferenceParts: string[] = [];
    if (hiddenCategoryIds.length) {
        preferenceParts.push(
            `(category_id IS NULL OR category_id NOT IN ${quotedList(hiddenCategoryIds)})`,
        );
    }
    if (hiddenSubcategoryIds.length) {
        preferenceParts.push(
            `(subcategory_id IS NULL OR subcategory_id NOT IN ${quotedList(hiddenSubcategoryIds)})`,
        );
    }

    const nextParts = [...parts, ...preferenceParts];
    if (!nextParts.length) {
        return undefined;
    }

    return nextParts;
}

function pbNotEqualsAll(field: string, ids: string[]) {
    return ids.map((id) => `${field} != "${id}"`).join(" && ");
}

export async function withTrailPreferencePocketBaseFilter(
    event: RequestEvent,
    filter?: string,
) {
    if (!event.locals.user) {
        return filter;
    }

    const { hiddenCategoryIds, hiddenSubcategoryIds } =
        await trailPreferenceExclusions(event);

    const parts = filter?.trim() ? [filter] : [];
    if (hiddenCategoryIds.length) {
        parts.push(
            `(category = "" || (${pbNotEqualsAll("category", hiddenCategoryIds)}))`,
        );
    }
    if (hiddenSubcategoryIds.length) {
        parts.push(
            `(subcategory = "" || (${pbNotEqualsAll("subcategory", hiddenSubcategoryIds)}))`,
        );
    }

    return parts.length ? parts.join(" && ") : undefined;
}
