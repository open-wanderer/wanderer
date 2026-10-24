import { describe, expect, it } from "vitest";
import type { ListFilter } from "$lib/models/list";
import { fromDraft, toDraft } from "./list_search_draft";

const filter: ListFilter = { q: "Bern", author: "author1", public: true, shared: false, sort: "created", sortOrder: "+" };

describe("list search drafts", () => {
    it("keeps criteria separate from sorting and the applied filter", () => {
        const draft = toDraft(filter);
        expect(draft).not.toHaveProperty("sort");
        expect(draft).not.toHaveProperty("sortOrder");
        draft.q = "Grat";
        expect(filter.q).toBe("Bern");
        const current = { ...filter, sort: "name" as const, sortOrder: "-" as const };
        expect(fromDraft(draft, current)).toEqual({ ...current, q: "Grat" });
    });

    it("does not treat an external sort change as a criteria change", () => {
        expect(toDraft({ ...filter, sort: "name", sortOrder: "-" })).toEqual(toDraft(filter));
    });

    it("keeps absent sort preferences absent", () => {
        expect(fromDraft(toDraft(filter), { q: "" }).sort).toBeUndefined();
        expect(fromDraft(toDraft(filter), { q: "" }).sortOrder).toBeUndefined();
    });
});
