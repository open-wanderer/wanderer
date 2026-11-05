import { describe, expect, it } from "vitest";
import { normalizeSearchLayout, searchPositionForViewport } from "./search_layout";

describe("search layout preferences", () => {
    it("uses usable defaults for absent, obsolete and malformed preferences", () => {
        for (const value of [undefined, null, [], "left", { position: "bottom", expansion: "unknown" }]) {
            expect(normalizeSearchLayout(value)).toEqual({ position: "top", expansion: "overlay" });
        }
        expect(normalizeSearchLayout({ position: "right", expansion: "push" })).toEqual({ position: "right", expansion: "push" });
    });

    it("retains the chosen side only in wide viewports", () => {
        for (const position of ["left", "right"] as const) {
            expect(searchPositionForViewport(position, true)).toBe(position);
            expect(searchPositionForViewport(position, false)).toBe("top");
        }
    });
});
