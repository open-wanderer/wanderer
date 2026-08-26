import { describe, expect, it } from "vitest";
import { externalHttpUrl, plugins_index } from "./plugin_store";

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

describe("plugins_index", () => {
    it("maps and sanitizes optional plugin metadata", async () => {
        const plugin: Record<string, any> = {
            id: "example",
            type: "trails",
            name: "Example",
            version: "1.0.0",
            runtime: "wasm",
            capabilities: [],
            status: "available",
            setupErrorCode: "manifest_invalid",
            manifest: {
                manifestVersion: "1.0",
                id: "example",
                type: "trails",
                name: "Example",
                version: "1.0.0",
                runtime: { type: "wasm", entrypoint: "plugin.wasm" },
                capabilities: [],
                metadata: {
                    homepageUrl: "https://example.com/plugin",
                    information: 42,
                },
            },
        };

        const items = await plugins_index(async () =>
            new Response(JSON.stringify({ items: [plugin] }), {
                status: 200,
                headers: { "content-type": "application/json" },
            }),
        );

        expect(items[0].homepageUrl).toBe("https://example.com/plugin");
        expect(items[0].information).toBeUndefined();
        expect(items[0].setupErrorCode).toBe("manifest_invalid");

        plugin.manifest.metadata.homepageUrl = "javascript:alert(1)";
        plugin.manifest.metadata.information = {
            en: "Long information",
            de: 42,
        };
        plugin.setupErrorCode = 42;
        const unsafeItems = await plugins_index(async () =>
            new Response(JSON.stringify({ items: [plugin] }), {
                status: 200,
                headers: { "content-type": "application/json" },
            }),
        );

        expect(unsafeItems[0].homepageUrl).toBeUndefined();
        expect(unsafeItems[0].information).toEqual({ en: "Long information" });
        expect(unsafeItems[0].setupErrorCode).toBeUndefined();
    });
});
