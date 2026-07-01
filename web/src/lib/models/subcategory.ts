import type { Category, CategoryTranslation } from "./category";

interface Subcategory {
    id: string;
    category: string;
    name: string;
    short_name?: string | null;
    icon?: string | null;
    badge_icon?: string | null;
    translations?: Record<string, CategoryTranslation> | null;
    expand?: {
        category?: Category;
    };
}

export type { Subcategory };
