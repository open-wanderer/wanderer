import type { StatisticActivity } from "$lib/models/statistic_activity";
import type { Subcategory } from "$lib/models/subcategory";
import type { SummitLog } from "$lib/models/summit_log";
import type { Trail } from "$lib/models/trail";
import { nextDateValue } from "$lib/util/date_util";
import {
    buildPocketBaseCategoryFilter,
    isPocketBaseRecordId,
    noSubcategoryFilterCategory,
} from "$lib/util/trail_filter_util";
import { z } from "zod";

type PocketBaseTrail = Trail & {
    collectionId?: string;
    collectionName?: string;
};

type PocketBaseSummitLog = SummitLog & {
    collectionId?: string;
    collectionName?: string;
};

export const ProfileStatisticsFilterSchema = z.object({
    startDate: z.string().date().optional(),
    endDate: z.string().date().optional(),
    category: z.array(
        z.string().refine(isPocketBaseRecordId, "Invalid category filter"),
    ),
    subcategory: z.array(
        z.string().refine((value) => {
            const noSubcategoryId = noSubcategoryFilterCategory(value);
            return (
                isPocketBaseRecordId(value) ||
                (noSubcategoryId !== undefined &&
                    isPocketBaseRecordId(noSubcategoryId))
            );
        }, "Invalid subcategory filter"),
    ),
});

export type ProfileStatisticsFilter = z.infer<
    typeof ProfileStatisticsFilterSchema
>;

export function summitLogToStatisticActivity(
    log: PocketBaseSummitLog,
): StatisticActivity {
    return {
        ...log,
        source: "summit_log",
        collectionId: log.collectionId ?? "summit_logs",
        collectionName: log.collectionName ?? "summit_logs",
    };
}

export function completedTrailToStatisticActivity(
    trail: PocketBaseTrail,
): StatisticActivity {
    return {
        id: trail.id,
        date: trail.completed_at!,
        photos: [],
        distance: trail.distance,
        elevation_gain: trail.elevation_gain,
        elevation_loss: trail.elevation_loss,
        duration: trail.duration,
        gpx: trail.gpx,
        author: trail.author,
        trail: trail.id,
        created: trail.created,
        source: "completed_trail",
        collectionId: trail.collectionId ?? "trails",
        collectionName: trail.collectionName ?? "trails",
        expand: { trail },
    };
}

export function mergeStatisticActivities(
    summitLogs: PocketBaseSummitLog[],
    completedTrails: PocketBaseTrail[],
): StatisticActivity[] {
    const trailsWithLoadedSummitLogs = new Set(
        summitLogs
            .map((log) => log.trail)
            .filter((trailId): trailId is string => Boolean(trailId)),
    );

    return [
        ...summitLogs.map(summitLogToStatisticActivity),
        ...completedTrails
            .filter(
                (trail) =>
                    !trail.id || !trailsWithLoadedSummitLogs.has(trail.id),
            )
            .map(completedTrailToStatisticActivity),
    ].sort((a, b) => a.date.localeCompare(b.date));
}

export function buildCompletedTrailFilter(
    actorId: string,
    filter: ProfileStatisticsFilter,
    subcategories: Subcategory[] = [],
): string {
    const clauses = [
        `author='${actorId}'`,
        "completed=true",
        "completed_at!=''",
    ];

    const categoryFilter = buildPocketBaseCategoryFilter(
        filter,
        subcategories,
    );
    if (categoryFilter) {
        clauses.push(categoryFilter);
    }
    if (filter.startDate) {
        clauses.push(`completed_at>='${filter.startDate}'`);
    }
    if (filter.endDate) {
        clauses.push(`completed_at<'${nextDateValue(filter.endDate)}'`);
    }

    return clauses.join("&&");
}

export function buildSummitLogStatisticsFilter(
    filter: ProfileStatisticsFilter,
    subcategories: Subcategory[] = [],
): string {
    const clauses: string[] = [];
    const categoryFilter = buildPocketBaseCategoryFilter(
        filter,
        subcategories,
        "trail",
    );
    if (categoryFilter) {
        clauses.push(categoryFilter);
    }
    if (filter.startDate) {
        clauses.push(`date>='${filter.startDate}'`);
    }
    if (filter.endDate) {
        clauses.push(`date<'${nextDateValue(filter.endDate)}'`);
    }

    return clauses.join("&&");
}
