import { describe, expect, it } from "vitest";
import {
    dateInputValue,
    datePeriodRange,
    datePeriodPresetForRange,
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

    it.each([
        ["current_month", { start: "2026-08-01", end: "2026-08-31" }],
        ["current_quarter", { start: "2026-07-01", end: "2026-09-30" }],
        ["current_year", { start: "2026-01-01", end: "2026-12-31" }],
        ["last_12_months", { start: "2025-08-11", end: "2026-08-11" }],
    ] as const)("returns the %s period", (preset, expected) => {
        expect(datePeriodRange(preset, new Date(2026, 7, 11))).toEqual(
            expected,
        );
    });

    it("clamps the rolling period across a leap day", () => {
        expect(datePeriodRange("last_12_months", new Date(2024, 1, 29))).toEqual(
            {
                start: "2023-02-28",
                end: "2024-02-29",
            },
        );
    });

    it("identifies the preset represented by a date range", () => {
        const today = new Date(2026, 7, 11);

        expect(
            datePeriodPresetForRange(
                "2026-07-01",
                "2026-09-30",
                today,
            ),
        ).toBe("current_quarter");
        expect(
            datePeriodPresetForRange(
                "2026-07-02",
                "2026-09-30",
                today,
            ),
        ).toBeUndefined();
    });
});
