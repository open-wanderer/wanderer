import type { TrailFilter } from "$lib/models/trail";

type TrailSearchSort = Pick<TrailFilter, "sort" | "sortOrder">;

// Only search criteria belong in a draft. External changes to these criteria
// must pass through publishFilter so an open editor stays in sync.
export type TrailSearchDraft = Omit<TrailFilter, "sort" | "sortOrder">;

export function toDraft(filter: TrailSearchDraft & Partial<TrailSearchSort>): TrailSearchDraft {
    const { sort, sortOrder, ...criteria } = filter;
    return {
        ...criteria,
        category: [...criteria.category],
        subcategory: [...criteria.subcategory],
        tags: [...criteria.tags],
        difficulty: [...criteria.difficulty],
        near: { ...criteria.near },
    };
}

export function fromDraft(draft: TrailSearchDraft, filter: TrailSearchSort): TrailFilter {
    return { ...toDraft(draft), sort: filter.sort, sortOrder: filter.sortOrder };
}
