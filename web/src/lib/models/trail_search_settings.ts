export type TrailSearchSection = "general" | "advanced";

export const trailSearchFilters = [
    { id: "activity", label: "categories", icon: "fa-person-hiking", section: "general" },
    { id: "location", label: "trail-search-place", icon: "fa-location-dot", section: "general" },
    { id: "distance", label: "distance", icon: "fa-route", section: "general" },
    { id: "difficulty", label: "difficulty", icon: "fa-mountain", section: "general" },
    { id: "tags", label: "tags", icon: "fa-tags", section: "advanced" },
    { id: "author", label: "author", icon: "fa-user", section: "advanced", requiresUser: true },
    { id: "visibility", label: "visibilty-status", icon: "fa-eye", section: "advanced", requiresUser: true },
    { id: "completed", label: "completion-status", icon: "fa-circle-check", section: "general" },
    { id: "gain", label: "elevation-gain", icon: "fa-arrow-trend-up", section: "advanced" },
    { id: "loss", label: "elevation-loss", icon: "fa-arrow-trend-down", section: "advanced" },
    { id: "date", label: "date", icon: "fa-calendar", section: "advanced" },
    { id: "liked", label: "like-status", icon: "fa-heart", section: "advanced" },
] as const;

export type TrailSearchFilterId = (typeof trailSearchFilters)[number]["id"];
export type TrailSearchFilterSections = Partial<Record<TrailSearchFilterId, TrailSearchSection>>;

/** Older accounts and newly introduced filters retain the standard placement. */
export function normalizeTrailSearchFilterSections(value?: unknown): Record<TrailSearchFilterId, TrailSearchSection> {
    const stored = value && typeof value === "object" ? value as Record<string, unknown> : {};
    return Object.fromEntries(trailSearchFilters.map(filter => [
        filter.id,
        stored[filter.id] === "general" || stored[filter.id] === "advanced" ? stored[filter.id] : filter.section,
    ])) as Record<TrailSearchFilterId, TrailSearchSection>;
}
