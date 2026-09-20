import { describe, expect, it } from "vitest";
import { SettingsCreateSchema } from "./settings_schema";

describe("upload duplicate check settings", () => {
    it.each([
        {},
        { uploadDuplicateCheck: null },
        { uploadDuplicateCheck: {} },
        { uploadDuplicateCheck: { includePublic: false, includeShared: false } },
        { uploadDuplicateCheck: { includePublic: true } },
        { uploadDuplicateCheck: { includeShared: true } },
        { uploadDuplicateCheck: { includePublic: true, includeShared: true } },
    ])("preserves the optional scope in %j", (input) => {
        expect(SettingsCreateSchema.parse(input)).toEqual(input);
    });

    it.each(["includePublic", "includeShared"])("requires actual booleans for %s", (field) => {
        for (const value of ["true", "false", 1, 0, null, [], {}]) {
            const result = SettingsCreateSchema.safeParse({
                uploadDuplicateCheck: { [field]: value },
            });
            expect(result.success, `${field}=${JSON.stringify(value)}`).toBe(false);
        }
    });

    it.each([false, true, "true", [], 1])("rejects a non-object scope: %j", (value) => {
        expect(SettingsCreateSchema.safeParse({ uploadDuplicateCheck: value }).success).toBe(false);
    });

    it("keeps map behavior independent from upload preferences", () => {
        const input = {
            behavior: { allowAutoGeolocate: true, mapClusteringMaxZoom: 9 },
            uploadDuplicateCheck: { includeShared: true },
        };
        expect(SettingsCreateSchema.parse(input)).toEqual(input);
    });
});
