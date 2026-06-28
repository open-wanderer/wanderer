import type { Category } from "$lib/models/category";
import type { UserCategoryPreference } from "$lib/models/category_preference";
import type { Subcategory } from "$lib/models/subcategory";
import type { UserSubcategoryPreference } from "$lib/models/subcategory_preference";

type CategoryDisplayEntity = Pick<Category | Subcategory, "name" | "translations">;
type CategoryShortNameEntity = Pick<
    Category | Subcategory,
    "name" | "short_name" | "translations"
>;

export type CategoryMappingTarget =
    | string
    | {
          category?: string;
          subcategory?: string;
      };

export type ResolvedCategoryMappingTarget = {
    categoryId?: string;
    subcategoryId?: string;
};

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

export function categoryMappingTargetFromUnknown(
    value: unknown,
): CategoryMappingTarget | undefined {
    if (typeof value === "string") {
        return value;
    }
    if (!value || typeof value !== "object" || Array.isArray(value)) {
        return undefined;
    }

    const raw = value as Record<string, unknown>;
    return {
        category: typeof raw.category === "string" ? raw.category : "",
        subcategory: typeof raw.subcategory === "string" ? raw.subcategory : "",
    };
}

function resolveCategoryTargetValue(
    value: string,
    categories: Category[],
    subcategories: Subcategory[],
): ResolvedCategoryMappingTarget {
    const trimmed = value.trim();
    if (!trimmed) {
        return {};
    }

    const directCategory = categories.find((category) => category.id === trimmed);
    if (directCategory) {
        return { categoryId: directCategory.id };
    }

    const directSubcategory = subcategories.find(
        (subcategory) => subcategory.id === trimmed,
    );
    if (directSubcategory) {
        return {
            categoryId: directSubcategory.category,
            subcategoryId: directSubcategory.id,
        };
    }

    if (trimmed.includes("/")) {
        const [rawCategory, rawSubcategory] = trimmed.split("/", 2);
        const category = categories.find(
            (candidate) =>
                normalizeCategoryName(candidate.name) ===
                normalizeCategoryName(rawCategory),
        );
        const subcategory = subcategories.find(
            (candidate) =>
                candidate.category === category?.id &&
                normalizeCategoryName(candidate.name) ===
                    normalizeCategoryName(rawSubcategory),
        );
        if (subcategory) {
            return {
                categoryId: subcategory.category,
                subcategoryId: subcategory.id,
            };
        }
    }

    const namedCategory = categories.find(
        (category) =>
            normalizeCategoryName(category.name) === normalizeCategoryName(trimmed),
    );
    return namedCategory ? { categoryId: namedCategory.id } : {};
}

export function resolveCategoryMappingTarget(
    target: CategoryMappingTarget,
    categories: Category[],
    subcategories: Subcategory[],
): ResolvedCategoryMappingTarget {
    if (typeof target === "string") {
        return resolveCategoryTargetValue(target, categories, subcategories);
    }

    const categoryValue = resolveCategoryTargetValue(
        target.category ?? "",
        categories,
        subcategories,
    );
    const subcategoryValue = resolveCategoryTargetValue(
        target.subcategory ?? "",
        categories,
        subcategories,
    );
    if (subcategoryValue.subcategoryId) {
        return subcategoryValue;
    }

    if (target.subcategory?.trim() && categoryValue.categoryId) {
        const subcategory = subcategories.find(
            (candidate) =>
                candidate.category === categoryValue.categoryId &&
                normalizeCategoryName(candidate.name) ===
                    normalizeCategoryName(target.subcategory ?? ""),
        );
        if (subcategory) {
            return {
                categoryId: subcategory.category,
                subcategoryId: subcategory.id,
            };
        }
    }

    return categoryValue;
}

export function categoryMappingTargetToPickerValue(
    target: CategoryMappingTarget,
    categories: Category[],
    subcategories: Subcategory[],
): string {
    const resolved = resolveCategoryMappingTarget(target, categories, subcategories);
    if (resolved.subcategoryId) {
        return `subcategory:${resolved.subcategoryId}`;
    }
    if (resolved.categoryId) {
        return `category:${resolved.categoryId}`;
    }
    return "";
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

export function displaySubcategoryShortBadge(
    subcategory?: Subcategory | null,
    locale?: string | null,
): string {
    if (!subcategory) {
        return "";
    }

    const shortName = displayCategoryShortName(subcategory, locale);
    const label = displaySubcategoryLabel(subcategory, locale);

    if (shortName && shortName !== label) {
        return shortName.toUpperCase();
    }

    if (subcategory.short_name?.trim()) {
        return subcategory.short_name.trim().toUpperCase();
    }

    const normalizedLabel = label.trim();
    if (normalizedLabel.length <= 5) {
        return normalizedLabel.toUpperCase();
    }

    const words = normalizedLabel.match(/[\p{L}\p{N}]+/gu) ?? [];
    if (words.length > 1) {
        return words
            .map((word) => word.at(0))
            .join("")
            .slice(0, 5)
            .toUpperCase();
    }

    return normalizedLabel.slice(0, 4).toUpperCase();
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

export function sortedSubcategoriesByPreference(
    subcategories: Subcategory[],
    preferences: UserSubcategoryPreference[],
    locale?: string | null,
): Subcategory[] {
    return [...subcategories].sort((a, b) => {
        const aPriority = preferenceForSubcategory(preferences, a.id)?.priority;
        const bPriority = preferenceForSubcategory(preferences, b.id)?.priority;
        const aPrioritized = typeof aPriority === "number" && aPriority > 0;
        const bPrioritized = typeof bPriority === "number" && bPriority > 0;

        if (aPrioritized && bPrioritized && aPriority !== bPriority) {
            return aPriority - bPriority;
        }
        if (aPrioritized && !bPrioritized) {
            return -1;
        }
        if (bPrioritized && !aPrioritized) {
            return 1;
        }

        return displaySubcategoryLabel(a, locale).localeCompare(
            displaySubcategoryLabel(b, locale),
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

    return preferenceForCategory(preferences, category.id)?.visible !== false;
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
