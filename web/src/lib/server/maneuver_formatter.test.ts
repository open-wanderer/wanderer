import { LOCALE_LOADERS, SUPPORTED_LOCALES } from "$lib/i18n/locales";
import type {
    RoutingInternalManeuverResponse,
    RoutingManeuverType,
} from "$lib/models/routing";
import { describe, expect, it } from "vitest";
import {
    formatManeuverResponse,
    resolveManeuverLanguage,
} from "./maneuver_formatter";

const maneuverTypes: RoutingManeuverType[] = [
    "start",
    "destination",
    "continue",
    "turn_left",
    "turn_right",
    "turn_slight_left",
    "turn_slight_right",
    "turn_sharp_left",
    "turn_sharp_right",
    "keep_left",
    "keep_right",
    "uturn_left",
    "uturn_right",
    "uturn",
    "roundabout_enter",
    "roundabout_exit",
    "exit_left",
    "exit_right",
    "ramp_straight",
    "ramp_left",
    "ramp_right",
    "merge",
    "merge_left",
    "merge_right",
    "ferry",
    "unknown",
];

const maneuverCatalogKeys = [
    "maneuver-continue",
    "maneuver-destination",
    "maneuver-exit-left",
    "maneuver-exit-right",
    "maneuver-ferry",
    "maneuver-keep-left",
    "maneuver-keep-right",
    "maneuver-merge",
    "maneuver-merge-left",
    "maneuver-merge-right",
    "maneuver-ramp-left",
    "maneuver-ramp-right",
    "maneuver-ramp-straight",
    "maneuver-roundabout-enter",
    "maneuver-roundabout-enter-exit",
    "maneuver-roundabout-exit",
    "maneuver-route",
    "maneuver-start",
    "maneuver-turn-left",
    "maneuver-turn-right",
    "maneuver-turn-sharp-left",
    "maneuver-turn-sharp-right",
    "maneuver-turn-slight-left",
    "maneuver-turn-slight-right",
    "maneuver-uturn",
    "maneuver-uturn-left",
    "maneuver-uturn-right",
] as const;

describe("maneuver language resolution", () => {
    it("uses explicit, setting, header and default locales in order", () => {
        expect(resolveManeuverLanguage({ explicit: "de" })).toEqual({
            language: "de",
            languageFallback: false,
        });
        expect(resolveManeuverLanguage({ settingsLanguage: "fr", acceptLanguage: "de" }))
            .toEqual({ language: "fr", languageFallback: false });
        expect(resolveManeuverLanguage({ acceptLanguage: "zz;q=1, de;q=0.8" }))
            .toEqual({ language: "de", languageFallback: false });
        expect(resolveManeuverLanguage({ acceptLanguage: "fr;q=0, de;q=0.8" }))
            .toEqual({ language: "de", languageFallback: false });
        expect(resolveManeuverLanguage({})).toEqual({
            language: "en",
            languageFallback: false,
        });
    });

    it("emits language_fallback only for a differing explicit locale", () => {
        expect(resolveManeuverLanguage({ explicit: "DE" }).languageFallback).toBe(false);
        expect(resolveManeuverLanguage({ explicit: "de-CH" })).toEqual({
            language: "de",
            languageFallback: true,
        });
        expect(resolveManeuverLanguage({ explicit: "not-a-locale" })).toEqual({
            language: "en",
            languageFallback: true,
        });
        expect(resolveManeuverLanguage({ settingsLanguage: "not-a-locale" }).languageFallback)
            .toBe(false);
    });
});

describe("maneuver response formatting", () => {
    it("has translated templates for every supported locale", async () => {
        const english = await LOCALE_LOADERS.en();
        for (const language of SUPPORTED_LOCALES) {
            const messages = await LOCALE_LOADERS[language]();
            for (const key of maneuverCatalogKeys) {
                expect(messages[key], `${language}:${key}`).toEqual(expect.any(String));
                expect(String(messages[key]).length, `${language}:${key}`).toBeGreaterThan(0);
                if (language !== "en") {
                    expect(messages[key], `${language}:${key}`).not.toBe(english[key]);
                }
            }
        }
    });

    it("formats every maneuver type in every supported locale", async () => {
        for (const language of SUPPORTED_LOCALES) {
            const internal = responseWithTypes(maneuverTypes);
            const result = await formatManeuverResponse(internal, {
                language,
                languageFallback: false,
            });
            expect(result.language).toBe(language);
            expect(result.maneuvers).toHaveLength(maneuverTypes.length);
            for (const maneuver of result.maneuvers) {
                expect(maneuver.instruction.length).toBeGreaterThan(0);
                expect(maneuver).not.toHaveProperty("providerInstruction");
            }
        }
    });

    it("uses provider prose only for unknown and records the fallback", async () => {
        const internal = responseWithTypes(["unknown"]);
        internal.maneuvers[0].providerInstruction = "Provider text";
        const result = await formatManeuverResponse(internal, {
            language: "en",
            languageFallback: false,
        });
        expect(result.maneuvers[0].instruction).toBe("Provider text");
        expect(result.maneuvers[0].warnings).toContain("provider_instruction_fallback");
        expect(result.maneuvers[0]).not.toHaveProperty("providerInstruction");
    });

    it("uses a localized generic instruction when unknown prose is absent", async () => {
        const internal = responseWithTypes(["unknown"]);
        delete internal.maneuvers[0].providerInstruction;
        const result = await formatManeuverResponse(internal, {
            language: "de",
            languageFallback: false,
        });
        expect(result.maneuvers[0].instruction).toContain("Weiter");
        expect(result.maneuvers[0].warnings ?? []).not.toContain(
            "provider_instruction_fallback",
        );
    });

    it("includes the exit number for roundabout exit maneuvers", async () => {
        const internal = responseWithTypes(["roundabout_exit"]);
        internal.maneuvers[0].roundaboutExit = 3;
        const result = await formatManeuverResponse(internal, {
            language: "en",
            languageFallback: false,
        });
        expect(result.maneuvers[0].instruction).toContain("3");
    });

    it("omits an empty warnings list from the public maneuver", async () => {
        const result = await formatManeuverResponse(responseWithTypes(["continue"]), {
            language: "en",
            languageFallback: false,
        });
        expect(result.maneuvers[0]).not.toHaveProperty("warnings");
    });

    it("truncates Unicode safely and removes the provider-only field", async () => {
        const internal = responseWithTypes(["unknown"]);
        internal.maneuvers[0].providerInstruction = "🧭".repeat(600);
        const result = await formatManeuverResponse(internal, {
            language: "en",
            languageFallback: true,
        });
        expect(Array.from(result.maneuvers[0].instruction)).toHaveLength(500);
        expect(result.maneuvers[0].instruction.endsWith("…")).toBe(true);
        expect(result.maneuvers[0].warnings).toEqual([
            "provider_instruction_fallback",
            "instruction_truncated",
        ]);
        expect(result.warnings).toContain("language_fallback");
        expect(result.maneuvers[0]).not.toHaveProperty("providerInstruction");
    });
});

function responseWithTypes(types: RoutingManeuverType[]): RoutingInternalManeuverResponse {
    return {
        geometry: {
            format: "encoded_polyline",
            precision: 6,
            coordinates: "_c`|@_oyo@_pR_pR",
        },
        maneuvers: types.map((type, index) => ({
            type,
            providerInstruction: "ignored provider prose",
            distanceMeters: index === types.length - 1 ? 0 : 10,
            beginShapeIndex: index,
            endShapeIndex: index + 1,
            streetNames: ["Example Way"],
            ...(type === "roundabout_enter" ? { roundaboutExit: 2 } : {}),
        })),
    };
}
