import { describe, expect, it } from "vitest";
import { visibleComboboxItems } from "./combobox_util";

const item = (text: string, value: unknown = null) => ({ text, value });

describe("visibleComboboxItems", () => {
    const suggestions = [item("wildflowers"), item("wild camping"), item("winter")];

    it("leaves out items that are already selected in multiple mode", () => {
        const result = visibleComboboxItems(suggestions, [item("wildflowers")], true);
        expect(result.map((i) => i.text)).toEqual(["wild camping", "winter"]);
    });

    it("keeps the order and the original item objects", () => {
        const result = visibleComboboxItems(suggestions, [item("wild camping")], true);
        expect(result[0]).toBe(suggestions[0]);
        expect(result[1]).toBe(suggestions[2]);
    });

    it("matches selected items by text, not by value", () => {
        const stored = { id: "abc", name: "wildflowers" };
        const result = visibleComboboxItems(
            [item("wildflowers", { id: "other" }), item("winter")],
            [item("wildflowers", stored)],
            true,
        );
        expect(result.map((i) => i.text)).toEqual(["winter"]);
    });

    it("returns every suggestion when nothing is selected", () => {
        expect(visibleComboboxItems(suggestions, [], true)).toEqual(suggestions);
    });

    it("returns an empty list when every suggestion is already selected", () => {
        const result = visibleComboboxItems(
            suggestions,
            [item("winter"), item("wildflowers"), item("wild camping")],
            true,
        );
        expect(result).toEqual([]);
    });

    it("does not filter in single mode", () => {
        expect(visibleComboboxItems(suggestions, "wildflowers", false)).toEqual(suggestions);
        expect(visibleComboboxItems(suggestions, undefined, false)).toEqual(suggestions);
    });

    it("tolerates a missing value in multiple mode", () => {
        expect(visibleComboboxItems(suggestions, undefined, true)).toEqual(suggestions);
    });
});
