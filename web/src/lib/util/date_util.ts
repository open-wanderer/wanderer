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

export function nextDateValue(value: string): string {
    const date = parseDateValue(value);
    date.setDate(date.getDate() + 1);
    return dateInputValue(date);
}
