interface Category {
    id: string;
    name: string;
    short_name?: string | null;
    icon?: string | null;
    translations?: Record<string, CategoryTranslation> | null;
    settings?: Settings | null;
}

interface CategoryTranslation {
    name?: string;
    short_name?: string;
}

interface Settings {
    wp_merge_enabled?: boolean;
    wp_merge_radius?: number;
}

export type {Category}
export type {CategoryTranslation}
export type {Settings}
