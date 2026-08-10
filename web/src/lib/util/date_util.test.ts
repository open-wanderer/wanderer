import { describe, expect, it } from "vitest";
import {
    dateInputValue,
    monthDateRange,
    nextDateValue,
    parseDateValue,
} from "./date_util";

describe("date utilities", () => {
    it("formats and parses calendar dates without a UTC offset", () => {
        const date = new Date(2026, 7, 10);

        expect(dateInputValue(date)).toBe("2026-08-10");
        expect(parseDateValue("2026-08-10T22:30:00Z")).toEqual(date);
    });

    it("returns the complete selected month", () => {
        expect(monthDateRange(new Date(2024, 1, 15))).toEqual({
            start: "2024-02-01",
            end: "2024-02-29",
        });
    });

    it("advances across month and year boundaries", () => {
        expect(nextDateValue("2026-08-31")).toBe("2026-09-01");
        expect(nextDateValue("2026-12-31")).toBe("2027-01-01");
    });
});
