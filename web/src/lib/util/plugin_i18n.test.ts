import type { PluginProvider } from "$lib/models/plugin_provider";
import { describe, expect, it } from "vitest";
import { pluginInformation } from "./plugin_i18n";

function plugin(overrides: Partial<PluginProvider> = {}): PluginProvider {
    return {
        id: "example",
        type: "trails",
        name: "Example",
        description: "Short fallback",
        version: "1.2.3",
        auth: { type: "none" },
        status: "available",
        ...overrides,
    };
}

describe("pluginInformation", () => {
    it("selects the localized information text", () => {
        expect(
            pluginInformation(
                plugin({ information: { de: "Langer Infotext", en: "Long information" } }),
                "de-CH",
            ),
        ).toBe("Langer Infotext");
    });

    it("prefers the localized card description over English information", () => {
        expect(
            pluginInformation(
                plugin({
                    descriptions: { en: "Short description", ru: "Краткое описание" },
                    information: { de: "Langer Infotext", en: "Long information" },
                }),
                "ru",
            ),
        ).toBe("Краткое описание");
    });

    it("falls back to English information when no description is available", () => {
        expect(
            pluginInformation(
                plugin({ description: "", information: { en: "Long information" } }),
                "ru",
            ),
        ).toBe("Long information");
    });
});
