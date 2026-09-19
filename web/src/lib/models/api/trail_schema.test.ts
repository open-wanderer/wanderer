import { describe, expect, it } from "vitest";
import { TrailCreateSchema, TrailUpdateSchema } from "./trail_schema";

describe.each([
    ["create", TrailCreateSchema.pick({ difficulty: true })],
    ["update", TrailUpdateSchema.pick({ difficulty: true })],
] as const)("trail %s difficulty", (_operation, schema) => {
    it("leaves omitted difficulty unset", () => {
        expect(schema.parse({})).toEqual({});
    });

    it("preserves an explicit empty value for unknown or cleared difficulty", () => {
        expect(schema.parse({ difficulty: "" })).toEqual({ difficulty: "" });
    });

    it.each(["easy", "moderate", "difficult"])(
        "preserves the known difficulty %s",
        (difficulty) => {
            expect(schema.parse({ difficulty })).toEqual({ difficulty });
        },
    );

    it.each([null, 0, "unknown", "hard"])(
        "rejects invalid difficulty %j",
        (difficulty) => {
            expect(schema.safeParse({ difficulty }).success).toBe(false);
        },
    );
});
