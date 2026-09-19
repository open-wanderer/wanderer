import { describe, expect, it, vi } from "vitest";
import type { RoutingControl } from "$lib/models/routing";
import {
    cloneRoutingControlValues,
    formatRoutingControlUnit,
    parseFiniteRoutingControlNumber,
    routingControlBucketTranslationKey,
    routingControlTranslationKey,
} from "./routing_control_util";

function control(overrides: Partial<RoutingControl> = {}): RoutingControl {
    return {
        key: "speedPreference",
        type: "number",
        target: "preference",
        ...overrides,
    };
}

describe("routing controls", () => {
    it("clones reactive-style proxy values through the JSON boundary", () => {
        const nested = new Proxy({ profile: "gravel" }, {});
        const values = new Proxy<Record<string, unknown>>(
            { costing: nested },
            {},
        );

        expect(() => structuredClone(values)).toThrow();
        const cloned = cloneRoutingControlValues(values);

        expect(cloned).toEqual({ costing: { profile: "gravel" } });
        expect(cloned).not.toBe(values);
        expect(cloned.costing).not.toBe(nested);
    });

    it("parses complete finite number drafts without coercing empty input", () => {
        expect(parseFiniteRoutingControlNumber("-1.25")).toBe(-1.25);
        expect(parseFiniteRoutingControlNumber(" 2.5 ")).toBe(2.5);
        expect(parseFiniteRoutingControlNumber("1e3")).toBe(1000);
        expect(parseFiniteRoutingControlNumber(3.75)).toBe(3.75);

        expect(parseFiniteRoutingControlNumber("")).toBeUndefined();
        expect(parseFiniteRoutingControlNumber("   ")).toBeUndefined();
        expect(parseFiniteRoutingControlNumber("-")).toBeUndefined();
        expect(parseFiniteRoutingControlNumber("Infinity")).toBeUndefined();
        expect(parseFiniteRoutingControlNumber(Number.NaN)).toBeUndefined();
        expect(parseFiniteRoutingControlNumber(null)).toBeUndefined();
        expect(parseFiniteRoutingControlNumber("not-a-number")).toBeUndefined();
    });

    it("routes standard labels through the client translations", () => {
        expect(routingControlTranslationKey(control())).toBe(
            "routing-control-speedPreference",
        );
        expect(
            routingControlTranslationKey(
                control({ key: "bicycle_type", target: "native_config" }),
            ),
        ).toBeUndefined();
    });

    it("restores localized buckets for canonical preferences", () => {
        const hills = control({ key: "hillPreference" });
        expect(routingControlBucketTranslationKey(hills, 0)).toBe(
            "routing-preference-hills-none",
        );
        expect(routingControlBucketTranslationKey(hills, 0.25)).toBe(
            "routing-preference-hills-low",
        );
        expect(routingControlBucketTranslationKey(hills, 0.5)).toBe(
            "routing-preference-hills-medium",
        );
        expect(routingControlBucketTranslationKey(hills, 0.75)).toBe(
            "routing-preference-hills-high",
        );
    });

    it("converts speed controls from their declared source unit", () => {
        const formatUserSpeed = vi.fn((value: number) => `${value.toFixed(3)} m/s`);

        expect(
            formatRoutingControlUnit(
                control({ unit: "km/h" }),
                5.1,
                formatUserSpeed,
            ),
        ).toBe("1.417 m/s");
        expect(formatUserSpeed).toHaveBeenCalledWith(5.1 / 3.6);

        expect(
            formatRoutingControlUnit(
                control({ unit: "m/s" }),
                5.1,
                formatUserSpeed,
            ),
        ).toBe("5.100 m/s");
    });

    it("shows opaque native units without provider-specific logic", () => {
        expect(
            formatRoutingControlUnit(
                control({ key: "penalty", target: "native_config", unit: "%" }),
                25,
                () => "unused",
            ),
        ).toBe("25 %");
    });
});
