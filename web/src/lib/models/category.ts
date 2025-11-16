interface CategoryIntegrationMapping {
    integrationType: string;
    providerCategories: string[];
}

interface Category {
    id: string;
    name: string;
    img: string;
    integrations?: CategoryIntegrationMapping[];
}

export type { Category, CategoryIntegrationMapping }
