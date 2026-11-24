import { addMessages, init, locale } from "svelte-i18n";
import { beforeAll, expect, it } from "vitest";
import en from "#lib/i18n/locales/en.json";
import de from "#lib/i18n/locales/de.json";
import { translateTrailMergeError } from "./trail_merge_i18n";

beforeAll(() => {
    addMessages("en", en);
    addMessages("de", de);
    init({ initialLocale: "en", fallbackLocale: "en" });
});

it.each(["en", "de"])("translates formatted trail merge errors in %s", (language) => {
    locale.set(language);
    const messages = language === "en" ? en : de;
    expect(translateTrailMergeError(" Trail_merge_target_not_found. ")).toBe(messages.trail_merge_target_not_found);
    expect(translateTrailMergeError("TRAIL_MERGE_SOURCE_NOT_FOUND!")).toBe(messages.trail_merge_source_not_found);
    expect(translateTrailMergeError("Provider error.")).toBe("Provider error.");
});
