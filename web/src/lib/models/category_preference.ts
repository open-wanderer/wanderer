interface UserCategoryPreference {
    id?: string;
    user: string;
    category: string;
    visible?: boolean;
    priority?: number | null;
}

export type { UserCategoryPreference };
