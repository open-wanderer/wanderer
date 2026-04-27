interface Category {
    id: string;
    name: string;
    img: string;
    settings?: Settings | null;
}

interface Settings {
    wp_merge_enabled?: boolean;
    wp_merge_radius?: number;
}

export type {Category}
export type {Settings}
