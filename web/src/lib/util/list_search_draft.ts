import type { ListFilter } from "$lib/models/list";

// Only search criteria belong in a draft. External criteria changes must pass
// through the search bar's publish function to keep an open editor in sync.
export type ListSearchDraft = Omit<ListFilter, "sort" | "sortOrder">;

export function toDraft(filter: ListFilter): ListSearchDraft {
    const { sort, sortOrder, ...criteria } = filter;
    return criteria;
}

export function fromDraft(draft: ListSearchDraft, filter: ListFilter): ListFilter {
    return { ...toDraft(draft), sort: filter.sort, sortOrder: filter.sortOrder };
}
