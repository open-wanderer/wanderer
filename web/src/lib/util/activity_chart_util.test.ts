import { describe, expect, it } from "vitest";
import {
    activityChartBucketKey,
    activityChartBuckets,
    activityChartDataUnit,
    activityChartTickUnit,
    dateAxisValue,
} from "./activity_chart_util";

describe("activity chart utilities", () => {
    it("uses increasingly coarse data units for longer periods", () => {
        expect(activityChartDataUnit("2026-08-01", "2026-08-31")).toBe(
            "day",
        );
        expect(activityChartDataUnit("2026-01-01", "2026-12-31")).toBe(
            "month",
        );
        expect(activityChartDataUnit("2020-01-01", "2026-12-31")).toBe(
            "quarter",
        );
        expect(activityChartDataUnit("2010-01-01", "2026-12-31")).toBe(
            "year",
        );
    });

    it("uses coarser tick labels before aggregating the data", () => {
        expect(activityChartTickUnit("2026-08-01", "2026-08-10")).toBe(
            "day",
        );
        expect(activityChartTickUnit("2026-08-01", "2026-08-31")).toBe(
            "month",
        );
        expect(activityChartTickUnit("2025-01-01", "2026-12-31")).toBe(
            "quarter",
        );
    });

    it("builds complete buckets and clips partial boundary months", () => {
        expect(
            activityChartBuckets("2026-01-15", "2026-03-04", "month"),
        ).toEqual([
            {
                key: "2026-01-01",
                start: "2026-01-15",
                end: "2026-01-31",
                position:
                    (dateAxisValue("2026-01-15") +
                        dateAxisValue("2026-01-31")) /
                    2,
            },
            {
                key: "2026-02-01",
                start: "2026-02-01",
                end: "2026-02-28",
                position:
                    (dateAxisValue("2026-02-01") +
                        dateAxisValue("2026-02-28")) /
                    2,
            },
            {
                key: "2026-03-01",
                start: "2026-03-01",
                end: "2026-03-04",
                position:
                    (dateAxisValue("2026-03-01") +
                        dateAxisValue("2026-03-04")) /
                    2,
            },
        ]);
    });

    it("maps dates to calendar quarters", () => {
        expect(activityChartBucketKey("2026-08-11", "quarter")).toBe(
            "2026-07-01",
        );
    });
});
