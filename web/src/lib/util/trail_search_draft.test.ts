import { describe, expect, it } from "vitest";
import type { TrailFilter } from "$lib/models/trail";
import { fromDraft, toDraft } from "./trail_search_draft";

const filter: TrailFilter = {
    q: "Bern", category: ["hiking"], subcategory: ["alpine"], tags: ["forest"], difficulty: [1],
    author: "author1", public: true, private: false, shared: false,
    near: { lat: 46, lon: 7, radius: 25_000, name: "Bern" },
    distanceMin: 0, distanceMax: 50_000, distanceLimit: 50_000,
    elevationGainMin: 0, elevationGainMax: 2_000, elevationGainLimit: 2_000,
    elevationLossMin: 0, elevationLossMax: 2_000, elevationLossLimit: 2_000,
    sort: "created", sortOrder: "+",
};

describe("trail search drafts", () => {
    it("keeps draft edits separate from the applied criteria", () => {
        const applied = structuredClone(filter);
        const draft = toDraft(applied);
        draft.category.push("walking");
        draft.subcategory.push("ridge");
        draft.tags.push("alpine");
        draft.difficulty.push(2);
        draft.near.radius = 1_000;
        expect(applied).toEqual(filter);
        expect(draft).not.toHaveProperty("sort");
        expect(draft).not.toHaveProperty("sortOrder");
    });

    it("publishes pending criteria with the latest external sorting", () => {
        const draft = toDraft(filter);
        draft.q = "Grat";
        const current = { ...filter, sort: "name" as const, sortOrder: "-" as const };
        const applied = fromDraft(draft, current);
        expect(applied).toEqual({ ...current, q: "Grat" });
        draft.category.push("walking");
        draft.near.radius = 1_000;
        expect(applied.category).toEqual(["hiking"]);
        expect(applied.near.radius).toBe(25_000);
    });

    it("does not treat an external sort change as a change to search criteria", () => {
        const draft = toDraft(filter);
        expect(toDraft({ ...filter, sort: "distance", sortOrder: "-" })).toEqual(draft);
    });
});
