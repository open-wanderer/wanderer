import { describe, expect, it } from "vitest";
import de from "$lib/i18n/locales/de.json";
import en from "$lib/i18n/locales/en.json";
import { pluginSetupErrorKey } from "./plugin_error_i18n";

describe("pluginSetupErrorKey", () => {
    it.each([
        ["manifest_missing", "plugin-setup-error-manifest-missing"],
        ["manifest_unreadable", "plugin-setup-error-manifest-unreadable"],
        ["manifest_invalid", "plugin-setup-error-manifest-invalid"],
        [
            "runtime_entrypoint_invalid",
            "plugin-setup-error-runtime-entrypoint-invalid",
        ],
        [
            "runtime_entrypoint_missing",
            "plugin-setup-error-runtime-entrypoint-missing",
        ],
        [
            "runtime_entrypoint_unreadable",
            "plugin-setup-error-runtime-entrypoint-unreadable",
        ],
        ["setup_failed", "plugin-setup-error-details"],
    ])("maps %s to %s", (code, key) => {
        expect(pluginSetupErrorKey(code)).toBe(key);
    });

    it("falls back safely for missing, malformed, unknown, and inherited names", () => {
        expect(pluginSetupErrorKey(undefined)).toBe("plugin-setup-error-details");
        expect(pluginSetupErrorKey(42)).toBe("plugin-setup-error-details");
        expect(pluginSetupErrorKey("future_private_error")).toBe(
            "plugin-setup-error-details",
        );
        expect(pluginSetupErrorKey("toString")).toBe("plugin-setup-error-details");
        expect(pluginSetupErrorKey("constructor")).toBe("plugin-setup-error-details");
        expect(pluginSetupErrorKey("__proto__")).toBe("plugin-setup-error-details");
    });

    it("maps every public code to an existing English and German message", () => {
        for (const code of [
            "manifest_missing",
            "manifest_unreadable",
            "manifest_invalid",
            "runtime_entrypoint_invalid",
            "runtime_entrypoint_missing",
            "runtime_entrypoint_unreadable",
            "setup_failed",
        ]) {
            const key = pluginSetupErrorKey(code);
            expect(Object.hasOwn(en, key), `missing English translation ${key}`).toBe(true);
            expect(Object.hasOwn(de, key), `missing German translation ${key}`).toBe(true);
        }
    });
});
