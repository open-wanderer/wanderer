import { describe, expect, it } from "vitest";
import type { PluginInstance } from "#lib/models/plugin_instance.ts";
import { isPluginInstanceConnected } from "./plugin_instance_status";

describe("isPluginInstanceConnected", () => {
    it.each(["configured", "syncing"] as const)("keeps %s connected", (status) => {
        expect(isPluginInstanceConnected({ status })).toBe(true);
    });

    it("keeps an incomplete sync connected for toggles and settings saves", () => {
        expect(isPluginInstanceConnected({
            status: "error",
            last_error: { code: "sync_incomplete" },
        })).toBe(true);
    });

    it("does not infer a connection from unknown or human-readable errors", () => {
        expect(isPluginInstanceConnected()).toBe(false);
        expect(isPluginInstanceConnected({ status: "error" })).toBe(false);
        expect(isPluginInstanceConnected({
            status: "error",
            last_error: { code: "future_error", message: "sync_incomplete" },
        })).toBe(false);
        expect(isPluginInstanceConnected({
            status: "error",
            last_error: { code: " sync_incomplete " },
        })).toBe(false);
    });

    it.each([
        "needs_auth", "needs_reauth", "rate_limited", "unavailable",
        "unsupported_protocol", "disabled",
    ] as PluginInstance["status"][])("does not override %s even with an old partial error", (status) => {
        expect(isPluginInstanceConnected({
            status,
            last_error: { code: "sync_incomplete" },
        })).toBe(false);
    });
});
