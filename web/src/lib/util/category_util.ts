import type { Category } from "$lib/models/category";
import type { UserCategoryPreference } from "$lib/models/category_preference";
import type { Subcategory } from "$lib/models/subcategory";
import type { UserSubcategoryPreference } from "$lib/models/subcategory_preference";

type CategoryDisplayEntity = Pick<Category | Subcategory, "name" | "translations">;
type CategoryShortNameEntity = Pick<
    Category | Subcategory,
    "name" | "short_name" | "translations"
>;

export function normalizeCategoryName(name: string): string {
    // Best-effort mirror of the backend normalization for resolving ?category= links.
    // Full Unicode casefold parity would require a dedicated frontend casefold implementation.
    return name
        .normalize("NFD")
        .replace(/\p{Mn}/gu, "")
        .toLowerCase()
        .replace(/[\s_-]+/g, " ")
        .trim();
}

function localeCandidates(locale?: string | null): string[] {
    if (!locale) {
        return [];
    }

    const normalized = locale.trim();
    if (!normalized) {
        return [];
    }

    const lower = normalized.toLowerCase();
    const base = lower.split("-")[0];

    return [...new Set([normalized, lower, base])];
}

export function displayCategoryName(
    category?: CategoryDisplayEntity | null,
    locale?: string | null,
): string {
    if (!category) {
        return "";
    }

    for (const candidate of localeCandidates(locale)) {
        const translatedName = category.translations?.[candidate]?.name;
        if (translatedName) {
            return translatedName;
        }
    }

    // Fall back to the English translation before the raw canonical name.
    return category.translations?.["en"]?.name || category.name || "";
}

export function displayCategoryShortName(
    category?: CategoryShortNameEntity | null,
    locale?: string | null,
): string {
    if (!category) {
        return "";
    }

    for (const candidate of localeCandidates(locale)) {
        const translatedShortName = category.translations?.[candidate]?.short_name;
        if (translatedShortName?.trim()) {
            return translatedShortName;
        }
    }

    return category.short_name?.trim() || displayCategoryName(category, locale);
}

export function displayCategoryIcon(
    category?: Pick<Category | Subcategory, "icon"> | null,
): string {
    const icon = category?.icon?.trim().replace(/^fa-/, "");
    return icon ? `fa-${icon}` : "fa-shapes";
}

export function displaySubcategoryIcon(
    subcategory?: Pick<Subcategory, "icon"> | null,
    parentCategory?: Pick<Category, "icon"> | null,
): string {
    return displayCategoryIcon(subcategory?.icon ? subcategory : parentCategory);
}

export function displaySubcategoryBadgeIcon(
    subcategory?: Pick<Subcategory, "badge_icon"> | null,
): string {
    const icon = subcategory?.badge_icon?.trim().replace(/^fa-/, "");
    return icon ? `fa-${icon}` : "";
}

type TrailCategoryIconEntity = {
    expand?: {
        category?: Pick<Category, "icon"> | null;
        subcategory?: Pick<Subcategory, "icon" | "badge_icon"> | null;
    } | null;
};

export function displayTrailCategoryIcon(
    trail?: TrailCategoryIconEntity | null,
): string {
    const subcategory = trail?.expand?.subcategory;
    if (subcategory) {
        return displaySubcategoryIcon(subcategory, trail?.expand?.category);
    }
    return displayCategoryIcon(trail?.expand?.category);
}

export function displayTrailCategoryBadgeIcon(
    trail?: TrailCategoryIconEntity | null,
): string {
    return displaySubcategoryBadgeIcon(trail?.expand?.subcategory);
}

export function displaySubcategoryName(
    subcategory?: Subcategory | null,
    locale?: string | null,
): string {
    return displayCategoryName(subcategory, locale);
}

export function displaySubcategoryLabel(
    subcategory?: Subcategory | null,
    locale?: string | null,
): string {
    return displaySubcategoryName(subcategory, locale);
}

export function preferenceForCategory(
    preferences: UserCategoryPreference[],
    categoryId?: string | null,
): UserCategoryPreference | undefined {
    return preferences.find((preference) => preference.category === categoryId);
}

export function preferenceForSubcategory(
    preferences: UserSubcategoryPreference[],
    subcategoryId?: string | null,
): UserSubcategoryPreference | undefined {
    return preferences.find(
        (preference) => preference.subcategory === subcategoryId,
    );
}

export function subcategoryVisible(
    subcategoryId: string | undefined | null,
    preferences: UserSubcategoryPreference[],
): boolean {
    return preferenceForSubcategory(preferences, subcategoryId)?.visible !== false;
}

export function sortedCategoriesByPreference(
    categories: Category[],
    preferences: UserCategoryPreference[],
    locale?: string | null,
): Category[] {
    return [...categories].sort((a, b) => {
        const aPriority = preferenceForCategory(preferences, a.id)?.priority;
        const bPriority = preferenceForCategory(preferences, b.id)?.priority;
        const aPrioritized = typeof aPriority === "number" && aPriority > 0;
        const bPrioritized = typeof bPriority === "number" && bPriority > 0;

        if (aPrioritized && bPrioritized) {
            return aPriority - bPriority;
        }
        if (aPrioritized) {
            return -1;
        }
        if (bPrioritized) {
            return 1;
        }

        return displayCategoryName(a, locale).localeCompare(
            displayCategoryName(b, locale),
            locale ?? undefined,
            { sensitivity: "base" },
        );
    });
}

export function categoryVisibleInDesign(
    category: Category,
    preferences: UserCategoryPreference[],
    currentCategoryId?: string | null,
): boolean {
    if (category.id === currentCategoryId) {
        return true;
    }

    return !preferenceForCategory(preferences, category.id)?.hide_design;
}

export function designSelectableCategories(
    categories: Category[],
    preferences: UserCategoryPreference[],
    locale?: string | null,
    currentCategoryId?: string | null,
): Category[] {
    return sortedCategoriesByPreference(categories, preferences, locale).filter(
        (category) =>
            categoryVisibleInDesign(category, preferences, currentCategoryId),
    );
}
