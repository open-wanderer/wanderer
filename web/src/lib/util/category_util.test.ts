import { describe, expect, it } from "vitest";
import {
    displayTrailCategoryLabel,
    trailCategoryKey,
} from "./category_util";

describe("displayTrailCategoryLabel", () => {
    it("includes the localized subcategory", () => {
        expect(
            displayTrailCategoryLabel(
                {
                    expand: {
                        category: {
                            name: "cycling",
                            translations: { de: { name: "Radfahren" } },
                        },
                        subcategory: {
                            name: "mountain_biking",
                            translations: { de: { name: "Mountainbike" } },
                        },
                    },
                },
                "de",
            ),
        ).toBe("Radfahren / Mountainbike");
    });

    it("keeps the category label for trails without a subcategory", () => {
        expect(
            displayTrailCategoryLabel({
                expand: { category: { name: "Hiking" } },
            }),
        ).toBe("Hiking");
    });
});

describe("trailCategoryKey", () => {
    it("uses stable relation IDs instead of localized labels", () => {
        expect(
            trailCategoryKey({
                category: "category000001",
                subcategory: "subcat00000001",
            }),
        ).toBe("category000001:subcat00000001");
    });

    it("falls back to expanded relation IDs", () => {
        expect(
            trailCategoryKey({
                expand: {
                    category: { id: "category000001" },
                    subcategory: { id: "subcat00000001" },
                },
            }),
        ).toBe("category000001:subcat00000001");
    });

    it("returns a stable key for trails without a category", () => {
        expect(trailCategoryKey()).toBe("-:-");
        expect(trailCategoryKey({})).toBe("-:-");
    });
});
