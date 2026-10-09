export function isToday(date: Date) {
    const today = new Date();
    return date.setHours(0, 0, 0, 0) == today.setHours(0, 0, 0, 0)
}

export function dateExistsInList(targetDate: Date, dateList: Date[]): boolean {
    const targetYear = targetDate.getFullYear();
    const targetMonth = targetDate.getMonth();
    const targetDay = targetDate.getDate();

    for (const date of dateList) {
        const year = date.getFullYear();
        const month = date.getMonth();
        const day = date.getDate();

        if (year === targetYear && month === targetMonth && day === targetDay) {
            return true;
        }
    }

    return false;
}

export function isSameDay(d1: Date, d2: Date) {
    return d1.getFullYear() === d2.getFullYear() &&
        d1.getMonth() === d2.getMonth() &&
        d1.getDate() === d2.getDate();
}

export function dateInputValue(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
}

export function parseDateValue(value: string): Date {
    const [datePart] = value.split(/[T ]/, 1);
    const [year, month, day] = datePart.split("-").map(Number);
    return new Date(year, month - 1, day);
}

export function monthDateRange(date: Date): { start: string; end: string } {
    const year = date.getFullYear();
    const month = date.getMonth();
    return {
        start: dateInputValue(new Date(year, month, 1)),
        end: dateInputValue(new Date(year, month + 1, 0)),
    };
}

export function calendarMonthForDateRange(
    start: string,
    end: string,
    today: Date = new Date(),
): { start: string; end: string } {
    const currentMonth = monthDateRange(today);
    const currentMonthOverlapsRange =
        currentMonth.start <= end && currentMonth.end >= start;

    return currentMonthOverlapsRange
        ? currentMonth
        : monthDateRange(parseDateValue(start));
}

export type DatePeriodPreset =
    | "current_month"
    | "current_quarter"
    | "current_year"
    | "last_12_months";

export const datePeriodPresets: DatePeriodPreset[] = [
    "current_month",
    "current_quarter",
    "current_year",
    "last_12_months",
];

export function datePeriodRange(
    preset: DatePeriodPreset,
    today: Date = new Date(),
): { start: string; end: string } {
    const year = today.getFullYear();
    const month = today.getMonth();

    switch (preset) {
        case "current_month":
            return monthDateRange(today);
        case "current_quarter": {
            const quarterStartMonth = Math.floor(month / 3) * 3;
            return {
                start: dateInputValue(new Date(year, quarterStartMonth, 1)),
                end: dateInputValue(new Date(year, quarterStartMonth + 3, 0)),
            };
        }
        case "current_year":
            return {
                start: dateInputValue(new Date(year, 0, 1)),
                end: dateInputValue(new Date(year, 11, 31)),
            };
        case "last_12_months": {
            const startYear = year - 1;
            const lastDayInStartMonth = new Date(
                startYear,
                month + 1,
                0,
            ).getDate();
            const start = new Date(
                startYear,
                month,
                Math.min(today.getDate(), lastDayInStartMonth),
            );
            return {
                start: dateInputValue(start),
                end: dateInputValue(today),
            };
        }
    }
}

export function datePeriodPresetForRange(
    start: string | undefined,
    end: string | undefined,
    today: Date = new Date(),
): DatePeriodPreset | undefined {
    if (!start || !end) {
        return undefined;
    }

    return datePeriodPresets.find((preset) => {
        const range = datePeriodRange(preset, today);
        return range.start === start && range.end === end;
    });
}

export function nextDateValue(value: string): string {
    const date = parseDateValue(value);
    date.setDate(date.getDate() + 1);
    return dateInputValue(date);
}

// EXIF GPSDateStamp ("YYYY:MM:DD") and GPSTimeStamp ([h, m, s]) are always UTC,
// unlike DateTimeOriginal, which is camera-local time.
export function convertGPSTimestampToDate(dateStamp: unknown, timeStamp: unknown): Date | undefined {
    if (typeof dateStamp !== "string" || !Array.isArray(timeStamp) || timeStamp.length !== 3) {
        return undefined;
    }
    const [year, month, day] = dateStamp.split(":").map(Number);
    const [hours, minutes, seconds] = timeStamp.map(Number);
    const time = Date.UTC(year, month - 1, day, hours, minutes) + seconds * 1000;
    return Number.isFinite(time) ? new Date(time) : undefined;
}

// EXIF DateTimeOriginal ("YYYY:MM:DD HH:MM:SS") is camera-local time, so it
// only gives an exact time together with OffsetTimeOriginal ("+02:00").
export function convertDateTimeOriginalToDate(dateTime: unknown, offset: unknown): Date | undefined {
    if (typeof dateTime !== "string" || typeof offset !== "string") {
        return undefined;
    }
    const date = dateTime.trim().match(/^(\d{4}):(\d{2}):(\d{2}) (\d{2}):(\d{2}):(\d{2})$/);
    const zone = offset.trim().match(/^([+-])(\d{2}):(\d{2})$/);
    if (!date || !zone) {
        return undefined;
    }
    const [year, month, day, hours, minutes, seconds] = date.slice(1).map(Number);
    const offsetMinutes = (zone[1] === "-" ? -1 : 1) * (Number(zone[2]) * 60 + Number(zone[3]));
    const time = Date.UTC(year, month - 1, day, hours, minutes - offsetMinutes, seconds);
    return Number.isFinite(time) ? new Date(time) : undefined;
}
