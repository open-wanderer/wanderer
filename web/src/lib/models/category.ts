interface CategoryIntegrationMapping {
    integrationType: string;
    providerCategories: string[];
}

interface Category {
    id: string;
    name: string;
    img: string;
    integrations?: CategoryIntegrationMapping[];
    settings?: Settings | null;
}

interface Settings {
    wp_merge_enabled?: boolean;
    wp_merge_radius?: number;
}

export type { Category, CategoryIntegrationMapping, Settings }
