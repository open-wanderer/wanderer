import type { PluginProvider } from "$lib/models/plugin_provider";
import { describe, expect, it } from "vitest";
import { pluginInformation, providerCategoryLabel } from "./plugin_i18n";

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
                plugin({
                    descriptions: { "de-CH": "Kurze regionale Beschreibung" },
                    information: { de: "Langer Infotext", en: "Long information" },
                }),
                "de_CH",
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

    it("prefers English information over an English description for an unsupported locale", () => {
        expect(
            pluginInformation(
                plugin({
                    descriptions: { en: "Short description" },
                    information: { en: "Long information" },
                }),
                "fr",
            ),
        ).toBe("Long information");
    });

    it("uses the English and base descriptions as final fallbacks", () => {
        expect(
            pluginInformation(
                plugin({ descriptions: { en: "English description" } }),
                "fr",
            ),
        ).toBe("English description");
        expect(
            pluginInformation(plugin({ description: "  Short fallback  " }), "fr"),
        ).toBe("Short fallback");
        expect(pluginInformation(plugin({ description: "   " }), "fr")).toBe("");
    });

    it("skips empty localized values before applying fallbacks", () => {
        expect(
            pluginInformation(
                plugin({
                    descriptions: { "de-CH": "   ", de: "Kurze Beschreibung" },
                    information: { "DE_ch": "   ", en: "Long information" },
                }),
                "de_CH",
            ),
        ).toBe("Kurze Beschreibung");
    });
});

describe("providerCategoryLabel", () => {
    it("ignores non-string labels from an unvalidated manifest", () => {
        const provider = plugin({
            metadata: {
                providerCategories: {
                    hiking: { labels: { en: 42, de: "Wandern" } },
                },
            },
        });

        expect(providerCategoryLabel(provider, "hiking", "en")).toBe("hiking");
        expect(providerCategoryLabel(provider, "hiking", "de")).toBe("Wandern");
    });
});
