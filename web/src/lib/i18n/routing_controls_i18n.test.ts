import { describe, expect, it } from "vitest";

const localeModules = import.meta.glob<{ default: Record<string, unknown> }>(
    "./locales/*.json",
    { eager: true },
);

const standardControlKeys = [
    "speedPreference",
    "hillPreference",
    "maxHikingDifficulty",
    "roadPreference",
    "avoidBadSurfaces",
    "vehicleWidth",
    "vehicleHeight",
];

describe("routing control translations", () => {
    it("keeps every standard control in every supported locale", () => {
        for (const [path, module] of Object.entries(localeModules)) {
            for (const key of standardControlKeys) {
                expect(
                    module.default[`routing-control-${key}`],
                    `${path} is missing routing-control-${key}`,
                ).toEqual(expect.any(String));
            }
        }
    });

    it("keeps the dynamically selected preference buckets in the fallback locale", () => {
        const fallback = localeModules["./locales/en.json"].default;
        for (const family of ["hills", "roads", "surfaces"]) {
            for (const bucket of ["none", "low", "medium", "high"]) {
                expect(
                    fallback[`routing-preference-${family}-${bucket}`],
                    `English fallback is missing routing-preference-${family}-${bucket}`,
                ).toEqual(expect.any(String));
            }
        }
    });
});
