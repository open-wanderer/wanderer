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
    const parts = dateStamp.trim().match(/^(\d{4}):(\d{2}):(\d{2})$/);
    if (!parts) return undefined;
    const [year, month, day] = parts.slice(1).map(Number);
    const [hours, minutes, seconds] = timeStamp.map(Number);
    if (year < 1 || month < 1 || month > 12 || day < 1 || day > 31 ||
        !Number.isInteger(hours) || hours < 0 || hours > 23 ||
        !Number.isInteger(minutes) || minutes < 0 || minutes > 59 ||
        !Number.isFinite(seconds) || seconds < 0 || seconds >= 60) return undefined;
    const date = new Date(0);
    date.setUTCFullYear(year, month - 1, day);
    date.setUTCHours(hours, minutes, 0, 0);
    if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) return undefined;
    date.setTime(date.getTime() + seconds * 1000);
    return date;
}
