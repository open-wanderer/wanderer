import { describe, expect, it } from "vitest";
import { externalHttpUrl } from "./plugin_store";

describe("externalHttpUrl", () => {
    it("accepts absolute HTTP(S) URLs", () => {
        expect(externalHttpUrl("https://example.com/donate")).toBe(
            "https://example.com/donate",
        );
        expect(externalHttpUrl("http://example.com/support")).toBe(
            "http://example.com/support",
        );
    });

    it("rejects URLs without a scheme", () => {
        expect(externalHttpUrl("example.com/donate")).toBeUndefined();
    });

    it("rejects unsafe schemes and non-string values", () => {
        expect(externalHttpUrl("javascript:alert(1)")).toBeUndefined();
        expect(externalHttpUrl({ url: "https://example.com" })).toBeUndefined();
    });
});
