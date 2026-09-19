import { afterEach, describe, expect, it, vi } from "vitest";
import { trailFilterDateBoundary } from "./trail_filter_util";

describe("trail filter calendar boundaries", () => {
    afterEach(() => vi.unstubAllEnvs());

    it.each([
        ["UTC", "2026-09-07", "2026-09-07T00:00:00Z", "2026-09-08T00:00:00Z"],
        ["Europe/Zurich", "2026-09-07", "2026-09-06T22:00:00Z", "2026-09-07T22:00:00Z"],
        ["America/New_York", "2026-09-07", "2026-09-07T04:00:00Z", "2026-09-08T04:00:00Z"],
        ["Asia/Kathmandu", "2026-09-07", "2026-09-06T18:15:00Z", "2026-09-07T18:15:00Z"],
        ["Europe/Zurich", "2026-03-29", "2026-03-28T23:00:00Z", "2026-03-29T22:00:00Z"],
        ["Europe/Zurich", "2026-10-25", "2026-10-24T22:00:00Z", "2026-10-25T23:00:00Z"],
        ["America/New_York", "2026-03-08", "2026-03-08T05:00:00Z", "2026-03-09T04:00:00Z"],
        ["America/New_York", "2026-11-01", "2026-11-01T04:00:00Z", "2026-11-02T05:00:00Z"],
        ["Australia/Lord_Howe", "2026-04-05", "2026-04-04T13:00:00Z", "2026-04-05T13:30:00Z"],
        ["Australia/Lord_Howe", "2026-10-04", "2026-10-03T13:30:00Z", "2026-10-04T13:00:00Z"],
    ])("resolves the whole day in %s on %s", (timezone, day, start, end) => {
        vi.stubEnv("TZ", timezone);

        expect(trailFilterDateBoundary(day)).toBe(Date.parse(start) / 1000);
        expect(trailFilterDateBoundary(day, true)).toBe(Date.parse(end) / 1000);
    });

    it("ends at the next midnight when DST skips the selected day's midnight", () => {
        vi.stubEnv("TZ", "America/Santiago");

        expect(trailFilterDateBoundary("2026-09-06")).toBe(
            Date.parse("2026-09-06T04:00:00Z") / 1000,
        );
        expect(trailFilterDateBoundary("2026-09-06", true)).toBe(
            Date.parse("2026-09-07T03:00:00Z") / 1000,
        );
    });

    it("includes both occurrences when DST repeats midnight", () => {
        vi.stubEnv("TZ", "America/Havana");

        expect(trailFilterDateBoundary("2026-11-01")).toBe(
            Date.parse("2026-11-01T04:00:00Z") / 1000,
        );
        expect(trailFilterDateBoundary("2026-11-01", true)).toBe(
            Date.parse("2026-11-02T05:00:00Z") / 1000,
        );
    });

    it("keeps a skipped local calendar day as an empty interval", () => {
        vi.stubEnv("TZ", "Pacific/Apia");
        const skipBoundary = Date.parse("2011-12-30T10:00:00Z") / 1000;

        expect(trailFilterDateBoundary("2011-12-29", true)).toBe(skipBoundary);
        expect(trailFilterDateBoundary("2011-12-30")).toBe(skipBoundary);
        expect(trailFilterDateBoundary("2011-12-30", true)).toBe(skipBoundary);
        expect(trailFilterDateBoundary("2011-12-31")).toBe(skipBoundary);
    });

    it.each([
        ["2024-02-29", "2024-03-01"],
        ["2026-04-30", "2026-05-01"],
        ["2026-12-31", "2027-01-01"],
        ["0000-02-29", "0000-03-01"],
        ["0099-12-31", "0100-01-01"],
        ["9999-12-31", "+010000-01-01"],
    ])("advances the calendar from %s to %s", (day, followingDay) => {
        vi.stubEnv("TZ", "UTC");

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
