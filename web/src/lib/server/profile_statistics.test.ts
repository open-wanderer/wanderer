import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import { describe, expect, it } from "vitest";
import {
    buildCompletedTrailFilter,
    buildSummitLogStatisticsFilter,
    mergeStatisticActivities,
} from "./profile_statistics";

describe("profile statistics", () => {
    it("adds a completed trail without summit logs", () => {
        const trail = completedTrail();

        const activities = mergeStatisticActivities([], [trail]);

        expect(activities).toHaveLength(1);
        expect(activities[0]).toMatchObject({
            id: trail.id,
            date: trail.completed_at,
            source: "completed_trail",
            collectionId: "trails",
            distance: trail.distance,
        });
        expect(activities[0].expand?.trail).toBe(trail);
    });

    it("ignores completed_at when the trail has a summit log", () => {
        const trail = completedTrail();
        trail.expand = {
            summit_logs_via_trail: [summitLog(trail.id!)],
        };

        const activities = mergeStatisticActivities(
            [summitLog(trail.id!)],
            [trail],
        );

        expect(activities).toHaveLength(1);
        expect(activities[0].source).toBe("summit_log");
    });

    it("builds an inclusive date range for timestamped completions", () => {
        const filter = buildCompletedTrailFilter("actor0000000001", {
            startDate: "2026-08-01",
            endDate: "2026-08-31",
            category: ["category0000001"],
            subcategory: [],
        });

        expect(filter).toContain("completed_at>='2026-08-01'");
        expect(filter).toContain("completed_at<'2026-09-01'");
        expect(filter).toContain("'category0000001'~category");
    });

    it("combines broad categories and selected subcategories", () => {
        const filter = buildSummitLogStatisticsFilter(
            {
                category: ["category0000001", "category0000002"],
                subcategory: ["subcategory001"],
            },
            [
                {
                    id: "subcategory001",
                    category: "category0000002",
                    name: "Alpine",
                },
            ],
        );

        expect(filter).toContain("'category0000001'~trail.category");
        expect(filter).toContain("'subcategory001'~trail.subcategory");
        expect(filter).not.toContain(
            "'category0000001,category0000002'~trail.category",
        );
    });

    it("supports filtering trails without a subcategory", () => {
        const filter = buildCompletedTrailFilter(
            "actor000000001",
            {
                category: ["category0000001"],
                subcategory: ["__no_subcategory__:category0000001"],
            },
        );

        expect(filter).toContain(
            "category='category0000001'&&subcategory=''",
        );
    });
});

function completedTrail(): Trail & {
    collectionId: string;
    collectionName: string;
} {
    return {
        id: "trail000000001",
        name: "Completed trail",
        public: true,
        completed: true,
        completed_at: "2026-08-10T10:30:00.000Z",
        distance: 12000,
        elevation_gain: 800,
        elevation_loss: 800,
        duration: 7200,
        photos: [],
        tags: [],
        like_count: 0,
        author: "actor0000000001",
        collectionId: "trails",
        collectionName: "trails",
    };
}

function summitLog(trail: string): SummitLog & {
    collectionId: string;
    collectionName: string;
} {
    return {
        id: "summitlog000001",
        date: "2026-08-11",
        photos: [],
        author: "actor0000000001",
        trail,
        collectionId: "summit_logs",
        collectionName: "summit_logs",
    };
}
