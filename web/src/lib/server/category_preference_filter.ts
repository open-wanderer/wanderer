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

function hasExplicitCategoryFilter(parts: string[]) {
    return parts.some((part) => /\bcategory_id\s*(=|IN)\b/i.test(part));
}

function hasExplicitSubcategoryFilter(parts: string[]) {
    return parts.some((part) => /\bsubcategory_id\s*(=|IN|IS)\b/i.test(part));
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
        excludedSearchIds: preferences
            .filter((preference) => preference.exclude_search)
            .map((preference) => preference.category),
        excludedFederatedIds: preferences
            .filter((preference) => preference.exclude_federated)
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

    const { excludedSearchIds, excludedFederatedIds, hiddenSubcategoryIds } =
        await trailPreferenceExclusions(event);

    const preferenceParts: string[] = [];
    if (excludedSearchIds.length && !hasExplicitCategoryFilter(parts)) {
        preferenceParts.push(
            `(category_id IS NULL OR category_id NOT IN ${quotedList(excludedSearchIds)})`,
        );
    }
    if (excludedFederatedIds.length) {
        preferenceParts.push(
            `NOT (is_federated = true AND category_id IN ${quotedList(excludedFederatedIds)})`,
        );
    }
    if (
        hiddenSubcategoryIds.length &&
        !hasExplicitCategoryFilter(parts) &&
        !hasExplicitSubcategoryFilter(parts)
    ) {
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

function pbEqualsAny(field: string, ids: string[]) {
    return ids.map((id) => `${field} = "${id}"`).join(" || ");
}

function pbNotEqualsAll(field: string, ids: string[]) {
    return ids.map((id) => `${field} != "${id}"`).join(" && ");
}

function hasExplicitPocketBaseCategoryFilter(filter?: string) {
    return /\bcategory\s*(=|~|\?=)\b/i.test(filter ?? "");
}

function hasExplicitPocketBaseSubcategoryFilter(filter?: string) {
    return /\bsubcategory\s*(=|~|\?=)\b/i.test(filter ?? "");
}

export async function withTrailPreferencePocketBaseFilter(
    event: RequestEvent,
    filter?: string,
) {
    if (!event.locals.user) {
        return filter;
    }

    const { excludedSearchIds, excludedFederatedIds, hiddenSubcategoryIds } =
        await trailPreferenceExclusions(event);

    const parts = filter?.trim() ? [filter] : [];
    if (excludedSearchIds.length && !hasExplicitPocketBaseCategoryFilter(filter)) {
        parts.push(`(category = "" || (${pbNotEqualsAll("category", excludedSearchIds)}))`);
    }
    if (excludedFederatedIds.length) {
        parts.push(
            `(author.is_local = true || category = "" || !(${pbEqualsAny("category", excludedFederatedIds)}))`,
        );
    }
    if (
        hiddenSubcategoryIds.length &&
        !hasExplicitPocketBaseCategoryFilter(filter) &&
        !hasExplicitPocketBaseSubcategoryFilter(filter)
    ) {
        parts.push(
            `(subcategory = "" || (${pbNotEqualsAll("subcategory", hiddenSubcategoryIds)}))`,
        );
    }

    return parts.length ? parts.join(" && ") : undefined;
}
