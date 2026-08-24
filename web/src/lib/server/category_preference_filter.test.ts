import type { RequestEvent } from "@sveltejs/kit";
import { describe, expect, it } from "vitest";
import { withTrailPreferenceMeiliFilter } from "./category_preference_filter";

describe("withTrailPreferenceMeiliFilter", () => {
    it("normalizes optional filter entries when no user is authenticated", async () => {
        const filter = await withTrailPreferenceMeiliFilter(eventWithoutUser(), [
            undefined,
            null,
            "",
            "author = actor123",
        ]);

        expect(filter).toEqual(["author = actor123"]);
    });

    it("returns undefined when no usable filter remains", async () => {
        const filter = await withTrailPreferenceMeiliFilter(eventWithoutUser(), [
            undefined,
            null,
            "",
        ]);

        expect(filter).toBeUndefined();
    });
});

function eventWithoutUser() {
    return { locals: {} } as RequestEvent;
}
