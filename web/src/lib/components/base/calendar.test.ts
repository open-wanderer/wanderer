import { render } from "svelte/server";
import { addMessages, init } from "svelte-i18n";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { StatisticActivity } from "$lib/models/statistic_activity";
import Calendar from "./calendar.svelte";

beforeAll(() => {
    addMessages("en", {
        activity: "{n, plural, =1 {Activity} other {Activities}}",
        calendar: {
            weekdays: [
                "Monday",
                "Tuesday",
                "Wednesday",
                "Thursday",
                "Friday",
                "Saturday",
                "Sunday",
            ],
        },
    });
    init({
        fallbackLocale: "en",
        initialLocale: "en",
        formats: {
            date: { monthName: { month: "long" } },
            number: {},
            time: {},
        },
    });
});

describe("Calendar", () => {
    beforeEach(() => {
        vi.useFakeTimers({ toFake: ["Date"] });
        vi.setSystemTime(new Date(2027, 0, 15, 12));
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it("renders the requested month and activity counts even in another year", () => {
        const activity = {
            date: "2026-09-15",
        } as StatisticActivity;

        const { body } = render(Calendar, {
            props: {
                month: "2026-09-01",
                activities: [activity],
                onclick: () => {},
            },
        });

        expect(body).toContain(
            'aria-label="Tuesday, September 15, 2026: 1 Activity"',
        );
    });

    it("defaults to the current month when no month is provided", () => {
        const { body } = render(Calendar, {
            props: {
                activities: [{ date: "2027-01-15" } as StatisticActivity],
                onclick: () => {},
            },
        });

        expect(body).toContain(
            'aria-label="Friday, January 15, 2027: 1 Activity"',
        );
    });
});
