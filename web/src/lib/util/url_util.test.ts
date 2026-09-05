import { describe, expect, it } from "vitest";
import { withShareToken } from "./url_util";

describe("withShareToken", () => {
    it("preserves the share token without forwarding unrelated state", () => {
        const searchParams = new URLSearchParams({
            share: "token+/=",
            t: "2",
        });

        expect(
            withShareToken("/map/trail/@alice/trail-id", searchParams),
        ).toBe("/map/trail/@alice/trail-id?share=token%2B%2F%3D");
    });

    it("leaves paths without a share token unchanged", () => {
        expect(
            withShareToken(
                "/map/trail/@alice/trail-id",
                new URLSearchParams({ t: "2" }),
            ),
        ).toBe("/map/trail/@alice/trail-id");
    });
});
