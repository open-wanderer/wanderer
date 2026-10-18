import { describe, expect, it } from "vitest";
import { TrailCreateSchema, TrailUpdateSchema } from "./trail_schema";

// Regression: `moving_duration` is advertised by three OpenAPI response
// schemas and carried on the `Trail` model, but was declared by neither Zod
// request schema. A Zod object strips unknown keys by default, so a client
// following the published contract had the field silently discarded with no
// error - the documented interface and the enforced interface disagreed. Only
// the multipart `/trail/form` route (which bypasses Zod via
// `uploadCreate`/`uploadUpdate`) happened to work, which is why the Flutter
// app never noticed.
describe("Trail request schemas - moving_duration", () => {
    const baseCreate = {
        name: "Test trail",
        public: true,
        completed: false,
        author: "abcdefghijklmno",
    };

    it("TrailCreateSchema keeps moving_duration instead of stripping it", () => {
        const parsed = TrailCreateSchema.parse({ ...baseCreate, moving_duration: 1234 });
        expect(parsed.moving_duration).toBe(1234);
    });

    it("TrailUpdateSchema keeps moving_duration instead of stripping it", () => {
        const parsed = TrailUpdateSchema.parse({ name: "Test trail", moving_duration: 1234 });
        expect(parsed.moving_duration).toBe(1234);
    });

    it("coerces a numeric string, matching the sibling duration field", () => {
        const parsed = TrailUpdateSchema.parse({ name: "Test trail", moving_duration: "600" });
        expect(parsed.moving_duration).toBe(600);
    });

    it("stays optional - omitting it is still valid", () => {
        expect(TrailCreateSchema.parse(baseCreate).moving_duration).toBeUndefined();
        expect(TrailUpdateSchema.parse({ name: "Test trail" }).moving_duration).toBeUndefined();
    });

    it("rejects a negative value", () => {
        expect(() => TrailUpdateSchema.parse({ name: "Test trail", moving_duration: -1 })).toThrow();
    });
});
