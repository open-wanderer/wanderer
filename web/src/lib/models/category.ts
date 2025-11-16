interface Category {
    id: string;
    name: string;
    img: string;
    settings: Settings;
}

interface Settings {
    wp_merge_radius: number;
}

export type {Category}
export type {Settings}