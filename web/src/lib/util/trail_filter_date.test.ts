import { describe, expect, it } from "vitest";
import { trailFilterDateBoundary } from "./trail_filter_util";

describe("trail filter UTC calendar boundaries", () => {
    it.each([
        ["2026-09-07", "2026-09-08"],
        ["2024-02-29", "2024-03-01"],
        ["2026-04-30", "2026-05-01"],
        ["2026-12-31", "2027-01-01"],
        ["0000-02-29", "0000-03-01"],
        ["0099-12-31", "0100-01-01"],
        ["9999-12-31", "+010000-01-01"],
    ])("spans the UTC day from %s to %s", (day, followingDay) => {
        expect(trailFilterDateBoundary(day)).toBe(
            Date.parse(`${day}T00:00:00Z`) / 1000,
        );
        expect(trailFilterDateBoundary(day, true)).toBe(
            Date.parse(`${followingDay}T00:00:00Z`) / 1000,
        );
    });

    it.each([
        undefined,
        "",
        "invalid",
        "2026-2-01",
        "2026-02-1",
        "2026-00-10",
        "2026-13-01",
        "2026-01-00",
        "2026-04-31",
        "2026-02-29",
        "2026-02-30",
        "1900-02-29",
        "2026-09-07T08:00:00Z",
        "2026-09-07Z",
        " 2026-09-07",
        "2026-09-07 ",
    ])("omits invalid or absent calendar input %s", (value) => {
        expect(trailFilterDateBoundary(value)).toBeUndefined();
        expect(trailFilterDateBoundary(value, true)).toBeUndefined();
    });
});
