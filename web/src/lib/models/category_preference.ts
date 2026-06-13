interface UserCategoryPreference {
    id?: string;
    user: string;
    category: string;
    exclude_search?: boolean;
    hide_design?: boolean;
    exclude_federated?: boolean;
    priority?: number | null;
}

export type { UserCategoryPreference };
