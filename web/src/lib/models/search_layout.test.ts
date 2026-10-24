import { describe, expect, it } from "vitest";
import { normalizeSearchLayout, searchPositionForView } from "./search_layout";

describe("search layout preferences", () => {
    it("uses usable defaults for absent, obsolete and malformed preferences", () => {
        for (const value of [undefined, null, [], "left", { position: "bottom", expansion: "unknown" }]) {
            expect(normalizeSearchLayout(value)).toEqual({ position: "top", expansion: "overlay" });
        }
        expect(normalizeSearchLayout({ position: "right", expansion: "push" })).toEqual({ position: "right", expansion: "push" });
    });

    it("retains the chosen side for wide trail views while maps and narrow windows use the top", () => {
        for (const position of ["left", "right"] as const) {
            for (const view of ["cards", "list", "table"]) {
                expect(searchPositionForView(position, view, true)).toBe(position);
                expect(searchPositionForView(position, view, false)).toBe("top");
            }
            expect(searchPositionForView(position, "map", true)).toBe("top");
        }
    });
});
