export type ActivityChartTimeUnit = "day" | "month" | "quarter" | "year";

export type ActivityChartBucket = {
    key: string;
    start: string;
    end: string;
    position: number;
};

export const millisecondsPerDay = 24 * 60 * 60 * 1000;

export function dateAxisValue(value: string): number {
    return Date.parse(value) / millisecondsPerDay;
}

function dateValueFromAxis(value: number): string {
    return new Date(value * millisecondsPerDay).toISOString().slice(0, 10);
}

function periodDayCount(start: string, end: string): number {
    return Math.max(0, dateAxisValue(end) - dateAxisValue(start) + 1);
}

export function activityChartDataUnit(
    start: string,
    end: string,
): ActivityChartTimeUnit {
    const days = periodDayCount(start, end);
    if (days <= 100) {
        return "day";
    }
    if (days <= 3 * 366) {
        return "month";
    }
    if (days <= 10 * 366) {
        return "quarter";
    }
    return "year";
}

export function activityChartTickUnit(
    start: string,
    end: string,
): ActivityChartTimeUnit {
    const days = periodDayCount(start, end);
    if (days <= 14) {
        return "day";
    }
    if (days <= 18 * 31) {
        return "month";
    }
    if (days <= 3 * 366) {
        return "quarter";
    }
    return "year";
}

export function activityChartBucketKey(
    value: string,
    unit: ActivityChartTimeUnit,
): string {
    const date = new Date(value);
    const year = date.getUTCFullYear();
    const month = date.getUTCMonth();

    switch (unit) {
        case "day":
            return value.slice(0, 10);
        case "month":
            return dateValueFromAxis(
                Date.UTC(year, month, 1) / millisecondsPerDay,
            );
        case "quarter":
            return dateValueFromAxis(
                Date.UTC(year, Math.floor(month / 3) * 3, 1) /
                    millisecondsPerDay,
            );
        case "year":
            return `${year}-01-01`;
    }
}

function calendarBucketEnd(
    key: string,
    unit: ActivityChartTimeUnit,
): string {
    const date = new Date(key);
    const year = date.getUTCFullYear();
    const month = date.getUTCMonth();

    switch (unit) {
        case "day":
            return key;
        case "month":
            return dateValueFromAxis(
                Date.UTC(year, month + 1, 0) / millisecondsPerDay,
            );
        case "quarter":
            return dateValueFromAxis(
                Date.UTC(year, month + 3, 0) / millisecondsPerDay,
            );
        case "year":
            return `${year}-12-31`;
    }
}

export function activityChartBuckets(
    start: string,
    end: string,
    unit: ActivityChartTimeUnit,
): ActivityChartBucket[] {
    if (start > end) {
        return [];
    }

    const buckets: ActivityChartBucket[] = [];
    let cursor = start;

    while (cursor <= end) {
        const key = activityChartBucketKey(cursor, unit);
        const bucketEnd = calendarBucketEnd(key, unit);
        const effectiveEnd = bucketEnd < end ? bucketEnd : end;
        buckets.push({
            key,
            start: cursor,
            end: effectiveEnd,
            position:
                (dateAxisValue(cursor) + dateAxisValue(effectiveEnd)) / 2,
        });
        cursor = dateValueFromAxis(dateAxisValue(effectiveEnd) + 1);
    }

    return buckets;
}
