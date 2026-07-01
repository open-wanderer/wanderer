interface UserSubcategoryPreference {
    id?: string;
    user: string;
    subcategory: string;
    visible?: boolean;
    priority?: number | null;
}

export type { UserSubcategoryPreference };
