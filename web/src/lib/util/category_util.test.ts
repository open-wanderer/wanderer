import { describe, expect, it } from "vitest";
import { displayTrailCategoryLabel } from "./category_util";

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
