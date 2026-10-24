import { describe, expect, it } from "vitest";
import { consumeTrailSearchShortcut, findTrailSearchShortcut, trailSearchAuthorHandle } from "./trail_search_shortcuts";
import type { ActorSearchResult } from "$lib/models/activitypub/actor";

describe("search filter capture shortcuts", () => {
    it.each([
        ["@", "author", ""],
        ["Grat @anna", "author", "anna"],
        ["@anna@example.org", "author", "anna@example.org"],
        ["#", "tag", ""],
        ["Grat #alpin", "tag", "alpin"],
    ])("recognizes only the capture expression at the caret: %s", (value, kind, query) => {
        expect(findTrailSearchShortcut(value)).toMatchObject({ kind, query });
    });

    it.each(["", "Grat", "in:", "Grat in:St. Gallen", "IN: Bern", "name@example.org", "C#", "https://example.org/in:Bern", "@anna weiter", "#alpin weiter"])(
        "leaves ordinary text alone: %s", value => expect(findTrailSearchShortcut(value)).toBeUndefined(),
    );

    it("uses the caret and ignores a non-empty text selection", () => {
        const value = "Grat @anna Schluss";
        expect(findTrailSearchShortcut(value, 10)).toEqual({ kind: "author", query: "anna", start: 5, end: 10 });
        expect(findTrailSearchShortcut(value, 5, 10)).toBeUndefined();
        expect(findTrailSearchShortcut(value, -1)).toBeUndefined();
    });

    it("removes a confirmed expression without consuming neighboring text or shortcuts", () => {
        const value = "Grat @anna #alpin";
        expect(consumeTrailSearchShortcut(value, findTrailSearchShortcut(value, 10)!))
            .toEqual({ value: "Grat #alpin", caret: 5 });
        expect(consumeTrailSearchShortcut("#alpin", findTrailSearchShortcut("#alpin")!))
            .toEqual({ value: "", caret: 0 });
    });

    it("distinguishes local and federated author handles", () => {
        const author = { preferred_username: "anna", domain: "example.org", is_local: true } as ActorSearchResult;
        expect(trailSearchAuthorHandle(author)).toBe("@anna");
        expect(trailSearchAuthorHandle({ ...author, is_local: false })).toBe("@anna@example.org");
    });
});
