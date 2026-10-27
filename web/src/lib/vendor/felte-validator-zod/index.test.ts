import { describe, expect, it } from "vitest";
import { z } from "zod";
import { validateSchema } from "./index";

describe("validateSchema", () => {
    const schema = z.object({
        name: z.string().min(3, "too-short"),
        tags: z.array(z.object({ label: z.string().min(1, "required") })),
        password: z.string(),
        passwordConfirm: z.string(),
    }).refine((data) => data.password === data.passwordConfirm, {
        message: "passwords-must-match",
        path: ["passwordConfirm"],
    });

    it("returns undefined for valid values", async () => {
        const validate = validateSchema(schema);
        expect(await validate({ name: "abc", tags: [], password: "x", passwordConfirm: "x" })).toBeUndefined();
    });

    it("shapes issues into nested felte errors", async () => {
        const validate = validateSchema(schema);
        expect(await validate({ name: "a", tags: [{ label: "" }], password: "x", passwordConfirm: "y" })).toEqual({
            name: ["too-short"],
            tags: [{ label: ["required"] }],
            passwordConfirm: ["passwords-must-match"],
        });
    });
});
