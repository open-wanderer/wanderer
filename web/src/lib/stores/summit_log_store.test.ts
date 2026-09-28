import { describe, expect, it } from "vitest";
import { buildFilterText } from "./summit_log_store";

describe("summit log filters", () => {
    it("includes the entire end date", () => {
        const filter = buildFilterText({
            startDate: "2026-07-27",
            endDate: "2026-07-27",
            category: [],
            subcategory: [],
        });

        expect(filter).toBe(
            "date>='2026-07-27'&&date<'2026-07-28'",
        );
    });
});
