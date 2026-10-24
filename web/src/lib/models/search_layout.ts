export type SearchPosition = "top" | "left" | "right";
export type SearchExpansion = "overlay" | "push";
export type SearchLayout = { position: SearchPosition; expansion: SearchExpansion };

export function normalizeSearchLayout(value: unknown): SearchLayout {
    const stored = value && typeof value === "object" ? value as Partial<SearchLayout> : {};
    return {
        position: stored.position === "left" || stored.position === "right" ? stored.position : "top",
        expansion: stored.expansion === "push" ? "push" : "overlay",
    };
}

export function searchPositionForView(position: SearchPosition, view: string, wide: boolean): SearchPosition {
    return view === "map" || !wide ? "top" : position;
}
