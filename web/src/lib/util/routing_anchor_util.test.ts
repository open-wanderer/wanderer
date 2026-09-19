import { describe, expect, it } from "vitest";
import {
    routingAnchorDisplay,
    routingAnchorListEntries,
    routingAnchorTitle,
} from "./routing_anchor_util";

const translate = (key: string) => key;

describe("routing anchor display", () => {
    it("keeps the final anchor as the finish of an open route", () => {
        expect(routingAnchorDisplay(0, 3)).toMatchObject({
            icon: "fa-bullseye",
            number: null,
            isStartAndFinish: false,
        });
        expect(routingAnchorDisplay(1, 3)).toMatchObject({
            icon: "fa-location-dot",
            number: 1,
        });
        expect(routingAnchorDisplay(2, 3)).toMatchObject({
            icon: "fa-flag-checkered",
            number: null,
            isStartAndFinish: false,
        });
    });

    it("identifies the shared map anchor as start and finish", () => {
        expect(routingAnchorDisplay(0, 3, true)).toMatchObject({
            icon: "fa-bullseye",
            number: null,
            isStartAndFinish: true,
        });
        expect(routingAnchorTitle(0, 3, translate, true)).toBe("start / finish");
    });

    it("keeps every other closed-loop anchor as a numbered route point", () => {
        expect(routingAnchorDisplay(1, 3, true)).toMatchObject({
            icon: "fa-location-dot",
            number: 1,
        });
        expect(routingAnchorDisplay(2, 3, true)).toMatchObject({
            icon: "fa-location-dot",
            number: 2,
        });
        expect(routingAnchorTitle(2, 3, translate, true)).toBe("route-point #2");
    });

    it("projects the start anchor again as the final list entry", () => {
        const start = { id: "start" };
        const middle = { id: "middle" };
        const lastIntermediate = { id: "last-intermediate" };
        const entries = routingAnchorListEntries(
            [start, middle, lastIntermediate],
            true,
        );

        expect(entries.map((entry) => entry.anchor)).toEqual([
            start,
            middle,
            lastIntermediate,
            start,
        ]);
        expect(entries.at(-1)).toMatchObject({
            anchorIndex: 0,
            isFinishCopy: true,
        });
        expect(entries.at(-1)?.anchor).toBe(start);
        expect(routingAnchorDisplay(0, entries.length).titleKey).toBe("start");
        expect(routingAnchorDisplay(entries.length - 1, entries.length).titleKey).toBe(
            "finish",
        );
    });

    it("does not duplicate the start of an open route", () => {
        const anchors = [{ id: "start" }, { id: "finish" }];
        expect(routingAnchorListEntries(anchors)).toHaveLength(2);
    });
});
